package main

import (
	"log"

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

	// Recover jobs that were pending when the worker previously stopped.
	log.Println("Recovering pending jobs...")

	worker.RecoverPendingJobs(
		db,
		redisClient,
		processJob,
	)

	// Start consuming new jobs.
	log.Println("Worker started")

	worker.RunWorker(
		db,
		redisClient,
		processJob,
	)
}
