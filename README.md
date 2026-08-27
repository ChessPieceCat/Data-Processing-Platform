# Data Processing Platform

A locally hosted web application for submitting data-processing jobs, tracking their status, and viewing or downloading generated results.

The platform currently supports **dataset processing** and **image processing**. Additional job types and machine-learning models are planned for later iterations.

## Current Features

The current implementation supports:

- CSV upload and dataset inspection
- Image upload and image processing configuration
- Temporary upload handling and per-job storage
- PostgreSQL-backed job records
- Redis Streams for asynchronous job processing
- Generic job processing and dispatch based on job type
- Job lifecycle tracking:
  - queued
  - processing
  - completed
  - failed

### Dataset Processing

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

Modeling currently supports the Random Forest Regressor. Additional models are planned for later iterations.

### Image Processing

- Image resizing
- Image compression with format-specific handling
- Image format conversion
- Metadata extraction
- Image-specific processing parameters
- Original and processed image result presentation
- Image processing operation reporting
- Original and resulting image dimensions and formats
- Compression statistics including original size, result size, and compression ratio
- Cleaned metadata summaries for result presentation
- Full metadata storage as a separate JSON artifact
- Downloadable full image metadata
- Controlled serving of original and processed image files

Image processing currently uses Python and Pillow.

### Platform

- Automatic cleanup of older jobs while retaining the most recent jobs
- Automated Go and Python tests
- GitHub Actions CI for Go tests, dataset processor tests, image processor tests, and Go builds

## Architecture

The application uses Go for the web server, job management, job dispatching, Redis queueing, and worker execution; PostgreSQL for persistent job state; and Python for domain-specific processing.

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
      +------+------+------+
      |      |      |
      v      v      v
   Dataset Image  Route
  processor processor processor
      |      |      |
      v      v      v
   Python Python  Future
   dataset image  processor
   processor processor
      |      |
      |      +---- Pillow
      |
      +---- pandas
      +---- scikit-learn
      +---- matplotlib / seaborn
```

The Go application creates a job in PostgreSQL and enqueues its job ID in a Redis Stream. A worker consumes messages from the stream and passes the job ID to `ProcessJob`.

`ProcessJob` retrieves the job from PostgreSQL, validates the shared job information, manages the common job lifecycle, and dispatches the job to the appropriate processor based on its type. The processor is discovered from the job type and executed with the job's input, output, and configuration paths.

Job-specific processors are responsible for performing domain-specific work and producing result artifacts. The shared job system does not need to be rewritten when another processor is added; a processor can be added for a new job type while preserving the common Redis and worker infrastructure.

Redis Streams use at-least-once delivery semantics. The worker acknowledges messages after processing has completed successfully or after a permanent processing failure has been recorded. Pending messages can be recovered when a worker is restarted.

PostgreSQL remains responsible for persistent job state and metadata, while Redis is responsible for transporting work between the API and worker.

Generated JSON, processed files, and visualization artifacts are stored under each job's directory in `uploads/`.

## Project Structure

```text
cmd/server/

    Go application entry point

internal/database/

    PostgreSQL connection and embedded migrations

internal/http/

    HTTP handlers, result models, and templates

internal/jobs/

    Job lifecycle, job dispatching, processor execution,
    and processing configuration

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

processors/image/

    Python image processor and tests

    config.py

        Image-processing configuration and validation

    metadata.py

        Metadata extraction, normalization, and summaries

    processing.py

        Image processing and result generation

    main.py

        Processing pipeline orchestration

    requirements.txt

        Image processor dependencies

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
- Python packages listed in:
  - `processors/dataset/requirements.txt`
  - `processors/image/requirements.txt`

The repository contains a Docker Compose configuration for local development, including PostgreSQL, Redis, and pgAdmin.

## Running Locally

Install the dataset processor dependencies:

```bash
cd processors/dataset
python3 -m pip install -r requirements.txt
cd ../..
```

Install the image processor dependencies:

```bash
cd processors/image
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

The web interface supports submitting dataset and image jobs.

For dataset jobs, the interface allows a CSV to be inspected before submission. Modeling options include target selection, feature-selection mode, Random Forest configuration, and requested visualizations.

For image jobs, the interface allows image-specific operations such as resizing, compression, format conversion, and metadata extraction to be selected.

When a job is submitted, the API creates the job record and places a message containing the job ID on the Redis Stream. The worker consumes the message and passes the job to `ProcessJob` for asynchronous processing.

## Testing

Run the Go tests from the repository root:

```bash
go test ./...
```

Run the dataset processor tests:

```bash
cd processors/dataset
pytest -v
cd ../..
```

Run the image processor tests:

```bash
cd processors/image
pytest -v
cd ../..
```

The GitHub Actions workflow runs the Go tests, both Python test suites, and builds the Go server.

## Results and Artifacts

Each job has a job-specific directory under `uploads/` containing its input, configuration, generated results, and any additional artifacts produced during processing.

A typical completed dataset job may contain:

```text
uploads/<job-id>/

├── dataset.csv
├── config.json
├── results.json
├── results_model_results.json
└── visualization artifacts
```

A typical completed image job may contain:

```text
uploads/<job-id>/

├── input.jpg
├── config.json
├── processed.jpg
├── results.json
└── metadata.json
```

### Dataset Results

The dataset results page displays:

- dataset analysis
- model configuration and results when modeling was requested
- feature importances
- regression metrics
- generated visualizations

Dataset and model result JSON files can also be downloaded from the results page.

### Image Results

The image results page displays:

- original image
- processed image
- processing operations performed
- original dimensions and format
- resulting dimensions and format
- compression statistics when compression was requested
- selected metadata summary when metadata extraction was requested

When full metadata is extracted, the complete metadata is stored separately as `metadata.json` and can be downloaded from the results page.

## Current Scope

This repository represents the first implementation of the platform.

The current architecture separates generic job orchestration from domain-specific processing. Redis and the worker handle asynchronous job delivery, `ProcessJob` handles the common job lifecycle and processor dispatching, and individual processors handle job-specific execution.

Dataset and image processing currently share the same job system and worker infrastructure. Adding another processor does not require rewriting the core queue or worker architecture.

Current job types:

- dataset
- image

Additional job types and processors are planned for later iterations.

Planned future work includes:

- additional machine-learning models
- additional job types and processors
- route processing and optimization
- further worker and processing abstractions where shared behavior warrants them
- improved concurrency and reliability handling
- broader integration testing
- deployment and infrastructure work
- monitoring and observability
- security hardening

## Documentation

- [Development and Design](docs/data_processing_platform.md)