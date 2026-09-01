package jobs

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/metrics"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"
	"github.com/redis/go-redis/v9"
)

const MaxOutstandingJobs = 100

var ErrJobQueueFull = errors.New("job queue is full")

type errorCategory string

const (
	errorTimeout       errorCategory = "timeout"
	errorCancelled     errorCategory = "cancelled"
	errorProcessor     errorCategory = "processor"
	errorValidation    errorCategory = "validation"
	errorStorage       errorCategory = "storage"
	errorFilesystem    errorCategory = "filesystem"
	errorDatabase      errorCategory = "database"
	errorConfiguration errorCategory = "configuration"
	errorRetryLimit    errorCategory = "retry_limit"
)

type categorizedError struct {
	category errorCategory
	err      error
}

func (e *categorizedError) Error() string {
	return e.err.Error()
}

func (e *categorizedError) Unwrap() error {
	return e.err
}

func categorizeError(
	category errorCategory,
	err error,
) error {
	return &categorizedError{
		category: category,
		err:      err,
	}
}

func CreateJobWithLimit(
	db *sql.DB,
	jobType string,
	m *metrics.Metrics,
) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	defer tx.Rollback()

	// Serialize job admission checks so concurrent submissions
	// cannot both pass the limit check.
	if _, err := tx.Exec(
		`SELECT pg_advisory_xact_lock($1)`,
		int64(1001),
	); err != nil {
		return 0, err
	}

	var outstandingJobs int

	if err := tx.QueryRow(
		`SELECT COUNT(*)
		 FROM jobs
		 WHERE status IN ('queued', 'processing')`,
	).Scan(&outstandingJobs); err != nil {
		return 0, err
	}

	if outstandingJobs >= MaxOutstandingJobs {
		return 0, ErrJobQueueFull
	}

	var jobID int64

	if err := tx.QueryRow(
		`INSERT INTO jobs (type, status, created_at)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		jobType,
		"queued",
		time.Now(),
	).Scan(&jobID); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	m.JobsSubmitted.WithLabelValues(jobType).Inc()

	log.Printf(
		"Created job %d of type %s",
		jobID,
		jobType,
	)

	return jobID, nil
}

// CreateJob creates a new job and returns its database-generated ID.
func CreateJob(db *sql.DB, jobType string) (int64, error) {
	var id int64

	err := db.QueryRow(
		`INSERT INTO jobs (type, status, created_at)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		jobType,
		"queued",
		time.Now(),
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	log.Printf("Created job %d of type %s", id, jobType)
	return id, nil
}

// ProcessJob manages the lifecycle of a job.
func ProcessJob(
	db *sql.DB,
	jobID int64,
	store storage.Storage,
	m *metrics.Metrics,
) error {
	job, err := GetJob(
		db,
		jobID,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to retrieve job %d: %w",
			jobID,
			err,
		)
	}

	if job.InputReference == nil {
		return fmt.Errorf(
			"job %d has no input reference",
			jobID,
		)
	}

	// Mark the job as processing.
	start := time.Now()

	defer func() {
		m.JobProcessingDuration.
			WithLabelValues(job.Type).
			Observe(time.Since(start).Seconds())
	}()

	if err := startJob(
		db,
		jobID,
	); err != nil {
		return err
	}

	if err := UpdateQueueDepth(db, m); err != nil {
		log.Printf("Error updating queue depth: %v", err)
	}

	workspace, err := prepareJobWorkspace(
		context.Background(),
		store,
		job,
	)
	if err != nil {
		return failJob(
			db,
			jobID,
			job.Type,
			err,
			m,
		)
	}

	defer os.RemoveAll(
		workspace.Dir,
	)

	localResultPath := filepath.Join(
		workspace.Dir,
		"results.json",
	)

	if _, err := runProcessor(
		job.Type,
		workspace.InputPath,
		localResultPath,
		workspace.ConfigPath,
	); err != nil {
		return failJob(
			db,
			jobID,
			job.Type,
			err,
			m,
		)
	}

	if err := persistJobArtifacts(
		context.Background(),
		db,
		store,
		job,
		workspace,
		localResultPath,
	); err != nil {
		return failJob(
			db,
			jobID,
			job.Type,
			err,
			m,
		)
	}

	if err := completeJob(
		db,
		jobID,
	); err != nil {
		return err
	}

	if err := UpdateQueueDepth(db, m); err != nil {
		log.Printf("Error updating queue depth: %v", err)
	}

	m.JobsCompleted.WithLabelValues(job.Type).Inc()

	return nil
}

func persistJobArtifacts(
	ctx context.Context,
	db *sql.DB,
	store storage.Storage,
	job *Job,
	workspace *JobWorkspace,
	localResultPath string,
) error {
	resultData, err := os.ReadFile(localResultPath)
	if err != nil {
		return categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to read processor result: %w",
				err,
			),
		)
	}

	var result map[string]interface{}

	if err := json.Unmarshal(
		resultData,
		&result,
	); err != nil {
		return categorizeError(
			errorProcessor,
			fmt.Errorf(
				"failed to decode processor result: %w",
				err,
			),
		)
	}

	switch job.Type {
	case "image":
		if err := persistImageArtifacts(
			ctx,
			store,
			job.ID,
			workspace,
			result,
		); err != nil {
			return err
		}

	case "dataset":
		if err := persistDatasetArtifacts(
			ctx,
			store,
			job.ID,
			workspace,
			result,
		); err != nil {
			return err
		}
	}

	updatedResult, err := json.MarshalIndent(
		result,
		"",
		"  ",
	)
	if err != nil {
		return categorizeError(
			errorProcessor,
			fmt.Errorf(
				"failed to encode updated processor result: %w",
				err,
			),
		)
	}

	if err := os.WriteFile(
		localResultPath,
		updatedResult,
		0644,
	); err != nil {
		return categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to write updated processor result: %w",
				err,
			),
		)
	}

	resultFile, err := os.Open(localResultPath)
	if err != nil {
		return categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to reopen processor result: %w",
				err,
			),
		)
	}
	defer resultFile.Close()

	resultKey := fmt.Sprintf(
		"jobs/%d/results.json",
		job.ID,
	)

	if err := store.Put(
		ctx,
		resultKey,
		resultFile,
	); err != nil {
		return categorizeError(
			errorStorage,
			fmt.Errorf(
				"failed to store job result: %w",
				err,
			),
		)
	}

	if err := saveResultReference(
		db,
		job.ID,
		resultKey,
	); err != nil {
		return categorizeError(
			errorDatabase,
			fmt.Errorf(
				"failed to save result reference: %w",
				err,
			),
		)
	}

	return nil
}

func uploadLocalFile(
	ctx context.Context,
	store storage.Storage,
	localPath string,
	key string,
) error {
	file, err := os.Open(localPath)
	if err != nil {
		return categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to open local artifact: %w",
				err,
			),
		)
	}
	defer file.Close()

	if err := store.Put(
		ctx,
		key,
		file,
	); err != nil {
		return categorizeError(
			errorStorage,
			fmt.Errorf(
				"failed to upload artifact: %w",
				err,
			),
		)
	}

	return nil
}

func persistImageArtifacts(
	ctx context.Context,
	store storage.Storage,
	jobID int64,
	workspace *JobWorkspace,
	result map[string]interface{},
) error {
	if processedPath, ok := result["processed_path"].(string); ok {
		key := fmt.Sprintf(
			"jobs/%d/%s",
			jobID,
			filepath.Base(processedPath),
		)

		if err := uploadLocalFile(
			ctx,
			store,
			processedPath,
			key,
		); err != nil {
			return fmt.Errorf(
				"failed to store processed image: %w",
				err,
			)
		}

		result["processed_path"] = key
	}

	if metadataPath, ok := result["metadata_reference"].(string); ok &&
		metadataPath != "" {
		key := fmt.Sprintf(
			"jobs/%d/metadata.json",
			jobID,
		)

		if err := uploadLocalFile(
			ctx,
			store,
			metadataPath,
			key,
		); err != nil {
			return fmt.Errorf(
				"failed to store image metadata: %w",
				err,
			)
		}

		result["metadata_reference"] = key
	}

	// The original image is already stored persistently.
	result["original_path"] = fmt.Sprintf(
		"jobs/%d/%s",
		jobID,
		filepath.Base(workspace.InputPath),
	)

	return nil
}

func persistDatasetArtifacts(
	ctx context.Context,
	store storage.Storage,
	jobID int64,
	workspace *JobWorkspace,
	result map[string]interface{},
) error {
	visualizations, ok := result["visualizations"].(map[string]interface{})
	if ok {
		for name, value := range visualizations {
			path, ok := value.(string)
			if !ok || path == "" {
				continue
			}

			key := fmt.Sprintf(
				"jobs/%d/%s",
				jobID,
				filepath.Base(path),
			)

			if err := uploadLocalFile(
				ctx,
				store,
				path,
				key,
			); err != nil {
				return fmt.Errorf(
					"failed to store visualization %q: %w",
					name,
					err,
				)
			}

			visualizations[name] = key
		}
	}

	return persistModelResults(
		ctx,
		store,
		jobID,
		workspace,
	)
}

func persistModelResults(
	ctx context.Context,
	store storage.Storage,
	jobID int64,
	workspace *JobWorkspace,
) error {
	path := filepath.Join(
		workspace.Dir,
		"results_model_results.json",
	)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to access model results: %w",
				err,
			),
		)
	}

	key := fmt.Sprintf(
		"jobs/%d/results_model_results.json",
		jobID,
	)

	if err := uploadLocalFile(
		ctx,
		store,
		path,
		key,
	); err != nil {
		return fmt.Errorf(
			"failed to store model results: %w",
			err,
		)
	}

	return nil
}

// failJob records a processing failure and returns the original error.
func failJob(
	db *sql.DB,
	jobID int64,
	jobType string,
	err error,
	m *metrics.Metrics,
) error {
	if failErr := FailJob(
		db,
		jobID,
		err.Error(),
	); failErr != nil {
		log.Printf(
			"Error marking job %d as failed: %v",
			jobID,
			failErr,
		)
	} else {
		if err := UpdateQueueDepth(db, m); err != nil {
			log.Printf(
				"Error updating queue depth: %v",
				err,
			)
		}

		m.JobsFailed.
			WithLabelValues(
				jobType,
				classifyError(err),
			).
			Inc()
	}

	return err
}

func classifyError(err error) string {
	var categorized *categorizedError

	if errors.As(err, &categorized) {
		return string(categorized.category)
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	default:
		return "unknown"
	}
}

// findProcessor searches for the processor script for a given job type.
func findProcessor(processorType string) (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to determine working directory: %w",
				err,
			),
		)
	}

	directory := workingDirectory

	for {
		processorPath := filepath.Join(
			directory,
			"processors",
			processorType,
			"main.py",
		)

		if _, err := os.Stat(processorPath); err == nil {
			return processorPath, nil
		} else if !os.IsNotExist(err) {
			return "", categorizeError(
				errorConfiguration,
				fmt.Errorf(
					"failed to access processor for type %q: %w",
					processorType,
					err,
				),
			)
		}

		parent := filepath.Dir(directory)

		if parent == directory {
			return "", categorizeError(
				errorConfiguration,
				fmt.Errorf(
					"could not find processor for type %q from %s",
					processorType,
					workingDirectory,
				),
			)
		}

		directory = parent
	}
}

// runProcessor executes the specified processor script with the given arguments.
func runProcessor(
	processorType string,
	filePath string,
	resultPath string,
	configPath string,
) (string, error) {
	processorPath, err := findProcessor(processorType)
	if err != nil {
		return "", err
	}

	args := []string{
		processorPath,
		filePath,
	}

	if processorType == "route" {
		distancePath := filepath.Join(
			filepath.Dir(filePath),
			"distances.csv",
		)

		if _, err := os.Stat(distancePath); err != nil {
			return "", categorizeError(
				errorValidation,
				fmt.Errorf(
					"route distance file not found: %w",
					err,
				),
			)
		}

		args = append(
			args,
			distancePath,
		)
	}

	args = append(
		args,
		resultPath,
		configPath,
	)

	pythonPath := os.Getenv("PYTHON_BIN")
	if pythonPath == "" {
		pythonPath = "python3"
	}

	const processorTimeout = 60 * time.Second

	ctx, cancel := context.WithTimeout(
		context.Background(),
		processorTimeout,
	)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		pythonPath,
		args...,
	)

	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf(
			"%s processor timed out after 60 seconds: %w",
			processorType,
			context.DeadlineExceeded,
		)
	}

	if err != nil {
		return "", categorizeError(
			errorProcessor,
			fmt.Errorf(
				"%s processor failed: %w: %s",
				processorType,
				err,
				string(output),
			),
		)
	}

	return resultPath, nil
}

// saveResultReference stores the result file reference for a job.
func saveResultReference(db *sql.DB, jobID int64, resultPath string) error {
	_, err := db.Exec(
		`UPDATE jobs
		 SET result_reference = $1
		 WHERE id = $2`,
		resultPath,
		jobID,
	)

	return err
}

// startJob marks a job as processing and increments its attempt count.
func startJob(db *sql.DB, jobID int64) error {
	result, err := db.Exec(
		`UPDATE jobs
		 SET status = $1,
		     started_at = $2,
			 attempts = attempts + 1
		 WHERE id = $3
		 	AND status = $4`,
		"processing",
		time.Now(),
		jobID,
		"queued",
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job %d is not in a queued state", jobID)
	}

	log.Printf("Job %d started", jobID)

	return nil
}

// completeJob marks a job as completed.
func completeJob(db *sql.DB, jobID int64) error {
	_, err := db.Exec(
		`UPDATE jobs
		 SET status = $1,
		     completed_at = $2
		 WHERE id = $3`,
		"completed",
		time.Now(),
		jobID,
	)

	if err != nil {
		return err
	}

	log.Printf("Job %d completed", jobID)
	return nil
}

// FailJob marks a job as failed and records the error.
func FailJob(db *sql.DB, jobID int64, errorMessage string) error {
	_, err := db.Exec(
		`UPDATE jobs
		 SET status = $1,
		     error_message = $2
		 WHERE id = $3`,
		"failed",
		errorMessage,
		jobID,
	)

	if err != nil {
		return err
	}

	log.Printf("Job %d failed: %s", jobID, errorMessage)
	return nil
}

type Job struct {
	ID              int64
	Type            string
	Status          string
	Attempts        int
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	ErrorMessage    *string
	ResultReference *string
	InputReference  *string
}

func GetJobs(db *sql.DB) ([]Job, error) {
	rows, err := db.Query(
		`SELECT id, type, status, attempts, created_at, started_at, completed_at, error_message
		 FROM jobs
		 ORDER BY created_at DESC`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var jobs []Job

	for rows.Next() {
		var j Job

		if err := rows.Scan(
			&j.ID,
			&j.Type,
			&j.Status,
			&j.Attempts,
			&j.CreatedAt,
			&j.StartedAt,
			&j.CompletedAt,
			&j.ErrorMessage,
		); err != nil {
			return nil, err
		}

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetJob retrieves a single job by its ID.
func GetJob(db *sql.DB, jobID int64) (*Job, error) {
	var j Job

	err := db.QueryRow(
		`SELECT id, type, status, attempts, created_at, started_at, completed_at,
	        error_message, result_reference, input_reference
		 FROM jobs
		 WHERE id = $1`,
		jobID,
	).Scan(
		&j.ID,
		&j.Type,
		&j.Status,
		&j.Attempts,
		&j.CreatedAt,
		&j.StartedAt,
		&j.CompletedAt,
		&j.ErrorMessage,
		&j.ResultReference,
		&j.InputReference,
	)

	if err != nil {
		return nil, err
	}

	return &j, nil
}

// ValidateCSV verifies that the file can be parsed as CSV.
func ValidateCSV(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	reader := csv.NewReader(file)

	// Read through the entire file so malformed CSV anywhere
	// in the file is detected.
	for {
		_, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// GetCSVColumns reads the first row of a CSV file and returns the column names.
func GetCSVColumns(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	return header, nil
}

// Delete a job and its associated directory.
func DeleteJob(
	db *sql.DB,
	jobID int64,
	store storage.Storage,
) error {
	if err := store.DeletePrefix(
		context.Background(),
		fmt.Sprintf(
			"jobs/%d/",
			jobID,
		),
	); err != nil {
		return fmt.Errorf(
			"failed to delete job objects: %w",
			err,
		)
	}

	if _, err := db.Exec(
		`DELETE FROM jobs WHERE id = $1`,
		jobID,
	); err != nil {
		return err
	}

	log.Printf(
		"Job %d deleted",
		jobID,
	)

	return nil
}

func DeleteOldJobs(db *sql.DB, keep int, store storage.Storage) error {
	rows, err := db.Query(`
        SELECT id
        FROM jobs
        ORDER BY created_at DESC
        OFFSET $1
    `, keep)
	if err != nil {
		return err
	}
	defer rows.Close()

	var oldJobIDs []int64

	for rows.Next() {
		var jobID int64

		if err := rows.Scan(&jobID); err != nil {
			return err
		}

		oldJobIDs = append(oldJobIDs, jobID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for _, jobID := range oldJobIDs {
		if err := DeleteJob(db, jobID, store); err != nil {
			return err
		}
	}

	return nil
}

// EnqueueJob adds a job ID to the Redis queue for processing.
func EnqueueJob(redisClient *redis.Client, jobID int64) error {
	_, err := redisClient.XAdd(
		context.Background(),
		&redis.XAddArgs{
			Stream: "job_queue",
			ID:     "*",
			Values: map[string]interface{}{
				"job_id": jobID,
			},
		},
	).Result()

	if err != nil {
		return fmt.Errorf("failed to enqueue job %d: %w", jobID, err)
	}

	return nil
}

func prepareJobWorkspace(
	ctx context.Context,
	store storage.Storage,
	job *Job,
) (*JobWorkspace, error) {
	if job.InputReference == nil {
		return nil, fmt.Errorf(
			"job %d has no input reference",
			job.ID,
		)
	}

	workspace, err := os.MkdirTemp(
		"",
		fmt.Sprintf("job-%d-", job.ID),
	)
	if err != nil {
		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to create job workspace: %w",
				err,
			),
		)
	}

	cleanup := func() {
		_ = os.RemoveAll(workspace)
	}

	inputReader, err := store.Get(
		ctx,
		*job.InputReference,
	)
	if err != nil {
		cleanup()

		return nil, categorizeError(
			errorStorage,
			fmt.Errorf(
				"failed to retrieve job input: %w",
				err,
			),
		)
	}
	defer inputReader.Close()

	inputPath := filepath.Join(
		workspace,
		filepath.Base(*job.InputReference),
	)

	inputFile, err := os.Create(inputPath)
	if err != nil {
		cleanup()

		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to create local input file: %w",
				err,
			),
		)
	}

	if _, err := io.Copy(
		inputFile,
		inputReader,
	); err != nil {
		inputFile.Close()
		cleanup()

		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to download job input: %w",
				err,
			),
		)
	}

	if err := inputFile.Close(); err != nil {
		cleanup()

		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to close local input file: %w",
				err,
			),
		)
	}

	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		job.ID,
	)

	configReader, err := store.Get(
		ctx,
		configKey,
	)
	if err != nil {
		cleanup()

		return nil, categorizeError(
			errorStorage,
			fmt.Errorf(
				"failed to retrieve job config: %w",
				err,
			),
		)
	}
	defer configReader.Close()

	configPath := filepath.Join(
		workspace,
		"config.json",
	)

	configFile, err := os.Create(configPath)
	if err != nil {
		cleanup()

		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to create local config file: %w",
				err,
			),
		)
	}

	if _, err := io.Copy(
		configFile,
		configReader,
	); err != nil {
		configFile.Close()
		cleanup()

		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to download job config: %w",
				err,
			),
		)
	}

	if err := configFile.Close(); err != nil {
		cleanup()

		return nil, categorizeError(
			errorFilesystem,
			fmt.Errorf(
				"failed to close local config file: %w",
				err,
			),
		)
	}

	workspaceInfo := &JobWorkspace{
		Dir:        workspace,
		InputPath:  inputPath,
		ConfigPath: configPath,
	}

	if job.Type == "route" {
		distanceKey := fmt.Sprintf(
			"jobs/%d/distances.csv",
			job.ID,
		)

		distanceReader, err := store.Get(
			ctx,
			distanceKey,
		)
		if err != nil {
			cleanup()

			return nil, categorizeError(
				errorStorage,
				fmt.Errorf(
					"failed to retrieve route distance table: %w",
					err,
				),
			)
		}
		defer distanceReader.Close()

		distancePath := filepath.Join(
			workspace,
			"distances.csv",
		)

		distanceFile, err := os.Create(distancePath)
		if err != nil {
			cleanup()

			return nil, categorizeError(
				errorFilesystem,
				fmt.Errorf(
					"failed to create local distance table: %w",
					err,
				),
			)
		}

		if _, err := io.Copy(
			distanceFile,
			distanceReader,
		); err != nil {
			distanceFile.Close()
			cleanup()

			return nil, categorizeError(
				errorFilesystem,
				fmt.Errorf(
					"failed to download route distance table: %w",
					err,
				),
			)
		}

		if err := distanceFile.Close(); err != nil {
			cleanup()

			return nil, categorizeError(
				errorFilesystem,
				fmt.Errorf(
					"failed to close local distance table: %w",
					err,
				),
			)
		}

		workspaceInfo.DistancePath = distancePath
	}

	return workspaceInfo, nil
}

type JobWorkspace struct {
	Dir          string
	InputPath    string
	DistancePath string
	ConfigPath   string
}

func GetQueueDepth(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`
        SELECT type, COUNT(*)
        FROM jobs
        WHERE status = 'queued'
        GROUP BY type
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	depths := make(map[string]int)

	for rows.Next() {
		var jobType string
		var count int

		if err := rows.Scan(&jobType, &count); err != nil {
			return nil, err
		}

		depths[jobType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return depths, nil
}

func UpdateQueueDepth(
	db *sql.DB,
	m *metrics.Metrics,
) error {
	depths, err := GetQueueDepth(db)
	if err != nil {
		return err
	}

	jobTypes := []string{
		"dataset",
		"image",
		"route",
	}

	for _, jobType := range jobTypes {
		m.QueueDepth.
			WithLabelValues(jobType).
			Set(float64(depths[jobType]))
	}

	return nil
}
