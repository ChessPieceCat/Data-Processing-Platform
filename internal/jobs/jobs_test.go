package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	redisclient "github.com/redis/go-redis/v9"
)

// setupTestDatabase runs migrations and returns a database connection
// for tests that require PostgreSQL.
func setupTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	m := database.RunMigrations()
	if m == nil {
		t.Fatal("RunMigrations returned nil")
	}

	t.Cleanup(func() {
		database.CloseMigrations(m)
	})

	db := database.OpenDatabase()

	if db == nil {
		t.Fatal("OpenDatabase returned nil")
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("database ping failed: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// createTestJob creates a job and registers cleanup for the database row.
func createTestJob(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	jobID, err := CreateJob(db, "dataset")
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM jobs WHERE id = $1",
			jobID,
		)
	})

	return jobID
}

// TestCreateJob verifies that a new job is created with the expected
// initial values.
func TestCreateJob(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)

	if jobID <= 0 {
		t.Fatalf("expected positive job ID, got %d", jobID)
	}

	job, err := GetJob(db, jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.ID != jobID {
		t.Fatalf("expected ID %d, got %d", jobID, job.ID)
	}

	if job.Type != "dataset" {
		t.Fatalf("expected type dataset, got %q", job.Type)
	}

	if job.Status != "queued" {
		t.Fatalf("expected status queued, got %q", job.Status)
	}

	if job.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if job.StartedAt != nil {
		t.Fatal("expected StartedAt to be nil")
	}

	if job.CompletedAt != nil {
		t.Fatal("expected CompletedAt to be nil")
	}
}

// TestGetJobNotFound verifies that requesting a nonexistent job returns
// sql.ErrNoRows.
func TestGetJobNotFound(t *testing.T) {
	db := setupTestDatabase(t)

	_, err := GetJob(db, 999999999)

	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}

	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

// TestGetJobs verifies that jobs are returned in newest-first order.
func TestGetJobs(t *testing.T) {
	db := setupTestDatabase(t)

	firstID := createTestJob(t, db)

	time.Sleep(time.Millisecond)

	secondID := createTestJob(t, db)

	jobList, err := GetJobs(db)
	if err != nil {
		t.Fatalf("GetJobs failed: %v", err)
	}

	if len(jobList) < 2 {
		t.Fatalf("expected at least 2 jobs, got %d", len(jobList))
	}

	if jobList[0].ID != secondID {
		t.Fatalf(
			"expected newest job to be %d, got %d",
			secondID,
			jobList[0].ID,
		)
	}

	if jobList[1].ID != firstID {
		t.Fatalf(
			"expected second job to be %d, got %d",
			firstID,
			jobList[1].ID,
		)
	}
}

// TestStartJob verifies that a queued job becomes processing and receives
// a start timestamp.
func TestStartJob(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)

	if err := startJob(db, jobID); err != nil {
		t.Fatalf("startJob failed: %v", err)
	}

	job, err := GetJob(db, jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != "processing" {
		t.Fatalf(
			"expected status processing, got %q",
			job.Status,
		)
	}

	if job.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
}

// TestCompleteJob verifies that a job becomes completed and receives a
// completion timestamp.
func TestCompleteJob(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)

	if err := startJob(db, jobID); err != nil {
		t.Fatalf("startJob failed: %v", err)
	}

	if err := completeJob(db, jobID); err != nil {
		t.Fatalf("completeJob failed: %v", err)
	}

	job, err := GetJob(db, jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != "completed" {
		t.Fatalf(
			"expected status completed, got %q",
			job.Status,
		)
	}

	if job.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}

	if job.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

// TestFailJob verifies that a job becomes failed and stores its error
// message.
func TestFailJob(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)
	errorMessage := "test processing failure"

	if err := failJob(db, jobID, errorMessage); err != nil {
		t.Fatalf("failJob failed: %v", err)
	}

	job, err := GetJob(db, jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != "failed" {
		t.Fatalf(
			"expected status failed, got %q",
			job.Status,
		)
	}

	if job.ErrorMessage == nil {
		t.Fatal("expected ErrorMessage to be set")
	}

	if *job.ErrorMessage != errorMessage {
		t.Fatalf(
			"expected error message %q, got %q",
			errorMessage,
			*job.ErrorMessage,
		)
	}
}

// TestSaveResultReference verifies that a result path is stored on the job.
func TestSaveResultReference(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)
	resultPath := "uploads/123/results.json"

	if err := saveResultReference(
		db,
		jobID,
		resultPath,
	); err != nil {
		t.Fatalf("saveResultReference failed: %v", err)
	}

	job, err := GetJob(db, jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.ResultReference == nil {
		t.Fatal("expected ResultReference to be set")
	}

	if *job.ResultReference != resultPath {
		t.Fatalf(
			"expected result reference %q, got %q",
			resultPath,
			*job.ResultReference,
		)
	}
}

// TestValidateCSVValid verifies that a valid CSV is accepted.
func TestValidateCSVValid(t *testing.T) {
	filePath := filepath.Join(
		t.TempDir(),
		"valid.csv",
	)

	content := "name,value\nalpha,10\nbeta,20\n"

	if err := os.WriteFile(
		filePath,
		[]byte(content),
		0644,
	); err != nil {
		t.Fatalf("failed to write test CSV: %v", err)
	}

	if err := ValidateCSV(filePath); err != nil {
		t.Fatalf("expected valid CSV, got error: %v", err)
	}
}

// TestValidateCSVInvalid verifies that malformed CSV is rejected.
func TestValidateCSVInvalid(t *testing.T) {
	filePath := filepath.Join(
		t.TempDir(),
		"invalid.csv",
	)

	// The first row has two fields while the second has three.
	content := "name,value\nalpha,10,unexpected\n"

	if err := os.WriteFile(
		filePath,
		[]byte(content),
		0644,
	); err != nil {
		t.Fatalf("failed to write test CSV: %v", err)
	}

	if err := ValidateCSV(filePath); err == nil {
		t.Fatal("expected malformed CSV to return an error")
	}
}

// TestValidateCSVMissingFile verifies that a missing file returns an error.
func TestValidateCSVMissingFile(t *testing.T) {
	err := ValidateCSV(
		filepath.Join(
			t.TempDir(),
			"missing.csv",
		),
	)

	if err == nil {
		t.Fatal("expected error for missing CSV")
	}
}

// TestGetCSVColumns verifies that the first CSV row is returned as the
// column names.
func TestGetCSVColumns(t *testing.T) {
	filePath := filepath.Join(
		t.TempDir(),
		"columns.csv",
	)

	content := "temperature,humidity,co2\n20.5,50,400\n"

	if err := os.WriteFile(
		filePath,
		[]byte(content),
		0644,
	); err != nil {
		t.Fatalf("failed to write test CSV: %v", err)
	}

	columns, err := GetCSVColumns(filePath)
	if err != nil {
		t.Fatalf("GetCSVColumns failed: %v", err)
	}

	expected := []string{
		"temperature",
		"humidity",
		"co2",
	}

	if len(columns) != len(expected) {
		t.Fatalf(
			"expected %d columns, got %d",
			len(expected),
			len(columns),
		)
	}

	for i, expectedColumn := range expected {
		if columns[i] != expectedColumn {
			t.Fatalf(
				"expected column %q at index %d, got %q",
				expectedColumn,
				i,
				columns[i],
			)
		}
	}
}

// TestGetCSVColumnsMissingFile verifies that a missing CSV returns an error.
func TestGetCSVColumnsMissingFile(t *testing.T) {
	_, err := GetCSVColumns(
		filepath.Join(
			t.TempDir(),
			"missing.csv",
		),
	)

	if err == nil {
		t.Fatal("expected error for missing CSV")
	}
}

// TestRunDatasetProcessorMissingInput verifies that processor failures are
// returned with useful output.
func TestRunDatasetProcessorMissingInput(t *testing.T) {
	jobID := int64(999999)

	_, err := runDatasetProcessor(
		"does-not-exist.csv",
		jobID,
		"does-not-exist-config.json",
	)

	if err == nil {
		t.Fatal("expected processor error")
	}

	if !strings.Contains(
		err.Error(),
		"dataset processor failed",
	) {
		t.Fatalf(
			"expected processor failure message, got %v",
			err,
		)
	}
}

// TestProcessDatasetJobSuccess verifies the complete successful lifecycle
// of a dataset processing job.
func TestProcessDatasetJobSuccess(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)

	jobDirectory := filepath.Join(
		"uploads",
		fmt.Sprintf("%d", jobID),
	)

	// Use the actual job ID to create the expected job directory.
	jobDirectory = filepath.Join(
		"uploads",
		itoa(jobID),
	)

	if err := os.MkdirAll(jobDirectory, 0755); err != nil {
		t.Fatalf(
			"failed to create job directory: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(jobDirectory)
	})

	inputPath := filepath.Join(
		jobDirectory,
		"dataset.csv",
	)

	configPath := filepath.Join(
		jobDirectory,
		"config.json",
	)

	input := "temperature,humidity,co2\n" +
		"20,40,400\n" +
		"21,42,410\n" +
		"22,44,420\n" +
		"23,46,430\n" +
		"24,48,440\n" +
		"25,50,450\n" +
		"26,52,460\n" +
		"27,54,470\n" +
		"28,56,480\n" +
		"29,58,490\n"

	if err := os.WriteFile(
		inputPath,
		[]byte(input),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write dataset: %v",
			err,
		)
	}

	config := `{
		"model": "none",
		"visualizations": {}
	}`

	if err := os.WriteFile(
		configPath,
		[]byte(config),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write config: %v",
			err,
		)
	}

	if err := ProcessDatasetJob(
		db,
		jobID,
		inputPath,
		configPath,
	); err != nil {
		t.Fatalf(
			"ProcessDatasetJob failed: %v",
			err,
		)
	}

	job, err := GetJob(db, jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != "completed" {
		t.Fatalf(
			"expected status completed, got %q",
			job.Status,
		)
	}

	if job.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}

	if job.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}

	if job.ResultReference == nil {
		t.Fatal("expected ResultReference to be set")
	}

	if _, err := os.Stat(
		*job.ResultReference,
	); err != nil {
		t.Fatalf(
			"expected result file to exist: %v",
			err,
		)
	}

	resultData, err := os.ReadFile(
		*job.ResultReference,
	)
	if err != nil {
		t.Fatalf(
			"failed to read result file: %v",
			err,
		)
	}

	var result map[string]interface{}

	if err := json.Unmarshal(
		resultData,
		&result,
	); err != nil {
		t.Fatalf(
			"result file contains invalid JSON: %v",
			err,
		)
	}

	if result["num_rows"] != float64(10) {
		t.Fatalf(
			"expected 10 rows in results, got %v",
			result["num_rows"],
		)
	}
}

// TestProcessDatasetJobFailure verifies that a processor failure marks the
// job as failed and preserves the processor error.
func TestProcessDatasetJobFailure(t *testing.T) {
	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)

	err := ProcessDatasetJob(
		db,
		jobID,
		"does-not-exist.csv",
		"does-not-exist-config.json",
	)

	if err == nil {
		t.Fatal("expected ProcessDatasetJob to fail")
	}

	job, getErr := GetJob(db, jobID)
	if getErr != nil {
		t.Fatalf(
			"GetJob failed after processor failure: %v",
			getErr,
		)
	}

	if job.Status != "failed" {
		t.Fatalf(
			"expected status failed, got %q",
			job.Status,
		)
	}

	if job.ErrorMessage == nil {
		t.Fatal("expected ErrorMessage to be set")
	}

	if !strings.Contains(
		*job.ErrorMessage,
		"dataset processor failed",
	) {
		t.Fatalf(
			"expected processor error in ErrorMessage, got %q",
			*job.ErrorMessage,
		)
	}
}

// itoa is kept local to tests so job-directory construction does not require
// exposing filesystem path details from the production package.
func itoa(value int64) string {
	return fmt.Sprintf("%d", value)
}

func TestEnqueueJob(t *testing.T) {
	redisClient := redisclient.NewClient(&redisclient.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	ctx := context.Background()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	jobID := int64(12345)

	if err := EnqueueJob(redisClient, jobID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages, err := redisClient.XRange(
		ctx,
		"job_queue",
		"-",
		"+",
	).Result()
	if err != nil {
		t.Fatalf("failed to read Redis stream: %v", err)
	}

	var found bool

	for _, message := range messages {
		value, ok := message.Values["job_id"].(string)
		if !ok {
			continue
		}

		if value == "12345" {
			found = true

			if err := redisClient.XDel(
				ctx,
				"job_queue",
				message.ID,
			).Err(); err != nil {
				t.Logf(
					"failed to clean up test message %s: %v",
					message.ID,
					err,
				)
			}

			break
		}
	}

	if !found {
		t.Fatalf("expected job_id %d to be added to Redis stream", jobID)
	}
}
