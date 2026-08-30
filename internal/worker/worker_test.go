package worker

import (
	"context"
	"database/sql"
	"fmt"
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

	ctx := context.Background()

	// Create two independent jobs.
	jobID1 := createTestJob(t, db, "dataset")
	jobID2 := createTestJob(t, db, "dataset")

	// Put both jobs into the Redis stream.
	if err := jobs.EnqueueJob(
		redisClient,
		jobID1,
	); err != nil {
		t.Fatalf(
			"failed to enqueue job %d: %v",
			jobID1,
			err,
		)
	}

	if err := jobs.EnqueueJob(
		redisClient,
		jobID2,
	); err != nil {
		t.Fatalf(
			"failed to enqueue job %d: %v",
			jobID2,
			err,
		)
	}

	started := make(chan int64, 2)
	finished := make(chan int64, 2)

	// Both processors block here until the test knows that both workers
	// have received a job. This lets us prove that processing overlaps.
	release := make(chan struct{})

	processJob := func(jobID int64) error {
		started <- jobID

		<-release

		finished <- jobID
		return nil
	}

	// Start two workers with different Redis consumer names.
	go RunWorker(
		db,
		redisClient,
		processJob,
		"test-worker-1",
	)

	go RunWorker(
		db,
		redisClient,
		processJob,
		"test-worker-2",
	)

	// Wait until both workers have received a job.
	received := make(map[int64]bool)

	timeout := time.After(5 * time.Second)

	for len(received) < 2 {
		select {
		case jobID := <-started:
			received[jobID] = true

		case <-timeout:
			t.Fatalf(
				"timed out waiting for two workers to receive jobs; received %v",
				received,
			)
		}
	}

	if !received[jobID1] {
		t.Fatalf(
			"job %d was not received",
			jobID1,
		)
	}

	if !received[jobID2] {
		t.Fatalf(
			"job %d was not received",
			jobID2,
		)
	}

	// Release both processors simultaneously.
	close(release)

	finishedJobs := make(map[int64]bool)

	timeout = time.After(5 * time.Second)

	for len(finishedJobs) < 2 {
		select {
		case jobID := <-finished:
			finishedJobs[jobID] = true

		case <-timeout:
			t.Fatalf(
				"timed out waiting for both jobs to finish; finished %v",
				finishedJobs,
			)
		}
	}

	// Give the workers a moment to acknowledge their messages.
	time.Sleep(100 * time.Millisecond)

	pending, err := redisClient.XPending(
		ctx,
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
			"expected all concurrent jobs to be acknowledged, got %d pending messages",
			pending.Count,
		)
	}
}
