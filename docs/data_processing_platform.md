# Data Processing Platform - Development and Design

## [Roadmap](data_processing_platform_roadmap.md)
## Overview

A web-based application for submitting computational jobs and retrieving results.

This is primarily a backend and systems engineering portfolio project. The web interface should be useful and clean, but the main objective is to demonstrate the ability to design, build, test, containerize, deploy, and operate a multi-service application.

The platform will support three different workloads:
- Dataset analysis and modeling from uploaded CSV files
- Image processing such as resizing, compression, format conversion, and metadata extraction
- Route optimization using 2-opt and a defined set of constraints

The application will begin as a locally hosted dockerized system and will slowly evolve into a cloud-deployed application using AWS and Terraform.

The project should be developed incrementally. Build the simplest version first and introduce complexity as the application gives reason for it.

# Primary Goals

## Engineering Goals

- Go backend development
- REST APIs
- HTTP and web application design
- PostgreSQL and SQL
- File uploads and file storage
- Python data processing
- Asynchronous jobs
- Queues
- Worker services
- Concurrency
- Failure handling and retries
- Docker and Docker Compose
- AWS infrastructure
- Terraform and Infrastructure as Code
- GitHub Actions and CI/CD
- Monitoring and observability
- Technical documentation
- Architectural decision-making

## Portfolio Goals

- Build a nontrivial backend system from the ground up.
- Separate an API from expensive background work.
- Design persistent job state.
- Work across Go and Python in one system.
- Design around queues and workers.
- Handle failures instead of assuming everything succeeds.
- Containerize a multi-service application.
- Deploy an application to AWS.
- Define cloud infrastructure using Terraform.
- Build automated tests and CI/CD.
- Explain technical tradeoffs and architectural decisions.

## Learning Goals

Early:

- HTTP
- Go application structure
- SQL
- PostgreSQL
- file handling
- Python processing

Middle:

- queues
- worker coordination
- concurrency
- retries
- failure handling
- generic job abstractions

Late:

- AWS
- networking
- IAM
- managed storage and databases
- infrastructure as code
- deployment automation
- observability

# Product Concept

A user submits a job through a web interface by providing the required input and selecting the operation. The application creates a persistent job record and places the work onto a queue.

A worker retrieves the job, performs the computation, stores the resulting artifacts and metadata, and updates the job status.

The user can return to the job page to view its current state and inspect or download results when available.

# Product Scope

## Dataset Analysis

The platform will accept an uploaded CSV file and produce an analysis report.

### Dataset Overview

- Number of rows
- Number of columns
- Column names
- Inferred data types
- Number of unique values
- Basic descriptive statistics

### Data Quality

- Missing-value counts
- Missing-value percentages
- Potentially duplicated rows
- Potentially invalid or inconsistent values
- Outlier information where appropriate

### Feature Analysis

- Feature distributions
- Histograms or comparable visualizations
- Categorical frequency information
- Numeric summaries

### Relationships

- Correlation analysis for suitable numeric features
- Correlation matrix visualization
- Potentially additional relationship analysis later

### Visualization

Graphs should be generated when useful rather than forcing every possible graph to be created automatically.

Possible outputs include:

- Histograms
- Box plots
- Correlation heatmaps
- Scatter plots
- Predicted-vs-actual plots for regression

# Modeling

Classification or regression modeling is an optional part of dataset processing rather than running automatically on every upload.

The user should be able to configure a job or analysis option.

```text
Target variable:   [ target_column      v ]

Features:
    ( ) Automatic feature selection
    ( ) Select features manually

Selected features:
    [ feature_a ] [ feature_b ] [ feature_c ]

Model:
    [ Random Forest Regressor v ]

Options:
    [x] Generate prediction plot
```

## Automatic vs Manual Features

With the automatic option, the system determines a reasonable set of features based on the selected target and dataset characteristics with conservative assumptions and clear reporting on which features were selected.

The manual option allows the user to select the columns used by the model.

## Initial Model

A random forest regressor will be used as the first implemented model.

Additional models will be added later.

## Results

Possible results include:

- R²
- MSE
- RMSE
- MAE
- Training/test split information
- Feature importance where supported
- Predicted-vs-actual visualization
- Model configuration
- Processing duration

Results should record enough metadata to understand how they were produced.

The system should avoid presenting results without the information required to interpret them.

# Image Processing

Initial operations should include:

- Resize
- Compression
- Format conversion
- Metadata extraction

Potential later operations:

- Crop
- Rotate
- Thumbnail generation
- Strip metadata
- Additional transformations

A user may select one or more operations depending on the final design.

The worker should produce both the processed image and useful metadata such as the original dimensions, result dimensions, original vs result format, and so on.

Security will be important for this portion.

# Route Optimization

This system will accept a set of locations, constraints, and vehicle info and attempt to produce an optimized route.

Vehicle info may include capacity and number of vehicles.

The initial algorithm will be 2-opt, but more may be added as options.

## Potential Constraints

- Starting location
- Ending location
- Time windows
- Delivery groups that should remain together
- Fragile deliveries
- Delivery priority
- Required ordering relationships
- Maximum route duration
- Maximum distance
- Vehicle capacity

## Conceptual Model

```text
Input
  |
  v
Validate locations and constraints
  |
  v
Construct feasible initial route
  |
  v
Evaluate route
  |
  v
Try 2-opt improvement
  |
  v
Check whether proposed route remains valid
  |
  +---- invalid ----> reject swap
  |
  +---- valid ------> evaluate improvement
  |
  v
Repeat until no useful improvement remains
  |
  v
Return final route and statistics
```

## Route Results

Results should include:

- Final route
- Ordered locations
- Total distance
- Estimated duration if supported
- Algorithm used
- Constraints applied
- Whether constraints were satisfied
- Processing duration
- Potentially route improvement from the initial solution

# Common Job Model

The three processing types should share a common job concept despite using different domain logic.

```text
Job
├── ID
├── Type
├── Status
├── Input reference
├── Parameters
├── Created time
├── Started time
├── Completed time
├── Error information
└── Result reference
```

Initial job states should be:

- `queued`
- `processing`
- `completed`
- `failed`

# Architecture

## API and Worker Separation

The API should not perform any expensive processing inside the HTTP request.

```text
HTTP request
    |
    v
Create job
    |
    v
Queue work
    |
    v
Return job ID
```

The client can then retrieve job status independently.

## Persistent Job State

The job state belongs in a persistent system of record rather than only in application memory.

PostgreSQL should initially be the source of truth for job state.

## Files Are Not Database Rows

Large uploaded and generated files should not normally be stored directly in PostgreSQL.

The design should separate relational metadata in PostgreSQL and file artifacts in filesystem/object storage.

Local storage may be used during initial development.

## Complexity

Functions should be added only when justified and only when current versions are working.

# Technology Stack

## Backend

- Go
- HTTP/REST API
- HTML templates
- JavaScript
- CSS

## Data Processing

- Python
- scikit-learn
- Image-processing libraries

## Database

- PostgreSQL

## Local Infrastructure

- Docker
- Docker Compose

## Version Control

- Github

## CI/CD

- Github Actions

## Cloud

Eventually:

- AWS EC2
- SQS
- S3
- RDS
- IAM
- VPC
- Security Groups
- CloudWatch

## Infrastructure as Code

- Terraform

# Testing

Testing should grow alongside the application.

## Go Tests

- HTTP handlers
- validation
- job creation
- database interactions
- queue interactions
- job state transitions
- error handling

## Python Tests

- data analysis functions
- input validation
- regression preparation
- regression results
- image processing functions
- route optimization behavior
- constraint validation

## Integration Tests

```text
API
 -> Database
 -> Queue
 -> Worker
 -> File/Result Storage
```

# Documentation

Documentation should be written during development.

## README

- project overview
- features
- architecture diagram
- technology stack
- local setup
- Docker setup
- API overview
- dataset processing
- image processing
- route optimization
- AWS architecture
- Terraform usage
- CI/CD
- security notes
- troubleshooting
- lessons learned

## Decision Records

Important decisions should be documented with:

- Problem
- Considered alternatives
- Decision
- Reasoning
- Tradeoffs
- Consequences

## Troubleshooting Documentation

Document problems as they occur.

# Security

## File Upload Security

- reasonable upload limits
- validated file types
- safe filenames
- safe temporary directories
- safe output paths
- cleanup of temporary artifacts

## Input Validation

- values accepted by the API
- values consumed by workers

Workers should not assume that the API has already made every input safe.

## Cloud Security

- least-privilege IAM
- restricted database access
- appropriately scoped security groups
- private resources where appropriate
- protected secrets
- HTTPS for public communication

# Preventing Scope Creep

1. What problem does this solve?
2. Could the current architecture handle it simply?
3. What will be learned by adding it?
4. Will it make the core system harder to understand?
5. Is it necessary now?

Defer features that need not be added immediately.

# Milestones

## Milestone 1 — First Working Dataset Job

```text
Browser
  |
  v
Go API
  |
  +----> PostgreSQL
  |
  v
Dataset Worker
  |
  v
PostgreSQL / Result Storage
```

A user can upload a CSV and receive an analysis result.

## Milestone 2 — Real Asynchronous Processing

```text
Browser
  |
  v
Go API
  |
  v
Queue
  |
  v
Dataset Worker
```

The API no longer performs expensive work directly.

## Milestone 3 — Multi-Workload Platform

```text
             Queue
          /     |     \
      Dataset Image Route
       Worker  Worker Worker
```

The platform supports all three processing domains.

## Milestone 4 — Reliable Local System

The platform has:

- retries
- failure states
- concurrent workers
- integration testing
- useful logs

## Milestone 5 — Cloud Deployment

The working system runs on AWS.

## Milestone 6 — Infrastructure as Code

The AWS infrastructure can be reproduced using Terraform.

## Milestone 7 — Automated Deployment and Observability

The project has:

- automated testing
- container builds
- deployment automation
- monitoring
- useful operational telemetry

# 23. First Implementation Checklist

- [x] Create GitHub repository.
- [x] Initialize Go module.
- [x] Build minimal Go HTTP server.
- [x] Create initial HTML interface.
- [x] Add Docker Compose.
- [x] Run PostgreSQL locally.
- [x] Create the first migration.
- [x] Create the initial `jobs` table.
- [x] Implement basic job creation.
- [x] Implement basic job retrieval.
- [x] Add CSV upload.
- [x] Add safe local file storage.
- [x] Create the first dataset worker.
- [x] Implement dataset shape analysis.
- [x] Implement missing-value analysis.
- [x] Implement descriptive statistics.
- [x] Implement basic distributions.
- [x] Implement correlations where applicable.
- [x] Display results in the browser.
- [x] Add regression configuration.
- [x] Add Random Forest Regressor.
- [x] Add automatic feature selection.
- [x] Add manual feature selection.
- [x] Add useful regression metrics.
- [x] Add regression visualization.
- [ ] Add tests.
- [ ] Document the first architecture.

**Do not continue into queues, image processing, route optimization, AWS, or Terraform until this checklist has produced a stable first end-to-end workflow.**