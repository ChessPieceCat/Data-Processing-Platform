package server

import (
	"bytes"
	"io"

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

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"

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

// TestGetTemporaryImage verifies validation of temporary image upload IDs.

func TestGetTemporaryImage(t *testing.T) {

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

			_, err := getTemporaryImage(tt.uploadID)

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

	})

	got, err := getTemporaryImage(uploadID)

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

	handler := DatasetSubmissionHandler(
		db,
		redisClient,
		store,
	)

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

	ImageUploadHandler(rec, req)

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

	ImageUploadHandler(rec, req)

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

	ImageUploadHandler(rec, req)

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

	handler := ImageSubmissionHandler(
		db,
		redisClient,
		store,
	)

	handler(rec, req)

	if rec.Code != http.StatusSeeOther {

		t.Fatalf(

			"expected status %d, got %d",

			http.StatusSeeOther,

			rec.Code,
		)

	}

	var jobID int64

	err := db.QueryRow(`

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

	var status string

	if err := db.QueryRow(

		`SELECT status FROM jobs WHERE id = $1`,

		jobID,
	).Scan(&status); err != nil {

		t.Fatalf(

			"failed to retrieve job status: %v",

			err,
		)

	}

	if status != "queued" {

		t.Fatalf(

			"expected job status queued, got %q",

			status,
		)

	}

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
