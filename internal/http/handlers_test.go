package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/google/uuid"
	redisclient "github.com/redis/go-redis/v9"
)

// TestParseJobID verifies that job IDs are correctly parsed from requests.
func TestParseJobID(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    int64
		wantErr bool
	}{
		{
			name:  "valid ID",
			query: "id=42",
			want:  42,
		},
		{
			name:    "missing ID",
			query:   "",
			wantErr: true,
		},
		{
			name:    "non-numeric ID",
			query:   "id=abc",
			wantErr: true,
		},
		{
			name:  "negative ID",
			query: "id=-1",
			want:  -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				"/results?"+tt.query,
				nil,
			)

			got, err := parseJobID(req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected job ID %d, got %d", tt.want, got)
			}
		})
	}
}

// TestGetTemporaryDataset verifies validation of temporary upload IDs.
func TestGetTemporaryDataset(t *testing.T) {
	tests := []struct {
		name     string
		uploadID string
		wantErr  error
	}{
		{
			name:     "empty upload ID",
			uploadID: "",
			wantErr:  errDatasetNotInspected,
		},
		{
			name:     "invalid upload ID",
			uploadID: "not-a-uuid",
			wantErr:  errInvalidUploadID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getTemporaryDataset(tt.uploadID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"expected error %v, got %v",
					tt.wantErr,
					err,
				)
			}
		})
	}
}

// TestGetTemporaryDataset verifies that an existing temporary dataset
// is returned successfully.
func TestGetTemporaryDatasetExistingFile(t *testing.T) {
	if err := os.MkdirAll(temporaryUploadDirectory, 0755); err != nil {
		t.Fatalf("failed to create temporary upload directory: %v", err)
	}

	uploadID := "550e8400-e29b-41d4-a716-446655440000"
	tempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+".csv",
	)

	if err := os.WriteFile(tempPath, []byte("a,b\n1,2\n"), 0644); err != nil {
		t.Fatalf("failed to create temporary CSV: %v", err)
	}
	defer os.Remove(tempPath)

	got, err := getTemporaryDataset(uploadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != tempPath {
		t.Fatalf("expected path %q, got %q", tempPath, got)
	}
}

// TestGetDatasetConfig verifies that submitted form values are translated
// into the application dataset configuration.
func TestGetDatasetConfig(t *testing.T) {
	form := url.Values{
		"model":               {"random_forest_regressor"},
		"target":              {"co2"},
		"featureSelection":    {"manual"},
		"features":            {"temperature", "humidity"},
		"configType":          {"manual"},
		"n_estimators":        {"25"},
		"max_depth":           {"10"},
		"histograms":          {"true"},
		"box_plots":           {"true"},
		"correlation_heatmap": {"true"},
		"actual_vs_predicted": {"true"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/dataset",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	config, err := getDatasetConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Model != "random_forest_regressor" {
		t.Fatalf("unexpected model: %q", config.Model)
	}

	if config.Target != "co2" {
		t.Fatalf("unexpected target: %q", config.Target)
	}

	if config.FeatureSelection != "manual" {
		t.Fatalf(
			"unexpected feature selection: %q",
			config.FeatureSelection,
		)
	}

	if len(config.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(config.Features))
	}

	if config.ConfigurationType != "manual" {
		t.Fatalf(
			"unexpected configuration type: %q",
			config.ConfigurationType,
		)
	}

	if config.Configuration.NEstimators != 25 {
		t.Fatalf(
			"expected 25 estimators, got %d",
			config.Configuration.NEstimators,
		)
	}

	if config.Configuration.MaxDepth != 10 {
		t.Fatalf(
			"expected max depth 10, got %d",
			config.Configuration.MaxDepth,
		)
	}

	if !config.Visualizations.Histograms {
		t.Error("expected histograms to be enabled")
	}

	if !config.Visualizations.BoxPlots {
		t.Error("expected box plots to be enabled")
	}

	if !config.Visualizations.CorrelationHeatmap {
		t.Error("expected correlation heatmap to be enabled")
	}

	if !config.Visualizations.ActualVsPredicted {
		t.Error("expected actual vs predicted to be enabled")
	}
}

// TestGetDatasetConfigNoModel verifies that model-specific visualizations
// are disabled when no model is selected.
func TestGetDatasetConfigNoModel(t *testing.T) {
	form := url.Values{
		"model":               {"none"},
		"histograms":          {"true"},
		"box_plots":           {"true"},
		"correlation_heatmap": {"true"},
		"actual_vs_predicted": {"true"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/dataset",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	config, err := getDatasetConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Visualizations.ActualVsPredicted {
		t.Error("expected actual-vs-predicted to be disabled for no-model jobs")
	}

	if !config.Visualizations.Histograms {
		t.Error("expected histograms to remain enabled")
	}

	if !config.Visualizations.BoxPlots {
		t.Error("expected box plots to remain enabled")
	}

	if !config.Visualizations.CorrelationHeatmap {
		t.Error("expected correlation heatmap to remain enabled")
	}
}

// TestGetDatasetConfigInvalidEstimators verifies invalid Random Forest
// estimator values are rejected.
func TestGetDatasetConfigInvalidEstimators(t *testing.T) {
	form := url.Values{
		"model":        {"random_forest_regressor"},
		"configType":   {"manual"},
		"n_estimators": {"not-a-number"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/dataset",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	_, err := getDatasetConfig(req)
	if err == nil {
		t.Fatal("expected error for invalid n_estimators")
	}
}

// TestGetDatasetConfigInvalidMaxDepth verifies invalid Random Forest
// max-depth values are rejected.
func TestGetDatasetConfigInvalidMaxDepth(t *testing.T) {
	form := url.Values{
		"model":      {"random_forest_regressor"},
		"configType": {"manual"},
		"max_depth":  {"not-a-number"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/dataset",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	_, err := getDatasetConfig(req)
	if err == nil {
		t.Fatal("expected error for invalid max_depth")
	}
}

// TestLoadDatasetResults verifies that dataset results JSON is decoded
// into the expected Go structure.
func TestLoadDatasetResults(t *testing.T) {
	tempDir := t.TempDir()
	resultPath := filepath.Join(tempDir, "results.json")

	data := map[string]interface{}{
		"num_rows":    10,
		"num_columns": 2,
		"column_names": []string{
			"temperature",
			"humidity",
		},
	}

	fileData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal test results: %v", err)
	}

	if err := os.WriteFile(resultPath, fileData, 0644); err != nil {
		t.Fatalf("failed to write test results: %v", err)
	}

	results, err := loadDatasetResults(resultPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results.NumRows != 10 {
		t.Fatalf("expected 10 rows, got %d", results.NumRows)
	}

	if results.NumColumns != 2 {
		t.Fatalf("expected 2 columns, got %d", results.NumColumns)
	}

	if len(results.ColumnNames) != 2 {
		t.Fatalf(
			"expected 2 column names, got %d",
			len(results.ColumnNames),
		)
	}
}

// TestLoadDatasetResultsMissingFile verifies missing result files return
// an error.
func TestLoadDatasetResultsMissingFile(t *testing.T) {
	_, err := loadDatasetResults(
		filepath.Join(t.TempDir(), "missing.json"),
	)

	if err == nil {
		t.Fatal("expected error for missing results file")
	}
}

// TestLoadModelResultsMissingFile verifies that a missing model-results
// file means no model was requested.
func TestLoadModelResultsMissingFile(t *testing.T) {
	jobID := int64(999999)

	results, err := loadModelResults(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results != nil {
		t.Fatal("expected nil model results for missing file")
	}
}

// TestIndexHandlerRejectsNonRoot verifies that IndexHandler returns 404
// for paths other than "/".
func TestIndexHandlerRejectsNonRoot(t *testing.T) {
	handler := IndexHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/not-root", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

// TestDownloadResultsHandlerRejectsWrongMethod verifies that the download
// endpoint only accepts GET requests.
func TestDownloadResultsHandlerRejectsWrongMethod(t *testing.T) {
	handler := DownloadResultsHandler(nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/results/download?id=1",
		nil,
	)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}
}

// TestDownloadModelHandlerRejectsWrongMethod verifies that the model-results
// download endpoint only accepts GET requests.
func TestDownloadModelHandlerRejectsWrongMethod(t *testing.T) {
	handler := DownloadModelHandler(nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/results/download/model?id=1",
		nil,
	)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}
}

// TestDatasetInspectionHandlerRejectsWrongMethod verifies that dataset
// inspection only accepts POST requests.
func TestDatasetInspectionHandlerRejectsWrongMethod(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/inspect/dataset",
		nil,
	)
	rec := httptest.NewRecorder()

	DatasetInspectionHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}
}

// TestDatasetInspectionHandlerRequiresCSV verifies that inspection fails
// cleanly when no CSV file is uploaded.
func TestDatasetInspectionHandlerRequiresCSV(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/inspect/dataset",
		nil,
	)
	rec := httptest.NewRecorder()

	DatasetInspectionHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

// TestVisualizationHandlerRequiresID verifies that visualization requests
// require a valid job ID.
func TestVisualizationHandlerRequiresID(t *testing.T) {
	handler := VisualizationHandler(nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/results/visualization?type=actual_vs_predicted",
		nil,
	)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

// TestVisualizationHandlerRequiresType verifies that a visualization type
// must be supplied.
func TestVisualizationHandlerRequiresType(t *testing.T) {
	// This test reaches database access after parsing the job ID, so a real
	// database connection is required.
	//
	// Keep this case covered by the integration tests once the database test
	// helper is established.
	t.Skip("requires database integration test setup")
}

// Ensure *sql.DB remains referenced in this test package as handlers use it.
var _ *sql.DB

// Ensure jobs remains referenced because handler behavior is coupled to the
// jobs package.
var _ = jobs.Job{}

// TestDatasetSubmissionHandlerEnqueuesJob verifies that submitting a dataset
// creates a queued job and adds its ID to the Redis job stream.
func TestDatasetSubmissionHandlerEnqueuesJob(t *testing.T) {
	db := database.OpenDatabase()
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL is not available: %v", err)
	}

	redisClient := redisclient.NewClient(&redisclient.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	uploadID := uuid.New().String()
	tempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+".csv",
	)

	if err := os.MkdirAll(temporaryUploadDirectory, 0755); err != nil {
		t.Fatalf("failed to create temporary upload directory: %v", err)
	}

	if err := os.WriteFile(
		tempPath,
		[]byte("temperature,humidity\n20,50\n21,55\n"),
		0644,
	); err != nil {
		t.Fatalf("failed to create temporary dataset: %v", err)
	}
	defer os.Remove(tempPath)

	form := url.Values{
		"uploadID":         {uploadID},
		"model":            {"none"},
		"featureSelection": {""},
		"configType":       {""},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/dataset",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	handler := DatasetSubmissionHandler(db, redisClient)
	handler(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	// The handler has redirected successfully, so recover the created
	// job from the database by looking for the most recent dataset job.
	var jobID int64
	err := db.QueryRow(`
		SELECT id
		FROM jobs
		WHERE type = 'dataset'
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&jobID)

	if err != nil {
		t.Fatalf("failed to retrieve created job: %v", err)
	}

	defer jobs.DeleteJob(db, jobID)

	// Verify the job is still queued.
	var status string
	if err := db.QueryRow(
		`SELECT status FROM jobs WHERE id = $1`,
		jobID,
	).Scan(&status); err != nil {
		t.Fatalf("failed to retrieve job status: %v", err)
	}

	if status != "queued" {
		t.Fatalf("expected job status queued, got %q", status)
	}

	// Find the Redis message containing this job ID.
	streams, err := redisClient.XRange(
		context.Background(),
		"job_queue",
		"-",
		"+",
	).Result()

	if err != nil {
		t.Fatalf("failed to read Redis stream: %v", err)
	}

	var found bool

	for _, message := range streams {
		value, ok := message.Values["job_id"].(string)
		if !ok {
			continue
		}

		if value == fmt.Sprintf("%d", jobID) {
			found = true

			// Remove the test message so it doesn't affect other tests.
			if err := redisClient.XDel(
				context.Background(),
				"job_queue",
				message.ID,
			).Err(); err != nil {
				t.Logf(
					"failed to remove test Redis message %s: %v",
					message.ID,
					err,
				)
			}

			break
		}
	}

	if !found {
		t.Fatalf(
			"expected Redis stream to contain job_id %d",
			jobID,
		)
	}
}
