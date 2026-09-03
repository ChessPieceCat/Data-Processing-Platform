# Data Processing Platform

A web application for submitting data-processing jobs, tracking their status, and viewing or downloading generated results.

The platform currently supports **dataset processing**, **image processing**, and **route optimization**. Additional job types and machine-learning models are planned for later iterations.

---

## Current Features

The current implementation supports:

- CSV upload and dataset inspection
- Image upload and image processing configuration
- Route CSV and distance-table CSV upload
- Route optimization configuration
- Temporary upload handling and per-job object storage
- PostgreSQL-backed job records
- Redis Streams for asynchronous job processing
- Separate Go HTTP server and Go worker processes
- Generic job processing and dispatch based on job type
- Storage abstraction supporting local filesystem and Amazon S3 backends
- Environment-based selection between local and S3 object storage
- Shared persistent job input, configuration, result, and artifact storage between the application and worker
- Job lifecycle tracking:

  - queued
  - processing
  - completed
  - failed
- Multiple concurrent worker replicas consuming from a shared Redis consumer group
- Unique Redis consumer identities for individual workers
- Pending-job recovery and reassignment after worker interruption
- Bounded retry handling for interrupted jobs
- Atomic job claiming to prevent duplicate processing
- Maximum outstanding-job limit of 100
- HTTP 429 backpressure when the outstanding-job limit is reached
- Context-aware worker cancellation and graceful shutdown
- Processor timeout handling
- Configurable PostgreSQL, Redis, and object-storage settings
- Worker throughput and latency benchmarking
- Verified end-to-end AWS deployment using Amazon S3 object storage
- Verified dataset, image, and route jobs execute successfully in AWS
- Verified generated result artifacts, including images and visualizations, are persisted to and served from Amazon S3
- AWS Secrets Manager integration for RDS database credentials
- IAM-based access to AWS Secrets Manager from EC2
- Systemd services for automatic API and worker startup and restart
- CloudWatch logging for API and worker services
- CloudWatch monitoring for EC2 CPU, memory, disk usage, and status checks
- Prometheus metrics for application, worker, queue, latency, and API error monitoring
- Prometheus Docker service discovery for application and worker containers
- Grafana monitoring dashboard
- Version-controlled Grafana datasource and dashboard provisioning
- Terraform infrastructure as code for AWS resources
- Terraform-managed AWS provider, networking references, security groups, EC2, RDS, S3, IAM, and CloudWatch resources
- Terraform remote state stored in Amazon S3 with state locking
- Existing AWS infrastructure imported and reconciled into Terraform state
- Terraform lifecycle validation using a disposable resource creation and destruction test
- GitHub Actions CI for Terraform formatting and validation, Go tests, Go vet, all Python processor tests, and ARM64 Docker image builds
- Terraform pull-request plans against the remote AWS state using a dedicated read-only AWS role
- ARM64 application images published to Amazon ECR from the main branch
- Automated EC2 deployment through AWS Systems Manager after successful main-branch image publication
- Separate GitHub OIDC roles for ECR publishing, Terraform planning, and EC2 deployment
- Protected main-branch workflow requiring CI and infrastructure checks before merging

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
- Duplicate-delivery protection through Redis consumer groups and atomic PostgreSQL job claiming
- Cleanup of temporary and per-job files
- Database and Redis health checks in the containerized development environment

### Monitoring and Observability

The application exposes Prometheus metrics for job processing, queue state, worker health, and API errors.

#### Application Metrics

The following metrics are exposed:

- `jobs_submitted_total` — number of successfully submitted jobs, labeled by job type
- `jobs_completed_total` — number of successfully completed jobs, labeled by job type
- `jobs_failed_total` — number of permanently failed jobs, labeled by job type and error type
- `job_processing_duration_seconds` — job processing duration histogram, labeled by job type
- `queue_depth` — current number of queued jobs, labeled by job type
- `worker_errors_total` — worker-level errors, labeled by error type
- `api_errors_total` — HTTP 5xx responses, labeled by method, route, and status

Processing throughput can be derived from `jobs_completed_total` using Prometheus rate calculations rather than requiring a separate throughput metric.

Queue depth is derived from PostgreSQL job state, which remains the source of truth for queued work. Metrics are refreshed when job state changes rather than requiring continuous database polling.

The Go HTTP server exposes its metrics through the `/metrics` endpoint. The worker process exposes its own metrics endpoint because the server and worker run as separate processes.

#### Error Classification

Application and worker errors are classified into controlled error types for use as Prometheus labels rather than using raw error messages or job-specific values. This keeps metric cardinality bounded while still allowing failures to be analyzed by category.

#### Monitoring Stack

Prometheus collects metrics from the Go HTTP server and each worker replica. Docker service discovery automatically discovers containers labeled for monitoring, allowing worker replicas to be added or removed without changing the Prometheus configuration.

Grafana provides the monitoring dashboard and uses Prometheus as its datasource. The datasource and dashboard are provisioned from version-controlled files in the repository so the monitoring configuration can be recreated automatically during deployment.

The current dashboard includes:

- job processing throughput

- job failure rate

- queue depth by job type

- p95 processing time by job type

- worker errors

- API errors

#### Observability Roadmap

The current observability implementation provides Prometheus instrumentation, automatic metric collection, Grafana dashboards, and contextual application and worker logging. Additional alerting, dashboard refinement, and production observability hardening are planned for later iterations.

### Platform

- Automatic cleanup of older jobs while retaining the most recent jobs
- Automated Go and Python tests
- GitHub Actions CI/CD for Terraform formatting and validation, Go tests and vet, Python processor tests, ARM64 Docker image builds, Amazon ECR publication, Terraform pull-request plans, and automated EC2 deployment
- Docker image builds are validated in CI, and images from the main branch are published to Amazon ECR
- Three worker replicas by default in Docker Compose
- Configurable worker scaling through Docker Compose
- Queue backpressure with a maximum of 100 outstanding queued or processing jobs

---

## Architecture

The application uses Go for the web server, job management, job submission, Redis queueing, worker execution, and storage orchestration; PostgreSQL for persistent job state; Redis Streams for asynchronous delivery; an object-storage abstraction backed by either the local filesystem or Amazon S3; and Python for domain-specific processing.

```text
Local Development

Browser
   |
   v
Go HTTP server
   |
   +---- PostgreSQL
   |
   +---- Redis
   |
   +---- Local filesystem storage
   |
   v
Go worker
   |
   v
ProcessJob
   |
   +---- Dataset processor
   +---- Image processor
   +---- Route processor


AWS Deployment

Browser
    |
    v
EC2 Docker Compose
    |
    +---- Go HTTP server
    |
    +---- Go worker
    |
    +---- Redis
    |
    +---- Amazon RDS PostgreSQL
    |
    +---- Amazon S3
    |
    +---- Prometheus
    |   |
    |   +---- Go HTTP server metrics
    |   |
    |   +---- Worker metrics
    |   |
    |   +---- Grafana dashboard
    v
ProcessJob
    |
    +---- Dataset processor
    +---- Image processor
    +---- Route processor
```

The application and worker are separate processes:

- The **Go HTTP server** handles requests, uploads, job creation, results, and queue submission.
- The **Go worker** consumes Redis Stream messages, recovers pending jobs on startup, and invokes `ProcessJob`.
- **PostgreSQL** stores persistent job state and metadata.
- **Redis** transports job IDs asynchronously between the application and worker.
- The application and worker use a shared object-storage abstraction for persistent job inputs, configuration, results, and generated artifacts.

The current AWS deployment runs the Go server, Go worker, Redis, Prometheus, and Grafana as Docker containers on the same EC2 instance, with Amazon RDS providing managed PostgreSQL and Amazon S3 providing persistent job storage. Prometheus discovers the application and worker containers through the Docker socket and scrapes their `/metrics` endpoints. Grafana uses Prometheus as its datasource and loads its dashboard and datasource configuration from version-controlled provisioning files. RDS master credentials are managed through AWS Secrets Manager and retrieved using the EC2 IAM role. CloudWatch collects application logs and EC2 monitoring metrics.

The production application image is built for ARM64 to match the t4g.micro EC2 instance and is published to Amazon ECR. The main branch deployment workflow uses GitHub OIDC to authenticate to AWS and AWS Systems Manager to update the EC2 deployment to the image associated with the triggering commit.

`ProcessJob` retrieves the job from PostgreSQL, validates the shared job information, manages the common job lifecycle, applies processor timeouts, and dispatches the job to the appropriate processor based on its type. Persistent job inputs and configuration are retrieved through the object-storage abstraction and materialized into a temporary local workspace because the Python processors operate on filesystem paths.

After processing, generated results and processor-specific artifacts are uploaded back through the object-storage abstraction. Result references stored in PostgreSQL and processor result JSON files point to persistent storage keys rather than temporary local filesystem paths.

Job-specific processors are responsible for performing domain-specific work and producing result artifacts. The shared job system does not need to be rewritten when another processor is added; a processor can be added for a new job type while preserving the common Redis and worker infrastructure.

Redis Streams use at-least-once delivery semantics through a shared consumer group. Each message is delivered to a single consumer within the group at a time, while remaining subject to at-least-once redelivery. The worker acknowledges messages after processing has completed successfully or after a permanent processing failure has been recorded.

Pending messages can be recovered and reassigned with Redis `XAUTOCLAIM` when they have remained idle longer than the configured recovery threshold. PostgreSQL job state is checked before recovered jobs are processed so completed and permanently failed jobs are not reprocessed.

Job claiming is also protected at the database level. A job can transition from `queued` to `processing` only once, and concurrent attempts to claim the same job are rejected after the first successful claim.

Workers use cancellable Redis reads and respond to process termination signals for graceful shutdown. Multiple workers can therefore operate concurrently while still allowing individual workers to stop and restart without taking down the worker pool.

The application also enforces a maximum of 100 outstanding jobs, counting both `queued` and `processing` jobs. Submissions beyond this limit are rejected with HTTP 429 rather than allowing an unbounded backlog to accumulate.

PostgreSQL remains responsible for persistent job state and metadata, while Redis is responsible for transporting work between the API and worker.

Persistent job inputs, configuration, results, and other job artifacts are stored through the object-storage abstraction. Local development uses the filesystem-backed implementation under uploads/, while AWS deployments use Amazon S3. Processor execution uses a temporary local workspace regardless of the configured persistent storage backend.

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

    Redis worker pool, consumer-group processing,

    pending-job recovery, graceful shutdown,

    concurrency tests, and performance benchmarks

internal/storage/

    Object-storage abstraction and implementations

monitoring/

    Prometheus configuration

    Grafana datasource provisioning

    Grafana dashboard provisioning

    Data Processing Platform dashboard

terraform/

    AWS infrastructure as code

    Provider and backend configuration

    EC2, RDS, S3, IAM, security groups, and CloudWatch resources

    Terraform variables and outputs
```
    storage.go
        Storage interface

    local.go
        Local filesystem storage implementation

    s3.go
        Amazon S3 storage implementation

    factory.go
        Environment-based storage backend selection
```


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
- Python 3.12 or Python 3.14
- PostgreSQL
- Redis
- Python packages listed in:

  - `processors/dataset/requirements.txt`
  - `processors/image/requirements.txt`

The dataset, image, and route processor test suites have been verified with Python 3.12 in CI and Python 3.14.4 on ARM64.

Amazon S3 is optional for local development. The application defaults to local filesystem storage, while S3 can be selected through `STORAGE_BACKEND=s3` and `S3_BUCKET`. AWS credentials are provided through the standard AWS SDK credential chain when using S3.

The route processor uses only Python standard-library modules and therefore does **not** require a `processors/route/requirements.txt` file.

For containerized development, the repository includes a Docker Compose configuration with PostgreSQL, Redis, pgAdmin, the application server, the worker pool, Prometheus, and Grafana. The application image installs Python and the dataset/image dependencies into `/opt/venv`.

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

By default, the application uses the local filesystem-backed storage implementation with `uploads/` as its base directory. Object storage can be selected through environment variables.

For local filesystem storage:

```bash
export STORAGE_BACKEND=local
```

For local PostgreSQL instances that do not support SSL, set:

```bash
export DB_SSLMODE=disable
```
Then start the Go server:

```bash
go run ./cmd/server
```

If you also want asynchronous job processing outside Docker, start one or more worker processes separately:

```bash
go run ./cmd/worker
```

The same storage configuration must be available to both the server and worker processes.

To run against Amazon S3 from a local development environment:

```bash
export STORAGE_BACKEND=s3
export S3_BUCKET=data-processing-platform-chess
export AWS_REGION=us-east-2
```

The AWS SDK uses the standard AWS credential/configuration chain. No AWS credentials are stored in the application source code.

Each worker uses its process hostname as its Redis consumer name, allowing multiple locally started workers to participate in the same Redis consumer group.

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
- `worker` — Go background worker pool
- `db` — PostgreSQL
- `redis` — Redis
- `pgadmin` — pgAdmin
- `prometheus` — Prometheus metrics collection and time-series storage
- `grafana` — Grafana monitoring dashboard

The default Compose configuration runs three worker replicas. Worker replicas share the same Redis consumer group while using unique consumer names derived from their container hostnames.

Additional worker capacity can be requested through Compose scaling, for example:

```bash
docker compose up -d --scale worker=5
```

The worker pool can therefore be scaled independently from the HTTP server.

Prometheus uses Docker service discovery to discover the application and worker containers. Worker replicas expose their own metrics endpoints, so scaling the worker service automatically adds or removes Prometheus targets without requiring changes to the Prometheus configuration.

Grafana connects to Prometheus through the Compose service name and loads the Data Processing Platform dashboard from the repository's provisioning files.

The application and worker use the Compose service names for PostgreSQL and Redis. Local Docker development uses the filesystem-backed object-storage implementation with the existing `uploads/` storage directory and shared `uploads_data` volume.

The same storage abstraction can be switched to Amazon S3 in AWS deployments without changing the job-processing interfaces.

The application is available at:

```text
http://localhost:8082
```

pgAdmin is available at:

```text
http://localhost:8083
```

Grafana is available at:

```text
http://localhost:3000
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

### Worker Performance Benchmark

Run the worker throughput benchmark with:

```bash
go test ./internal/worker \
  -bench BenchmarkWorkerThroughput \
  -benchtime=1x \
  -run '^$'
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

Worker tests additionally cover:

- concurrent processing across multiple workers
- Redis consumer-group message claiming
- pending-job recovery and reassignment
- prevention of duplicate job claims
- graceful worker shutdown
- queue acknowledgement behavior
- backpressure and outstanding-job limits
- concurrent submissions at the queue limit

Worker performance benchmarking measures:

- throughput in jobs per second
- average queue latency
- p95 queue latency
- average processing latency
- p95 processing latency
- average total latency
- p95 total latency

A controlled benchmark using a fixed 100 ms processing workload was used to compare one, two, and three concurrent workers.

The benchmark produced the following results on the development system:

| Workers | Throughput | Avg Queue Latency | Avg Total Latency |
|---:|---:|---:|---:|
| 1 | 9.9 jobs/sec | 1.47 s | 1.57 s |
| 2 | 19.8 jobs/sec | 718 ms | 818 ms |
| 3 | 29.6 jobs/sec | 466 ms | 567 ms |

Processing latency remained approximately 100 ms across all configurations. Increasing the worker count therefore increased throughput while reducing queue and total job latency.

The containerized monitoring stack has also been validated by confirming that Prometheus discovers the application and all configured worker replicas and reports healthy scrapes.

The GitHub Actions workflow runs the Go tests, all three Python test suites, and builds both `cmd/server` and `cmd/worker`.

---

## Results and Artifacts

Each job has a job-specific storage prefix containing its persistent input, configuration, generated result, and any additional artifacts produced by the processor. With local filesystem storage, these prefixes map to directories under `uploads/`; with Amazon S3, they map to object-key prefixes such as `jobs/<job-id>/`.

### Dataset Job

A typical completed dataset job may contain:

```text
jobs/<job-id>/

├── dataset.csv
├── config.json
├── results.json
├── results_model_results.json
├── results_feature_distributions.png
├── results_correlation_heatmap.png
└── results_actual_vs_predicted.png
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
jobs/<job-id>/

├── input.<ext>
├── config.json
├── processed.<ext>
├── results.json
└── metadata.json
```

The processed image extension depends on the selected output format. The original image, processed image, and metadata are stored as persistent object-storage artifacts and can be served or downloaded through the application.

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
jobs/<job-id>/

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

Redis pending messages are recovered when workers start and can be reassigned between consumers when they become eligible for recovery. Jobs left in a non-terminal state can be retried, while completed and permanently failed jobs are acknowledged without reprocessing.

Job attempts are tracked in PostgreSQL. Interrupted jobs are retried only up to the configured maximum attempt count. The current maximum is three processing attempts.

Job claiming is protected at the PostgreSQL level. Only jobs that are still in the `queued` state can be transitioned to `processing`, and the update result is checked so concurrent attempts to claim the same job cannot both succeed.

Workers run as independent Redis consumers within a shared consumer group. A worker can be stopped and restarted without stopping the remaining workers. Worker processes use signal-aware contexts and cancellable Redis reads so they can terminate cleanly.

The application also enforces backpressure by limiting the number of outstanding `queued` and `processing` jobs to 100. Attempts to submit work after the limit is reached receive HTTP 429 responses and do not create additional job records.

Processor execution is bounded by a 60-second timeout. A processor that exceeds the timeout is terminated and the job is recorded as failed.

Persistent job inputs, configuration, results, and processor-generated artifacts are accessed through the object-storage abstraction. Local development uses filesystem-backed storage, while AWS deployments use Amazon S3. Workers download persistent inputs into temporary local workspaces before invoking Python processors and upload generated results and artifacts back to object storage after processing. AWS deployments retrieve database credentials through AWS Secrets Manager using the EC2 IAM role rather than storing the database password in application configuration.

The system has been tested against:

- malformed input
- oversized uploads
- missing input files
- invalid parameters and configuration
- worker crashes during processing
- database failure
- Redis/queue failure
- duplicate job delivery
- concurrent job processing
- concurrent job claiming
- pending-job reassignment
- partial processing
- processor timeouts
- application/worker restart during processing
- worker shutdown and restart
- queue-limit enforcement
- concurrent submissions at the queue limit

Redis uses at-least-once delivery semantics, so duplicate delivery is possible. Redis consumer-group ownership and PostgreSQL job claiming work together to prevent an already-claimed job from being started a second time.

---

## Current Scope

This repository represents the first implementation of the platform.

The current architecture separates generic job orchestration from domain-specific processing. The Go HTTP server handles API requests and submission, Redis and the worker pool handle asynchronous delivery, `ProcessJob` handles the common job lifecycle and processor dispatching, and individual processors handle job-specific execution.

The worker pool uses Redis consumer groups for concurrent processing, PostgreSQL for atomic job claiming and job-state authority, and bounded queue admission to prevent unbounded outstanding work.

Dataset, image, and route processing currently share the same job system and worker infrastructure. Adding another processor does not require rewriting the core queue or worker architecture.

Current job types:

- dataset
- image
- route

Additional job types and processors are planned for later iterations.

AWS deployment is now operational for the core application and is represented as Terraform-managed infrastructure. The deployment uses an ARM64 Docker image running the Go HTTP server, Go worker, and Redis on an EC2 instance; Amazon RDS for managed PostgreSQL; Amazon S3 for persistent job storage; AWS Secrets Manager for managed RDS credentials; IAM-based access from EC2; and CloudWatch logging and monitoring. Dataset, image, and route jobs have been successfully executed end-to-end in AWS, including persistence and retrieval of generated result artifacts.

GitHub Actions provides continuous integration, ARM64 image publication to Amazon ECR, Terraform pull-request planning, and automated EC2 deployment through AWS Systems Manager. GitHub OIDC is used for AWS authentication, with separate roles for Terraform planning, image publication, and deployment.

Planned future work includes:

- additional machine-learning models
- additional routing algorithms
- additional route constraints and time-window handling
- route visualizations or other specialized artifacts
- additional job types and processors
- further worker and processing abstractions where shared behavior warrants them
- broader integration testing
- additional AWS production hardening
- additional monitoring dashboards, alerting, and production observability hardening

---

## Documentation

- [Development and Design](docs/data_processing_platform.md)