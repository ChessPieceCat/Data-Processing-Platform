# Data Processing Platform

A locally hosted web application for submitting data-processing jobs, tracking their status, and viewing or downloading generated results.

The project is currently in its first implementation. The initial job type is **dataset processing**; additional job types and models are planned for later iterations.

## Current Features

The current implementation supports:

- CSV upload and dataset inspection
- Job records stored in PostgreSQL
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
- Numeric and categorical feature preprocessing
- Missing-value imputation
- One-hot encoding for categorical model features
- Aggregated feature importance for original dataset columns
- Regression metrics including R² and mean squared error
- Optional actual-vs-predicted visualization
- Downloadable dataset and model result JSON files
- Job status tracking for queued, processing, completed, and failed jobs
- Automated Go and Python tests

Modeling currently supports the Random Forest Regressor; additional models are intended for later iterations.

## Architecture

The application currently uses Go for the web server and job management, PostgreSQL for persistent job state, and Python for dataset analysis and machine-learning processing.

```text
Browser
   |
   v
Go HTTP server
   |
   +---- PostgreSQL
   |
   +---- Job/filesystem storage
   |
   +---- Python dataset processor
             |
             +---- pandas
             +---- scikit-learn
             +---- matplotlib / seaborn
```

The Go application creates and manages jobs and invokes the Python dataset processor for dataset jobs. Dataset processing is performed by the Python processor as a separate process, and the resulting JSON and visualization artifacts are stored under the job's upload directory.

## Project Structure

```text
cmd/server/              Go application entry point

internal/database/       PostgreSQL connection and migrations

internal/http/           HTTP handlers, result models, and templates

internal/jobs/           Job lifecycle and dataset configuration

processors/dataset/      Python dataset processor and tests

web/                      HTML and browser-side JavaScript

docs/                     Development and architecture documentation
```

## Requirements

For the current local development setup, you need:

- Go 1.26.5
- Python 3.12
- PostgreSQL
- Python packages listed in `processors/dataset/requirements.txt`

The repository also contains a Docker Compose configuration for local PostgreSQL and pgAdmin development.

## Running Locally

Install the Python dependencies:

```bash
cd processors/dataset
python3 -m pip install -r requirements.txt
cd ../..
```

Start PostgreSQL using the repository's Compose configuration, then start the Go server:

```bash
go run ./cmd/server
```

The server listens on:

```text
http://localhost:8082
```

The web interface allows a CSV to be inspected before submitting a dataset job. For modeling jobs, the interface allows a target, feature-selection mode, Random Forest configuration, and requested visualizations to be selected.

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

The GitHub Actions workflow runs both test suites and builds the Go server.

## Results and Artifacts

Each job has a job-specific directory under `uploads/` containing the dataset, configuration, generated results, and any visualization or model artifacts that were produced.

The web results page displays the available dataset analysis, model results, and visualizations. Generated JSON files can also be downloaded from the results page.

## Current Scope

This repository intentionally represents an early stage of the platform. The current implementation is focused on establishing the core job workflow and the first dataset-processing pipeline.

Planned future work includes additional job types, additional machine-learning models, improved worker architecture and queuing, broader deployment infrastructure, and further operational and security work.

## Documentation

- [Development and Design](docs/data_processing_platform.md)