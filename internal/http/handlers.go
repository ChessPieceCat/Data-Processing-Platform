package server

import (
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
func ResultsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		page := ResultsPage{Job: job}

		if job.Status == "completed" && job.ResultReference != nil {
			page.Results, err = loadDatasetResults(*job.ResultReference)
			if err != nil {
				log.Println("Error loading result JSON:", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			page.VisualizationResults = page.Results.Visualizations

			// If the job has model results, populate them as well.
			page.ModelResults, err = loadModelResults(job.ID)
			if err != nil {
				log.Println("Error loading model results:", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		if err := ResultsTemplate.Execute(w, page); err != nil {
			log.Println("Error executing results template:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// DatasetSubmissionHandler handles job submission.
func DatasetSubmissionHandler(db *sql.DB, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get the temporary upload ID.
		uploadID := r.FormValue("uploadID")

		// Find the temporary dataset.
		tempPath, err := getTemporaryDataset(uploadID)
		if err != nil {
			switch {
			case errors.Is(err, errDatasetNotInspected):
				http.Error(w, "Dataset has not been inspected.", http.StatusBadRequest)
			case errors.Is(err, errInvalidUploadID):
				http.Error(w, "Invalid upload ID.", http.StatusBadRequest)
			case errors.Is(err, os.ErrNotExist):
				http.Error(w, "Temporary dataset was not found.", http.StatusBadRequest)
			default:
				log.Println("Error checking temporary dataset:", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Read the model configuration.
		config, err := getDatasetConfig(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Create the job.
		jobID, err := jobs.CreateJob(db, "dataset")
		if err != nil {
			log.Println("Error creating job:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		jobDirectory := fmt.Sprintf("uploads/%d", jobID)
		filePath := filepath.Join(jobDirectory, "dataset.csv")

		// Create the directory for this job.
		if err := os.MkdirAll(jobDirectory, 0755); err != nil {
			log.Println("Error creating job directory:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Move the temporary dataset into the job directory.
		if err := os.Rename(tempPath, filePath); err != nil {
			os.RemoveAll(jobDirectory)
			log.Println("Error moving dataset file:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Moved temporary dataset to %s for job %d", filePath, jobID)

		// Save the input reference.
		if err := saveInputReference(db, filePath, jobID); err != nil {
			os.RemoveAll(jobDirectory)
			log.Println("Error updating job input reference:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Save the processing configuration.
		configPath, err := jobs.SaveDatasetConfig(config, jobID)
		if err != nil {
			os.RemoveAll(jobDirectory)
			log.Println("Error saving dataset configuration:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Saved dataset configuration for job %d at %s", jobID, configPath)

		// Enqueue the job ID in Redis for processing by the worker.
		if err := jobs.EnqueueJob(redisClient, jobID); err != nil {
			log.Printf("Error enqueuing job %d: %v", jobID, err)

			if deleteErr := jobs.DeleteJob(db, jobID); deleteErr != nil {
				log.Printf(
					"Error cleaning up job %d after enqueue failure: %v",
					jobID,
					deleteErr,
				)
			}

			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// DownloadResultsHandler handles downloading a job's results.
func DownloadResultsHandler(db *sql.DB) http.HandlerFunc {
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

		resultFile, err := os.Open(*job.ResultReference)
		if err != nil {
			log.Println("Error opening result file:", err)
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
func DownloadModelHandler(db *sql.DB) http.HandlerFunc {
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

		modelResultsPath := fmt.Sprintf("uploads/%d/results_model_results.json", job.ID)
		modelFile, err := os.Open(modelResultsPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "Model results are not available.", http.StatusNotFound)
				return
			}

			log.Println("Error opening model results file:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
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

// parseJobID parses the job ID from a request.
func parseJobID(r *http.Request) (int64, error) {
	jobID := r.URL.Query().Get("id")
	return strconv.ParseInt(jobID, 10, 64)
}

// loadDatasetResults loads the dataset-analysis results for a job.
func loadDatasetResults(resultPath string) (*DatasetResults, error) {
	resultFile, err := os.Open(resultPath)
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

// loadModelResults loads the model results for a job.
// A missing model-results file means no model was requested.
func loadModelResults(jobID int64) (*ModelResults, error) {
	modelResultsPath := fmt.Sprintf("uploads/%d/results_model_results.json", jobID)
	modelFile, err := os.Open(modelResultsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer modelFile.Close()

	var modelResults ModelResults
	if err := json.NewDecoder(modelFile).Decode(&modelResults); err != nil {
		return nil, err
	}

	return &modelResults, nil
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

func VisualizationHandler(db *sql.DB) http.HandlerFunc {
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

		visualizationPath := filepath.Join(
			fmt.Sprintf("uploads/%d", job.ID),
			filename,
		)

		if _, err := os.Stat(visualizationPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(
					w,
					"Requested visualization is not available.",
					http.StatusNotFound,
				)
				return
			}

			log.Println("Error checking visualization file:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.ServeFile(w, r, visualizationPath)
	}
}
