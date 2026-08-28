package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/redis/go-redis/v9"
)

const maxJobAttempts = 3

// Connect to job_queue and wait for new jobs.
func RunWorker(db *sql.DB, redisClient *redis.Client, processJob func(jobID int64) error) {
	for {
		// Read from the Redis stream.
		streams, err := redisClient.XReadGroup(
			context.Background(),
			&redis.XReadGroupArgs{
				Group:    "job_workers",
				Consumer: "worker_1",
				Streams:  []string{"job_queue", ">"},
				Block:    0, // Block indefinitely until a new message arrives.
			},
		).Result()

		if err != nil {
			fmt.Println("Error reading from Redis stream:", err)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				jobIDStr, ok := message.Values["job_id"].(string)
				if !ok {
					fmt.Println("Invalid job_id in message:", message.Values)
					continue
				}

				jobID, err := strconv.ParseInt(jobIDStr, 10, 64)

				if err != nil {
					fmt.Println("Error converting job_id to int64:", err)
					continue
				}

				ack := true
				if err := processJob(jobID); err != nil {
					log.Printf(
						"Error processing job %d: %v",
						jobID,
						err,
					)
					// Check if the job reached a terminal failed state in PostgreSQL.
					job, err := jobs.GetJob(db, jobID)
					if err != nil {
						ack = false
					} else if job.Status != "failed" {
						// If the job is not in a terminal state, we do not acknowledge it, allowing it to be retried later.
						ack = false
					}
				}

				if ack {
					if err := redisClient.XAck(context.Background(), "job_queue", "job_workers", message.ID).Err(); err != nil {
						fmt.Println("Error acknowledging message:", err)
					}
				}

			}
		}
	}
}

// RecoverPendingJobs checks for pending jobs in the Redis stream and
// reprocesses jobs that have not reached a terminal state.
func RecoverPendingJobs(
	db *sql.DB,
	redisClient *redis.Client,
	processJob func(jobID int64) error,
) {
	// Get pending messages for the consumer group.
	pendingMessages, err := redisClient.XPendingExt(
		context.Background(),
		&redis.XPendingExtArgs{
			Stream: "job_queue",
			Group:  "job_workers",
			Start:  "-",
			End:    "+",
			Count:  100,
		},
	).Result()

	if err != nil {
		log.Println("Error fetching pending messages:", err)
		return
	}

	for _, pending := range pendingMessages {

		// Get the Redis ID of the pending message and retrieve the
		// job ID from the message.
		messages, err := redisClient.XRange(
			context.Background(),
			"job_queue",
			pending.ID,
			pending.ID,
		).Result()

		if err != nil {
			log.Printf(
				"Error retrieving message for pending ID %s: %v",
				pending.ID,
				err,
			)
			continue
		}

		if len(messages) == 0 {
			log.Printf(
				"No message found for pending ID %s",
				pending.ID,
			)
			continue
		}

		jobIDStr, ok := messages[0].Values["job_id"].(string)
		if !ok {
			log.Printf(
				"Invalid job_id in message for pending ID %s: %v",
				pending.ID,
				messages[0].Values,
			)
			continue
		}

		jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
		if err != nil {
			log.Printf(
				"Error converting job_id to int64 for pending ID %s: %v",
				pending.ID,
				err,
			)
			continue
		}

		// Check the current PostgreSQL state of the job.
		job, err := jobs.GetJob(db, jobID)
		if err != nil {
			log.Printf(
				"Error retrieving job %d: %v",
				jobID,
				err,
			)
			continue
		}

		// Completed and failed jobs are already in terminal states,
		// so acknowledge their Redis messages without reprocessing them.
		if job.Status == "completed" || job.Status == "failed" {
			if err := redisClient.XAck(
				context.Background(),
				"job_queue",
				"job_workers",
				pending.ID,
			).Err(); err != nil {
				log.Printf(
					"Error acknowledging terminal job %d: %v",
					jobID,
					err,
				)
			}

			continue
		}

		// Do not reprocess jobs that have reached the maximum number of attempts.
		if job.Attempts >= maxJobAttempts {
			errorMessage := fmt.Sprintf("Job %d reached maximum attempts (%d)", jobID, maxJobAttempts)
			if err := jobs.FailJob(db, jobID, errorMessage); err != nil {
				log.Printf(
					"Error marking job %d as failed: %v",
					jobID,
					err,
				)
				continue
			}

			if err := redisClient.XAck(
				context.Background(),
				"job_queue",
				"job_workers",
				pending.ID,
			).Err(); err != nil {
				log.Printf(
					"Error acknowledging job %d after marking as failed: %v",
					jobID,
					err,
				)
			}

			continue
		}

		// Reprocess jobs that have not reached a terminal state.
		if err := processJob(jobID); err != nil {
			log.Printf(
				"Error reprocessing job %d: %v",
				jobID,
				err,
			)
			continue
		}

		// Acknowledge the message after successful reprocessing.
		if err := redisClient.XAck(
			context.Background(),
			"job_queue",
			"job_workers",
			pending.ID,
		).Err(); err != nil {
			log.Printf(
				"Error acknowledging message for job %d: %v",
				jobID,
				err,
			)
		}
	}
}
