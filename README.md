# Data Processing Platform

A locally hosted web application for submitting data-processing jobs, tracking their status, and viewing or downloading generated results.

The platform currently supports **dataset processing**, **image processing**, and **route optimization**. Additional job types and machine-learning models are planned for later iterations.

---

## Current Features

The current implementation supports:

- CSV upload and dataset inspection
- Image upload and image processing configuration
- Route CSV and distance-table CSV upload
- Route optimization configuration
- Temporary upload handling and per-job storage
- PostgreSQL-backed job records
- Redis Streams for asynchronous job processing
- Separate Go HTTP server and Go worker processes
- Generic job processing and dispatch based on job type
- Shared job input/result storage between the application and worker
- Job lifecycle tracking:
  - queued
  - processing
  - completed
  - failed
- Worker recovery of pending jobs after worker crashes or restarts
- Retry limits for interrupted jobs
- Processor timeout handling
- Configurable database and Redis connection settings

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

### Route Optimization

- Generalized route/location input using CSV
- Separate distance-table CSV input
- Configurable starting location
- Optional ending location
- Nearest-neighbor initial route construction
- 2-opt route optimization
- Support for open and closed routes
- Configurable maximum distance constraint
- Configurable maximum stop constraint
- Route feasibility checking
- Distance improvement and improvement-percentage reporting
- Algorithm and runtime reporting
- Route result JSON generation
- Route result presentation through the existing job results system
- Independent Python unit tests for configuration, locations, distance tables, routing, constraints, and processing

The current route processor uses the **nearest-neighbor + 2-opt** algorithm. Location demand, priority, and time-window values are stored and validated, but they do not automatically alter the routing algorithm unless an explicit supported rule or constraint is introduced.

### Reliability

- Predictable completed and failed job states
- Detailed job error information
- Pending-job recovery after worker interruption or restart
- Bounded retries for interrupted jobs
- Processor execution timeouts
- Safe handling of duplicate job delivery where practical
- Cleanup of temporary and per-job files
- Database and Redis health checks in the containerized development environment

### Platform

- Automatic cleanup of older jobs while retaining the most recent jobs
- Automated Go and Python tests
- GitHub Actions CI for Go tests, all Python processor tests, and both Go binaries

---

## Architecture

The application uses Go for the web server, job management, job submission, Redis queueing, and worker execution; PostgreSQL for persistent job state; Redis Streams for asynchronous delivery; and Python for domain-specific processing.

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
       +-----+-----+-----+
       |           |     |
       v           v     v
    Dataset      Image  Route
    processor    processor processor
       |           |       |
       v           v       v
     Python      Python  Python
     dataset      image   route
    processor   processor processor
       |           |
       |           +--- Pillow
       |
       +--- pandas
       +--- scikit-learn
       +--- matplotlib / seaborn
```

The application and worker are separate processes:

- The **Go HTTP server** handles requests, uploads, job creation, results, and queue submission.
- The **Go worker** consumes Redis Stream messages, recovers pending jobs on startup, and invokes `ProcessJob`.
- **PostgreSQL** stores persistent job state and metadata.
- **Redis** transports job IDs asynchronously between the application and worker.
- A shared `uploads/` volume makes job inputs, configuration, and generated artifacts available to both application and worker containers.

The Go application creates a job in PostgreSQL and enqueues its job ID in a Redis Stream. The worker consumes the message and passes the job to `ProcessJob`.

`ProcessJob` retrieves the job from PostgreSQL, validates the shared job information, manages the common job lifecycle, applies processor timeouts, and dispatches the job to the appropriate processor based on its type. The processor is discovered from the job type and executed with the job's input, output, and configuration paths.

Job-specific processors are responsible for performing domain-specific work and producing result artifacts. The shared job system does not need to be rewritten when another processor is added; a processor can be added for a new job type while preserving the common Redis and worker infrastructure.

Redis Streams use at-least-once delivery semantics. The worker acknowledges messages after processing has completed successfully or after a permanent processing failure has been recorded. Pending messages can be recovered when a worker is restarted. Interrupted jobs are tracked with an attempt count and are prevented from retrying indefinitely.

PostgreSQL remains responsible for persistent job state and metadata, while Redis is responsible for transporting work between the API and worker.

Generated JSON, processed files, route inputs, and visualization artifacts are stored under each job's directory in `uploads/`. In the containerized development setup, this directory is backed by a shared Compose volume mounted by both the application and worker.

---

## Project Structure

```text
cmd/server/
    Go HTTP server entry point

cmd/worker/
    Go worker entry point

internal/database/
    PostgreSQL connection and embedded migrations

internal/http/
    HTTP handlers, result models, and templates

internal/jobs/
    Job lifecycle, job dispatching, processor execution,
    processing configuration, retries, and timeouts

internal/redis/
    Redis connection and Stream configuration

internal/worker/
    Redis worker, pending-job recovery, and retry limits

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

    requirements.txt
        Dataset processor dependencies

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

processors/route/
    Python route optimization processor and tests

    config.py
        Routing configuration loading and validation

    constraints.py
        Constraint validation and route feasibility

    distance_table.py
        Distance-table loading and validation

    locations.py
        Route location model and CSV loading

    routing.py
        Nearest-neighbor and 2-opt routing

    main.py
        Processing pipeline orchestration

    test_*.py
        Python tests

web/
    HTML and browser-side JavaScript

docs/
    Development and architecture documentation
```

---

## Requirements

For local development outside Docker, you need:

- Go 1.26.5
- Python 3.12
- PostgreSQL
- Redis
- Python packages listed in:
  - `processors/dataset/requirements.txt`
  - `processors/image/requirements.txt`

The route processor uses only Python standard-library modules and therefore does **not** require a `processors/route/requirements.txt` file.

For containerized development, the repository includes a Docker Compose configuration with PostgreSQL, Redis, pgAdmin, the application server, and the worker. The application image installs Python and the dataset/image dependencies into `/opt/venv`.

---

## Running Locally

### Host-based development

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

The route processor has no third-party runtime dependencies.

Start PostgreSQL and Redis using your local development environment.

Then start the Go server:

```bash
go run ./cmd/server
```

If you also want asynchronous job processing outside Docker, start the worker separately:

```bash
go run ./cmd/worker
```

The server listens on:

```text
http://localhost:8082
```

The web interface supports submitting dataset, image, and route jobs.

For dataset jobs, the interface allows a CSV to be inspected before submission. Modeling options include target selection, feature-selection mode, Random Forest configuration, and requested visualizations.

For image jobs, the interface allows image-specific operations such as resizing, compression, format conversion, and metadata extraction to be selected.

For route jobs, the interface accepts a route CSV and distance-table CSV and allows the starting location, optional ending location, optimization algorithm, maximum distance, and maximum stops to be configured.

When a job is submitted, the API creates the job record and places a message containing the job ID on the Redis Stream. The worker consumes the message and passes the job to `ProcessJob` for asynchronous processing.

### Docker Compose development

Build and start the full stack:

```bash
docker compose up -d --build
```

The Compose environment includes:

- `app` — Go HTTP server
- `worker` — Go background worker
- `db` — PostgreSQL
- `redis` — Redis
- `pgadmin` — pgAdmin

The application and worker use the Compose service names for PostgreSQL and Redis. The application and worker share the `uploads_data` volume so both can access the same job directories.

The application is available at:

```text
http://localhost:8082
```

pgAdmin is available at:

```text
http://localhost:8083
```

The Docker image installs Python processor dependencies in `/opt/venv`. The `PYTHON_BIN` environment variable is used to select the processor interpreter inside the container.

---

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

Run the route processor tests:

```bash
cd processors/route
pytest -v
cd ../..
```

The route test suite covers:

- configuration loading and validation
- location parsing and validation
- distance-table loading and validation
- nearest-neighbor routing
- route-distance calculation
- 2-opt optimization
- constraint validation
- route feasibility checking
- processor orchestration

The Go test suite covers application handlers, job processing, configuration, Redis behavior, worker behavior, database integration, and route integration.

The GitHub Actions workflow runs the Go tests, all three Python test suites, and builds both `cmd/server` and `cmd/worker`.

---

## Results and Artifacts

Each job has a job-specific directory under `uploads/` containing its input, configuration, generated results, and any additional artifacts produced during processing.

### Dataset Job

A typical completed dataset job may contain:

```text
uploads/<job-id>/

├── dataset.csv
├── config.json
├── results.json
├── results_model_results.json
└── visualization artifacts
```

The dataset results page displays:

- dataset analysis
- model configuration and results when modeling was requested
- feature importances
- regression metrics
- generated visualizations

Dataset and model result JSON files can also be downloaded from the results page.

### Image Job

A typical completed image job may contain:

```text
uploads/<job-id>/

├── input.jpg
├── config.json
├── processed.jpg
├── results.json
└── metadata.json
```

The image results page displays:

- original image
- processed image
- processing operations performed
- original dimensions and format
- resulting dimensions and format
- compression statistics when compression was requested
- selected metadata summary when metadata extraction was requested

When full metadata is extracted, the complete metadata is stored separately as `metadata.json` and can be downloaded from the results page.

### Route Job

A typical completed route job may contain:

```text
uploads/<job-id>/

├── route.csv
├── distances.csv
├── config.json
└── results.json
```

The route results page displays:

- starting location
- ending location
- initial route
- optimized route
- initial distance
- optimized distance
- distance improvement
- improvement percentage
- feasibility
- optimization algorithm
- 2-opt status
- runtime

Route jobs use `results.json` as their primary processing artifact. Additional binary artifacts or route visualizations can be added in later iterations.

---

## Route Input Format

The route processor expects a location CSV with the following columns:

```csv
id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,08:00,17:00
2,Customer A,10,1,09:00,12:00
3,Customer B,5,2,10:00,14:00
4,Customer C,8,1,11:00,15:00
```

Each location has a unique ID and name. Demand and priority are non-negative integers. Time windows use `HH:MM` and are stored internally as minutes from midnight.

The distance input is a complete CSV distance matrix:

```csv
,Warehouse,Customer A,Customer B,Customer C
Warehouse,0,5.2,7.1,3.4
Customer A,5.2,0,2.8,6.0
Customer B,7.1,2.8,0,4.1
Customer C,3.4,6.0,4.1,0
```

The distance table must contain every supported location and every required distance relationship. Distances must be numeric and non-negative.

Route configuration is stored separately from the location data:

```json
{
  "start_location": "Warehouse",
  "end_location": "Warehouse",
  "optimization": {
    "algorithm": "nearest_neighbor_2opt"
  },
  "constraints": {
    "max_distance": 140.0,
    "max_stops": 40
  }
}
```

The route processor first constructs a nearest-neighbor route and then applies 2-opt optimization. The optimized route is checked against the configured feasibility constraints before the job is considered successful.

---

## Reliability and Failure Handling

The job system distinguishes between successful and failed processing states and records useful error information in PostgreSQL.

Redis pending messages are recovered when the worker starts. Jobs left in a non-terminal state can be retried, while completed and permanently failed jobs are acknowledged without reprocessing.

Job attempts are tracked in PostgreSQL. Interrupted jobs are retried only up to the configured maximum attempt count. The current maximum is three processing attempts.

Processor execution is bounded by a 60-second timeout. A processor that exceeds the timeout is terminated and the job is recorded as failed.

The system has been tested against:

- malformed input
- oversized uploads
- missing input files
- invalid parameters and configuration
- worker crashes during processing
- database failure
- Redis/queue failure
- duplicate job delivery
- partial processing
- processor timeouts
- application/worker restart during processing

Redis uses at-least-once delivery semantics, so duplicate delivery is possible. Processing is designed to tolerate duplicate execution where practical without creating duplicate PostgreSQL job records.

---

## Current Scope

This repository represents the first implementation of the platform.

The current architecture separates generic job orchestration from domain-specific processing. The Go HTTP server handles API requests and submission, Redis and the dedicated worker handle asynchronous delivery, `ProcessJob` handles the common job lifecycle and processor dispatching, and individual processors handle job-specific execution.

Dataset, image, and route processing currently share the same job system and worker infrastructure. Adding another processor does not require rewriting the core queue or worker architecture.

Current job types:

- dataset
- image
- route

Additional job types and processors are planned for later iterations.

Planned future work includes:

- additional machine-learning models
- additional routing algorithms
- additional route constraints and time-window handling
- route visualizations or other specialized artifacts
- additional job types and processors
- further worker and processing abstractions where shared behavior warrants them
- improved concurrency and reliability handling
- broader integration testing
- deployment and infrastructure work
- monitoring and observability
- security hardening

---

## Documentation

- [Development and Design](docs/data_processing_platform.md)