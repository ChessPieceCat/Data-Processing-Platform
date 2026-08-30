package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/redis"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/worker"
)

func main() {
	// Open the application database connection.
	db := database.OpenDatabase()
	defer db.Close()

	// Verify that the database is reachable.
	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// Open the Redis connection.
	redisClient := redis.OpenRedis()
	defer redisClient.Close()

	if err := redis.PingRedis(redisClient); err != nil {
		log.Fatal(err)
	}

	// Initialize the Redis consumer group.
	if err := redis.InitializeGroup(redisClient); err != nil {
		log.Fatal(err)
	}

	// Define the job processing function.
	processJob := func(jobID int64) error {
		return jobs.ProcessJob(
			db,
			jobID,
		)
	}

	// Use the container hostname as the Redis consumer name.
	consumerName, err := os.Hostname()
	if err != nil {
		log.Fatal("Error getting hostname:", err)
	}

	// Create a context that is cancelled when the worker receives
	// an interrupt or termination signal.
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	// Recover jobs that were pending when the worker previously stopped.
	log.Printf(
		"Recovering pending jobs for consumer %s...",
		consumerName,
	)

	worker.RecoverPendingJobs(
		db,
		redisClient,
		processJob,
		consumerName,
	)

	// Start consuming new jobs.
	log.Printf(
		"Worker started as consumer %s",
		consumerName,
	)

	worker.RunWorker(
		ctx,
		db,
		redisClient,
		processJob,
		consumerName,
	)

	// RunWorker returns after the context is cancelled.
	log.Printf(
		"Worker %s shut down cleanly",
		consumerName,
	)
}
