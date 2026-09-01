package jobs

import (
	"context"

	"database/sql"

	"encoding/json"

	"errors"

	"fmt"

	"image"

	"image/color"

	"image/png"

	"io"

	"os"

	"path/filepath"

	"strings"

	"sync"

	"testing"

	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"

	redisclient "github.com/redis/go-redis/v9"
)

// setupTestDatabase runs migrations and returns a database connection*

// for tests that require PostgreSQL.*

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

func clearJobsTable(t *testing.T, db *sql.DB) {

	t.Helper()

	if _, err := db.Exec(`DELETE FROM jobs`); err != nil {

		t.Fatalf(

			"failed to clear jobs table: %v",

			err,
		)

	}

	t.Cleanup(func() {

		_, _ = db.Exec(`DELETE FROM jobs`)

	})

}

// createTestJob creates a dataset job and registers cleanup for the database row.*

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

// TestCreateJob verifies that a new job is created with the expected*

// initial values.*

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

// TestGetJobNotFound verifies that requesting a nonexistent job returns*

// sql.ErrNoRows.*

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

// TestGetJobs verifies that jobs are returned in newest-first order.*

func TestGetJobs(t *testing.T) {

	db := setupTestDatabase(t)

	firstID := createTestJob(t, db)

	time.Sleep(time.Millisecond)

	secondID := createTestJob(t, db)

	jobList, err := GetJobs(db)

	if err != nil {

		t.Fatalf(

			"GetJobs failed: %v",

			err,
		)

	}

	var firstJobIndex = -1

	var secondJobIndex = -1

	for index, job := range jobList {

		switch job.ID {

		case firstID:

			firstJobIndex = index

		case secondID:

			secondJobIndex = index

		}

	}

	if firstJobIndex == -1 {

		t.Fatalf(

			"first test job %d was not returned",

			firstID,
		)

	}

	if secondJobIndex == -1 {

		t.Fatalf(

			"second test job %d was not returned",

			secondID,
		)

	}

	if secondJobIndex >= firstJobIndex {

		t.Fatalf(

			"expected job %d to appear before job %d, got indexes %d and %d",

			secondID,

			firstID,

			secondJobIndex,

			firstJobIndex,
		)

	}

}

// TestStartJob verifies that a queued job becomes processing and receives*

// a start timestamp.*

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

// TestCompleteJob verifies that a job becomes completed and receives*

// a completion timestamp.*

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

// TestFailJob verifies that a job becomes failed and stores its error*

// message.*

func TestFailJob(t *testing.T) {

	db := setupTestDatabase(t)

	jobID := createTestJob(t, db)

	errorMessage := "test processing failure"

	if err := FailJob(db, jobID, errorMessage); err != nil {

		t.Fatalf("FailJob failed: %v", err)

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

// TestSaveResultReference verifies that a result path is stored on the job.*

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

// TestValidateCSVValid verifies that a valid CSV is accepted.*

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

// TestValidateCSVInvalid verifies that malformed CSV is rejected.*

func TestValidateCSVInvalid(t *testing.T) {

	filePath := filepath.Join(

		t.TempDir(),

		"invalid.csv",
	)

	// The first row has two fields while the second has three.*

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

// TestValidateCSVMissingFile verifies that a missing file returns an error.*

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

// TestGetCSVColumns verifies that the first CSV row is returned as*

// the column names.*

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

// TestGetCSVColumnsMissingFile verifies that a missing CSV returns an error.*

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

// TestFindProcessor verifies that the dataset processor path can be found.*

func TestFindProcessor(t *testing.T) {

	processorPath, err := findProcessor("dataset")

	if err != nil {

		t.Fatalf("findProcessor failed: %v", err)

	}

	expectedSuffix := filepath.Join(

		"processors",

		"dataset",

		"main.py",
	)

	if !strings.HasSuffix(

		processorPath,

		expectedSuffix,
	) {

		t.Fatalf(

			"expected processor path to end with %q, got %q",

			expectedSuffix,

			processorPath,
		)

	}

	if _, err := os.Stat(processorPath); err != nil {

		t.Fatalf(

			"expected processor path to exist: %v",

			err,
		)

	}

}

// TestFindImageProcessor verifies that the image processor path can be found.*

func TestFindImageProcessor(t *testing.T) {

	processorPath, err := findProcessor("image")

	if err != nil {

		t.Fatalf("findProcessor failed for image: %v", err)

	}

	expectedSuffix := filepath.Join(

		"processors",

		"image",

		"main.py",
	)

	if !strings.HasSuffix(

		processorPath,

		expectedSuffix,
	) {

		t.Fatalf(

			"expected processor path to end with %q, got %q",

			expectedSuffix,

			processorPath,
		)

	}

	if _, err := os.Stat(processorPath); err != nil {

		t.Fatalf(

			"expected image processor path to exist: %v",

			err,
		)

	}

}

// TestFindProcessorMissing verifies that an unknown processor type*

// returns an error.*

func TestFindProcessorMissing(t *testing.T) {

	_, err := findProcessor("does-not-exist")

	if err == nil {

		t.Fatal("expected findProcessor to return an error")

	}

	if !strings.Contains(

		err.Error(),

		"could not find processor",
	) {

		t.Fatalf(

			"expected processor lookup error, got %v",

			err,
		)

	}

}

// TestRunProcessorMissingInput verifies that processor failures are*

// returned with useful output.*

func TestRunProcessorMissingInput(t *testing.T) {
	resultPath := filepath.Join(
		t.TempDir(),
		"results.json",
	)

	_, err := runProcessor(
		"dataset",
		"does-not-exist.csv",
		resultPath,
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

func TestProcessJobSuccess(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())
	jobID := createTestJob(t, db)

	inputKey := fmt.Sprintf(
		"jobs/%d/dataset.csv",
		jobID,
	)

	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
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

	if err := store.Put(
		context.Background(),
		inputKey,
		strings.NewReader(input),
	); err != nil {
		t.Fatalf(
			"failed to store dataset: %v",
			err,
		)
	}

	config := `{
		"model": "none",
		"visualizations": {}
	}`

	if err := store.Put(
		context.Background(),
		configKey,
		strings.NewReader(config),
	); err != nil {
		t.Fatalf(
			"failed to store config: %v",
			err,
		)
	}

	if _, err := db.Exec(
		`UPDATE jobs
		 SET input_reference = $1
		 WHERE id = $2`,
		inputKey,
		jobID,
	); err != nil {
		t.Fatalf(
			"failed to set input reference: %v",
			err,
		)
	}
	m := createTestMetrics()

	if err := ProcessJob(
		db,
		jobID,
		store,
		m,
	); err != nil {
		t.Fatalf(
			"ProcessJob failed: %v",
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

	resultReader, err := store.Get(
		context.Background(),
		*job.ResultReference,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve result from storage: %v",
			err,
		)
	}
	defer resultReader.Close()

	resultData, err := io.ReadAll(resultReader)
	if err != nil {
		t.Fatalf(
			"failed to read result: %v",
			err,
		)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultData, &result); err != nil {
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

func TestProcessImageJobSuccess(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())

	jobID, err := CreateJob(
		db,
		"image",
	)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	t.Cleanup(func() {
		_ = DeleteJob(
			db,
			jobID,
			store,
		)
	})

	inputTempPath := filepath.Join(
		t.TempDir(),
		"input.png",
	)

	img := image.NewRGBA(
		image.Rect(0, 0, 1, 1),
	)
	img.Set(
		0,
		0,
		color.RGBA{
			R: 255,
			G: 0,
			B: 0,
			A: 255,
		},
	)

	imageFile, err := os.Create(inputTempPath)
	if err != nil {
		t.Fatalf(
			"failed to create image: %v",
			err,
		)
	}

	if err := png.Encode(imageFile, img); err != nil {
		imageFile.Close()
		t.Fatalf(
			"failed to encode test PNG: %v",
			err,
		)
	}

	if err := imageFile.Close(); err != nil {
		t.Fatalf(
			"failed to close test image: %v",
			err,
		)
	}

	inputKey := fmt.Sprintf(
		"jobs/%d/input.png",
		jobID,
	)

	storedInput, err := os.Open(inputTempPath)
	if err != nil {
		t.Fatalf(
			"failed to reopen test image: %v",
			err,
		)
	}

	if err := store.Put(
		context.Background(),
		inputKey,
		storedInput,
	); err != nil {
		storedInput.Close()
		t.Fatalf(
			"failed to store test image: %v",
			err,
		)
	}

	if err := storedInput.Close(); err != nil {
		t.Fatalf(
			"failed to close stored image: %v",
			err,
		)
	}

	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	config := `{
		"resize": false,
		"compression": false,
		"format_conversion": false,
		"output_format": "original",
		"extract_metadata": false
	}`

	if err := store.Put(
		context.Background(),
		configKey,
		strings.NewReader(config),
	); err != nil {
		t.Fatalf(
			"failed to store image config: %v",
			err,
		)
	}

	if _, err := db.Exec(
		`UPDATE jobs
		 SET input_reference = $1
		 WHERE id = $2`,
		inputKey,
		jobID,
	); err != nil {
		t.Fatalf(
			"failed to set input reference: %v",
			err,
		)
	}
	m := createTestMetrics()
	if err := ProcessJob(
		db,
		jobID,
		store,
		m,
	); err != nil {
		t.Fatalf(
			"ProcessJob failed for image job: %v",
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

	if job.ResultReference == nil {
		t.Fatal("expected ResultReference to be set")
	}

	resultReader, err := store.Get(
		context.Background(),
		*job.ResultReference,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve image result: %v",
			err,
		)
	}
	defer resultReader.Close()

	resultData, err := io.ReadAll(resultReader)
	if err != nil {
		t.Fatalf(
			"failed to read image result: %v",
			err,
		)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultData, &result); err != nil {
		t.Fatalf(
			"image result file contains invalid JSON: %v",
			err,
		)
	}

	if result["original_path"] == nil {
		t.Fatal("expected original_path in image results")
	}

	if result["processed_path"] == nil {
		t.Fatal("expected processed_path in image results")
	}
}

func TestProcessJobFailure(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())
	jobID := createTestJob(t, db)

	inputReference := fmt.Sprintf(
		"jobs/%d/does-not-exist.csv",
		jobID,
	)

	if _, err := db.Exec(
		`UPDATE jobs
		 SET input_reference = $1
		 WHERE id = $2`,
		inputReference,
		jobID,
	); err != nil {
		t.Fatalf(
			"failed to set input reference: %v",
			err,
		)
	}
	m := createTestMetrics()
	err := ProcessJob(
		db,
		jobID,
		store,
		m,
	)
	if err == nil {
		t.Fatal("expected ProcessJob to fail")
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
		"failed to retrieve job input",
	) {
		t.Fatalf(
			"expected storage error in ErrorMessage, got %q",
			*job.ErrorMessage,
		)
	}
}

func TestProcessJobMissingInput(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())
	jobID := createTestJob(t, db)
	m := createTestMetrics()
	err := ProcessJob(
		db,
		jobID,
		store,
		m,
	)
	if err == nil {
		t.Fatal("expected ProcessJob to fail")
	}

	if !strings.Contains(
		err.Error(),
		"has no input reference",
	) {
		t.Fatalf(
			"expected missing input reference error, got %v",
			err,
		)
	}

	job, getErr := GetJob(db, jobID)
	if getErr != nil {
		t.Fatalf(
			"GetJob failed: %v",
			getErr,
		)
	}

	if job.Status != "queued" {
		t.Fatalf(
			"expected job to remain queued when input is missing, got %q",
			job.Status,
		)
	}
}

func TestEnqueueJob(t *testing.T) {

	redisClient := redisclient.NewClient(

		&redisclient.Options{

			Addr: "localhost:6379",
		},
	)

	defer redisClient.Close()

	ctx := context.Background()

	if err := redisClient.Ping(ctx).Err(); err != nil {

		t.Skipf(

			"Redis is not available: %v",

			err,
		)

	}

	jobID := int64(12345)

	if err := EnqueueJob(

		redisClient,

		jobID,
	); err != nil {

		t.Fatalf(

			"unexpected error: %v",

			err,
		)

	}

	messages, err := redisClient.XRange(

		ctx,

		"job_queue",

		"-",

		"+",
	).Result()

	if err != nil {

		t.Fatalf(

			"failed to read Redis stream: %v",

			err,
		)

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

		t.Fatalf(

			"expected job_id %d to be added to Redis stream",

			jobID,
		)

	}

}

// TestProcessJobUnsupportedType verifies that a job with an unsupported*

// processor type fails processor lookup without being claimed.*

func TestProcessJobUnsupportedType(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())
	jobID := createTestJob(t, db)

	inputKey := fmt.Sprintf(
		"jobs/%d/test-input.csv",
		jobID,
	)

	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	if err := store.Put(
		context.Background(),
		inputKey,
		strings.NewReader("a,b\n1,2\n"),
	); err != nil {
		t.Fatalf(
			"failed to store test input: %v",
			err,
		)
	}

	if err := store.Put(
		context.Background(),
		configKey,
		strings.NewReader(`{}`),
	); err != nil {
		t.Fatalf(
			"failed to store test config: %v",
			err,
		)
	}

	if _, err := db.Exec(
		`UPDATE jobs
		 SET type = $1,
		     input_reference = $2
		 WHERE id = $3`,
		"unsupported",
		inputKey,
		jobID,
	); err != nil {
		t.Fatalf(
			"failed to update job: %v",
			err,
		)
	}
	m := createTestMetrics()
	err := ProcessJob(
		db,
		jobID,
		store,
		m,
	)
	if err == nil {
		t.Fatal("expected ProcessJob to fail")
	}

	if !strings.Contains(
		err.Error(),
		"could not find processor",
	) {
		t.Fatalf(
			"expected processor lookup error, got %v",
			err,
		)
	}

	job, getErr := GetJob(db, jobID)
	if getErr != nil {
		t.Fatalf(
			"GetJob failed: %v",
			getErr,
		)
	}

	if job.Status != "failed" {
		t.Fatalf(
			"expected status failed when processor lookup fails, got %q",
			job.Status,
		)
	}
}

func TestCreateRouteJob(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())

	jobID, err := CreateJob(
		db,
		"route",
	)
	if err != nil {
		t.Fatalf(
			"CreateJob failed for route job: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = DeleteJob(
			db,
			jobID,
			store,
		)
	})

	job, err := GetJob(
		db,
		jobID,
	)
	if err != nil {
		t.Fatalf(
			"GetJob failed: %v",
			err,
		)
	}

	if job.Type != "route" {
		t.Fatalf(
			"expected job type route, got %q",
			job.Type,
		)
	}

	if job.Status != "queued" {
		t.Fatalf(
			"expected status queued, got %q",
			job.Status,
		)
	}
}

func TestFindRouteProcessor(t *testing.T) {

	processorPath, err := findProcessor("route")

	if err != nil {

		t.Fatalf(

			"findProcessor failed for route: %v",

			err,
		)

	}

	expectedSuffix := filepath.Join(

		"processors",

		"route",

		"main.py",
	)

	if !strings.HasSuffix(

		processorPath,

		expectedSuffix,
	) {

		t.Fatalf(

			"expected processor path to end with %q, got %q",

			expectedSuffix,

			processorPath,
		)

	}

	if _, err := os.Stat(processorPath); err != nil {

		t.Fatalf(

			"expected route processor to exist: %v",

			err,
		)

	}

}

// TestProcessRouteJobSuccess verifies that the shared ProcessJob lifecycle*

// successfully dispatches a route job to the route processor.*

func TestProcessRouteJobSuccess(t *testing.T) {
	db := setupTestDatabase(t)
	store := storage.NewLocalStorage(t.TempDir())

	jobID, err := CreateJob(
		db,
		"route",
	)
	if err != nil {
		t.Fatalf(
			"CreateJob failed: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = DeleteJob(
			db,
			jobID,
			store,
		)
	})

	routeKey := fmt.Sprintf(
		"jobs/%d/route.csv",
		jobID,
	)

	distanceKey := fmt.Sprintf(
		"jobs/%d/distances.csv",
		jobID,
	)

	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	routeData :=
		"id,name,demand,priority,window_start,window_end\n" +
			"1,Warehouse,0,0,08:00,17:00\n" +
			"2,Customer A,10,1,08:00,17:00\n"

	distanceData :=
		"location,Warehouse,Customer A\n" +
			"Warehouse,0,10\n" +
			"Customer A,10,0\n"

	configData := `{
		"start_location": "Warehouse",
		"end_location": "Warehouse",
		"optimization": {
			"algorithm": "nearest_neighbor_2opt"
		},
		"constraints": {
			"max_distance": 100.0,
			"max_stops": 10
		}
	}`

	if err := store.Put(
		context.Background(),
		routeKey,
		strings.NewReader(routeData),
	); err != nil {
		t.Fatalf(
			"failed to store route CSV: %v",
			err,
		)
	}

	if err := store.Put(
		context.Background(),
		distanceKey,
		strings.NewReader(distanceData),
	); err != nil {
		t.Fatalf(
			"failed to store distance CSV: %v",
			err,
		)
	}

	if err := store.Put(
		context.Background(),
		configKey,
		strings.NewReader(configData),
	); err != nil {
		t.Fatalf(
			"failed to store route config: %v",
			err,
		)
	}

	if _, err := db.Exec(
		`UPDATE jobs
		 SET input_reference = $1
		 WHERE id = $2`,
		routeKey,
		jobID,
	); err != nil {
		t.Fatalf(
			"failed to set route input reference: %v",
			err,
		)
	}
	m := createTestMetrics()
	if err := ProcessJob(
		db,
		jobID,
		store,
		m,
	); err != nil {
		t.Fatalf(
			"ProcessJob failed for route job: %v",
			err,
		)
	}

	job, err := GetJob(
		db,
		jobID,
	)
	if err != nil {
		t.Fatalf(
			"GetJob failed: %v",
			err,
		)
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

	resultReader, err := store.Get(
		context.Background(),
		*job.ResultReference,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve route result: %v",
			err,
		)
	}
	defer resultReader.Close()

	resultData, err := io.ReadAll(resultReader)
	if err != nil {
		t.Fatalf(
			"failed to read route result: %v",
			err,
		)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultData, &result); err != nil {
		t.Fatalf(
			"route result file contains invalid JSON: %v",
			err,
		)
	}

	if result["start_location"] != "Warehouse" {
		t.Fatalf(
			"expected start_location Warehouse, got %v",
			result["start_location"],
		)
	}

	if result["end_location"] != "Warehouse" {
		t.Fatalf(
			"expected end_location Warehouse, got %v",
			result["end_location"],
		)
	}

	if result["algorithm"] != "nearest_neighbor_2opt" {
		t.Fatalf(
			"expected nearest_neighbor_2opt algorithm, got %v",
			result["algorithm"],
		)
	}

	if result["feasible"] != true {
		t.Fatal("expected route result to be feasible")
	}

	if result["initial_distance"] == nil {
		t.Fatal("expected initial_distance in route results")
	}

	if result["optimized_distance"] == nil {
		t.Fatal("expected optimized_distance in route results")
	}

	if result["runtime_seconds"] == nil {
		t.Fatal("expected runtime_seconds in route results")
	}
}

func TestStartJobPreventsDuplicateProcessing(t *testing.T) {

	db := setupTestDatabase(t)

	jobID := createTestJob(

		t,

		db,
	)

	var wg sync.WaitGroup

	wg.Add(2)

	results := make(chan error, 2)

	start := make(chan struct{})

	for i := 0; i < 2; i++ {

		go func() {

			defer wg.Done()

			<-start

			err := startJob(

				db,

				jobID,
			)

			results <- err

		}()

	}

	// Release both goroutines at approximately the same time.*

	close(start)

	wg.Wait()

	close(results)

	var errorsSeen []error

	successCount := 0

	for err := range results {

		if err == nil {

			successCount++

		} else {

			errorsSeen = append(errorsSeen, err)

		}

	}

	if successCount != 1 {

		t.Fatalf(

			"expected exactly one successful job claim, got %d; errors: %v",

			successCount,

			errorsSeen,
		)

	}

	job, err := GetJob(

		db,

		jobID,
	)

	if err != nil {

		t.Fatalf(

			"failed to retrieve job: %v",

			err,
		)

	}

	if job.Status != "processing" {

		t.Fatalf(

			"expected job status processing, got %q",

			job.Status,
		)

	}

	if job.Attempts != 1 {

		t.Fatalf(

			"expected exactly one processing attempt, got %d",

			job.Attempts,
		)

	}

}

func TestCreateJobWithLimit(t *testing.T) {

	db := setupTestDatabase(t)

	clearJobsTable(t, db)

	for i := 0; i < MaxOutstandingJobs; i++ {
		m := createTestMetrics()
		if _, err := CreateJobWithLimit(
			db,
			"dataset",
			m,
		); err != nil {

			t.Fatalf(

				"failed to create job %d: %v",

				i+1,

				err,
			)

		}

	}
	m := createTestMetrics()
	if _, err := CreateJobWithLimit(

		db,

		"dataset",
		m,
	); !errors.Is(err, ErrJobQueueFull) {

		t.Fatalf(

			"expected ErrJobQueueFull, got %v",

			err,
		)

	}

}

func TestCreateJobWithLimitIgnoresCompletedJobs(

	t *testing.T,

) {

	db := setupTestDatabase(t)

	clearJobsTable(t, db)
	m := createTestMetrics()
	for i := 0; i < MaxOutstandingJobs; i++ {

		if _, err := CreateJobWithLimit(

			db,

			"dataset",
			m,
		); err != nil {

			t.Fatalf(
				"failed to create job %d: %v",
				i+1,
				err,
			)

		}

	}

	_, err := db.Exec(

		`UPDATE jobs

         SET status = 'completed'

         WHERE id = (

            SELECT id

            FROM jobs

            ORDER BY id

            LIMIT 1

         )`,
	)

	if err != nil {

		t.Fatalf(

			"failed to complete test job: %v",

			err,
		)

	}

	if _, err := CreateJobWithLimit(

		db,

		"dataset",
		m,
	); err != nil {

		t.Fatalf(

			"expected completed job to free capacity: %v",

			err,
		)

	}

}

func TestCreateJobWithLimitPreventsConcurrentOverflow(

	t *testing.T,

) {

	db := setupTestDatabase(t)

	clearJobsTable(t, db)
	m := createTestMetrics()
	for i := 0; i < MaxOutstandingJobs-1; i++ {

		if _, err := CreateJobWithLimit(

			db,

			"dataset",
			m,
		); err != nil {

			t.Fatalf(

				"failed to create initial job %d: %v",

				i+1,

				err,
			)

		}

	}

	var wg sync.WaitGroup

	wg.Add(2)

	results := make(chan error, 2)

	for i := 0; i < 2; i++ {

		go func() {

			defer wg.Done()

			_, err := CreateJobWithLimit(

				db,

				"dataset",
				m,
			)

			results <- err

		}()

	}

	wg.Wait()

	close(results)

	successes := 0

	queueFullErrors := 0

	for err := range results {

		switch {

		case err == nil:

			successes++

		case errors.Is(err, ErrJobQueueFull):

			queueFullErrors++

		default:

			t.Fatalf(

				"unexpected error: %v",

				err,
			)

		}

	}

	if successes != 1 {

		t.Fatalf(

			"expected exactly one concurrent submission to succeed, got %d",

			successes,
		)

	}

	if queueFullErrors != 1 {

		t.Fatalf(

			"expected exactly one concurrent submission to be rejected, got %d",

			queueFullErrors,
		)

	}

}

func createTestMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()
	return metrics.NewMetrics(registry)
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "storage",
			err: categorizeError(
				errorStorage,
				errors.New("storage failed"),
			),
			expected: "storage",
		},
		{
			name: "filesystem",
			err: categorizeError(
				errorFilesystem,
				errors.New("filesystem failed"),
			),
			expected: "filesystem",
		},
		{
			name: "processor",
			err: categorizeError(
				errorProcessor,
				errors.New("processor failed"),
			),
			expected: "processor",
		},
		{
			name: "validation",
			err: categorizeError(
				errorValidation,
				errors.New("validation failed"),
			),
			expected: "validation",
		},
		{
			name: "configuration",
			err: categorizeError(
				errorConfiguration,
				errors.New("configuration failed"),
			),
			expected: "configuration",
		},
		{
			name: "database",
			err: categorizeError(
				errorDatabase,
				errors.New("database failed"),
			),
			expected: "database",
		},
		{
			name:     "timeout",
			err:      context.DeadlineExceeded,
			expected: "timeout",
		},
		{
			name:     "cancelled",
			err:      context.Canceled,
			expected: "cancelled",
		},
		{
			name:     "unknown",
			err:      errors.New("something unexpected happened"),
			expected: "unknown",
		},
		{
			name: "wrapped categorized error",
			err: fmt.Errorf(
				"outer error: %w",
				categorizeError(
					errorStorage,
					errors.New("storage failed"),
				),
			),
			expected: "storage",
		},
		{
			name: "retry limit",
			err: categorizeError(
				errorRetryLimit,
				errors.New("maximum attempts reached"),
			),
			expected: "retry_limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyError(tt.err)

			if got != tt.expected {
				t.Fatalf(
					"expected error category %q, got %q",
					tt.expected,
					got,
				)
			}
		})
	}
}
