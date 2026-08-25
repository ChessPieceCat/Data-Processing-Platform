package jobs

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

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

// ProcessDatasetJob manages the lifecycle of a dataset processing job.
func ProcessDatasetJob(
	db *sql.DB,
	jobID int64,
	filePath string,
	configPath string,
) error {
	// Mark the job as processing.
	if err := startJob(db, jobID); err != nil {
		return err
	}

	// Run the Python dataset processor.
	resultPath, err := runDatasetProcessor(filePath, jobID, configPath)
	if err != nil {
		if failErr := failJob(db, jobID, err.Error()); failErr != nil {
			log.Printf("Error marking job %d as failed: %v", jobID, failErr)
		}

		return err
	}

	// Save the result reference.
	if err := saveResultReference(db, jobID, resultPath); err != nil {
		if failErr := failJob(db, jobID, err.Error()); failErr != nil {
			log.Printf("Error marking job %d as failed: %v", jobID, failErr)
		}

		return err
	}

	// Mark the job as complete.
	if err := completeJob(db, jobID); err != nil {
		return err
	}

	return nil
}

// findDatasetProcessor finds the Python data processor relative to the project root
func findDatasetProcessor() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	directory := workingDirectory

	for {
		processorPath := filepath.Join(
			directory,
			"processors",
			"dataset",
			"main.py",
		)

		if _, err := os.Stat(processorPath); err == nil {
			return processorPath, nil
		}

		parent := filepath.Dir(directory)

		if parent == directory {
			return "", fmt.Errorf(
				"could not find processors/dataset/main.py from %s",
				workingDirectory,
			)
		}

		directory = parent
	}
}

// runDatasetProcessor runs the Python dataset processor.
func runDatasetProcessor(
	filePath string,
	jobID int64,
	configPath string,
) (string, error) {

	resultPath := fmt.Sprintf(
		"uploads/%d/results.json",
		jobID,
	)

	processorPath, err := findDatasetProcessor()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(
		"python3",
		processorPath,
		filePath,
		resultPath,
		configPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"dataset processor failed: %w: %s",
			err,
			string(output),
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

// startJob marks a job as processing.
func startJob(db *sql.DB, jobID int64) error {
	_, err := db.Exec(
		`UPDATE jobs
		 SET status = $1,
		     started_at = $2
		 WHERE id = $3`,
		"processing",
		time.Now(),
		jobID,
	)

	if err != nil {
		return err
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

// failJob marks a job as failed and records the error.
func failJob(db *sql.DB, jobID int64, errorMessage string) error {
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
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	ErrorMessage    *string
	ResultReference *string
}

func GetJobs(db *sql.DB) ([]Job, error) {
	rows, err := db.Query(
		`SELECT id, type, status, created_at, started_at, completed_at, error_message
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
		`SELECT id, type, status, created_at, started_at, completed_at,
	        error_message, result_reference
		 FROM jobs
		 WHERE id = $1`,
		jobID,
	).Scan(
		&j.ID,
		&j.Type,
		&j.Status,
		&j.CreatedAt,
		&j.StartedAt,
		&j.CompletedAt,
		&j.ErrorMessage,
		&j.ResultReference,
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
func DeleteJob(db *sql.DB, jobID int64) error {
	_, err := db.Exec(
		`DELETE FROM jobs WHERE id = $1`,
		jobID,
	)

	if err != nil {
		return err
	}

	// Delete the associated directory for the job.
	jobDir := fmt.Sprintf("uploads/%d", jobID)
	if err := os.RemoveAll(jobDir); err != nil {
		return fmt.Errorf("failed to delete job directory %s: %w", jobDir, err)
	}

	log.Printf("Job %d deleted", jobID)
	return nil
}

func DeleteOldJobs(db *sql.DB, keep int) error {
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
		if err := DeleteJob(db, jobID); err != nil {
			return err
		}
	}

	return nil
}
