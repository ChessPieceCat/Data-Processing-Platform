package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/redis/go-redis/v9"
)

const (
	benchmarkJobCount    = 30
	benchmarkProcessTime = 100 * time.Millisecond
	benchmarkMaxWait     = 30 * time.Second
)

// BenchmarkWorkerThroughput measures worker throughput and job latency
// with one, two, and three concurrent workers.
//
// Run with:
//
//	go test ./internal/worker \
//		-bench BenchmarkWorkerThroughput \
//		-benchtime=1x \
//		-run '^$'
//
// Reported metrics:
//
//   - jobs/sec
//   - avg-queue-latency-ms
//   - p95-queue-latency-ms
//   - avg-processing-latency-ms
//   - p95-processing-latency-ms
//   - avg-total-latency-ms
//   - p95-total-latency-ms
//
// A fixed processing delay is used so the benchmark measures worker
// concurrency and queue behavior rather than Python processor performance.
func BenchmarkWorkerThroughput(b *testing.B) {
	for _, workerCount := range []int{1, 2, 3} {
		b.Run(
			workersLabel(workerCount),
			func(b *testing.B) {
				for iteration := 0; iteration < b.N; iteration++ {
					runWorkerThroughputTrial(
						b,
						workerCount,
					)
				}
			},
		)
	}
}

// runWorkerThroughputTrial executes one complete benchmark workload.
func runWorkerThroughputTrial(
	b *testing.B,
	workerCount int,
) {
	b.Helper()

	db, redisClient := setupWorkerBenchmarkEnvironment(b)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	var workerWG sync.WaitGroup
	workerWG.Add(workerCount)

	// This channel confirms that every worker has started before
	// the workload measurement begins.
	workersReady := make(
		chan struct{},
		workerCount,
	)

	// Record when each job enters the Redis queue.
	submissionTimes := sync.Map{}

	// Record latency measurements.
	queueLatencies := make(
		chan time.Duration,
		benchmarkJobCount,
	)

	processingLatencies := make(
		chan time.Duration,
		benchmarkJobCount,
	)

	totalLatencies := make(
		chan time.Duration,
		benchmarkJobCount,
	)

	var completed atomic.Int64

	// The benchmark processor simulates a fixed amount of work.
	//
	// It records:
	//   queue latency     = enqueue -> processing start
	//   processing latency = processing start -> processing finish
	//   total latency      = enqueue -> processing finish
	processJob := func(jobID int64) error {
		submittedAtValue, ok := submissionTimes.Load(jobID)
		if !ok {
			return errors.New(
				"missing submission timestamp",
			)
		}

		submittedAt, ok := submittedAtValue.(time.Time)
		if !ok {
			return errors.New(
				"invalid submission timestamp",
			)
		}

		processingStartedAt := time.Now()

		queueLatencies <- processingStartedAt.Sub(
			submittedAt,
		)

		time.Sleep(
			benchmarkProcessTime,
		)

		completedAt := time.Now()

		processingLatencies <- completedAt.Sub(
			processingStartedAt,
		)

		totalLatencies <- completedAt.Sub(
			submittedAt,
		)

		completed.Add(1)

		return nil
	}

	// Prepare the complete workload before starting the workers.

	for jobIndex := 0; jobIndex < benchmarkJobCount; jobIndex++ {
		jobID := createBenchmarkJob(
			b,
			db,
		)

		submissionTimes.Store(
			jobID,
			time.Now(),
		)

		if err := jobs.EnqueueJob(
			redisClient,
			jobID,
		); err != nil {
			cancel()
			workerWG.Wait()

			b.Fatalf(
				"failed to enqueue benchmark job %d: %v",
				jobID,
				err,
			)
		}
	}

	// Start the requested number of workers after the queue is populated.
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		consumerName := workerConsumerName(
			workerIndex,
		)

		go func() {
			defer workerWG.Done()

			workersReady <- struct{}{}
			m := createTestMetrics()
			RunWorker(
				m,
				ctx,
				db,
				redisClient,
				processJob,
				consumerName,
			)
		}()
	}

	// Wait until every worker has started.
	for i := 0; i < workerCount; i++ {
		select {
		case <-workersReady:

		case <-time.After(5 * time.Second):
			cancel()
			workerWG.Wait()

			b.Fatalf(
				"timed out waiting for %d workers to start",
				workerCount,
			)
		}
	}

	// Start measuring once all workers are ready to drain the populated queue.
	workloadStart := time.Now()

	// Wait for every job to complete.
	deadline := time.NewTimer(
		benchmarkMaxWait,
	)
	defer deadline.Stop()

	for completed.Load() < benchmarkJobCount {
		select {
		case <-deadline.C:
			cancel()
			workerWG.Wait()

			b.Fatalf(
				"timed out waiting for benchmark workload: completed %d/%d jobs",
				completed.Load(),
				benchmarkJobCount,
			)

		case <-time.After(5 * time.Millisecond):
		}
	}

	elapsed := time.Since(
		workloadStart,
	)

	// Read the latency measurements.
	queueLatencyValues := readDurations(
		queueLatencies,
		benchmarkJobCount,
	)

	processingLatencyValues := readDurations(
		processingLatencies,
		benchmarkJobCount,
	)

	totalLatencyValues := readDurations(
		totalLatencies,
		benchmarkJobCount,
	)

	averageQueueLatency := calculateAverageDuration(
		queueLatencyValues,
	)

	p95QueueLatency := calculatePercentileDuration(
		queueLatencyValues,
		0.95,
	)

	averageProcessingLatency := calculateAverageDuration(
		processingLatencyValues,
	)

	p95ProcessingLatency := calculatePercentileDuration(
		processingLatencyValues,
		0.95,
	)

	averageTotalLatency := calculateAverageDuration(
		totalLatencyValues,
	)

	p95TotalLatency := calculatePercentileDuration(
		totalLatencyValues,
		0.95,
	)

	throughput := float64(
		benchmarkJobCount,
	) / elapsed.Seconds()

	b.ReportMetric(
		throughput,
		"jobs/sec",
	)

	b.ReportMetric(
		averageQueueLatency.Seconds()*1000,
		"avg-queue-latency-ms",
	)

	b.ReportMetric(
		p95QueueLatency.Seconds()*1000,
		"p95-queue-latency-ms",
	)

	b.ReportMetric(
		averageProcessingLatency.Seconds()*1000,
		"avg-processing-latency-ms",
	)

	b.ReportMetric(
		p95ProcessingLatency.Seconds()*1000,
		"p95-processing-latency-ms",
	)

	b.ReportMetric(
		averageTotalLatency.Seconds()*1000,
		"avg-total-latency-ms",
	)

	b.ReportMetric(
		p95TotalLatency.Seconds()*1000,
		"p95-total-latency-ms",
	)

	b.Logf(
		"workers=%d jobs=%d duration=%s throughput=%.2f jobs/sec queue_avg=%s queue_p95=%s processing_avg=%s processing_p95=%s total_avg=%s total_p95=%s",
		workerCount,
		benchmarkJobCount,
		elapsed,
		throughput,
		averageQueueLatency,
		p95QueueLatency,
		averageProcessingLatency,
		p95ProcessingLatency,
		averageTotalLatency,
		p95TotalLatency,
	)

	// Stop workers before benchmark cleanup closes Redis.
	cancel()
	workerWG.Wait()
}

// setupWorkerBenchmarkEnvironment creates an isolated PostgreSQL/Redis
// environment for the benchmark.
func setupWorkerBenchmarkEnvironment(
	b *testing.B,
) (*sql.DB, *redis.Client) {
	b.Helper()

	m := database.RunMigrations()

	if m == nil {
		b.Fatal(
			"RunMigrations returned nil",
		)
	}

	b.Cleanup(func() {
		database.CloseMigrations(m)
	})

	db := database.OpenDatabase()

	if db == nil {
		b.Fatal(
			"OpenDatabase returned nil",
		)
	}

	if err := db.Ping(); err != nil {
		db.Close()

		b.Skipf(
			"PostgreSQL is not available: %v",
			err,
		)
	}

	redisClient := redis.NewClient(
		&redis.Options{
			Addr: "localhost:6379",
			DB:   1,
		},
	)

	if err := redisClient.Ping(
		context.Background(),
	).Err(); err != nil {
		redisClient.Close()
		db.Close()

		b.Skipf(
			"Redis is not available: %v",
			err,
		)
	}

	// Start with a clean Redis benchmark database.
	if err := redisClient.FlushDB(
		context.Background(),
	).Err(); err != nil {
		redisClient.Close()
		db.Close()

		b.Fatalf(
			"failed to clear Redis benchmark database: %v",
			err,
		)
	}

	// Create the stream and consumer group.
	if err := redisClient.XGroupCreateMkStream(
		context.Background(),
		"job_queue",
		"job_workers",
		"0",
	).Err(); err != nil {
		redisClient.Close()
		db.Close()

		b.Fatalf(
			"failed to initialize Redis consumer group: %v",
			err,
		)
	}

	// The benchmark creates its own jobs, so remove jobs left by
	// previous tests.
	if _, err := db.Exec(
		`DELETE FROM jobs`,
	); err != nil {
		redisClient.Close()
		db.Close()

		b.Fatalf(
			"failed to clear jobs table: %v",
			err,
		)
	}

	b.Cleanup(func() {
		_, _ = db.Exec(
			`DELETE FROM jobs`,
		)

		_ = redisClient.FlushDB(
			context.Background(),
		)

		_ = redisClient.Close()
		_ = db.Close()
	})

	return db, redisClient
}

// createBenchmarkJob creates one queued job for the benchmark.
func createBenchmarkJob(
	b *testing.B,
	db *sql.DB,
) int64 {
	b.Helper()

	token := fmt.Sprintf("benchmark-session-%d", time.Now().UnixNano())

	var sessionID int64
	err := db.QueryRow(`
		INSERT INTO sessions (token, created_at, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`,
		token,
		time.Now(),
		time.Now().Add(24*time.Hour),
	).Scan(&sessionID)

	if err != nil {
		b.Fatalf(
			"failed to create benchmark session: %v",
			err,
		)
	}

	owner := jobs.JobOwner{
		SessionID: &sessionID,
	}

	jobID, err := jobs.CreateJob(
		db,
		"dataset",
		owner,
	)
	if err != nil {
		b.Fatalf(
			"failed to create benchmark job: %v",
			err,
		)
	}

	return jobID
}

// workerConsumerName returns a unique Redis consumer name.
func workerConsumerName(index int) string {
	return "benchmark-worker-" + strconv.Itoa(index)
}

// workersLabel returns the benchmark subtest name.
func workersLabel(workerCount int) string {
	return strconv.Itoa(workerCount) + "_workers"
}

// readDurations drains exactly count measurements from a channel.
func readDurations(
	values <-chan time.Duration,
	count int,
) []time.Duration {
	result := make(
		[]time.Duration,
		0,
		count,
	)

	for i := 0; i < count; i++ {
		result = append(
			result,
			<-values,
		)
	}

	return result
}

// calculateAverageDuration calculates the average duration.
func calculateAverageDuration(
	values []time.Duration,
) time.Duration {
	if len(values) == 0 {
		return 0
	}

	var total time.Duration

	for _, value := range values {
		total += value
	}

	return total / time.Duration(
		len(values),
	)
}

// calculatePercentileDuration calculates a percentile duration.
func calculatePercentileDuration(
	values []time.Duration,
	percentile float64,
) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sorted := append(
		[]time.Duration(nil),
		values...,
	)

	sort.Slice(
		sorted,
		func(i, j int) bool {
			return sorted[i] < sorted[j]
		},
	)

	index := int(
		float64(len(sorted)-1) * percentile,
	)

	return sorted[index]
}
