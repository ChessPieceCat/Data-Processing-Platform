package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/redis/go-redis/v9"
)

const maxJobAttempts = 3
const pendingJobMinIdle = 30 * time.Second

// Connect to job_queue and wait for new jobs.
func RunWorker(ctx context.Context, db *sql.DB, redisClient *redis.Client, processJob func(jobID int64) error, consumerName string) {
	for {
		// Read from the Redis stream.
		streams, err := redisClient.XReadGroup(
			ctx,
			&redis.XReadGroupArgs{
				Group:    "job_workers",
				Consumer: consumerName,
				Streams:  []string{"job_queue", ">"},
				Count:    1,
				Block:    time.Second,
			},
		).Result()

		if err != nil {
			if ctx.Err() != nil {
				return
			}

			if err != redis.Nil {
				log.Printf(
					"Error reading from Redis stream: %v",
					err,
				)
			}

			continue
		}

		if ctx.Err() != nil {
			return
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
					if err := redisClient.XAck(ctx, "job_queue", "job_workers", message.ID).Err(); err != nil {
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
	consumerName string,
) {
	recoverPendingJobs(
		db,
		redisClient,
		processJob,
		consumerName,
		pendingJobMinIdle,
	)
}

func recoverPendingJobs(
	db *sql.DB,
	redisClient *redis.Client,
	processJob func(jobID int64) error,
	consumerName string,
	minIdle time.Duration,
) {
	ctx := context.Background()

	messages, _, err := redisClient.XAutoClaim(
		ctx,
		&redis.XAutoClaimArgs{
			Stream:   "job_queue",
			Group:    "job_workers",
			Consumer: consumerName,
			MinIdle:  minIdle,
			Start:    "0-0",
			Count:    100,
		},
	).Result()

	if err != nil {
		log.Printf(
			"Error claiming pending messages: %v",
			err,
		)
		return
	}

	for _, message := range messages {
		jobIDStr, ok := message.Values["job_id"].(string)
		if !ok {
			log.Printf(
				"Invalid job_id in pending message %s: %v",
				message.ID,
				message.Values,
			)
			continue
		}

		jobID, err := strconv.ParseInt(
			jobIDStr,
			10,
			64,
		)
		if err != nil {
			log.Printf(
				"Error converting job_id to int64 for pending message %s: %v",
				message.ID,
				err,
			)
			continue
		}

		job, err := jobs.GetJob(
			db,
			jobID,
		)
		if err != nil {
			log.Printf(
				"Error retrieving job %d: %v",
				jobID,
				err,
			)
			continue
		}

		if job.Status == "completed" ||
			job.Status == "failed" {

			if err := redisClient.XAck(
				ctx,
				"job_queue",
				"job_workers",
				message.ID,
			).Err(); err != nil {
				log.Printf(
					"Error acknowledging terminal job %d: %v",
					jobID,
					err,
				)
			}

			continue
		}

		if job.Attempts >= maxJobAttempts {
			errorMessage := fmt.Sprintf(
				"Job %d reached maximum attempts (%d)",
				jobID,
				maxJobAttempts,
			)

			if err := jobs.FailJob(
				db,
				jobID,
				errorMessage,
			); err != nil {
				log.Printf(
					"Error marking job %d as failed: %v",
					jobID,
					err,
				)
				continue
			}

			if err := redisClient.XAck(
				ctx,
				"job_queue",
				"job_workers",
				message.ID,
			).Err(); err != nil {
				log.Printf(
					"Error acknowledging job %d after marking it failed: %v",
					jobID,
					err,
				)
			}

			continue
		}

		if err := processJob(jobID); err != nil {
			log.Printf(
				"Error reprocessing job %d: %v",
				jobID,
				err,
			)

			continue
		}

		if err := redisClient.XAck(
			ctx,
			"job_queue",
			"job_workers",
			message.ID,
		).Err(); err != nil {
			log.Printf(
				"Error acknowledging reprocessed job %d: %v",
				jobID,
				err,
			)
		}
	}
}
