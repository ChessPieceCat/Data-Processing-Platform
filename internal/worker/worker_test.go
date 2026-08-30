package worker

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/redis/go-redis/v9"
)

func setupWorkerTest(
	t *testing.T,
) (*sql.DB, *redis.Client) {
	t.Helper()

	m := database.RunMigrations()
	if m == nil {
		t.Fatal("RunMigrations returned nil")
	}

	t.Cleanup(func() {
		database.CloseMigrations(m)
	})

	db := database.OpenDatabase()

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("PostgreSQL is not available: %v", err)
	}

	// Use a separate Redis database so worker tests do not interfere
	// with the application's normal Redis data.
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		redisClient.Close()
		db.Close()
		t.Skipf("Redis is not available: %v", err)
	}

	// Start each test with a clean Redis database.
	if err := redisClient.FlushDB(context.Background()).Err(); err != nil {
		redisClient.Close()
		db.Close()
		t.Fatalf("failed to clear Redis test database: %v", err)
	}

	// Create the consumer group and stream used by the worker.
	if err := redisClient.XGroupCreateMkStream(
		context.Background(),
		"job_queue",
		"job_workers",
		"0",
	).Err(); err != nil {
		redisClient.Close()
		db.Close()
		t.Fatalf("failed to initialize Redis consumer group: %v", err)
	}

	t.Cleanup(func() {
		_ = redisClient.FlushDB(context.Background())
		_ = redisClient.Close()
		_ = db.Close()
	})

	return db, redisClient
}

func createTestJob(t *testing.T, db *sql.DB, status string) int64 {
	t.Helper()

	jobID, err := jobs.CreateJob(db, "dataset")
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	if _, err := db.Exec(
		`UPDATE jobs SET status = $1 WHERE id = $2`,
		status,
		jobID,
	); err != nil {
		t.Fatalf("failed to set job status: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			`DELETE FROM jobs WHERE id = $1`,
			jobID,
		)
	})

	return jobID
}

func enqueueAndPendTestJob(
	t *testing.T,
	redisClient *redis.Client,
	jobID int64,
) string {
	t.Helper()

	ctx := context.Background()

	messageID, err := redisClient.XAdd(
		ctx,
		&redis.XAddArgs{
			Stream: "job_queue",
			ID:     "*",
			Values: map[string]interface{}{
				"job_id": jobID,
			},
		},
	).Result()

	if err != nil {
		t.Fatalf("failed to enqueue test job: %v", err)
	}

	// Deliver the new message to worker_1. Because the message was
	// delivered but not acknowledged, it is now pending.
	streams, err := redisClient.XReadGroup(
		ctx,
		&redis.XReadGroupArgs{
			Group:    "job_workers",
			Consumer: "worker_1",
			Streams:  []string{"job_queue", ">"},
			Count:    1,
		},
	).Result()

	if err != nil {
		t.Fatalf("failed to make test message pending: %v", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		t.Fatal("expected test message to be delivered to worker_1")
	}

	if streams[0].Messages[0].ID != messageID {
		t.Fatalf(
			"expected pending message %s, got %s",
			messageID,
			streams[0].Messages[0].ID,
		)
	}

	return messageID
}

func TestRecoverPendingCompletedJob(t *testing.T) {
	db, redisClient := setupWorkerTest(t)

	jobID := createTestJob(t, db, "completed")

	messageID := enqueueAndPendTestJob(
		t,
		redisClient,
		jobID,
	)

	processed := false

	recoverPendingJobs(
		db,
		redisClient,
		func(jobID int64) error {
			processed = true
			return nil
		},
		"recovery-worker",
		0,
	)

	if processed {
		t.Fatal("completed job should not be reprocessed")
	}

	pending, err := redisClient.XPending(
		context.Background(),
		"job_queue",
		"job_workers",
	).Result()
	if err != nil {
		t.Fatalf(
			"failed to inspect pending messages: %v",
			err,
		)
	}

	if pending.Count != 0 {
		t.Fatalf(
			"expected message %s to be acknowledged, but %d messages remain pending",
			messageID,
			pending.Count,
		)
	}
}

func TestRecoverPendingFailedJob(t *testing.T) {
	db, redisClient := setupWorkerTest(t)

	jobID := createTestJob(t, db, "failed")

	messageID := enqueueAndPendTestJob(
		t,
		redisClient,
		jobID,
	)

	processed := false

	recoverPendingJobs(
		db,
		redisClient,
		func(jobID int64) error {
			processed = true
			return nil
		},
		"recovery-worker",
		0,
	)

	if processed {
		t.Fatal("failed job should not be reprocessed")
	}

	pending, err := redisClient.XPending(
		context.Background(),
		"job_queue",
		"job_workers",
	).Result()
	if err != nil {
		t.Fatalf(
			"failed to inspect pending messages: %v",
			err,
		)
	}

	if pending.Count != 0 {
		t.Fatalf(
			"expected failed job message %s to be acknowledged, got %d pending messages",
			messageID,
			pending.Count,
		)
	}
}

func TestRecoverPendingQueuedJob(t *testing.T) {
	db, redisClient := setupWorkerTest(t)

	jobID := createTestJob(t, db, "queued")

	messageID := enqueueAndPendTestJob(
		t,
		redisClient,
		jobID,
	)

	var processedJobID int64

	recoverPendingJobs(
		db,
		redisClient,
		func(jobID int64) error {
			processedJobID = jobID

			// Simulate successful processing.
			_, err := db.Exec(
				`UPDATE jobs
				 SET status = 'completed'
				 WHERE id = $1`,
				jobID,
			)

			return err
		},
		"recovery-worker",
		0,
	)

	if processedJobID != jobID {
		t.Fatalf(
			"expected job %d to be reprocessed, got %d",
			jobID,
			processedJobID,
		)
	}

	pending, err := redisClient.XPending(
		context.Background(),
		"job_queue",
		"job_workers",
	).Result()
	if err != nil {
		t.Fatalf(
			"failed to inspect pending messages: %v",
			err,
		)
	}

	if pending.Count != 0 {
		t.Fatalf(
			"expected successfully reprocessed job %d (%s) to be acknowledged, got %d pending messages",
			jobID,
			messageID,
			pending.Count,
		)
	}
}

func TestRecoverPendingJobProcessingFailure(t *testing.T) {
	db, redisClient := setupWorkerTest(t)

	jobID := createTestJob(t, db, "processing")

	enqueueAndPendTestJob(
		t,
		redisClient,
		jobID,
	)

	processed := false

	recoverPendingJobs(
		db,
		redisClient,
		func(jobID int64) error {
			processed = true

			return fmt.Errorf(
				"simulated processing failure",
			)
		},
		"recovery-worker",
		0,
	)

	if !processed {
		t.Fatal(
			"expected non-terminal job to be reprocessed",
		)
	}

	pending, err := redisClient.XPending(
		context.Background(),
		"job_queue",
		"job_workers",
	).Result()
	if err != nil {
		t.Fatalf(
			"failed to inspect pending messages: %v",
			err,
		)
	}

	if pending.Count == 0 {
		t.Fatal(
			"expected failed reprocessing attempt to leave message pending",
		)
	}
}

func TestWorkersProcessJobsConcurrently(t *testing.T) {
	db, redisClient := setupWorkerTest(t)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	var workerWG sync.WaitGroup
	workerWG.Add(2)

	firstStarted := make(chan int64, 1)
	secondStarted := make(chan int64, 1)
	release := make(chan struct{})

	// Give each worker its own processing callback so the test can
	// identify which consumer received a job.
	processJob1 := func(jobID int64) error {
		firstStarted <- jobID
		<-release
		return nil
	}

	processJob2 := func(jobID int64) error {
		secondStarted <- jobID
		<-release
		return nil
	}

	// Start both workers before enqueueing any jobs.
	go func() {
		defer workerWG.Done()

		RunWorker(
			ctx,
			db,
			redisClient,
			processJob1,
			"test-worker-1",
		)
	}()

	go func() {
		defer workerWG.Done()

		RunWorker(
			ctx,
			db,
			redisClient,
			processJob2,
			"test-worker-2",
		)
	}()

	// Give both workers time to enter XReadGroup and begin waiting.
	time.Sleep(100 * time.Millisecond)

	// Create and enqueue the first job.
	jobID1 := createTestJob(
		t,
		db,
		"dataset",
	)

	if err := jobs.EnqueueJob(
		redisClient,
		jobID1,
	); err != nil {
		close(release)
		cancel()
		workerWG.Wait()

		t.Fatalf(
			"failed to enqueue job %d: %v",
			jobID1,
			err,
		)
	}

	// Wait for one of the workers to receive the first job.
	firstWorker := ""

	select {
	case receivedID := <-firstStarted:
		if receivedID != jobID1 {
			close(release)
			cancel()
			workerWG.Wait()

			t.Fatalf(
				"worker 1 received unexpected job %d; expected %d",
				receivedID,
				jobID1,
			)
		}

		firstWorker = "test-worker-1"

	case receivedID := <-secondStarted:
		if receivedID != jobID1 {
			close(release)
			cancel()
			workerWG.Wait()

			t.Fatalf(
				"worker 2 received unexpected job %d; expected %d",
				receivedID,
				jobID1,
			)
		}

		firstWorker = "test-worker-2"

	case <-time.After(5 * time.Second):
		close(release)
		cancel()
		workerWG.Wait()

		t.Fatal(
			"timed out waiting for first worker to receive a job",
		)
	}

	// The worker that received the first job is blocked in processJob.
	// Enqueueing a second job should therefore make the other worker
	// available to receive it.
	jobID2 := createTestJob(
		t,
		db,
		"dataset",
	)

	if err := jobs.EnqueueJob(
		redisClient,
		jobID2,
	); err != nil {
		close(release)
		cancel()
		workerWG.Wait()

		t.Fatalf(
			"failed to enqueue job %d: %v",
			jobID2,
			err,
		)
	}

	// Wait for the other worker to receive the second job.
	timeout := time.After(5 * time.Second)

	switch firstWorker {
	case "test-worker-1":
		select {
		case receivedID := <-firstStarted:
			close(release)
			cancel()
			workerWG.Wait()

			t.Fatalf(
				"worker 1 received second job %d; expected worker 2",
				receivedID,
			)

		case receivedID := <-secondStarted:
			if receivedID != jobID2 {
				close(release)
				cancel()
				workerWG.Wait()

				t.Fatalf(
					"worker 2 received unexpected job %d; expected %d",
					receivedID,
					jobID2,
				)
			}

		case <-timeout:
			close(release)
			cancel()
			workerWG.Wait()

			t.Fatal(
				"timed out waiting for second worker to receive a job",
			)
		}

	case "test-worker-2":
		select {
		case receivedID := <-secondStarted:
			close(release)
			cancel()
			workerWG.Wait()

			t.Fatalf(
				"worker 2 received second job %d; expected worker 1",
				receivedID,
			)

		case receivedID := <-firstStarted:
			if receivedID != jobID2 {
				close(release)
				cancel()
				workerWG.Wait()

				t.Fatalf(
					"worker 1 received unexpected job %d; expected %d",
					receivedID,
					jobID2,
				)
			}

		case <-timeout:
			close(release)
			cancel()
			workerWG.Wait()

			t.Fatal(
				"timed out waiting for second worker to receive a job",
			)
		}
	}

	// Both workers are now blocked in processJob, which means both jobs
	// are in flight concurrently.
	close(release)

	// Wait for both messages to be acknowledged.
	timeout = time.After(5 * time.Second)

	for {
		pending, err := redisClient.XPending(
			context.Background(),
			"job_queue",
			"job_workers",
		).Result()

		if err != nil {
			cancel()
			workerWG.Wait()

			t.Fatalf(
				"failed to inspect pending messages: %v",
				err,
			)
		}

		if pending.Count == 0 {
			break
		}

		select {
		case <-timeout:
			cancel()
			workerWG.Wait()

			t.Fatalf(
				"timed out waiting for jobs to be acknowledged; %d messages remain pending",
				pending.Count,
			)

		case <-time.After(10 * time.Millisecond):
		}
	}

	// Stop both workers before setupWorkerTest closes the Redis client.
	cancel()
	workerWG.Wait()
}
