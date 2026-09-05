package server

import (
	"bytes"
	"io"
	"time"

	"context"

	"database/sql"

	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"

	"errors"

	"fmt"

	"mime/multipart"

	"net/http"

	"net/http/httptest"

	"net/url"

	"os"

	"path/filepath"

	"strings"

	"testing"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/auth"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/metrics"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/bcrypt"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"

	"github.com/google/uuid"

	redisclient "github.com/redis/go-redis/v9"
)

// TestParseJobID verifies that job IDs are correctly parsed from requests.

func TestParseJobID(t *testing.T) {

	tests := []struct {
		name string

		query string

		want int64

		wantErr bool
	}{

		{

			name: "valid ID",

			query: "id=42",

			want: 42,
		},

		{

			name: "missing ID",

			query: "",

			wantErr: true,
		},

		{

			name: "non-numeric ID",

			query: "id=abc",

			wantErr: true,
		},

		{

			name: "negative ID",

			query: "id=-1",

			want: -1,
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
	db := setupTestDatabase(t)
	tests := []struct {
		name string

		uploadID string

		wantErr error
	}{

		{

			name: "empty upload ID",

			uploadID: "",

			wantErr: errDatasetNotInspected,
		},

		{

			name: "invalid upload ID",

			uploadID: "not-a-uuid",

			wantErr: errInvalidUploadID,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			_, err := getTemporaryDataset(
				db,
				tt.uploadID,
				0,
			)

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
	db := setupTestDatabase(t)

	sessionToken, err := auth.GenerateSessionToken()

	if err != nil {
		t.Fatalf("failed to generate test session token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionToken); err != nil {
		t.Fatalf("failed to store test session: %v", err)
	}

	var sessionID int64
	if err := db.QueryRow(`SELECT id FROM sessions WHERE token = $1`, sessionToken).Scan(&sessionID); err != nil {
		t.Fatalf("failed to query session ID: %v", err)
	}

	if err := os.MkdirAll(temporaryUploadDirectory, 0755); err != nil {

		t.Fatalf("failed to create temporary upload directory: %v", err)

	}

	uploadID := "550e8400-e29b-41d4-a716-446655440000"

	if _, err := db.Exec(`INSERT INTO temporary_uploads (upload_id, session_id, created_at) VALUES ($1, $2, $3)`, uploadID, sessionID, time.Now()); err != nil {

		t.Fatalf("failed to insert temporary dataset record: %v", err)
	}

	tempPath := filepath.Join(

		temporaryUploadDirectory,

		uploadID+".csv",
	)

	if err := os.WriteFile(tempPath, []byte("a,b\n1,2\n"), 0644); err != nil {

		t.Fatalf("failed to create temporary CSV: %v", err)

	}

	t.Cleanup(func() {
		os.Remove(tempPath)
		_, _ = db.Exec(`DELETE FROM temporary_uploads WHERE upload_id = $1`, uploadID)
	})

	got, err := getTemporaryDataset(
		db,
		uploadID,
		sessionID,
	)

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if got != tempPath {

		t.Fatalf("expected path %q, got %q", tempPath, got)

	}

}

// TestGetTemporaryImage verifies validation of temporary image upload IDs.

func TestGetTemporaryImage(t *testing.T) {
	db := setupTestDatabase(t)
	tests := []struct {
		name string

		uploadID string

		wantErr error
	}{

		{

			name: "empty upload ID",

			uploadID: "",

			wantErr: errImageNotInspected,
		},

		{

			name: "invalid upload ID",

			uploadID: "not-a-uuid",

			wantErr: errInvalidUploadID,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			_, err := getTemporaryImage(
				db,
				tt.uploadID,
				0,
			)

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

// TestGetTemporaryImageExistingFile verifies that an existing

// temporary image is returned successfully.

func TestGetTemporaryImageExistingFile(t *testing.T) {
	db := setupTestDatabase(t)

	sessionToken, err := auth.GenerateSessionToken()

	if err != nil {
		t.Fatalf("failed to generate test session token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionToken); err != nil {
		t.Fatalf("failed to store test session: %v", err)
	}

	var sessionID int64
	if err := db.QueryRow(`SELECT id FROM sessions WHERE token = $1`, sessionToken).Scan(&sessionID); err != nil {
		t.Fatalf("failed to query session ID: %v", err)
	}

	if err := os.MkdirAll(

		temporaryUploadDirectory,

		0755,
	); err != nil {

		t.Fatalf(

			"failed to create temporary upload directory: %v",

			err,
		)

	}

	uploadID := "550e8400-e29b-41d4-a716-446655440000"

	if _, err := db.Exec(`INSERT INTO temporary_uploads (upload_id, session_id, created_at) VALUES ($1, $2, $3)`, uploadID, sessionID, time.Now()); err != nil {

		t.Fatalf("failed to insert temporary upload record: %v", err)
	}

	tempPath := filepath.Join(

		temporaryUploadDirectory,

		uploadID+".jpg",
	)

	if err := os.WriteFile(

		tempPath,

		[]byte("test image"),

		0644,
	); err != nil {

		t.Fatalf(

			"failed to create temporary image: %v",

			err,
		)

	}

	t.Cleanup(func() {

		os.Remove(tempPath)

		_, _ = db.Exec(`DELETE FROM temporary_uploads WHERE upload_id = $1`, uploadID)
	})

	got, err := getTemporaryImage(
		db,
		uploadID,
		sessionID,
	)

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if got != tempPath {

		t.Fatalf(

			"expected path %q, got %q",

			tempPath,

			got,
		)

	}

}

// TestGetDatasetConfig verifies that submitted form values are translated

// into the application dataset configuration.

func TestGetDatasetConfig(t *testing.T) {

	form := url.Values{

		"model": {"random_forest_regressor"},

		"target": {"co2"},

		"featureSelection": {"manual"},

		"features": {"temperature", "humidity"},

		"configType": {"manual"},

		"n_estimators": {"25"},

		"max_depth": {"10"},

		"histograms": {"true"},

		"box_plots": {"true"},

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

		"model": {"none"},

		"histograms": {"true"},

		"box_plots": {"true"},

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

		"model": {"random_forest_regressor"},

		"configType": {"manual"},

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

		"model": {"random_forest_regressor"},

		"configType": {"manual"},

		"max_depth": {"not-a-number"},
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

// TestGetImageConfig verifies that submitted image form values are

// translated into the application image configuration.

func TestGetImageConfig(t *testing.T) {

	form := url.Values{

		"resize": {"true"},

		"resize_width": {"100"},

		"resize_height": {"200"},

		"compression": {"true"},

		"compression_quality": {"85"},

		"format_conversion": {"true"},

		"output_format": {"png"},

		"extract_metadata": {"true"},
	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/submit/image",

		strings.NewReader(form.Encode()),
	)

	req.Header.Set(

		"Content-Type",

		"application/x-www-form-urlencoded",
	)

	config, err := getImageConfig(req)

	if err != nil {

		t.Fatalf(

			"unexpected error: %v",

			err,
		)

	}

	if !config.Resize {

		t.Error("expected resize to be enabled")

	}

	if config.ResizeWidth != 100 {

		t.Errorf(

			"expected resize width 100, got %d",

			config.ResizeWidth,
		)

	}

	if config.ResizeHeight != 200 {

		t.Errorf(

			"expected resize height 200, got %d",

			config.ResizeHeight,
		)

	}

	if !config.Compression {

		t.Error("expected compression to be enabled")

	}

	if config.CompressionQuality != 85 {

		t.Errorf(

			"expected compression quality 85, got %d",

			config.CompressionQuality,
		)

	}

	if !config.FormatConversion {

		t.Error("expected format conversion to be enabled")

	}

	if config.OutputFormat != "png" {

		t.Errorf(

			"expected output format png, got %q",

			config.OutputFormat,
		)

	}

	if !config.ExtractMetadata {

		t.Error("expected metadata extraction to be enabled")

	}

}

// TestLoadDatasetResults verifies that dataset results JSON is decoded

// into the expected Go structure.

func TestLoadDatasetResults(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())

	key := "results.json"

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

	if err := store.Put(
		context.Background(),
		key,
		bytes.NewReader(fileData),
	); err != nil {
		t.Fatalf("failed to store test results: %v", err)
	}

	results, err := loadDatasetResults(
		store,
		key,
	)
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
	store := storage.NewLocalStorage(t.TempDir())

	_, err := loadDatasetResults(
		store,
		"missing.json",
	)

	if err == nil {
		t.Fatal("expected error for missing results file")
	}
}

// TestLoadModelResultsMissingFile verifies that a missing model-results

// file means no model was requested.

func TestLoadModelResultsMissingFile(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())

	jobID := int64(999999)

	results, err := loadModelResults(
		store,
		jobID,
	)
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

	handler := DownloadResultsHandler(nil, nil)

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

	handler := DownloadModelHandler(nil, nil)

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

	db := setupTestDatabase(t)

	handler := DatasetInspectionHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

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

	db := setupTestDatabase(t)

	handler := DatasetInspectionHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

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

	handler := VisualizationHandler(nil, nil)

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

	m := database.RunMigrations()

	if m == nil {

		t.Fatal("RunMigrations returned nil")

	}

	t.Cleanup(func() {

		database.CloseMigrations(m)

	})

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

		"uploadID": {uploadID},

		"model": {"none"},

		"featureSelection": {""},

		"configType": {""},
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

	store := storage.NewLocalStorage(t.TempDir())
	metrics := createTestMetrics()

	sessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate test session token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionToken); err != nil {
		t.Fatalf("failed to store test guest session: %v", err)
	}

	var sessionID int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionToken,
	).Scan(&sessionID); err != nil {
		t.Fatalf("failed to query session ID: %v", err)
	}

	if _, err := db.Exec(`
    INSERT INTO temporary_uploads (
        upload_id,
        session_id,
        created_at
    )
    VALUES ($1, $2, $3)
`,
		uploadID,
		sessionID,
		time.Now(),
	); err != nil {
		t.Fatalf("failed to insert temporary upload record: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionToken,
	})

	handler := DatasetSubmissionHandler(
		db,
		redisClient,
		store,
		metrics,
	)

	auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

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

	err = db.QueryRow(`

        SELECT id

        FROM jobs

        WHERE type = 'dataset'

        AND input_reference LIKE 'jobs/%/dataset.csv'

        ORDER BY id DESC

        LIMIT 1

    `).Scan(&jobID)

	if err != nil {

		t.Fatalf("failed to retrieve created job: %v", err)

	}

	t.Cleanup(func() {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)
	})

	_, _ = db.Exec(
		`DELETE FROM temporary_uploads WHERE upload_id = $1`,
		uploadID,
	)

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

// TestGetImageConfigInvalidResizeWidth verifies that invalid resize

// widths are rejected.

func TestGetImageConfigInvalidResizeWidth(t *testing.T) {

	form := url.Values{

		"resize": {"true"},

		"resize_width": {"0"},

		"resize_height": {"100"},
	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/submit/image",

		strings.NewReader(form.Encode()),
	)

	req.Header.Set(

		"Content-Type",

		"application/x-www-form-urlencoded",
	)

	_, err := getImageConfig(req)

	if err == nil {

		t.Fatal("expected invalid resize width error")

	}

}

// TestGetImageConfigInvalidResizeHeight verifies that invalid resize

// heights are rejected.

func TestGetImageConfigInvalidResizeHeight(t *testing.T) {

	form := url.Values{

		"resize": {"true"},

		"resize_width": {"100"},

		"resize_height": {"0"},
	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/submit/image",

		strings.NewReader(form.Encode()),
	)

	req.Header.Set(

		"Content-Type",

		"application/x-www-form-urlencoded",
	)

	_, err := getImageConfig(req)

	if err == nil {

		t.Fatal("expected invalid resize height error")

	}

}

// TestGetImageConfigInvalidCompressionQuality verifies that invalid

// compression quality values are rejected.

func TestGetImageConfigInvalidCompressionQuality(t *testing.T) {

	form := url.Values{

		"compression": {"true"},

		"compression_quality": {"101"},

		"output_format": {"jpeg"},
	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/submit/image",

		strings.NewReader(form.Encode()),
	)

	req.Header.Set(

		"Content-Type",

		"application/x-www-form-urlencoded",
	)

	_, err := getImageConfig(req)

	if err == nil {

		t.Fatal("expected invalid compression quality error")

	}

}

// TestGetImageConfigInvalidOutputFormat verifies that unsupported

// output formats are rejected.

func TestGetImageConfigInvalidOutputFormat(t *testing.T) {

	form := url.Values{

		"format_conversion": {"true"},

		"output_format": {"bmp"},
	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/submit/image",

		strings.NewReader(form.Encode()),
	)

	req.Header.Set(

		"Content-Type",

		"application/x-www-form-urlencoded",
	)

	_, err := getImageConfig(req)

	if err == nil {

		t.Fatal("expected invalid output format error")

	}

}

// TestImageUploadHandlerRejectsWrongMethod verifies that image uploads

// only accept POST requests.

func TestImageUploadHandlerRejectsWrongMethod(t *testing.T) {

	req := httptest.NewRequest(

		http.MethodGet,

		"/upload/image",

		nil,
	)

	rec := httptest.NewRecorder()
	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusMethodNotAllowed {

		t.Fatalf(

			"expected status %d, got %d",

			http.StatusMethodNotAllowed,

			rec.Code,
		)

	}

}

// TestImageUploadHandlerRequiresImage verifies that image uploads fail

// when no image is provided.

func TestImageUploadHandlerRequiresImage(t *testing.T) {

	req := httptest.NewRequest(

		http.MethodPost,

		"/upload/image",

		nil,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {

		t.Fatalf(

			"expected status %d, got %d",

			http.StatusBadRequest,

			rec.Code,
		)

	}

}

// TestImageUploadHandlerUploadsImage verifies that a supported image

// is saved to the temporary upload directory and receives an upload ID.

func TestImageUploadHandlerUploadsImage(t *testing.T) {

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(

		"imageFile",

		"test.jpg",
	)

	if err != nil {

		t.Fatalf(

			"failed to create form file: %v",

			err,
		)

	}

	// Generate a valid 1x1 JPEG for upload testing.
	imageDataBuffer := new(bytes.Buffer)
	testImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	testImage.Set(0, 0, color.RGBA{R: 255, A: 255})

	if err := jpeg.Encode(imageDataBuffer, testImage, nil); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}

	if _, err := part.Write(imageDataBuffer.Bytes()); err != nil {

		t.Fatalf(

			"failed to write image data: %v",

			err,
		)

	}

	if err := writer.Close(); err != nil {

		t.Fatalf(

			"failed to close multipart writer: %v",

			err,
		)

	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/upload/image",

		&body,
	)

	req.Header.Set(

		"Content-Type",

		writer.FormDataContentType(),
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusOK {

		t.Fatalf(

			"expected status %d, got %d: %s",

			http.StatusOK,

			rec.Code,

			rec.Body.String(),
		)

	}

	var response inspectionResponse

	if err := json.NewDecoder(

		rec.Body,
	).Decode(&response); err != nil {

		t.Fatalf(

			"failed to decode response: %v",

			err,
		)

	}

	if response.UploadID == "" {

		t.Fatal("expected upload ID")

	}

	matches, err := filepath.Glob(

		filepath.Join(

			temporaryUploadDirectory,

			response.UploadID+".jpg",
		),
	)

	if err != nil {

		t.Fatalf(

			"failed to search for uploaded image: %v",

			err,
		)

	}

	if len(matches) != 1 {

		t.Fatalf(

			"expected uploaded image to exist, found %d files",

			len(matches),
		)

	}

	t.Cleanup(func() {

		os.Remove(matches[0])

	})

}

// TestImageSubmissionHandlerEnqueuesJob verifies that submitting an image

// creates a queued image job, stores its configuration, and adds its ID

// to the Redis job stream.

func TestImageSubmissionHandlerEnqueuesJob(t *testing.T) {

	m := database.RunMigrations()

	if m == nil {

		t.Fatal("RunMigrations returned nil")

	}

	t.Cleanup(func() {

		database.CloseMigrations(m)

	})

	db := database.OpenDatabase()

	if err := db.Ping(); err != nil {

		db.Close()

		t.Skipf(

			"PostgreSQL is not available: %v",

			err,
		)

	}

	redisClient := redisclient.NewClient(

		&redisclient.Options{

			Addr: "localhost:6379",
		},
	)

	if err := redisClient.Ping(

		context.Background(),
	).Err(); err != nil {

		db.Close()

		redisClient.Close()

		t.Skipf(

			"Redis is not available: %v",

			err,
		)

	}

	t.Cleanup(func() {

		_ = redisClient.Close()

		_ = db.Close()

	})

	uploadID := uuid.New().String()

	tempPath := filepath.Join(

		temporaryUploadDirectory,

		uploadID+".jpg",
	)

	if err := os.MkdirAll(

		temporaryUploadDirectory,

		0755,
	); err != nil {

		t.Fatalf(

			"failed to create temporary upload directory: %v",

			err,
		)

	}

	// The submission handler only needs the temporary image to exist.

	if err := os.WriteFile(

		tempPath,

		[]byte("test image"),

		0644,
	); err != nil {

		t.Fatalf(

			"failed to create temporary image: %v",

			err,
		)

	}

	t.Cleanup(func() {

		_ = os.Remove(tempPath)

	})

	form := url.Values{

		"uploadID": {uploadID},

		"resize": {"true"},

		"resize_width": {"50"},

		"resize_height": {"50"},

		"compression": {"true"},

		"compression_quality": {"85"},

		"format_conversion": {"true"},

		"output_format": {"png"},

		"extract_metadata": {"true"},
	}

	req := httptest.NewRequest(

		http.MethodPost,

		"/submit/image",

		strings.NewReader(form.Encode()),
	)

	req.Header.Set(

		"Content-Type",

		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	store := storage.NewLocalStorage(t.TempDir())
	metrics := createTestMetrics()

	sessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate test session token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionToken); err != nil {
		t.Fatalf("failed to store test guest session: %v", err)
	}

	var sessionID int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionToken,
	).Scan(&sessionID); err != nil {
		t.Fatalf("failed to query session ID: %v", err)
	}

	if _, err := db.Exec(`
    INSERT INTO temporary_uploads (
        upload_id,
        session_id,
        created_at
    )
    VALUES ($1, $2, $3)
`,
		uploadID,
		sessionID,
		time.Now(),
	); err != nil {
		t.Fatalf("failed to insert temporary upload record: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionToken,
	})

	handler := ImageSubmissionHandler(
		db,
		redisClient,
		store,
		metrics,
	)

	auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {

		t.Fatalf(

			"expected status %d, got %d",

			http.StatusSeeOther,

			rec.Code,
		)

	}

	var jobID int64

	err = db.QueryRow(`

        SELECT id

        FROM jobs

        WHERE type = 'image'

        AND input_reference LIKE 'jobs/%/input.jpg'

        ORDER BY id DESC

        LIMIT 1

    `).Scan(&jobID)

	if err != nil {

		t.Fatalf(

			"failed to retrieve created image job: %v",

			err,
		)

	}

	t.Cleanup(func() {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)
	})

	_, _ = db.Exec(
		`DELETE FROM temporary_uploads WHERE upload_id = $1`,
		uploadID,
	)

	var inputReference string

	if err := db.QueryRow(

		`SELECT input_reference FROM jobs WHERE id = $1`,

		jobID,
	).Scan(&inputReference); err != nil {

		t.Fatalf(

			"failed to retrieve input reference: %v",

			err,
		)

	}

	if !strings.HasSuffix(

		inputReference,

		"/input.jpg",
	) {

		t.Fatalf(

			"unexpected input reference: %q",

			inputReference,
		)

	}

	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	configReader, err := store.Get(
		context.Background(),
		configKey,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve image config: %v",
			err,
		)
	}
	defer configReader.Close()

	configData, err := io.ReadAll(
		configReader,
	)
	if err != nil {
		t.Fatalf(
			"failed to read image config: %v",
			err,
		)
	}

	var config jobs.ImageConfig

	if err := json.Unmarshal(

		configData,

		&config,
	); err != nil {

		t.Fatalf(

			"failed to decode image config: %v",

			err,
		)

	}

	if !config.Resize {

		t.Error("expected resize to be enabled")

	}

	if config.ResizeWidth != 50 {

		t.Errorf(

			"expected resize width 50, got %d",

			config.ResizeWidth,
		)

	}

	if config.OutputFormat != "png" {

		t.Errorf(

			"expected output format png, got %q",

			config.OutputFormat,
		)

	}

	streams, err := redisClient.XRange(

		context.Background(),

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

	for _, message := range streams {

		value, ok := message.Values["job_id"].(string)

		if !ok {

			continue

		}

		if value == fmt.Sprintf("%d", jobID) {

			found = true

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

func createTestMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()
	return metrics.NewMetrics(registry)
}

// TestValidateImageTypeAcceptsSupportedFormats verifies that supported image
// content types are accepted.
func TestValidateImageTypeAcceptsSupportedFormats(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "jpeg",
			data: []byte{
				0xff, 0xd8, 0xff, 0xe0,
				0x00, 0x10, 'J', 'F', 'I', 'F',
			},
		},
		{
			name: "png",
			data: []byte{
				0x89, 'P', 'N', 'G',
				0x0d, 0x0a, 0x1a, 0x0a,
			},
		},
		{
			name: "gif",
			data: []byte("GIF89a"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.CreateTemp("", "image-*")
			if err != nil {
				t.Fatalf("failed to create temporary file: %v", err)
			}
			t.Cleanup(func() {
				os.Remove(file.Name())
			})
			t.Cleanup(func() {
				file.Close()
			})

			if _, err := file.Write(tt.data); err != nil {
				t.Fatalf("failed to write test image: %v", err)
			}

			if _, err := file.Seek(0, 0); err != nil {
				t.Fatalf("failed to reset test file: %v", err)
			}

			if err := validateImageType(file); err != nil {
				t.Fatalf("expected image type to be accepted: %v", err)
			}
		})
	}
}

// TestValidateImageTypeAcceptsWebP verifies that valid WebP content is accepted.
func TestValidateImageTypeAcceptsWebP(t *testing.T) {
	data, err := os.ReadFile("testdata/test.webp")
	if err != nil {
		t.Fatalf("failed to read WebP fixture: %v", err)
	}

	file, err := os.CreateTemp("", "image-*")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}

	t.Cleanup(func() {
		file.Close()
		os.Remove(file.Name())
	})

	if _, err := file.Write(data); err != nil {
		t.Fatalf("failed to write WebP fixture: %v", err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to reset test file: %v", err)
	}

	if err := validateImageType(file); err != nil {
		t.Fatalf("expected WebP to be accepted: %v", err)
	}
}

// TestValidateImageTypeRejectsNonImage verifies that non-image content is rejected.
func TestValidateImageTypeRejectsNonImage(t *testing.T) {
	file, err := os.CreateTemp("", "image-*")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(file.Name())
	})
	t.Cleanup(func() {
		file.Close()
	})

	if _, err := file.Write([]byte("this is not an image")); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("failed to reset test file: %v", err)
	}

	if err := validateImageType(file); err == nil {
		t.Fatal("expected non-image content to be rejected")
	}
}

// TestValidateImageTypeResetsFilePointer verifies that the file is positioned
// at the beginning after validation.
func TestValidateImageTypeResetsFilePointer(t *testing.T) {
	file, err := os.CreateTemp("", "image-*")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(file.Name())
	})
	t.Cleanup(func() {
		file.Close()
	})

	imageData := []byte{
		0xff, 0xd8, 0xff, 0xe0,
		0x00, 0x10, 'J', 'F', 'I', 'F',
	}

	if _, err := file.Write(imageData); err != nil {
		t.Fatalf("failed to write test image: %v", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("failed to reset test file: %v", err)
	}

	if err := validateImageType(file); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	position, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatalf("failed to get file position: %v", err)
	}

	if position != 0 {
		t.Fatalf("expected file position 0, got %d", position)
	}
}

func makeMultipartRequest(
	t *testing.T,
	fieldName string,
	filename string,
	data []byte,
) *http.Request {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write file data: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/upload",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	return req
}

func TestImageUploadHandlerRejectsEmptyFile(t *testing.T) {
	req := makeMultipartRequest(
		t,
		"imageFile",
		"empty.jpg",
		nil,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestImageUploadHandlerRejectsUnsupportedExtension(t *testing.T) {
	data := loadTestFixture(t, "valid.jpg")

	req := makeMultipartRequest(
		t,
		"imageFile",
		"image.txt",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestImageUploadHandlerRejectsNonImageContent(t *testing.T) {
	data := loadTestFixture(t, "not_image.jpg")

	req := makeMultipartRequest(
		t,
		"imageFile",
		"not_image.jpg",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestImageUploadHandlerAcceptsSupportedFormats(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		fixture  string
	}{
		{
			name:     "JPEG",
			filename: "test.jpg",
			fixture:  "valid.jpg",
		},
		{
			name:     "PNG",
			filename: "test.png",
			fixture:  "valid.png",
		},
		{
			name:     "GIF",
			filename: "test.gif",
			fixture:  "valid.gif",
		},
		{
			name:     "WebP",
			filename: "test.webp",
			fixture:  "valid.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := loadTestFixture(t, tt.fixture)

			req := makeMultipartRequest(
				t,
				"imageFile",
				tt.filename,
				data,
			)

			rec := httptest.NewRecorder()

			db := setupTestDatabase(t)
			handler := ImageUploadHandler(db)

			serveWithTestSession(t, db, handler, rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusOK,
					rec.Code,
					rec.Body.String(),
				)
			}

			var response inspectionResponse
			if err := json.NewDecoder(
				rec.Body,
			).Decode(&response); err != nil {
				t.Fatalf(
					"failed to decode response: %v",
					err,
				)
			}

			if response.UploadID == "" {
				t.Fatal("expected upload ID")
			}

			t.Cleanup(func() {
				matches, err := filepath.Glob(
					filepath.Join(
						temporaryUploadDirectory,
						response.UploadID+"*",
					),
				)
				if err != nil {
					return
				}

				for _, match := range matches {
					os.Remove(match)
				}
			})
		})
	}
}

func TestImageUploadHandlerRejectsOversizedFile(t *testing.T) {
	data := bytes.Repeat(
		[]byte("A"),
		MaxUploadSize+1,
	)

	req := makeMultipartRequest(
		t,
		"imageFile",
		"large.jpg",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestDatasetInspectionHandlerRejectsEmptyFile(t *testing.T) {
	req := makeMultipartRequest(
		t,
		"csvFile",
		"empty.csv",
		nil,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)

	handler := DatasetInspectionHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestDatasetInspectionHandlerRejectsWrongExtension(t *testing.T) {
	data := loadTestFixture(t, "valid.csv")

	req := makeMultipartRequest(
		t,
		"csvFile",
		"dataset.txt",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)

	handler := DatasetInspectionHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestDatasetInspectionHandlerRejectsMalformedCSV(t *testing.T) {
	data := loadTestFixture(t, "malformed.csv")

	req := makeMultipartRequest(
		t,
		"csvFile",
		"malformed.csv",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)

	handler := DatasetInspectionHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestDatasetInspectionHandlerRejectsOversizedFile(t *testing.T) {
	data := bytes.Repeat(
		[]byte("A"),
		MaxUploadSize+1,
	)

	req := makeMultipartRequest(
		t,
		"csvFile",
		"large.csv",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)

	handler := DatasetInspectionHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func makeRouteMultipartRequest(
	t *testing.T,
	routeFilename string,
	routeData []byte,
	distanceFilename string,
	distanceData []byte,
) *http.Request {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	routePart, err := writer.CreateFormFile(
		"routeFile",
		routeFilename,
	)
	if err != nil {
		t.Fatalf("failed to create route form file: %v", err)
	}

	if _, err := routePart.Write(routeData); err != nil {
		t.Fatalf("failed to write route data: %v", err)
	}

	distancePart, err := writer.CreateFormFile(
		"distanceFile",
		distanceFilename,
	)
	if err != nil {
		t.Fatalf("failed to create distance form file: %v", err)
	}

	if _, err := distancePart.Write(distanceData); err != nil {
		t.Fatalf("failed to write distance data: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/upload/route",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	return req
}

func TestRouteUploadHandlerRejectsEmptyRouteFile(t *testing.T) {
	distanceData := loadTestFixture(t, "valid.csv")

	req := makeRouteMultipartRequest(
		t,
		"route.csv",
		nil,
		"distances.csv",
		distanceData,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := RouteUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestRouteUploadHandlerRejectsEmptyDistanceFile(t *testing.T) {
	routeData := loadTestFixture(t, "valid.csv")

	req := makeRouteMultipartRequest(
		t,
		"route.csv",
		routeData,
		"distances.csv",
		nil,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := RouteUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestRouteUploadHandlerRejectsWrongRouteExtension(t *testing.T) {
	routeData := loadTestFixture(t, "valid.csv")
	distanceData := loadTestFixture(t, "valid.csv")

	req := makeRouteMultipartRequest(
		t,
		"route.txt",
		routeData,
		"distances.csv",
		distanceData,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := RouteUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestRouteUploadHandlerRejectsWrongDistanceExtension(t *testing.T) {
	routeData := loadTestFixture(t, "valid.csv")
	distanceData := loadTestFixture(t, "valid.csv")

	req := makeRouteMultipartRequest(
		t,
		"route.csv",
		routeData,
		"distances.txt",
		distanceData,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := RouteUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func loadTestFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf(
			"failed to read test fixture %q: %v",
			name,
			err,
		)
	}

	return data
}

func TestValidateImageTypeAcceptsMalformedJPEGSignature(t *testing.T) {
	data := loadTestFixture(t, "malformed.jpg")

	tempFile, err := os.CreateTemp(t.TempDir(), "test-*.jpg")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to rewind temporary file: %v", err)
	}

	err = validateImageType(tempFile)
	if err != nil {
		t.Fatalf(
			"expected MIME validation to accept JPEG signature, got: %v",
			err,
		)
	}
}

func TestImageUploadHandlerAcceptsMalformedJPEGSignature(t *testing.T) {
	data := loadTestFixture(t, "malformed.jpg")

	req := makeMultipartRequest(
		t,
		"imageFile",
		"malformed.jpg",
		data,
	)

	rec := httptest.NewRecorder()

	db := setupTestDatabase(t)
	handler := ImageUploadHandler(db)

	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			rec.Code,
			rec.Body.String(),
		)
	}

	var response inspectionResponse

	if err := json.NewDecoder(
		rec.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.UploadID == "" {
		t.Fatal("expected upload ID")
	}

	t.Cleanup(func() {
		matches, err := filepath.Glob(
			filepath.Join(
				temporaryUploadDirectory,
				response.UploadID+"*",
			),
		)
		if err != nil {
			return
		}

		for _, match := range matches {
			os.Remove(match)
		}
	})
}

func TestGuestSessionHandler(t *testing.T) {
	m := database.RunMigrations()

	if m == nil {
		t.Fatal("RunMigrations returned nil")
	}

	t.Cleanup(func() {
		database.CloseMigrations(m)
	})

	db := database.OpenDatabase()
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL is not available: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/guest",
		nil,
	)

	rec := httptest.NewRecorder()

	handler := GuestSessionHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf("expected redirect to /, got %q", location)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf(
			"expected 1 response cookie, got %d",
			len(cookies),
		)
	}

	cookie := cookies[0]

	if cookie.Name != "session_id" {
		t.Fatalf(
			"expected cookie name session_id, got %q",
			cookie.Name,
		)
	}

	if cookie.Value == "" {
		t.Fatal("expected session cookie to contain a token")
	}

	if !cookie.HttpOnly {
		t.Fatal("expected session cookie to be HttpOnly")
	}

	if !cookie.Secure {
		t.Fatal("expected session cookie to be Secure")
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf(
			"expected SameSite=Lax, got %v",
			cookie.SameSite,
		)
	}

	if cookie.Path != "/" {
		t.Fatalf(
			"expected cookie path /, got %q",
			cookie.Path,
		)
	}

	var count int

	err := db.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE token = $1",
		cookie.Value,
	).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify stored session: %v", err)
	}

	if count != 1 {
		t.Fatalf(
			"expected exactly 1 stored session, got %d",
			count,
		)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM sessions WHERE token = $1",
			cookie.Value,
		)
	})
}

func TestGuestSessionHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/guest",
		nil,
	)

	rec := httptest.NewRecorder()

	handler := GuestSessionHandler(nil)
	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}
}

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

func serveWithTestSession(
	t *testing.T,
	db *sql.DB,
	handler http.Handler,
	rec http.ResponseWriter,
	req *http.Request,
) string {
	t.Helper()

	sessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate test session token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionToken); err != nil {
		t.Fatalf("failed to store test guest session: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionToken,
	})

	auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

	return sessionToken
}

func TestGetOwnedJob(t *testing.T) {
	db := setupTestDatabase(t)

	// Create owner/session A.
	sessionTokenA, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session A token: %v", err)
	}
	if err := auth.StoreGuestSession(db, sessionTokenA); err != nil {
		t.Fatalf("failed to store session A: %v", err)
	}

	var sessionIDA int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenA,
	).Scan(&sessionIDA); err != nil {
		t.Fatalf("failed to retrieve session A ID: %v", err)
	}

	// Create owner/session B.
	sessionTokenB, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session B token: %v", err)
	}
	if err := auth.StoreGuestSession(db, sessionTokenB); err != nil {
		t.Fatalf("failed to store session B: %v", err)
	}

	var sessionIDB int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenB,
	).Scan(&sessionIDB); err != nil {
		t.Fatalf("failed to retrieve session B ID: %v", err)
	}

	ownerA := jobs.JobOwner{
		SessionID: &sessionIDA,
	}

	jobID, err := jobs.CreateJob(db, "dataset", ownerA)
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM jobs WHERE id = $1`, jobID)
		_, _ = db.Exec(`DELETE FROM sessions WHERE id IN ($1, $2)`, sessionIDA, sessionIDB)
	})

	t.Run("owner can access own job", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/results?id="+fmt.Sprintf("%d", jobID),
			nil,
		)
		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: sessionTokenA,
		})

		auth.SessionMiddleware(
			db,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, err := getOwnedJob(r, db, jobID)
				if err != nil {
					t.Fatalf("getOwnedJob returned error: %v", err)
				}

				if got.ID != jobID {
					t.Fatalf("expected job ID %d, got %d", jobID, got.ID)
				}
			}),
		).ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("different owner cannot access job", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/results?id="+fmt.Sprintf("%d", jobID),
			nil,
		)
		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: sessionTokenB,
		})

		rec := httptest.NewRecorder()

		auth.SessionMiddleware(
			db,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := getOwnedJob(r, db, jobID)
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf(
						"expected sql.ErrNoRows, got %v",
						err,
					)
				}
			}),
		).ServeHTTP(rec, req)
	})
}

func TestResultsHandlerOwnership(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	store := storage.NewLocalStorage(t.TempDir())

	// Create owner/session A.
	sessionTokenA, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session A token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenA); err != nil {
		t.Fatalf("failed to store session A: %v", err)
	}

	var sessionIDA int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenA,
	).Scan(&sessionIDA); err != nil {
		t.Fatalf("failed to retrieve session A ID: %v", err)
	}

	// Create owner/session B.
	sessionTokenB, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session B token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenB); err != nil {
		t.Fatalf("failed to store session B: %v", err)
	}

	var sessionIDB int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenB,
	).Scan(&sessionIDB); err != nil {
		t.Fatalf("failed to retrieve session B ID: %v", err)
	}

	ownerA := jobs.JobOwner{
		SessionID: &sessionIDA,
	}

	jobID, err := jobs.CreateJob(db, "dataset", ownerA)
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM jobs WHERE id = $1`, jobID)
		_, _ = db.Exec(
			`DELETE FROM sessions WHERE id IN ($1, $2)`,
			sessionIDA,
			sessionIDB,
		)
	})

	handler := ResultsHandler(db, store)

	t.Run("owner can access own results", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/results?id="+fmt.Sprintf("%d", jobID),
			nil,
		)

		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: sessionTokenA,
		})

		rec := httptest.NewRecorder()

		auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

		// The job belongs to this session, so it must not be rejected
		// by the ownership check.
		if rec.Code == http.StatusNotFound {
			t.Fatalf("owner received 404 for own job")
		}
	})

	t.Run("different session cannot access results", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/results?id="+fmt.Sprintf("%d", jobID),
			nil,
		)

		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: sessionTokenB,
		})

		rec := httptest.NewRecorder()

		auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf(
				"expected 404 for job owned by different session, got %d",
				rec.Code,
			)
		}
	})
}

func TestProtectedJobHandlersRejectDifferentSession(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	store := storage.NewLocalStorage(t.TempDir())

	// Create owner/session A.
	sessionTokenA, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session A token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenA); err != nil {
		t.Fatalf("failed to store session A: %v", err)
	}

	var sessionIDA int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenA,
	).Scan(&sessionIDA); err != nil {
		t.Fatalf("failed to retrieve session A ID: %v", err)
	}

	// Create different/session B.
	sessionTokenB, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session B token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenB); err != nil {
		t.Fatalf("failed to store session B: %v", err)
	}

	var sessionIDB int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenB,
	).Scan(&sessionIDB); err != nil {
		t.Fatalf("failed to retrieve session B ID: %v", err)
	}

	ownerA := jobs.JobOwner{
		SessionID: &sessionIDA,
	}

	jobID, err := jobs.CreateJob(db, "dataset", ownerA)
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM jobs WHERE id = $1`, jobID)
		_, _ = db.Exec(
			`DELETE FROM sessions WHERE id IN ($1, $2)`,
			sessionIDA,
			sessionIDB,
		)
	})

	type protectedHandler struct {
		name    string
		handler http.Handler
		path    string
	}

	handlers := []protectedHandler{
		{
			name:    "visualization",
			handler: VisualizationHandler(db, store),
			path:    fmt.Sprintf("/results/visualization?id=%d&type=feature_distributions", jobID),
		},
		{
			name:    "image result",
			handler: ImageResultHandler(db, store),
			path:    fmt.Sprintf("/results/image?id=%d&type=original", jobID),
		},
		{
			name:    "result download",
			handler: DownloadResultsHandler(db, store),
			path:    fmt.Sprintf("/results/download?id=%d", jobID),
		},
		{
			name:    "model download",
			handler: DownloadModelHandler(db, store),
			path:    fmt.Sprintf("/results/download/model?id=%d", jobID),
		},
		{
			name:    "image metadata download",
			handler: DownloadImageMetadataHandler(db, store),
			path:    fmt.Sprintf("/results/download/metadata?id=%d", jobID),
		},
	}

	for _, endpoint := range handlers {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				endpoint.path,
				nil,
			)

			req.AddCookie(&http.Cookie{
				Name:  auth.SessionCookieName,
				Value: sessionTokenB,
			})

			rec := httptest.NewRecorder()

			auth.SessionMiddleware(db, endpoint.handler).ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf(
					"expected 404 when accessing another session's job, got %d",
					rec.Code,
				)
			}
		})
	}
}

func TestDownloadResultsHandlerRejectsDifferentSession(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	store := storage.NewLocalStorage(t.TempDir())

	// Create owner/session A.
	sessionTokenA, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session A token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenA); err != nil {
		t.Fatalf("failed to store session A: %v", err)
	}

	var sessionIDA int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenA,
	).Scan(&sessionIDA); err != nil {
		t.Fatalf("failed to retrieve session A ID: %v", err)
	}

	// Create different/session B.
	sessionTokenB, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session B token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenB); err != nil {
		t.Fatalf("failed to store session B: %v", err)
	}

	var sessionIDB int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenB,
	).Scan(&sessionIDB); err != nil {
		t.Fatalf("failed to retrieve session B ID: %v", err)
	}

	owner := jobs.JobOwner{
		SessionID: &sessionIDA,
	}

	jobID, err := jobs.CreateJob(db, "dataset", owner)
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	resultKey := fmt.Sprintf("jobs/%d/results.json", jobID)
	resultData := []byte(`{"secret":"session A data"}`)

	if err := store.Put(
		context.Background(),
		resultKey,
		bytes.NewReader(resultData),
	); err != nil {
		t.Fatalf("failed to store test result: %v", err)
	}

	// Mark the job completed and associate the result file.
	_, err = db.Exec(
		`UPDATE jobs
		 SET status = 'completed',
		     result_reference = $1
		 WHERE id = $2`,
		resultKey,
		jobID,
	)
	if err != nil {
		t.Fatalf("failed to update test job: %v", err)
	}

	t.Cleanup(func() {
		_ = jobs.DeleteJob(db, jobID, store)
		_, _ = db.Exec(
			`DELETE FROM sessions WHERE id IN ($1, $2)`,
			sessionIDA,
			sessionIDB,
		)
	})

	handler := DownloadResultsHandler(db, store)

	t.Run("owner can download result", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/results/download?id=%d", jobID),
			nil,
		)
		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: sessionTokenA,
		})

		rec := httptest.NewRecorder()

		auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf(
				"expected owner to receive 200, got %d",
				rec.Code,
			)
		}

		if rec.Body.String() != string(resultData) {
			t.Fatalf(
				"expected result data %q, got %q",
				string(resultData),
				rec.Body.String(),
			)
		}
	})

	t.Run("different session cannot download result", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/results/download?id=%d", jobID),
			nil,
		)
		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: sessionTokenB,
		})

		rec := httptest.NewRecorder()

		auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf(
				"expected different session to receive 404, got %d",
				rec.Code,
			)
		}

		if rec.Body.String() == string(resultData) {
			t.Fatal("different session received the protected result data")
		}
	})
}

func TestGetOwnedJobRejectsDifferentUser(t *testing.T) {
	db := setupTestDatabase(t)

	userAName := "test-user-a-" + uuid.NewString()
	userBName := "test-user-b-" + uuid.NewString()
	// Create user A.
	var userIDA int64
	err := db.QueryRow(`
	INSERT INTO users (username, password_hash, created_at)
	VALUES ($1, $2, $3)
	RETURNING id
`, userAName, "test-hash-a", time.Now()).Scan(&userIDA)
	if err != nil {
		t.Fatalf("failed to create user A: %v", err)
	}

	// Create user B.
	var userIDB int64
	err = db.QueryRow(`
	INSERT INTO users (username, password_hash, created_at)
	VALUES ($1, $2, $3)
	RETURNING id
`, userBName, "test-hash-b", time.Now()).Scan(&userIDB)
	if err != nil {
		t.Fatalf("failed to create user B: %v", err)
	}

	// Create an authenticated session for user A.
	sessionTokenA, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session A token: %v", err)
	}

	var sessionIDA int64
	err = db.QueryRow(`
		INSERT INTO sessions (token, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, sessionTokenA, userIDA, time.Now().Add(24*time.Hour), time.Now()).Scan(&sessionIDA)
	if err != nil {
		t.Fatalf("failed to create session A: %v", err)
	}

	// Create an authenticated session for user B.
	sessionTokenB, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session B token: %v", err)
	}

	_, err = db.Exec(`
	INSERT INTO sessions (token, user_id, expires_at, created_at)
	VALUES ($1, $2, $3, $4)
	`, sessionTokenB, userIDB, time.Now().Add(24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("failed to create session B: %v", err)
	}

	// Create a job owned by user A.
	ownerA := jobs.JobOwner{
		UserID: &userIDA,
	}

	store := storage.NewLocalStorage(t.TempDir())

	jobID, err := jobs.CreateJob(db, "dataset", ownerA)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	t.Cleanup(func() {
		_ = jobs.DeleteJob(db, jobID, store)
	})

	// User A should be able to access the job.
	reqA := httptest.NewRequest(
		http.MethodGet,
		"/results?id="+fmt.Sprint(jobID),
		nil,
	)
	reqA.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionTokenA,
	})

	var recA httptest.ResponseRecorder
	auth.SessionMiddleware(
		db,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			job, err := getOwnedJob(r, db, jobID)
			if err != nil {
				t.Fatalf("user A could not access their own job: %v", err)
			}
			if job.ID != jobID {
				t.Fatalf("expected job ID %d, got %d", jobID, job.ID)
			}
			w.WriteHeader(http.StatusOK)
		}),
	).ServeHTTP(&recA, reqA)

	if recA.Code != http.StatusOK {
		t.Fatalf("expected user A status %d, got %d", http.StatusOK, recA.Code)
	}

	// User B should not be able to access user A's job.
	reqB := httptest.NewRequest(
		http.MethodGet,
		"/results?id="+fmt.Sprint(jobID),
		nil,
	)
	reqB.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionTokenB,
	})

	var recB httptest.ResponseRecorder
	auth.SessionMiddleware(
		db,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := getOwnedJob(r, db, jobID)
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf(
					"expected sql.ErrNoRows for cross-user access, got %v",
					err,
				)
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
	).ServeHTTP(&recB, reqB)

	if recB.Code != http.StatusNotFound {
		t.Fatalf(
			"expected user B status %d, got %d",
			http.StatusNotFound,
			recB.Code,
		)
	}
}

func TestCrossSessionGuestAccess(t *testing.T) {
	db := setupTestDatabase(t)

	// Create guest session A.
	sessionTokenA, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session A token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenA); err != nil {
		t.Fatalf("failed to store session A: %v", err)
	}

	var sessionIDA int64
	if err := db.QueryRow(
		`SELECT id FROM sessions WHERE token = $1`,
		sessionTokenA,
	).Scan(&sessionIDA); err != nil {
		t.Fatalf("failed to retrieve session A ID: %v", err)
	}

	// Create guest session B.
	sessionTokenB, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session B token: %v", err)
	}

	if err := auth.StoreGuestSession(db, sessionTokenB); err != nil {
		t.Fatalf("failed to store session B: %v", err)
	}

	// Create a job owned by guest session A.
	ownerA := jobs.JobOwner{
		SessionID: &sessionIDA,
	}

	jobID, err := jobs.CreateJob(db, "dataset", ownerA)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	store := storage.NewLocalStorage(t.TempDir())
	t.Cleanup(func() {
		_ = jobs.DeleteJob(db, jobID, store)
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := getOwnedJob(r, db, jobID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}

			t.Fatalf("unexpected ownership error: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Session A should be able to access its own job.
	reqA := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/results?id=%d", jobID),
		nil,
	)
	reqA.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionTokenA,
	})

	recA := httptest.NewRecorder()

	auth.SessionMiddleware(db, handler).ServeHTTP(recA, reqA)

	if recA.Code != http.StatusOK {
		t.Fatalf(
			"expected session A status %d, got %d",
			http.StatusOK,
			recA.Code,
		)
	}

	// Session B must not be able to access session A's job.
	reqB := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/results?id=%d", jobID),
		nil,
	)
	reqB.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionTokenB,
	})

	recB := httptest.NewRecorder()

	auth.SessionMiddleware(db, handler).ServeHTTP(recB, reqB)

	if recB.Code != http.StatusNotFound {
		t.Fatalf(
			"expected session B status %d, got %d",
			http.StatusNotFound,
			recB.Code,
		)
	}
}

func TestRegisterHandlerGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()

	handler := RegisterHandler(nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Create an Account") {
		t.Error("expected registration page to contain 'Create an Account'")
	}

	if !strings.Contains(body, `name="username"`) {
		t.Error("expected registration page to contain username field")
	}

	if !strings.Contains(body, `name="password"`) {
		t.Error("expected registration page to contain password field")
	}

	if !strings.Contains(body, `action="/register"`) {
		t.Error("expected registration form to submit to /register")
	}
}

func TestLoginHandler(t *testing.T) {
	db := setupTestDatabase(t)

	username := "testuser-" + uuid.NewString()
	password := "testpassword"

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to hash test password: %v", err)
	}

	var userID int64
	err = db.QueryRow(
		`
		INSERT INTO users (
			username,
			password_hash,
			created_at
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		username,
		string(passwordHash),
		time.Now(),
	).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	form := url.Values{
		"username": {username},
		"password": {password},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	handler := LoginHandler(db)
	serveWithTestSession(t, db, handler, rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf(
			"expected redirect to /, got %q",
			location,
		)
	}

	cookie := req.Cookies()[0]
	var sessionUserID *int64

	err = db.QueryRow(
		`
		SELECT user_id
		FROM sessions
		WHERE token = $1
		`,
		cookie.Value,
	).Scan(&sessionUserID)
	if err != nil {
		t.Fatalf("failed to query logged-in session: %v", err)
	}

	if sessionUserID == nil {
		t.Fatal("expected session to be associated with a user")
	}

	if *sessionUserID != userID {
		t.Fatalf(
			"expected session user ID %d, got %d",
			userID,
			*sessionUserID,
		)
	}
}

func TestLogoutHandler(t *testing.T) {
	db := setupTestDatabase(t)

	username := "logout-test-user-" + uuid.NewString()
	password := "logout-test-password"

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to hash test password: %v", err)
	}

	var userID int64
	err = db.QueryRow(
		`
		INSERT INTO users (
			username,
			password_hash,
			created_at
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		username,
		string(passwordHash),
		time.Now(),
	).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessionToken, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate session token: %v", err)
	}

	var sessionID int64
	err = db.QueryRow(
		`
		INSERT INTO sessions (
			token,
			user_id,
			created_at,
			expires_at
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		sessionToken,
		userID,
		time.Now(),
		time.Now().Add(24*time.Hour),
	).Scan(&sessionID)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/logout",
		nil,
	)
	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionToken,
	})

	rec := httptest.NewRecorder()

	handler := LogoutHandler(db)
	auth.SessionMiddleware(db, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf(
			"expected redirect to /, got %q",
			location,
		)
	}

	var loggedInUserID *int64
	err = db.QueryRow(
		`
		SELECT user_id
		FROM sessions
		WHERE id = $1
		`,
		sessionID,
	).Scan(&loggedInUserID)
	if err != nil {
		t.Fatalf("failed to query session after logout: %v", err)
	}

	if loggedInUserID != nil {
		t.Fatalf(
			"expected session user ID to be NULL after logout, got %d",
			*loggedInUserID,
		)
	}
}

func TestResultsHandlerDoesNotLeakDatabaseErrors(t *testing.T) {
	db := database.OpenDatabase()
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("PostgreSQL is not available: %v", err)
	}
	db.Close()

	req := httptest.NewRequest(
		http.MethodGet,
		"/results?id=1",
		nil,
	)

	rec := httptest.NewRecorder()

	handler := ResultsHandler(
		db,
		nil,
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Internal server error") {
		t.Fatalf(
			"expected generic error message, got %q",
			body,
		)
	}

	for _, leaked := range []string{
		"sql.ErrConnDone",
		"database/sql",
		"connection",
		"postgres",
	} {
		if strings.Contains(body, leaked) {
			t.Fatalf(
				"response appears to leak internal error information: %q",
				body,
			)
		}
	}
}

func TestResultsHandlerReturnsSafeValidationError(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/results?id=not-a-number",
		nil,
	)

	rec := httptest.NewRecorder()

	handler := ResultsHandler(nil, nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if body := rec.Body.String(); body != "Invalid job ID\n" {
		t.Fatalf(
			"unexpected response body: %q",
			body,
		)
	}
}
