package main

import (
	"log"
	"net/http"
	"text/template"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
	apphttp "github.com/ChessPieceCat/Data-Processing-Platform/internal/http"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/redis"
	"github.com/ChessPieceCat/Data-Processing-Platform/internal/worker"
)

func main() {
	// Run database migrations.
	m := database.RunMigrations()
	defer database.CloseMigrations(m)

	// Open the application database connection.
	db := database.OpenDatabase()
	defer db.Close()

	// Verify that the database is reachable.
	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// Delete old jobs.
	if err := jobs.DeleteOldJobs(db, 10); err != nil {
		log.Fatal("Error deleting old jobs:", err)
	}

	// Open the Redis connection.
	redisClient := redis.OpenRedis()
	defer redisClient.Close()

	if err := redis.PingRedis(redisClient); err != nil {
		log.Fatal(err)
	}

	// Initialize the Redis consumer group for job processing.
	if err := redis.InitializeGroup(redisClient); err != nil {
		log.Fatal(err)
	}

	// Define the job processing function.
	processJob := func(jobID int64) error {

		return jobs.ProcessJob(
			db,
			jobID,
		)
	}

	// Recover any pending jobs that were not acknowledged due to worker crashes or failures.
	worker.RecoverPendingJobs(db, redisClient, processJob)

	// Start the worker in a separate goroutine.
	go worker.RunWorker(db, redisClient, processJob)

	// Serve the index.html file.
	tmpl, err := template.ParseFiles("web/index.html")
	if err != nil {
		log.Fatal(err)
	}

	// Serve static files.
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handle the root URL.
	http.HandleFunc("/", apphttp.IndexHandler(db, tmpl))

	// Handle the results page.
	http.HandleFunc("/results", apphttp.ResultsHandler(db))
	http.HandleFunc("/results/visualization", apphttp.VisualizationHandler(db))
	http.HandleFunc("/results/image", apphttp.ImageResultHandler(db))

	// Handle the download of result files.
	http.HandleFunc("/results/download", apphttp.DownloadResultsHandler(db))
	http.HandleFunc("/results/download/model", apphttp.DownloadModelHandler(db))
	http.HandleFunc("/results/download/metadata", apphttp.DownloadImageMetadataHandler(db))

	// Handle dataset inspection.
	http.HandleFunc("/inspect/dataset", apphttp.DatasetInspectionHandler)

	// handle uploads
	http.HandleFunc("/upload/image", apphttp.ImageUploadHandler)
	http.HandleFunc("/upload/route", apphttp.RouteUploadHandler)

	// Handle job submission.
	http.HandleFunc("/submit/dataset", apphttp.DatasetSubmissionHandler(db, redisClient))
	http.HandleFunc("/submit/image", apphttp.ImageSubmissionHandler(db, redisClient))
	http.HandleFunc("/submit/route", apphttp.RouteSubmissionHandler(db, redisClient))

	// Start the HTTP server.
	log.Println("Server is running on http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
