# Data Processing Platform

A locally hosted web application for submitting data-processing jobs, tracking their status, and viewing or downloading generated results.

The project is currently in its first implementation. The initial implemented job type is **dataset processing**; additional job types and machine-learning models are planned for later iterations.

## Current Features

The current implementation supports:

- CSV upload and dataset inspection
- Temporary upload handling and per-job dataset storage
- PostgreSQL-backed job records
- Redis Streams for asynchronous job processing
- Job lifecycle tracking:
  - queued
  - processing
  - completed
  - failed
- Dataset analysis including:
  - dataset dimensions and column names
  - inferred data types
  - missing-value counts and percentages
  - duplicate-row counts
  - unique-value counts
  - descriptive statistics
  - numeric and categorical summaries
  - outlier detection
  - feature-distribution analysis
  - numeric correlation analysis
- Optional dataset visualizations:
  - combined histograms and box plots
  - correlation heatmap
- Random Forest Regressor modeling
- Automatic or manual feature selection
- Configurable Random Forest parameters
- Numeric and categorical feature preprocessing
- Missing-value imputation
- One-hot encoding for categorical model features
- Feature-importance aggregation back to original dataset columns
- Regression metrics including R² and mean squared error
- Optional actual-vs-predicted visualization
- Separate dataset and model result artifacts
- Downloadable dataset and model result JSON files
- Controlled serving of generated visualization files
- Automatic cleanup of older jobs while retaining the most recent jobs
- Automated Go and Python tests

Modeling currently supports the Random Forest Regressor. Additional models and job types are intended for later iterations.

## Architecture

The application currently uses Go for the web server, job management, job dispatching, Redis queueing, and worker execution; PostgreSQL for persistent job state; and Python for dataset analysis and machine-learning processing.

```text
Browser
   |
   v
Go HTTP server
   |
   +---- PostgreSQL
   |
   +---- Redis Stream
             |
             v
          Go Worker
             |
             v
         ProcessJob
             |
       +-----+-----+
       |           |
       v           v
Dataset       Future job
processor     processors
   |
   v
Python dataset processor
   |
   +---- pandas
   +---- scikit-learn
   +---- matplotlib / seaborn
```

The Go application creates a job in PostgreSQL and enqueues its job ID in a Redis Stream. A worker consumes messages from the stream and passes the job ID to the generic job-processing layer.

`ProcessJob` retrieves the job from PostgreSQL, validates the shared job information, manages the common job lifecycle, and dispatches the job to the appropriate processor based on its type. Job-specific processor functions are responsible for performing the domain-specific work and producing result artifacts.

Redis Streams use at-least-once delivery semantics. The worker acknowledges messages after processing has completed or a permanent processing failure has been recorded. Pending messages can be recovered when a worker is restarted.

PostgreSQL remains responsible for persistent job state and metadata, while Redis is responsible for transporting work between the API and worker.

Generated JSON and visualization artifacts are stored under each job's directory in `uploads/`.

## Project Structure

```text
cmd/server/
    Go application entry point

internal/database/
    PostgreSQL connection and embedded migrations

internal/http/
    HTTP handlers, result models, and templates

internal/jobs/
    Job lifecycle, job dispatching, and dataset configuration

internal/redis/
    Redis connection and Stream configuration

internal/worker/
    Redis worker and pending-job recovery

processors/dataset/
    Python dataset processor and tests

    analysis.py
        Dataset analysis

    config.py
        Configuration loading and validation

    io_utils.py
        JSON serialization and result writing

    modeling.py
        Random Forest preprocessing, training, and evaluation

    visualizations.py
        Dataset and model visualizations

    main.py
        Processing pipeline orchestration

    test_*.py
        Python tests

web/
    HTML and browser-side JavaScript

docs/
    Development and architecture documentation
```

## Requirements

For the current local development setup, you need:

- Go 1.26.5
- Python 3.12
- PostgreSQL
- Redis
- Python packages listed in `processors/dataset/requirements.txt`

The repository contains a Docker Compose configuration for local development, including PostgreSQL, Redis, and pgAdmin.

## Running Locally

Install the Python dependencies:

```bash
cd processors/dataset
python3 -m pip install -r requirements.txt
cd ../..
```

Start the required services using the repository's Compose configuration.

Then start the Go server:

```bash
go run ./cmd/server
```

The server listens on:

```text
http://localhost:8082
```

The web interface allows a CSV to be inspected before submitting a dataset job. For modeling jobs, the interface allows a target, feature-selection mode, Random Forest configuration, and requested visualizations to be selected.

When a job is submitted, the API creates the job record and places a message containing the job ID on the Redis Stream. The worker consumes the message and passes the job to `ProcessJob` for asynchronous processing.

## Testing

Run the Go tests from the repository root:

```bash
go test ./...
```

Run the Python tests from the dataset processor directory:

```bash
cd processors/dataset
pytest -v
```

The GitHub Actions workflow runs the automated Go and Python test suites and builds the Go server.

## Results and Artifacts

Each job has a job-specific directory under `uploads/` containing the dataset, configuration, generated results, and any visualization or model artifacts that were produced.

A typical completed dataset job may contain:

```text
uploads/<job-id>/
├── dataset.csv
├── config.json
├── results.json
├── results_model_results.json
└── visualization artifacts
```

The results page displays:

- dataset analysis
- model configuration and results when modeling was requested
- feature importances
- regression metrics
- generated visualizations

Dataset and model result JSON files can also be downloaded from the results page.

## Current Scope

This repository represents the first implementation of the platform and is focused on establishing a complete dataset-processing workflow.

The current architecture separates generic job orchestration from domain-specific processing. Redis and the worker handle asynchronous job delivery, `ProcessJob` handles the common job lifecycle and dispatching, and individual processors handle job-specific execution.

Only dataset processing is currently implemented. Additional job types can be added through the job-processing registry without changing the Redis queue or worker architecture.

Planned future work includes:

- additional machine-learning models
- additional job types and processors
- further worker and processing abstractions where shared behavior warrants them
- improved concurrency and reliability handling
- broader integration testing
- deployment and infrastructure work
- monitoring and observability
- security hardening

## Documentation

- [Development and Design](docs/data_processing_platform.md)