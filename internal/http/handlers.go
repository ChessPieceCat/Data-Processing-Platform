package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/metrics"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const MaxUploadSize = 10 << 20 // 10 MiB
const temporaryUploadDirectory = "uploads/tmp"

// IndexHandler handles the root URL.
func IndexHandler(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		jobList, err := jobs.GetJobs(db)
		if err != nil {
			log.Println("Error retrieving jobs:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data := struct {
			Jobs []jobs.Job
		}{
			Jobs: jobList,
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Println("Error executing template:", err)
		}
	}
}

// ResultsHandler handles the results page.
func ResultsHandler(db *sql.DB, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID, err := parseJobID(r)
		if err != nil {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		job, err := jobs.GetJob(db, jobID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(
					w,
					fmt.Sprintf("job with ID %d not found", jobID),
					http.StatusNotFound,
				)
				return
			}

			log.Printf("Error retrieving job %d: %v", jobID, err)
			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		page := ResultsPage{
			Job: job,
		}

		switch job.Type {
		case "dataset":
			if err := loadDatasetPageResults(&page, store); err != nil {
				log.Printf(
					"Error loading dataset results for job %d: %v",
					jobID,
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
				return
			}

			if err := DatasetResultsTemplate.Execute(w, page); err != nil {
				log.Printf(
					"Error executing dataset results template: %v",
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
			}

		case "image":
			if err := loadImagePageResults(&page, store); err != nil {
				log.Printf(
					"Error loading image results for job %d: %v",
					jobID,
					err,
				)

				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
				return
			}

			if err := ImageResultsTemplate.Execute(w, page); err != nil {
				log.Printf(
					"Error executing image results template: %v",
					err,
				)

				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
				return
			}

		case "route":
			if err := loadRoutePageResults(&page, store); err != nil {
				log.Printf(
					"Error loading route results for job %d: %v",
					jobID,
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
				return
			}

			if err := RouteResultsTemplate.Execute(w, page); err != nil {
				log.Printf(
					"Error executing route results template: %v",
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
			}

		default:
			log.Printf("Unsupported job type: %s", job.Type)
			http.Error(
				w,
				"Unsupported job type",
				http.StatusInternalServerError,
			)
		}
	}
}

func loadRoutePageResults(
	page *ResultsPage,
	store storage.Storage,
) error {
	if page.Job.Status != "completed" ||
		page.Job.ResultReference == nil {
		return nil
	}

	results, err := loadRouteResults(
		store,
		*page.Job.ResultReference,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to load route results: %w",
			err,
		)
	}

	page.RouteResults = results
	return nil
}

func loadDatasetPageResults(
	page *ResultsPage,
	store storage.Storage,
) error {
	if page.Job.Status != "completed" ||
		page.Job.ResultReference == nil {
		return nil
	}

	results, err := loadDatasetResults(
		store,
		*page.Job.ResultReference,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to load dataset results: %w",
			err,
		)
	}

	page.Results = results
	page.VisualizationResults = results.Visualizations

	modelResults, err := loadModelResults(
		store,
		page.Job.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to load model results: %w",
			err,
		)
	}

	page.ModelResults = modelResults
	return nil
}

func loadRouteResults(
	store storage.Storage,
	key string,
) (*RouteResults, error) {
	resultFile, err := store.Get(
		context.Background(),
		key,
	)
	if err != nil {
		return nil, err
	}
	defer resultFile.Close()

	var results RouteResults

	if err := json.NewDecoder(resultFile).Decode(&results); err != nil {
		return nil, err
	}

	return &results, nil
}

// loadImagePageResults loads image-processing results for a completed job.
func loadImagePageResults(
	page *ResultsPage,
	store storage.Storage,
) error {
	if page.Job.Status != "completed" ||
		page.Job.ResultReference == nil {
		return nil
	}

	results, err := loadImageResults(
		store,
		*page.Job.ResultReference,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to load image results: %w",
			err,
		)
	}

	page.ImageResults = results
	return nil
}

// prepareJob creates the job directory, moves the input file,
// and saves the input reference.
func prepareJob(
	db *sql.DB,
	store storage.Storage,
	jobType string,
	sourcePath string,
	filename string,
	m *metrics.Metrics,
) (int64, string, error) {
	jobID, err := jobs.CreateJobWithLimit(
		db,
		jobType,
		m,
	)
	if err != nil {
		return 0, "", fmt.Errorf(
			"failed to create job: %w",
			err,
		)
	}

	inputKey := fmt.Sprintf(
		"jobs/%d/%s",
		jobID,
		filename,
	)

	inputFile, err := os.Open(sourcePath)
	if err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to open input file: %w",
			err,
		)
	}
	defer inputFile.Close()

	if err := store.Put(
		context.Background(),
		inputKey,
		inputFile,
	); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to store input file: %w",
			err,
		)
	}

	if err := os.Remove(sourcePath); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to remove temporary input file: %w",
			err,
		)
	}

	if err := saveInputReference(
		db,
		inputKey,
		jobID,
	); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to save input reference: %w",
			err,
		)
	}

	return jobID, inputKey, nil
}

func prepareRouteJob(
	db *sql.DB,
	store storage.Storage,
	routeTempPath string,
	distanceTempPath string,
	m *metrics.Metrics,
) (int64, string, error) {
	jobID, err := jobs.CreateJobWithLimit(
		db,
		"route",
		m,
	)
	if err != nil {
		return 0, "", fmt.Errorf(
			"failed to create route job: %w",
			err,
		)
	}

	routeKey := fmt.Sprintf(
		"jobs/%d/route.csv",
		jobID,
	)

	distanceKey := fmt.Sprintf(
		"jobs/%d/distances.csv",
		jobID,
	)

	routeFile, err := os.Open(routeTempPath)
	if err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to open route CSV: %w",
			err,
		)
	}

	if err := store.Put(
		context.Background(),
		routeKey,
		routeFile,
	); err != nil {
		routeFile.Close()
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to store route CSV: %w",
			err,
		)
	}

	if err := routeFile.Close(); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to close route CSV: %w",
			err,
		)
	}

	distanceFile, err := os.Open(distanceTempPath)
	if err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to open distance CSV: %w",
			err,
		)
	}

	if err := store.Put(
		context.Background(),
		distanceKey,
		distanceFile,
	); err != nil {
		distanceFile.Close()
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to store distance CSV: %w",
			err,
		)
	}

	if err := distanceFile.Close(); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to close distance CSV: %w",
			err,
		)
	}

	if err := os.Remove(routeTempPath); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to remove temporary route CSV: %w",
			err,
		)
	}

	if err := os.Remove(distanceTempPath); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to remove temporary distance CSV: %w",
			err,
		)
	}

	if err := saveInputReference(
		db,
		routeKey,
		jobID,
	); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return 0, "", fmt.Errorf(
			"failed to save route input reference: %w",
			err,
		)
	}

	return jobID, routeKey, nil
}

// finalizeJobSubmission saves the job configuration and enqueues the job.
func finalizeJobSubmission(
	db *sql.DB,
	redisClient *redis.Client,
	store storage.Storage,
	jobID int64,
	saveConfig func(int64) (string, error),
	m *metrics.Metrics,
) error {
	if _, err := saveConfig(jobID); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return fmt.Errorf(
			"failed to save job configuration: %w",
			err,
		)
	}

	if err := jobs.EnqueueJob(
		redisClient,
		jobID,
	); err != nil {
		_ = jobs.DeleteJob(
			db,
			jobID,
			store,
		)

		return fmt.Errorf(
			"failed to enqueue job: %w",
			err,
		)
	}

	if err := jobs.UpdateQueueDepth(db, m); err != nil {
		log.Printf("Error updating queue depth: %v", err)
	}

	return nil
}

// RouteSubmissionHandler handles route optimization job submission.
func RouteSubmissionHandler(
	db *sql.DB,
	redisClient *redis.Client,
	store storage.Storage,
	m *metrics.Metrics,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		uploadID := r.FormValue("uploadID")

		routeTempPath, distanceTempPath, err :=
			getTemporaryRouteFiles(uploadID)

		if err != nil {
			switch {
			case errors.Is(err, errRouteNotUploaded):
				http.Error(
					w,
					"Route files have not been uploaded.",
					http.StatusBadRequest,
				)
			case errors.Is(err, errInvalidUploadID):
				http.Error(
					w,
					"Invalid upload ID.",
					http.StatusBadRequest,
				)
			case errors.Is(err, os.ErrNotExist):
				http.Error(
					w,
					"Temporary route files were not found.",
					http.StatusBadRequest,
				)
			default:
				log.Println(
					"Error checking temporary route files:",
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
			}

			return
		}

		config, err := getRouteConfig(r)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		jobID, routeKey, err := prepareRouteJob(
			db,
			store,
			routeTempPath,
			distanceTempPath,
			m,
		)
		if err != nil {
			if errors.Is(
				err,
				jobs.ErrJobQueueFull,
			) {
				http.Error(
					w,
					"Job queue is full. Please try again later.",
					http.StatusTooManyRequests,
				)
				return
			}

			log.Println(
				"Error preparing route job:",
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf(
			"Stored route input for job %d as %s",
			jobID,
			routeKey,
		)

		if err := finalizeJobSubmission(
			db,
			redisClient,
			store,
			jobID,
			func(jobID int64) (string, error) {
				return jobs.SaveRouteConfig(
					config,
					jobID,
					store,
				)
			},
			m,
		); err != nil {
			log.Printf(
				"Error finalizing route job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/",
			http.StatusSeeOther,
		)
	}
}

// DatasetSubmissionHandler handles dataset job submission.
func DatasetSubmissionHandler(
	db *sql.DB,
	redisClient *redis.Client,
	store storage.Storage,
	m *metrics.Metrics,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		uploadID := r.FormValue("uploadID")

		tempPath, err := getTemporaryDataset(
			uploadID,
		)
		if err != nil {
			switch {
			case errors.Is(err, errDatasetNotInspected):
				http.Error(
					w,
					"Dataset has not been inspected.",
					http.StatusBadRequest,
				)
			case errors.Is(err, errInvalidUploadID):
				http.Error(
					w,
					"Invalid upload ID.",
					http.StatusBadRequest,
				)
			case errors.Is(err, os.ErrNotExist):
				http.Error(
					w,
					"Temporary dataset was not found.",
					http.StatusBadRequest,
				)
			default:
				log.Println(
					"Error checking temporary dataset:",
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
			}

			return
		}

		config, err := getDatasetConfig(r)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		jobID, fileKey, err := prepareJob(
			db,
			store,
			"dataset",
			tempPath,
			"dataset.csv",
			m,
		)
		if err != nil {
			if errors.Is(
				err,
				jobs.ErrJobQueueFull,
			) {
				http.Error(
					w,
					"Job queue is full. Please try again later.",
					http.StatusTooManyRequests,
				)
				return
			}

			log.Println(
				"Error preparing dataset job:",
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf(
			"Stored dataset input for job %d as %s",
			jobID,
			fileKey,
		)

		if err := finalizeJobSubmission(
			db,
			redisClient,
			store,
			jobID,
			func(jobID int64) (string, error) {
				return jobs.SaveDatasetConfig(
					config,
					jobID,
					store,
				)
			},
			m,
		); err != nil {
			log.Printf(
				"Error finalizing dataset job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/",
			http.StatusSeeOther,
		)
	}
}

// ImageSubmissionHandler handles image job submission.
func ImageSubmissionHandler(
	db *sql.DB,
	redisClient *redis.Client,
	store storage.Storage,
	m *metrics.Metrics,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		uploadID := r.FormValue("uploadID")

		tempPath, err := getTemporaryImage(
			uploadID,
		)
		if err != nil {
			switch {
			case errors.Is(err, errImageNotInspected):
				http.Error(
					w,
					"Image has not been inspected.",
					http.StatusBadRequest,
				)
			case errors.Is(err, errInvalidUploadID):
				http.Error(
					w,
					"Invalid upload ID.",
					http.StatusBadRequest,
				)
			case errors.Is(err, os.ErrNotExist):
				http.Error(
					w,
					"Temporary image was not found.",
					http.StatusBadRequest,
				)
			default:
				log.Println(
					"Error checking temporary image:",
					err,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
			}

			return
		}

		config, err := getImageConfig(r)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		jobID, fileKey, err := prepareJob(
			db,
			store,
			"image",
			tempPath,
			"input"+filepath.Ext(tempPath),
			m,
		)
		if err != nil {
			if errors.Is(
				err,
				jobs.ErrJobQueueFull,
			) {
				http.Error(
					w,
					"Job queue is full. Please try again later.",
					http.StatusTooManyRequests,
				)
				return
			}

			log.Println(
				"Error preparing image job:",
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf(
			"Stored image input for job %d as %s",
			jobID,
			fileKey,
		)

		if err := finalizeJobSubmission(
			db,
			redisClient,
			store,
			jobID,
			func(jobID int64) (string, error) {
				return jobs.SaveImageConfig(
					config,
					jobID,
					store,
				)
			},
			m,
		); err != nil {
			log.Printf(
				"Error finalizing image job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/",
			http.StatusSeeOther,
		)
	}
}

func getRouteConfig(
	r *http.Request,
) (jobs.RouteConfig, error) {
	config := jobs.RouteConfig{
		StartLocation: r.FormValue("start_location"),
		EndLocation:   r.FormValue("end_location"),
		Optimization: jobs.RouteOptimizationConfig{
			Algorithm: r.FormValue("algorithm"),
		},
		Constraints: jobs.RouteConstraintConfig{},
	}

	if config.StartLocation == "" {
		return jobs.RouteConfig{}, fmt.Errorf(
			"start location is required",
		)
	}

	if maxDistance := r.FormValue("max_distance"); maxDistance != "" {
		value, err := strconv.ParseFloat(
			maxDistance,
			64,
		)

		if err != nil || value <= 0 {
			return jobs.RouteConfig{}, fmt.Errorf(
				"maximum distance must be a positive number",
			)
		}

		config.Constraints.MaxDistance = &value
	}

	if maxStops := r.FormValue("max_stops"); maxStops != "" {
		value, err := strconv.Atoi(
			maxStops,
		)

		if err != nil || value <= 0 {
			return jobs.RouteConfig{}, fmt.Errorf(
				"maximum stops must be a positive integer",
			)
		}

		config.Constraints.MaxStops = &value
	}

	return config, nil
}

func getImageConfig(
	r *http.Request,
) (jobs.ImageConfig, error) {
	config := jobs.ImageConfig{
		Resize:           r.FormValue("resize") == "true",
		Compression:      r.FormValue("compression") == "true",
		FormatConversion: r.FormValue("format_conversion") == "true",
		ExtractMetadata:  r.FormValue("extract_metadata") == "true",
		OutputFormat:     r.FormValue("output_format"),
	}

	if config.Resize {
		width, err := strconv.Atoi(
			r.FormValue("resize_width"),
		)
		if err != nil || width <= 0 {
			return jobs.ImageConfig{}, fmt.Errorf(
				"resize width must be a positive integer",
			)
		}

		height, err := strconv.Atoi(
			r.FormValue("resize_height"),
		)
		if err != nil || height <= 0 {
			return jobs.ImageConfig{}, fmt.Errorf(
				"resize height must be a positive integer",
			)
		}

		config.ResizeWidth = width
		config.ResizeHeight = height
	}

	if config.Compression {
		quality, err := strconv.Atoi(
			r.FormValue("compression_quality"),
		)
		if err != nil || quality < 1 || quality > 100 {
			return jobs.ImageConfig{}, fmt.Errorf(
				"compression quality must be between 1 and 100",
			)
		}

		config.CompressionQuality = quality
	}

	if config.FormatConversion {
		switch config.OutputFormat {
		case "jpeg", "png", "webp":
		default:
			return jobs.ImageConfig{}, fmt.Errorf(
				"unsupported output format: %q",
				config.OutputFormat,
			)
		}
	}

	return config, nil
}

// DownloadResultsHandler handles downloading a job's results.
func DownloadResultsHandler(db *sql.DB, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobIDInt, err := parseJobID(r)
		if err != nil {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		job, err := jobs.GetJob(db, jobIDInt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, fmt.Sprintf("job with ID %d not found", jobIDInt), http.StatusNotFound)
				return
			}

			log.Printf("Error retrieving job %d: %v", jobIDInt, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if job.Status != "completed" || job.ResultReference == nil {
			http.Error(w, "Results are not available.", http.StatusNotFound)
			return
		}

		resultFile, err := store.Get(
			r.Context(),
			*job.ResultReference,
		)
		if err != nil {
			log.Println("Error opening result object:", err)
			http.Error(w, "Results could not be found.", http.StatusNotFound)
			return
		}
		defer resultFile.Close()

		// Tell the browser to download the file instead of displaying it.
		w.Header().Set("Content-Disposition", fmt.Sprintf(
			`attachment; filename="job_%s_results.json"`,
			strconv.FormatInt(jobIDInt, 10),
		))
		w.Header().Set("Content-Type", "application/json")

		if _, err := io.Copy(w, resultFile); err != nil {
			log.Println("Error sending result file:", err)
			return
		}
	}
}

// DownloadModelHandler handles downloading a job's model results.
func DownloadModelHandler(db *sql.DB, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobIDInt, err := parseJobID(r)
		if err != nil {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		job, err := jobs.GetJob(db, jobIDInt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, fmt.Sprintf("job with ID %d not found", jobIDInt), http.StatusNotFound)
				return
			}

			log.Printf("Error retrieving job %d: %v", jobIDInt, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if job.Status != "completed" {
			http.Error(w, "Model results are not available.", http.StatusNotFound)
			return
		}

		modelResultsKey := fmt.Sprintf(
			"jobs/%d/results_model_results.json",
			job.ID,
		)

		modelFile, err := store.Get(
			r.Context(),
			modelResultsKey,
		)
		if err != nil {
			exists, existsErr := store.Exists(
				r.Context(),
				modelResultsKey,
			)
			if existsErr != nil {
				log.Println(
					"Error checking model results object:",
					existsErr,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
				return
			}

			if !exists {
				http.Error(
					w,
					"Model results are not available.",
					http.StatusNotFound,
				)
				return
			}

			log.Println(
				"Error opening model results object:",
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}
		defer modelFile.Close()

		// Tell the browser to download the file instead of displaying it.
		w.Header().Set("Content-Disposition", fmt.Sprintf(
			`attachment; filename="job_%s_model_results.json"`,
			strconv.FormatInt(jobIDInt, 10),
		))
		w.Header().Set("Content-Type", "application/json")

		if _, err := io.Copy(w, modelFile); err != nil {
			log.Println("Error sending model results file:", err)
			return
		}
	}
}

// DownloadImageMetadataHandler handles downloading an image job's full metadata.
func DownloadImageMetadataHandler(db *sql.DB, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		jobID, err := parseJobID(r)
		if err != nil {
			http.Error(
				w,
				"Invalid job ID",
				http.StatusBadRequest,
			)
			return
		}

		job, err := jobs.GetJob(db, jobID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(
					w,
					fmt.Sprintf("job with ID %d not found", jobID),
					http.StatusNotFound,
				)
				return
			}

			log.Printf(
				"Error retrieving job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		if job.Type != "image" {
			http.Error(
				w,
				"Job is not an image job.",
				http.StatusBadRequest,
			)
			return
		}

		if job.Status != "completed" ||
			job.ResultReference == nil {
			http.Error(
				w,
				"Image results are not available.",
				http.StatusNotFound,
			)
			return
		}

		results, err := loadImageResults(
			store,
			*job.ResultReference,
		)
		if err != nil {
			log.Printf(
				"Error loading image results for job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Image results could not be loaded.",
				http.StatusInternalServerError,
			)
			return
		}

		if results.MetadataReference == "" {
			http.Error(
				w,
				"Image metadata is not available.",
				http.StatusNotFound,
			)
			return
		}

		metadataFile, err := store.Get(
			r.Context(),
			results.MetadataReference,
		)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(
					w,
					"Image metadata is not available.",
					http.StatusNotFound,
				)
				return
			}

			log.Printf(
				"Error opening metadata file for job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}
		defer metadataFile.Close()

		w.Header().Set(
			"Content-Disposition",
			fmt.Sprintf(
				`attachment; filename="job_%s_metadata.json"`,
				strconv.FormatInt(jobID, 10),
			),
		)

		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		if _, err := io.Copy(w, metadataFile); err != nil {
			log.Printf(
				"Error sending metadata file: %v",
				err,
			)
		}
	}
}

// DatasetInspectionHandler handles dataset inspection.
func DatasetInspectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit the total request size.
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

	// Retrieve the uploaded file.
	file, header, err := r.FormFile("csvFile")
	if err != nil {
		log.Println("Error retrieving file:", err)
		http.Error(w, "A CSV file is required.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Reject empty files.
	if header.Size == 0 {
		http.Error(w, "The uploaded file is empty.", http.StatusBadRequest)
		return
	}

	// Validate the filename extension.
	extension := strings.ToLower(filepath.Ext(header.Filename))
	if extension != ".csv" {
		http.Error(w, "Only CSV files are accepted.", http.StatusBadRequest)
		return
	}

	// Create the temporary upload directory.
	if err := os.MkdirAll(temporaryUploadDirectory, 0755); err != nil {
		log.Println("Error creating temporary upload directory:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	uploadID := uuid.New().String()
	tempPath := filepath.Join(temporaryUploadDirectory, uploadID+".csv")

	// Create a temporary file.
	tempFile, err := os.Create(tempPath)
	if err != nil {
		log.Println("Error creating temporary file:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Save the uploaded file.
	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		log.Println("Error saving temporary file:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		log.Println("Error closing temporary file:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify that the file is valid CSV.
	if err := jobs.ValidateCSV(tempPath); err != nil {
		os.Remove(tempPath)
		log.Println("Invalid CSV:", err)
		http.Error(w, "The uploaded file is not a valid CSV file.", http.StatusBadRequest)
		return
	}

	// Get the column names.
	columns, err := jobs.GetCSVColumns(tempPath)
	if err != nil {
		os.Remove(tempPath)
		log.Println("Error reading CSV columns:", err)
		http.Error(w, "Unable to read CSV columns.", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(inspectionResponse{
		UploadID: uploadID,
		Columns:  columns,
	}); err != nil {
		log.Println("Error encoding column response:", err)
	}
}

// ImageUploadHandler handles image uploads for image processing jobs.
func ImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// Limit the total request size.
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		MaxUploadSize,
	)

	// Retrieve the uploaded file.
	file, header, err := r.FormFile("imageFile")
	if err != nil {
		log.Println("Error retrieving image:", err)

		http.Error(
			w,
			"An image file is required.",
			http.StatusBadRequest,
		)
		return
	}
	defer file.Close()

	// Reject empty files.
	if header.Size == 0 {
		http.Error(
			w,
			"The uploaded image is empty.",
			http.StatusBadRequest,
		)
		return
	}

	// Validate the image extension.
	extension := strings.ToLower(
		filepath.Ext(header.Filename),
	)

	switch extension {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		// Supported image format.
	default:
		http.Error(
			w,
			"Unsupported image format.",
			http.StatusBadRequest,
		)
		return
	}

	// Create the temporary upload directory.
	if err := os.MkdirAll(
		temporaryUploadDirectory,
		0755,
	); err != nil {
		log.Println(
			"Error creating temporary upload directory:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	uploadID := uuid.New().String()

	tempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+extension,
	)

	// Create a temporary file.
	tempFile, err := os.Create(tempPath)
	if err != nil {
		log.Println(
			"Error creating temporary image file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	// Save the uploaded image.
	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		os.Remove(tempPath)

		log.Println(
			"Error saving temporary image:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)

		log.Println(
			"Error closing temporary image file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(
		inspectionResponse{
			UploadID: uploadID,
		},
	); err != nil {
		log.Println(
			"Error encoding image upload response:",
			err,
		)
	}
}

// RouteUploadHandler handles route file uploads for route optimization jobs.
func RouteUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// Limit the total request size.
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		MaxUploadSize,
	)

	// Retrieve the route CSV.
	routeFile, routeHeader, err := r.FormFile("routeFile")
	if err != nil {
		http.Error(
			w,
			"A route CSV file is required.",
			http.StatusBadRequest,
		)
		return
	}
	defer routeFile.Close()

	// Retrieve the distance-table CSV.
	distanceFile, distanceHeader, err := r.FormFile("distanceFile")
	if err != nil {
		routeFile.Close()

		http.Error(
			w,
			"A distance table CSV file is required.",
			http.StatusBadRequest,
		)
		return
	}
	defer distanceFile.Close()

	// Reject empty files.
	if routeHeader.Size == 0 {
		http.Error(
			w,
			"The route CSV is empty.",
			http.StatusBadRequest,
		)
		return
	}

	if distanceHeader.Size == 0 {
		http.Error(
			w,
			"The distance table CSV is empty.",
			http.StatusBadRequest,
		)
		return
	}

	// Validate file extensions.
	if strings.ToLower(filepath.Ext(routeHeader.Filename)) != ".csv" {
		http.Error(
			w,
			"Only CSV route files are accepted.",
			http.StatusBadRequest,
		)
		return
	}

	if strings.ToLower(filepath.Ext(distanceHeader.Filename)) != ".csv" {
		http.Error(
			w,
			"Only CSV distance table files are accepted.",
			http.StatusBadRequest,
		)
		return
	}

	// Create the temporary upload directory.
	if err := os.MkdirAll(
		temporaryUploadDirectory,
		0755,
	); err != nil {
		log.Println(
			"Error creating temporary upload directory:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	uploadID := uuid.New().String()

	routeTempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_route.csv",
	)

	distanceTempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_distances.csv",
	)

	// Save the route CSV.
	routeTempFile, err := os.Create(routeTempPath)
	if err != nil {
		log.Println(
			"Error creating temporary route file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	if _, err := io.Copy(routeTempFile, routeFile); err != nil {
		routeTempFile.Close()
		os.Remove(routeTempPath)

		log.Println(
			"Error saving temporary route file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	if err := routeTempFile.Close(); err != nil {
		os.Remove(routeTempPath)

		log.Println(
			"Error closing temporary route file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	// Save the distance table CSV.
	distanceTempFile, err := os.Create(distanceTempPath)
	if err != nil {
		os.Remove(routeTempPath)

		log.Println(
			"Error creating temporary distance table file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	if _, err := io.Copy(distanceTempFile, distanceFile); err != nil {
		distanceTempFile.Close()
		os.Remove(routeTempPath)
		os.Remove(distanceTempPath)

		log.Println(
			"Error saving temporary distance table file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	if err := distanceTempFile.Close(); err != nil {
		os.Remove(routeTempPath)
		os.Remove(distanceTempPath)

		log.Println(
			"Error closing temporary distance table file:",
			err,
		)

		http.Error(
			w,
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	// Verify that both files are valid CSV files.
	if err := jobs.ValidateCSV(routeTempPath); err != nil {
		os.Remove(routeTempPath)
		os.Remove(distanceTempPath)

		log.Println(
			"Invalid route CSV:",
			err,
		)

		http.Error(
			w,
			"The uploaded route file is not a valid CSV file.",
			http.StatusBadRequest,
		)
		return
	}

	if err := jobs.ValidateCSV(distanceTempPath); err != nil {
		os.Remove(routeTempPath)
		os.Remove(distanceTempPath)

		log.Println(
			"Invalid distance table CSV:",
			err,
		)

		http.Error(
			w,
			"The uploaded distance table is not a valid CSV file.",
			http.StatusBadRequest,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(
		struct {
			UploadID string `json:"upload_id"`
		}{
			UploadID: uploadID,
		},
	); err != nil {
		log.Println(
			"Error encoding route upload response:",
			err,
		)
	}
}

// parseJobID parses the job ID from a request.
func parseJobID(r *http.Request) (int64, error) {
	jobID := r.URL.Query().Get("id")
	return strconv.ParseInt(jobID, 10, 64)
}

// loadDatasetResults loads the dataset-analysis results for a job.
func loadDatasetResults(
	store storage.Storage,
	key string,
) (*DatasetResults, error) {
	resultFile, err := store.Get(
		context.Background(),
		key,
	)
	if err != nil {
		return nil, err
	}
	defer resultFile.Close()

	var results DatasetResults

	if err := json.NewDecoder(resultFile).Decode(&results); err != nil {
		return nil, err
	}

	return &results, nil
}

// loadImageResults loads image-processing results from a result JSON file.
func loadImageResults(
	store storage.Storage,
	key string,
) (*ImageResults, error) {
	resultFile, err := store.Get(
		context.Background(),
		key,
	)
	if err != nil {
		return nil, err
	}
	defer resultFile.Close()

	var results ImageResults

	if err := json.NewDecoder(resultFile).Decode(&results); err != nil {
		return nil, err
	}

	return &results, nil
}

// loadModelResults loads the model results for a job.
// A missing model-results file means no model was requested.
func loadModelResults(
	store storage.Storage,
	jobID int64,
) (*ModelResults, error) {
	modelResultsKey := fmt.Sprintf(
		"jobs/%d/results_model_results.json",
		jobID,
	)

	exists, err := store.Exists(
		context.Background(),
		modelResultsKey,
	)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	modelFile, err := store.Get(
		context.Background(),
		modelResultsKey,
	)
	if err != nil {
		return nil, err
	}
	defer modelFile.Close()

	var modelResults ModelResults

	if err := json.NewDecoder(modelFile).Decode(&modelResults); err != nil {
		return nil, err
	}

	return &modelResults, nil
}

var errRouteNotUploaded = errors.New(
	"route files have not been uploaded",
)

func getTemporaryRouteFiles(uploadID string) (
	string,
	string,
	error,
) {
	if uploadID == "" {
		return "", "", errRouteNotUploaded
	}

	if _, err := uuid.Parse(uploadID); err != nil {
		return "", "", errInvalidUploadID
	}

	routePath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_route.csv",
	)

	distancePath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+"_distances.csv",
	)

	if _, err := os.Stat(routePath); err != nil {
		return "", "", err
	}

	if _, err := os.Stat(distancePath); err != nil {
		return "", "", err
	}

	return routePath, distancePath, nil
}

// getTemporaryDataset validates the upload ID and returns the temporary dataset path.
func getTemporaryDataset(uploadID string) (string, error) {
	if uploadID == "" {
		return "", errDatasetNotInspected
	}

	// Make sure the upload ID is a valid UUID.
	if _, err := uuid.Parse(uploadID); err != nil {
		return "", errInvalidUploadID
	}

	tempPath := filepath.Join(
		temporaryUploadDirectory,
		uploadID+".csv",
	)

	if _, err := os.Stat(tempPath); err != nil {
		return "", err
	}

	return tempPath, nil
}

var errImageNotInspected = errors.New("image has not been inspected")

// getTemporaryImage validates the upload ID and returns the temporary image path.
func getTemporaryImage(uploadID string) (string, error) {
	if uploadID == "" {
		return "", errImageNotInspected
	}

	// Make sure the upload ID is a valid UUID.
	if _, err := uuid.Parse(uploadID); err != nil {
		return "", errInvalidUploadID
	}

	matches, err := filepath.Glob(
		filepath.Join(
			temporaryUploadDirectory,
			uploadID+".*",
		),
	)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", os.ErrNotExist
	}

	return matches[0], nil
}

// getDatasetConfig reads the processing configuration from the request.
func getDatasetConfig(r *http.Request) (jobs.DatasetConfig, error) {
	// Read the model configuration.
	config := jobs.DatasetConfig{
		Model:             r.FormValue("model"),
		Target:            r.FormValue("target"),
		FeatureSelection:  r.FormValue("featureSelection"),
		Features:          r.Form["features"],
		ConfigurationType: r.FormValue("configType"),
		Visualizations: jobs.VisualizationConfig{
			Histograms:         r.FormValue("histograms") == "true",
			BoxPlots:           r.FormValue("box_plots") == "true",
			CorrelationHeatmap: r.FormValue("correlation_heatmap") == "true",
			ActualVsPredicted:  r.FormValue("actual_vs_predicted") == "true",
		},
	}

	if config.Model == "none" {
		config.Visualizations.ActualVsPredicted = false
	}

	// Read manual Random Forest parameters.
	if config.Model == "random_forest_regressor" && config.ConfigurationType == "manual" {
		if nEstimators := r.FormValue("n_estimators"); nEstimators != "" {
			value, err := strconv.Atoi(nEstimators)
			if err != nil {
				return jobs.DatasetConfig{}, fmt.Errorf("Invalid number of estimators.")
			}
			config.Configuration.NEstimators = value
		}

		if maxDepth := r.FormValue("max_depth"); maxDepth != "" {
			value, err := strconv.Atoi(maxDepth)
			if err != nil {
				return jobs.DatasetConfig{}, fmt.Errorf("Invalid max depth.")
			}
			config.Configuration.MaxDepth = value
		}
	}

	return config, nil
}

// saveInputReference stores the dataset path in the job record.
func saveInputReference(db *sql.DB, filePath string, jobID int64) error {
	_, err := db.Exec(
		`UPDATE jobs
		 SET input_reference = $1
		 WHERE id = $2`,
		filePath,
		jobID,
	)

	return err
}

var (
	errDatasetNotInspected = errors.New("dataset has not been inspected")
	errInvalidUploadID     = errors.New("invalid upload ID")
)

type inspectionResponse struct {
	UploadID string   `json:"upload_id"`
	Columns  []string `json:"columns"`
}

func VisualizationHandler(db *sql.DB, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		jobIDInt, err := parseJobID(r)
		if err != nil {
			http.Error(w, "Invalid job ID", http.StatusBadRequest)
			return
		}

		job, err := jobs.GetJob(db, jobIDInt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(
					w,
					fmt.Sprintf("job with ID %d not found", jobIDInt),
					http.StatusNotFound,
				)
				return
			}

			log.Printf("Error retrieving job %d: %v", jobIDInt, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if job.Status != "completed" {
			http.Error(w, "Visualizations are not available.", http.StatusNotFound)
			return
		}

		visualizationType := r.URL.Query().Get("type")
		if visualizationType == "" {
			http.Error(w, "Visualization type is required.", http.StatusBadRequest)
			return
		}

		var filename string

		switch visualizationType {
		case "correlation_heatmap":
			filename = "results_correlation_heatmap.png"

		case "actual_vs_predicted":
			filename = "results_actual_vs_predicted.png"

		case "feature_distributions":
			filename = "results_feature_distributions.png"

		default:
			http.Error(w, "Unknown visualization type.", http.StatusBadRequest)
			return
		}

		visualizationKey := fmt.Sprintf(
			"jobs/%d/%s",
			job.ID,
			filename,
		)

		visualizationFile, err := store.Get(
			r.Context(),
			visualizationKey,
		)
		if err != nil {
			exists, existsErr := store.Exists(
				r.Context(),
				visualizationKey,
			)
			if existsErr != nil {
				log.Println(
					"Error checking visualization object:",
					existsErr,
				)
				http.Error(
					w,
					"Internal server error",
					http.StatusInternalServerError,
				)
				return
			}

			if !exists {
				http.Error(
					w,
					"Requested visualization is not available.",
					http.StatusNotFound,
				)
				return
			}

			log.Println(
				"Error opening visualization object:",
				err,
			)
			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}
		defer visualizationFile.Close()

		w.Header().Set(
			"Content-Type",
			"image/png",
		)

		if _, err := io.Copy(
			w,
			visualizationFile,
		); err != nil {
			log.Println(
				"Error sending visualization:",
				err,
			)
		}
	}
}

// ImageResultHandler serves original or processed images for a completed job.
func ImageResultHandler(db *sql.DB, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		jobID, err := parseJobID(r)
		if err != nil {
			http.Error(
				w,
				"Invalid job ID",
				http.StatusBadRequest,
			)
			return
		}

		job, err := jobs.GetJob(db, jobID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(
					w,
					fmt.Sprintf("job with ID %d not found", jobID),
					http.StatusNotFound,
				)
				return
			}

			log.Printf(
				"Error retrieving job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		if job.Type != "image" {
			http.Error(
				w,
				"Job is not an image job.",
				http.StatusBadRequest,
			)
			return
		}

		if job.Status != "completed" ||
			job.ResultReference == nil {
			http.Error(
				w,
				"Image results are not available.",
				http.StatusNotFound,
			)
			return
		}

		results, err := loadImageResults(
			store,
			*job.ResultReference,
		)
		if err != nil {
			log.Printf(
				"Error loading image results for job %d: %v",
				jobID,
				err,
			)

			http.Error(
				w,
				"Image results could not be loaded.",
				http.StatusInternalServerError,
			)
			return
		}

		imageType := r.URL.Query().Get("type")

		var imageKey string

		switch imageType {
		case "original":
			imageKey = results.OriginalPath
		case "processed":
			imageKey = results.ProcessedPath
		default:
			http.Error(
				w,
				"Unknown image type.",
				http.StatusBadRequest,
			)
			return
		}

		imageFile, err := store.Get(
			r.Context(),
			imageKey,
		)
		if err != nil {
			log.Printf(
				"Error opening image object for job %d: %v",
				jobID,
				err,
			)
			http.Error(
				w,
				"Requested image is not available.",
				http.StatusNotFound,
			)
			return
		}
		defer imageFile.Close()

		contentType := "application/octet-stream"

		switch strings.ToLower(
			filepath.Ext(imageKey),
		) {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".webp":
			contentType = "image/webp"
		case ".gif":
			contentType = "image/gif"
		}

		w.Header().Set(
			"Content-Type",
			contentType,
		)

		if _, err := io.Copy(
			w,
			imageFile,
		); err != nil {
			log.Printf(
				"Error sending image for job %d: %v",
				jobID,
				err,
			)
		}
	}
}
