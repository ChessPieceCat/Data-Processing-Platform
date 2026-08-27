package server

import (
	"bytes"

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

	tempDir := t.TempDir()

	resultPath := filepath.Join(tempDir, "results.json")

	data := map[string]interface{}{

		"num_rows": 10,

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

        AND input_reference LIKE 'uploads/%/dataset.csv'

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

	handler := ImageSubmissionHandler(

		db,

		redisClient,
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

        AND input_reference LIKE 'uploads/%/input.jpg'

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

		_ = jobs.DeleteJob(db, jobID)

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

	configPath := filepath.Join(

		"uploads",

		fmt.Sprintf("%d", jobID),

		"config.json",
	)

	configData, err := os.ReadFile(configPath)

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

// TestGetTemporaryRouteFiles verifies validation of temporary route upload IDs.
func TestGetTemporaryRouteFiles(t *testing.T) {
	tests := []struct {
		name     string
		uploadID string
		wantErr  error
	}{
		{
			name:     "empty upload ID",
			uploadID: "",
			wantErr:  errRouteNotUploaded,
		},
		{
			name:     "invalid upload ID",
			uploadID: "not-a-uuid",
			wantErr:  errInvalidUploadID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := getTemporaryRouteFiles(tt.uploadID)

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

// TestGetTemporaryRouteFilesExistingFiles verifies that both temporary route
// files are returned when they exist.
func TestGetTemporaryRouteFilesExistingFiles(t *testing.T) {
	if err := os.MkdirAll(
		temporaryUploadDirectory,
		0755,
	); err != nil {
		t.Fatalf(
			"failed to create temporary upload directory: %v",
			err,
		)
	}

	uploadID := uuid.New().String()

	routePath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_route.csv",
	)

	distancePath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_distances.csv",
	)

	if err := os.WriteFile(
		routePath,
		[]byte(
			"id,name,demand,priority,window_start,window_end\n"+
				"1,Warehouse,0,0,08:00,17:00\n",
		),
		0644,
	); err != nil {
		t.Fatalf("failed to create route CSV: %v", err)
	}

	if err := os.WriteFile(
		distancePath,
		[]byte(
			"location,Warehouse\n"+
				"Warehouse,0\n",
		),
		0644,
	); err != nil {
		os.Remove(routePath)
		t.Fatalf("failed to create distance CSV: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(routePath)
		os.Remove(distancePath)
	})

	gotRoutePath, gotDistancePath, err := getTemporaryRouteFiles(uploadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotRoutePath != routePath {
		t.Fatalf(
			"expected route path %q, got %q",
			routePath,
			gotRoutePath,
		)
	}

	if gotDistancePath != distancePath {
		t.Fatalf(
			"expected distance path %q, got %q",
			distancePath,
			gotDistancePath,
		)
	}
}

// TestRouteUploadHandlerRejectsWrongMethod verifies that route uploads only
// accept POST requests.
func TestRouteUploadHandlerRejectsWrongMethod(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/upload/route",
		nil,
	)

	rec := httptest.NewRecorder()

	RouteUploadHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}
}

// TestRouteUploadHandlerRequiresRouteFile verifies that a route file is
// required.
func TestRouteUploadHandlerRequiresRouteFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(
		"distanceFile",
		"distances.csv",
	)
	if err != nil {
		t.Fatalf(
			"failed to create distance form file: %v",
			err,
		)
	}

	if _, err := part.Write(
		[]byte(
			"location,Warehouse\n" +
				"Warehouse,0\n",
		),
	); err != nil {
		t.Fatalf(
			"failed to write distance file: %v",
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
		"/upload/route",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	rec := httptest.NewRecorder()

	RouteUploadHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

// TestRouteUploadHandlerRequiresDistanceFile verifies that a distance table
// file is required.
func TestRouteUploadHandlerRequiresDistanceFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(
		"routeFile",
		"route.csv",
	)
	if err != nil {
		t.Fatalf(
			"failed to create route form file: %v",
			err,
		)
	}

	if _, err := part.Write(
		[]byte(
			"id,name,demand,priority,window_start,window_end\n" +
				"1,Warehouse,0,0,08:00,17:00\n",
		),
	); err != nil {
		t.Fatalf(
			"failed to write route file: %v",
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
		"/upload/route",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	rec := httptest.NewRecorder()

	RouteUploadHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

// TestRouteUploadHandlerRejectsNonCSV verifies that route uploads reject a
// non-CSV route file.
func TestRouteUploadHandlerRejectsNonCSV(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	routePart, err := writer.CreateFormFile(
		"routeFile",
		"route.txt",
	)
	if err != nil {
		t.Fatalf(
			"failed to create route form file: %v",
			err,
		)
	}

	if _, err := routePart.Write(
		[]byte("route data"),
	); err != nil {
		t.Fatalf(
			"failed to write route file: %v",
			err,
		)
	}

	distancePart, err := writer.CreateFormFile(
		"distanceFile",
		"distances.csv",
	)
	if err != nil {
		t.Fatalf(
			"failed to create distance form file: %v",
			err,
		)
	}

	if _, err := distancePart.Write(
		[]byte(
			"location,Warehouse\n" +
				"Warehouse,0\n",
		),
	); err != nil {
		t.Fatalf(
			"failed to write distance file: %v",
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
		"/upload/route",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	rec := httptest.NewRecorder()

	RouteUploadHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

// TestRouteUploadHandlerUploadsFiles verifies that both route CSV files are
// saved and that the handler returns an upload ID.
func TestRouteUploadHandlerUploadsFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	routePart, err := writer.CreateFormFile(
		"routeFile",
		"route.csv",
	)
	if err != nil {
		t.Fatalf(
			"failed to create route form file: %v",
			err,
		)
	}

	routeData := []byte(
		"id,name,demand,priority,window_start,window_end\n" +
			"1,Warehouse,0,0,08:00,17:00\n" +
			"2,Customer A,10,1,09:00,12:00\n",
	)

	if _, err := routePart.Write(routeData); err != nil {
		t.Fatalf(
			"failed to write route file: %v",
			err,
		)
	}

	distancePart, err := writer.CreateFormFile(
		"distanceFile",
		"distances.csv",
	)
	if err != nil {
		t.Fatalf(
			"failed to create distance form file: %v",
			err,
		)
	}

	distanceData := []byte(
		"location,Warehouse,Customer A\n" +
			"Warehouse,0,10\n" +
			"Customer A,10,0\n",
	)

	if _, err := distancePart.Write(distanceData); err != nil {
		t.Fatalf(
			"failed to write distance file: %v",
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
		"/upload/route",
		&body,
	)

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	rec := httptest.NewRecorder()

	RouteUploadHandler(rec, req)

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

	routePath := filepath.Join(
		temporaryUploadDirectory,
		response.UploadID+"_route.csv",
	)

	distancePath := filepath.Join(
		temporaryUploadDirectory,
		response.UploadID+"_distances.csv",
	)

	savedRoute, err := os.ReadFile(routePath)
	if err != nil {
		t.Fatalf(
			"failed to read uploaded route file: %v",
			err,
		)
	}

	savedDistance, err := os.ReadFile(distancePath)
	if err != nil {
		t.Fatalf(
			"failed to read uploaded distance file: %v",
			err,
		)
	}

	if string(savedRoute) != string(routeData) {
		t.Fatal("uploaded route file does not match input")
	}

	if string(savedDistance) != string(distanceData) {
		t.Fatal("uploaded distance file does not match input")
	}

	t.Cleanup(func() {
		os.Remove(routePath)
		os.Remove(distancePath)
	})
}

// TestGetRouteConfig verifies that route form values are parsed into the
// expected route configuration.
func TestGetRouteConfig(t *testing.T) {
	form := url.Values{
		"start_location": {"Warehouse"},
		"end_location":   {"Customer B"},
		"algorithm":      {"nearest_neighbor_2opt"},
		"max_distance":   {"140.5"},
		"max_stops":      {"40"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/route",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	config, err := getRouteConfig(req)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if config.StartLocation != "Warehouse" {
		t.Fatalf(
			"expected start location Warehouse, got %q",
			config.StartLocation,
		)
	}

	if config.EndLocation != "Customer B" {
		t.Fatalf(
			"expected end location Customer B, got %q",
			config.EndLocation,
		)
	}

	if config.Optimization.Algorithm != "nearest_neighbor_2opt" {
		t.Fatalf(
			"unexpected algorithm: %q",
			config.Optimization.Algorithm,
		)
	}

	if config.Constraints.MaxDistance == nil {
		t.Fatal("expected maximum distance")
	}

	if *config.Constraints.MaxDistance != 140.5 {
		t.Fatalf(
			"expected maximum distance 140.5, got %f",
			*config.Constraints.MaxDistance,
		)
	}

	if config.Constraints.MaxStops == nil {
		t.Fatal("expected maximum stops")
	}

	if *config.Constraints.MaxStops != 40 {
		t.Fatalf(
			"expected maximum stops 40, got %d",
			*config.Constraints.MaxStops,
		)
	}
}

// TestGetRouteConfigWithoutEndLocation verifies that the end location is
// optional.
func TestGetRouteConfigWithoutEndLocation(t *testing.T) {
	form := url.Values{
		"start_location": {"Warehouse"},
		"algorithm":      {"nearest_neighbor_2opt"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/route",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	config, err := getRouteConfig(req)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if config.StartLocation != "Warehouse" {
		t.Fatalf(
			"expected start location Warehouse, got %q",
			config.StartLocation,
		)
	}

	if config.EndLocation != "" {
		t.Fatalf(
			"expected empty end location, got %q",
			config.EndLocation,
		)
	}
}

// TestGetRouteConfigRequiresStartLocation verifies that a start location is
// required.
func TestGetRouteConfigRequiresStartLocation(t *testing.T) {
	form := url.Values{
		"algorithm": {"nearest_neighbor_2opt"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/route",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	_, err := getRouteConfig(req)

	if err == nil {
		t.Fatal("expected error for missing start location")
	}

	if err.Error() != "start location is required" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

// TestGetRouteConfigInvalidMaxDistance verifies that invalid maximum
// distances are rejected.
func TestGetRouteConfigInvalidMaxDistance(t *testing.T) {
	values := []string{
		"not-a-number",
		"0",
		"-1",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			form := url.Values{
				"start_location": {"Warehouse"},
				"algorithm":      {"nearest_neighbor_2opt"},
				"max_distance":   {value},
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/submit/route",
				strings.NewReader(form.Encode()),
			)

			req.Header.Set(
				"Content-Type",
				"application/x-www-form-urlencoded",
			)

			_, err := getRouteConfig(req)

			if err == nil {
				t.Fatal(
					"expected invalid maximum distance error",
				)
			}
		})
	}
}

// TestGetRouteConfigInvalidMaxStops verifies that invalid maximum stop counts
// are rejected.
func TestGetRouteConfigInvalidMaxStops(t *testing.T) {
	values := []string{
		"not-a-number",
		"0",
		"-1",
		"1.5",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			form := url.Values{
				"start_location": {"Warehouse"},
				"algorithm":      {"nearest_neighbor_2opt"},
				"max_stops":      {value},
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/submit/route",
				strings.NewReader(form.Encode()),
			)

			req.Header.Set(
				"Content-Type",
				"application/x-www-form-urlencoded",
			)

			_, err := getRouteConfig(req)

			if err == nil {
				t.Fatal(
					"expected invalid maximum stops error",
				)
			}
		})
	}
}

// TestRouteSubmissionHandlerRejectsWrongMethod verifies that route submissions
// only accept POST requests.
func TestRouteSubmissionHandlerRejectsWrongMethod(t *testing.T) {
	handler := RouteSubmissionHandler(nil, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/submit/route",
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

// TestRouteSubmissionHandlerRequiresUpload verifies that route submission
// requires a previously uploaded pair of CSV files.
func TestRouteSubmissionHandlerRequiresUpload(t *testing.T) {
	form := url.Values{
		"start_location": {"Warehouse"},
		"algorithm":      {"nearest_neighbor_2opt"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/route",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	handler := RouteSubmissionHandler(nil, nil)
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

// TestRouteSubmissionHandlerRejectsInvalidConfiguration verifies that invalid
// route configuration is rejected before a job is prepared.
func TestRouteSubmissionHandlerRejectsInvalidConfiguration(t *testing.T) {
	if err := os.MkdirAll(
		temporaryUploadDirectory,
		0755,
	); err != nil {
		t.Fatalf(
			"failed to create temporary upload directory: %v",
			err,
		)
	}

	uploadID := uuid.New().String()

	routePath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_route.csv",
	)

	distancePath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_distances.csv",
	)

	if err := os.WriteFile(
		routePath,
		[]byte(
			"id,name,demand,priority,window_start,window_end\n"+
				"1,Warehouse,0,0,08:00,17:00\n",
		),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to create route CSV: %v",
			err,
		)
	}

	if err := os.WriteFile(
		distancePath,
		[]byte(
			"location,Warehouse\n"+
				"Warehouse,0\n",
		),
		0644,
	); err != nil {
		os.Remove(routePath)
		t.Fatalf(
			"failed to create distance CSV: %v",
			err,
		)
	}

	t.Cleanup(func() {
		os.Remove(routePath)
		os.Remove(distancePath)
	})

	form := url.Values{
		"uploadID":       {uploadID},
		"start_location": {"Warehouse"},
		"algorithm":      {"nearest_neighbor_2opt"},
		"max_distance":   {"invalid"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/route",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	handler := RouteSubmissionHandler(nil, nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if _, err := os.Stat(routePath); err != nil {
		t.Fatalf(
			"expected route temporary file to remain: %v",
			err,
		)
	}

	if _, err := os.Stat(distancePath); err != nil {
		t.Fatalf(
			"expected distance temporary file to remain: %v",
			err,
		)
	}
}

// TestRouteSubmissionHandlerEnqueuesJob verifies that a valid route submission
// creates a queued route job, stores its inputs and configuration, and
// enqueues the job.
func TestRouteSubmissionHandlerEnqueuesJob(t *testing.T) {
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

	if err := os.MkdirAll(
		temporaryUploadDirectory,
		0755,
	); err != nil {
		t.Fatalf(
			"failed to create temporary upload directory: %v",
			err,
		)
	}

	routeTempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_route.csv",
	)

	distanceTempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_distances.csv",
	)

	routeData := []byte(
		"id,name,demand,priority,window_start,window_end\n" +
			"1,Warehouse,0,0,08:00,17:00\n" +
			"2,Customer A,10,1,09:00,12:00\n",
	)

	distanceData := []byte(
		"location,Warehouse,Customer A\n" +
			"Warehouse,0,10\n" +
			"Customer A,10,0\n",
	)

	if err := os.WriteFile(
		routeTempPath,
		routeData,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to create temporary route CSV: %v",
			err,
		)
	}

	if err := os.WriteFile(
		distanceTempPath,
		distanceData,
		0644,
	); err != nil {
		os.Remove(routeTempPath)

		t.Fatalf(
			"failed to create temporary distance CSV: %v",
			err,
		)
	}

	t.Cleanup(func() {
		os.Remove(routeTempPath)
		os.Remove(distanceTempPath)
	})

	form := url.Values{
		"uploadID":       {uploadID},
		"start_location": {"Warehouse"},
		"end_location":   {"Warehouse"},
		"algorithm":      {"nearest_neighbor_2opt"},
		"max_distance":   {"100"},
		"max_stops":      {"1"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/submit/route",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	handler := RouteSubmissionHandler(
		db,
		redisClient,
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
        WHERE type = 'route'
          AND input_reference LIKE 'uploads/%/route.csv'
        ORDER BY id DESC
        LIMIT 1
    `).Scan(&jobID)

	if err != nil {
		t.Fatalf(
			"failed to retrieve created route job: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = jobs.DeleteJob(db, jobID)
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
			"failed to retrieve route input reference: %v",
			err,
		)
	}

	if !strings.HasSuffix(
		inputReference,
		"/route.csv",
	) {
		t.Fatalf(
			"unexpected route input reference: %q",
			inputReference,
		)
	}

	jobDirectory := filepath.Dir(inputReference)

	savedRoute, err := os.ReadFile(
		filepath.Join(jobDirectory, "route.csv"),
	)
	if err != nil {
		t.Fatalf(
			"failed to read saved route CSV: %v",
			err,
		)
	}

	savedDistance, err := os.ReadFile(
		filepath.Join(jobDirectory, "distances.csv"),
	)
	if err != nil {
		t.Fatalf(
			"failed to read saved distance CSV: %v",
			err,
		)
	}

	if string(savedRoute) != string(routeData) {
		t.Fatal(
			"saved route CSV does not match submitted data",
		)
	}

	if string(savedDistance) != string(distanceData) {
		t.Fatal(
			"saved distance CSV does not match submitted data",
		)
	}

	configData, err := os.ReadFile(
		filepath.Join(jobDirectory, "config.json"),
	)
	if err != nil {
		t.Fatalf(
			"failed to read route config: %v",
			err,
		)
	}

	var config jobs.RouteConfig

	if err := json.Unmarshal(
		configData,
		&config,
	); err != nil {
		t.Fatalf(
			"failed to decode route config: %v",
			err,
		)
	}

	if config.StartLocation != "Warehouse" {
		t.Fatalf(
			"expected start location Warehouse, got %q",
			config.StartLocation,
		)
	}

	if config.EndLocation != "Warehouse" {
		t.Fatalf(
			"expected end location Warehouse, got %q",
			config.EndLocation,
		)
	}

	if config.Optimization.Algorithm != "nearest_neighbor_2opt" {
		t.Fatalf(
			"unexpected algorithm: %q",
			config.Optimization.Algorithm,
		)
	}

	if config.Constraints.MaxDistance == nil ||
		*config.Constraints.MaxDistance != 100 {
		t.Fatal("unexpected maximum distance")
	}

	if config.Constraints.MaxStops == nil ||
		*config.Constraints.MaxStops != 1 {
		t.Fatal("unexpected maximum stops")
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

// TestLoadRouteResults verifies that route results JSON is decoded.
func TestLoadRouteResults(t *testing.T) {
	tempDir := t.TempDir()

	resultPath := filepath.Join(
		tempDir,
		"results.json",
	)

	data := map[string]interface{}{
		"start_location":         "Warehouse",
		"end_location":           "Warehouse",
		"initial_route":          []string{"Warehouse", "Customer A", "Warehouse"},
		"optimized_route":        []string{"Warehouse", "Customer A", "Warehouse"},
		"initial_distance":       20.0,
		"optimized_distance":     18.0,
		"distance_improvement":   2.0,
		"improvement_percentage": 10.0,
		"algorithm":              "nearest_neighbor_2opt",
		"two_opt_applied":        true,
		"runtime_seconds":        0.0021,
		"feasible":               true,
	}

	fileData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf(
			"failed to marshal route results: %v",
			err,
		)
	}

	if err := os.WriteFile(
		resultPath,
		fileData,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write route results: %v",
			err,
		)
	}

	results, err := loadRouteResults(resultPath)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if results == nil {
		t.Fatal("expected route results")
	}

	encodedResults, err := json.Marshal(results)
	if err != nil {
		t.Fatalf(
			"failed to re-encode route results: %v",
			err,
		)
	}

	var decoded map[string]interface{}

	if err := json.Unmarshal(
		encodedResults,
		&decoded,
	); err != nil {
		t.Fatalf(
			"failed to decode route results: %v",
			err,
		)
	}

	if decoded["start_location"] != "Warehouse" {
		t.Fatalf(
			"unexpected start location: %v",
			decoded["start_location"],
		)
	}

	if decoded["end_location"] != "Warehouse" {
		t.Fatalf(
			"unexpected end location: %v",
			decoded["end_location"],
		)
	}

	if decoded["algorithm"] != "nearest_neighbor_2opt" {
		t.Fatalf(
			"unexpected algorithm: %v",
			decoded["algorithm"],
		)
	}

	if decoded["feasible"] != true {
		t.Fatal("expected route to be feasible")
	}
}

// TestLoadRouteResultsMissingFile verifies that a missing result file returns
// an error.
func TestLoadRouteResultsMissingFile(t *testing.T) {
	resultPath := filepath.Join(
		t.TempDir(),
		"missing.json",
	)

	_, err := loadRouteResults(resultPath)

	if err == nil {
		t.Fatal("expected error for missing route results file")
	}
}

// TestLoadRouteResultsInvalidJSON verifies that malformed result JSON is
// rejected.
func TestLoadRouteResultsInvalidJSON(t *testing.T) {
	resultPath := filepath.Join(
		t.TempDir(),
		"results.json",
	)

	if err := os.WriteFile(
		resultPath,
		[]byte("{invalid json"),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write invalid route results: %v",
			err,
		)
	}

	_, err := loadRouteResults(resultPath)

	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

// TestLoadRoutePageResults verifies that completed route results are loaded
// into the results page.
func TestLoadRoutePageResults(t *testing.T) {
	resultPath := filepath.Join(
		t.TempDir(),
		"results.json",
	)

	resultData := []byte(`{
		"start_location": "Warehouse",
		"end_location": "Warehouse",
		"initial_route": ["Warehouse", "Customer A", "Warehouse"],
		"optimized_route": ["Warehouse", "Customer A", "Warehouse"],
		"initial_distance": 20.0,
		"optimized_distance": 20.0,
		"distance_improvement": 0.0,
		"improvement_percentage": 0.0,
		"algorithm": "nearest_neighbor_2opt",
		"two_opt_applied": true,
		"runtime_seconds": 0.0021,
		"feasible": true
	}`)

	if err := os.WriteFile(
		resultPath,
		resultData,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write route results: %v",
			err,
		)
	}

	job := &jobs.Job{
		ID:              1,
		Type:            "route",
		Status:          "completed",
		ResultReference: &resultPath,
	}

	page := ResultsPage{
		Job: job,
	}

	if err := loadRoutePageResults(&page); err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if page.RouteResults == nil {
		t.Fatal("expected route results to be loaded")
	}
}

// TestLoadRoutePageResultsSkipsIncompleteJob verifies that incomplete jobs
// do not attempt to load route results.
func TestLoadRoutePageResultsSkipsIncompleteJob(t *testing.T) {
	tests := []struct {
		name   string
		status string
		path   *string
	}{
		{
			name:   "queued job",
			status: "queued",
		},
		{
			name:   "processing job",
			status: "processing",
		},
		{
			name:   "completed job without result",
			status: "completed",
			path:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &jobs.Job{
				ID:              1,
				Type:            "route",
				Status:          tt.status,
				ResultReference: tt.path,
			}

			page := ResultsPage{
				Job: job,
			}

			if err := loadRoutePageResults(&page); err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

			if page.RouteResults != nil {
				t.Fatal(
					"expected route results to remain nil",
				)
			}
		})
	}
}

// TestLoadRoutePageResultsInvalidFile verifies that result-loading errors are
// returned by the page loader.
func TestLoadRoutePageResultsInvalidFile(t *testing.T) {
	resultPath := filepath.Join(
		t.TempDir(),
		"results.json",
	)

	if err := os.WriteFile(
		resultPath,
		[]byte("{invalid json"),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write invalid route results: %v",
			err,
		)
	}

	job := &jobs.Job{
		ID:              1,
		Type:            "route",
		Status:          "completed",
		ResultReference: &resultPath,
	}

	page := ResultsPage{
		Job: job,
	}

	err := loadRoutePageResults(&page)

	if err == nil {
		t.Fatal("expected error loading invalid route results")
	}

	if !strings.Contains(
		err.Error(),
		"failed to load route results",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

// TestRouteResultsTemplate verifies that the route results template renders
// a completed route results page.
func TestRouteResultsTemplate(t *testing.T) {
	resultPath := filepath.Join(
		t.TempDir(),
		"results.json",
	)

	resultData := []byte(`{
		"start_location": "Warehouse",
		"end_location": "Warehouse",
		"initial_route": ["Warehouse", "Customer A", "Warehouse"],
		"optimized_route": ["Warehouse", "Customer A", "Warehouse"],
		"initial_distance": 20.0,
		"optimized_distance": 18.0,
		"distance_improvement": 2.0,
		"improvement_percentage": 10.0,
		"algorithm": "nearest_neighbor_2opt",
		"two_opt_applied": true,
		"runtime_seconds": 0.0021,
		"feasible": true
	}`)

	if err := os.WriteFile(
		resultPath,
		resultData,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write route results: %v",
			err,
		)
	}

	job := &jobs.Job{
		ID:              123,
		Type:            "route",
		Status:          "completed",
		ResultReference: &resultPath,
	}

	page := ResultsPage{
		Job: job,
	}

	if err := loadRoutePageResults(&page); err != nil {
		t.Fatalf(
			"failed to load route results: %v",
			err,
		)
	}

	rec := httptest.NewRecorder()

	if err := RouteResultsTemplate.Execute(
		rec,
		page,
	); err != nil {
		t.Fatalf(
			"failed to execute route results template: %v",
			err,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(
		body,
		"Route Results for Job #123",
	) {
		t.Fatalf(
			"expected route results heading, got: %s",
			body,
		)
	}

	if !strings.Contains(
		body,
		"Warehouse",
	) {
		t.Fatalf(
			"expected route location in rendered output, got: %s",
			body,
		)
	}
}

// TestResultsHandlerRouteJob verifies that ResultsHandler dispatches completed
// route jobs to the route result loader and template.
func TestResultsHandlerRouteJob(t *testing.T) {
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

	t.Cleanup(func() {
		_ = db.Close()
	})

	jobID, err := jobs.CreateJob(
		db,
		"route",
	)
	if err != nil {
		t.Fatalf(
			"failed to create test route job: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = jobs.DeleteJob(db, jobID)
	})

	resultPath := filepath.Join(
		"uploads",
		fmt.Sprintf("%d", jobID),
		"results.json",
	)

	if err := os.MkdirAll(
		filepath.Dir(resultPath),
		0755,
	); err != nil {
		t.Fatalf(
			"failed to create result directory: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(
			filepath.Dir(resultPath),
		)
	})

	resultData := []byte(`{
		"start_location": "Warehouse",
		"end_location": "Warehouse",
		"initial_route": ["Warehouse", "Customer A", "Warehouse"],
		"optimized_route": ["Warehouse", "Customer A", "Warehouse"],
		"initial_distance": 20.0,
		"optimized_distance": 18.0,
		"distance_improvement": 2.0,
		"improvement_percentage": 10.0,
		"algorithm": "nearest_neighbor_2opt",
		"two_opt_applied": true,
		"runtime_seconds": 0.0021,
		"feasible": true
	}`)

	if err := os.WriteFile(
		resultPath,
		resultData,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write route result file: %v",
			err,
		)
	}

	if _, err := db.Exec(
		`UPDATE jobs
		 SET status = $1,
		     result_reference = $2,
		     completed_at = CURRENT_TIMESTAMP
		 WHERE id = $3`,
		"completed",
		resultPath,
		jobID,
	); err != nil {
		t.Fatalf(
			"failed to mark route job completed: %v",
			err,
		)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/results?id=%d",
			jobID,
		),
		nil,
	)

	rec := httptest.NewRecorder()

	handler := ResultsHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			rec.Code,
			rec.Body.String(),
		)
	}

	body := rec.Body.String()

	if !strings.Contains(
		body,
		fmt.Sprintf(
			"Route Results for Job #%d",
			jobID,
		),
	) {
		t.Fatalf(
			"expected route results page, got: %s",
			body,
		)
	}

	if !strings.Contains(
		body,
		"Warehouse",
	) {
		t.Fatalf(
			"expected route location in results page, got: %s",
			body,
		)
	}
}
