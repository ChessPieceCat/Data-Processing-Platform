package server

import "text/template"

var DatasetResultsTemplate = template.Must(template.New("dataset_results").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Job #{{.Job.ID}} Results</title>
    <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
<div class="results-page">
    <h1 class="result-title">Job #{{.Job.ID}} Results</h1>

    <section class="result-section result-info">
        <h2>Job Information</h2>
        <dl class="result-grid">

        <div class="result-item">
            <dt>Type</dt>
            <dd>{{.Job.Type}}</dd>
        </div>

        <div class="result-item">
            <dt>Status</dt>
            <dd class="result-status status-{{.Job.Status}}">
                {{.Job.Status}}
            </dd>
        </div>

        <div class="result-item">
            <dt>Created</dt>
            <dd>{{.Job.CreatedAt}}</dd>
        </div>

        {{with .Job.StartedAt}}
        <div class="result-item">
            <dt>Started</dt>
            <dd>{{.}}</dd>
        </div>
        {{end}}

        {{with .Job.CompletedAt}}
        <div class="result-item">
            <dt>Completed</dt>
            <dd>{{.}}</dd>
        </div>
        {{end}}

        </dl>
    </section>

    {{with .Job.ErrorMessage}}
    <section class="result-section result-error">
        <h2>Error</h2>
        <p>{{.}}</p>
    </section>
    {{end}}

    {{if .Results}}
    <section class="result-section">
        <h2>Dataset Overview</h2>

        <dl>
            <dt>Rows</dt>
            <dd>{{.Results.NumRows}}</dd>

            <dt>Columns</dt>
            <dd>{{.Results.NumColumns}}</dd>
        </dl>

        <h3>Columns</h3>

        <ul class="result-list">
            {{range .Results.ColumnNames}}
                <li>{{.}}</li>
            {{end}}
        </ul>
    </section>

    <section class="result-section">
        <h2>Data Types</h2>

        <table class="result-table">
            <thead>
                <tr>
                    <th>Column</th>
                    <th>Type</th>
                </tr>
            </thead>
            <tbody>
                {{range $column, $type := .Results.DataTypes}}
                <tr>
                    <td>{{$column}}</td>
                    <td>{{$type}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </section>

    <section class="result-section">
        <h2>Missing Values</h2>

        <table class="result-table">
            <thead>
                <tr>
                    <th>Column</th>
                    <th>Missing Values</th>
                </tr>
            </thead>
            <tbody>
                {{range $column, $count := .Results.MissingValues}}
                <tr>
                    <td>{{$column}}</td>
                    <td>{{$count}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </section>

    <section class="result-section">
        <h2>Duplicates and Unique Values</h2>

        <p>Duplicate rows: {{.Results.DuplicateRows}}</p>

        <table class="result-table">
            <thead>
                <tr>
                    <th>Column</th>
                    <th>Unique Values</th>
                </tr>
            </thead>
            <tbody>
                {{range $column, $count := .Results.UniqueValues}}
                <tr>
                    <td>{{$column}}</td>
                    <td>{{$count}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </section>

    <section class="result-section">
        <h2>Descriptive Statistics</h2>

        {{range $column, $stats := .Results.DescriptiveStats}}
        <h3>{{$column}}</h3>

        <table class="result-table">
            <tr>
                <th>Count</th>
                <td>{{$stats.Count}}</td>
            </tr>
            <tr>
                <th>Mean</th>
                <td>{{$stats.Mean}}</td>
            </tr>
            <tr>
                <th>Standard Deviation</th>
                <td>{{$stats.Std}}</td>
            </tr>
            <tr>
                <th>Minimum</th>
                <td>{{$stats.Min}}</td>
            </tr>
            <tr>
                <th>25th Percentile</th>
                <td>{{$stats.Q25}}</td>
            </tr>
            <tr>
                <th>Median</th>
                <td>{{$stats.Median}}</td>
            </tr>
            <tr>
                <th>75th Percentile</th>
                <td>{{$stats.Q75}}</td>
            </tr>
            <tr>
                <th>Maximum</th>
                <td>{{$stats.Max}}</td>
            </tr>
        </table>
        {{end}}
    </section>

    <section class="result-section">
        <h2>Numeric Summary</h2>

        {{range $column, $summary := .Results.NumericSummary}}
        <h3>{{$column}}</h3>

        <dl class="result-grid result-summary">
            <div class="result-item">
                <dt>Mean</dt>
                <dd>{{$summary.Mean}}</dd>
            </div>

            <div class="result-item">
                <dt>Median</dt>
                <dd>{{$summary.Median}}</dd>
            </div>

            <div class="result-item">
                <dt>Standard Deviation</dt>
                <dd>{{$summary.StdDev}}</dd>
            </div>

            <div class="result-item">
                <dt>Minimum</dt>
                <dd>{{$summary.Min}}</dd>
            </div>

            <div class="result-item">
                <dt>Maximum</dt>
                <dd>{{$summary.Max}}</dd>
            </div>
        </dl>
        {{end}}
    </section>

    <section class="result-section">
        <h2>Categorical Summary</h2>

        {{range $column, $summary := .Results.CategoricalSummary}}
        <h3>{{$column}}</h3>

        <p>Mode: {{$summary.Mode}}</p>
        <p>Unique values: {{$summary.UniqueValues}}</p>

        <table class="result-table">
            <thead>
                <tr>
                    <th>Value</th>
                    <th>Count</th>
                </tr>
            </thead>
            <tbody>
                {{range $value, $count := $summary.ValueCounts}}
                <tr>
                    <td>{{$value}}</td>
                    <td>{{$count}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{end}}
    </section>

    <section class="result-section">
        <h2>Outliers</h2>

        {{range $column, $summary := .Results.OutlierSummary}}
        <h3>{{$column}}</h3>

        <dl class="result-grid result-summary">
            <div class="result-item">
                <dt>Number of Outliers</dt>
                <dd>{{$summary.NumOutliers}}</dd>
            </div>
        </dl>
        {{end}}
    </section>

    <section class="result-section">
        <h2>Correlation Matrix</h2>

        {{range $column, $correlations := .Results.CorrelationMatrix}}
        <h3>{{$column}}</h3>

        <ul class="result-list">
            {{range $otherColumn, $value := $correlations}}
                <li>{{$otherColumn}}: {{$value}}</li>
            {{end}}
        </ul>
        {{end}}
    </section>

    <div class="result-actions">
        <a href="/results/download?id={{.Job.ID}}">
            Download results JSON
        </a>
    </div>

    {{if .ModelResults}}
    <section class="result-section">
        <h2>Model Results</h2>

        <p>Model Type: {{.ModelResults.Model}}</p>
        <p>R²: {{.ModelResults.Evaluation.R2}}</p>
        <p>Mean Squared Error: {{.ModelResults.Evaluation.MSE}}</p>
        <p>Features:</p>
        <ul class="result-list">
            {{range .ModelResults.FeaturesUsed}}
                <li>{{.}}</li>
            {{end}}
        </ul>
        <p>Feature Importances:</p>
        <ul class="result-list">
            {{range $feature, $importance := .ModelResults.FeatureImportances}}
                <li>{{$feature}}: {{$importance}}</li>
            {{end}}
        </ul>
        <p>Actual vs Predicted Values:</p>
        <ul class="result-list">
        {{range .ModelResults.ActualVsPredicted}}
            <li>Actual: {{.Actual}}, Predicted: {{.Predicted}}</li>
        {{end}}
        </ul>
        <p>Target Variable: {{.ModelResults.Target}}</p>
        <p>Configuration:</p>
        <dl>
            <dt>Number of Estimators</dt>
            <dd>{{index .ModelResults.Configuration "n_estimators"}}</dd>

            <dt>Max Depth</dt>
            <dd>{{index .ModelResults.Configuration "max_depth"}}</dd>
        </dl>
        <p>
            <a href="/results/download/model?id={{.Job.ID}}">
                Download model results JSON
            </a>
        </p>
    </section>
    {{end}}

    {{else}}
        {{if eq .Job.Status "processing"}}
        <div class="result-status-message status-processing">
            <p>The dataset is still being processed.</p>
        </div>
        {{else if eq .Job.Status "queued"}}
        <div class="result-status-message status-queued">
            <p>The job is waiting to be processed.</p>
        </div>
        {{else}}
        <div class="result-status-message">
            <p>No dataset results are currently available.</p>
        </div>
        {{end}}
    {{end}}

    {{if .VisualizationResults}}
    <section class="result-section">
        <h2>Visualizations</h2>

        <div class="result-visualizations">
            {{if .VisualizationResults.FeatureDistributions}}
            <div class="result-visualization">
                <h3>Feature Distributions</h3>
                <img src="/results/visualization?id={{.Job.ID}}&type=feature_distributions" alt="Feature Distributions">
            </div>
            {{end}}

            {{if .VisualizationResults.CorrelationHeatmap}}
            <div class="result-visualization">
                <h3>Correlation Heatmap</h3>
                <img src="/results/visualization?id={{.Job.ID}}&type=correlation_heatmap" alt="Correlation Heatmap">
            </div>
            {{end}}

            {{if .VisualizationResults.ActualVsPredicted}}
            <div class="result-visualization">
                <h3>Actual vs Predicted</h3>
                <img src="/results/visualization?id={{.Job.ID}}&type=actual_vs_predicted" alt="Actual vs Predicted">
            </div>
            {{end}}
        </div>
    </section>
    {{end}}

    <div class="result-actions">
        <a href="/">Back to jobs</a>
    </div>
</div>
</body>
</html>`))

var ImageResultsTemplate = template.Must(
	template.New("image_results").Parse(`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Results for Job #{{.Job.ID}}</title>
    <link rel="stylesheet" href="/static/styles.css">
</head>

<body>
<div class="results-page">
    <h1 class="result-title">Image Results for Job #{{.Job.ID}}</h1>

    <section class="result-section result-info">
        <h2>Job Information</h2>
        <dl class="result-grid">

        <div class="result-item">
            <dt>Type</dt>
            <dd>{{.Job.Type}}</dd>
        </div>

        <div class="result-item">
            <dt>Status</dt>
            <dd class="result-status status-{{.Job.Status}}">
                {{.Job.Status}}
            </dd>
        </div>

        <div class="result-item">
            <dt>Created</dt>
            <dd>{{.Job.CreatedAt}}</dd>
        </div>

        {{with .Job.StartedAt}}
        <div class="result-item">
            <dt>Started</dt>
            <dd>{{.}}</dd>
        </div>
        {{end}}

        {{with .Job.CompletedAt}}
        <div class="result-item">
            <dt>Completed</dt>
            <dd>{{.}}</dd>
        </div>
        {{end}}

        </dl>
    </section>

    {{with .Job.ErrorMessage}}
    <section class="result-section result-error">
        <h2>Error</h2>
        <p>{{.}}</p>
    </section>
    {{end}}

    {{if .ImageResults}}

    <section class="result-section">
        <h2>Images</h2>

        <div class="result-images">
        <div class="result-image">
            <h3>Original Image</h3>
            <img
                src="/results/image?id={{.Job.ID}}&type=original"
                alt="Original image"
            >
        </div>

        <div class="result-image">
            <h3>Processed Image</h3>
            <img
                src="/results/image?id={{.Job.ID}}&type=processed"
                alt="Processed image"
            >
        </div>
        </div>
    </section>

    <section class="result-section">
        <h2>Processing Operations</h2>

        {{if .ImageResults.Operations}}
        <ul class="result-list">
            {{range .ImageResults.Operations}}
            <li>{{.}}</li>
            {{end}}
        </ul>
        {{else}}
        <p>No image processing operations were performed.</p>
        {{end}}
    </section>

    <section class="result-section">
        <h2>Original Image</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Format</dt>
            <dd>{{.ImageResults.OriginalFormat}}</dd>
            </div>

            <div class="result-item">
            <dt>Dimensions</dt>
            <dd>
                {{.ImageResults.OriginalWidth}}
                ×
                {{.ImageResults.OriginalHeight}}
            </dd>
            </div>
        </dl>
    </section>

    <section class="result-section">
        <h2>Processed Image</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Format</dt>
            <dd>{{.ImageResults.ResultFormat}}</dd>
            </div>

            <div class="result-item">
            <dt>Dimensions</dt>
            <dd>
                {{.ImageResults.ResultWidth}}
                ×
                {{.ImageResults.ResultHeight}}
            </dd>
            </div>
        </dl>
    </section>

    {{with .ImageResults.Compression}}
    <section class="result-section">
        <h2>Compression</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Original Size</dt>
            <dd>{{.OriginalSize}} bytes</dd>
            </div>

            <div class="result-item">
            <dt>Result Size</dt>
            <dd>{{.ResultSize}} bytes</dd>
            </div>

            <div class="result-item">
            <dt>Compression Ratio</dt>
            <dd>{{.CompressionRatio}}</dd>
            </div>
        </dl>
    </section>
    {{end}}

    {{if .ImageResults.Metadata}}
    <section class="result-section">
        <h2>Metadata</h2>

        <dl class="result-grid">
            {{range $key, $value := .ImageResults.Metadata}}
            <div class="result-item">
            <dt>{{$key}}</dt>
            <dd>{{$value}}</dd>
            </div>
            {{end}}
        </dl>

        {{if .ImageResults.MetadataReference}}
        <p>
            <a href="/results/download/metadata?id={{.Job.ID}}">
                Download Full Metadata
            </a>
        </p>
        {{end}}
    </section>
    {{else}}
        {{if .ImageResults.MetadataReference}}
        <section class="result-section">
            <h2>Metadata</h2>
            <p>
                Full metadata was extracted but is not displayed in the
                results summary.
            </p>

            <a href="/results/download/metadata?id={{.Job.ID}}">
                Download Full Metadata
            </a>
        </section>
        {{end}}
    {{end}}

    {{else}}
        {{if eq .Job.Status "processing"}}
        <div class="result-status-message status-processing">
            <p>The image is still being processed.</p>
        </div>

        {{else if eq .Job.Status "queued"}}
        <div class="result-status-message status-queued">
            <p>The job is waiting to be processed.</p>
        </div>

        {{else if eq .Job.Status "failed"}}
        <div class="result-status-message status-failed">
            <p>Image processing failed.</p>
        </div>

        {{else}}
        <div class="result-status-message">
        <p>No image results are currently available.</p>
        </div>
        {{end}}
    {{end}}

    <div class="result-actions">
        <a href="/">Back to jobs</a>
    </div>
</div>
</body>
</html>`),
)

var RouteResultsTemplate = template.Must(
	template.New("route_results").Parse(`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Route Results for Job #{{.Job.ID}}</title>
    <link rel="stylesheet" href="/static/styles.css">
</head>

<body>
<div class="results-page">

    <h1 class="result-title">Route Results for Job #{{.Job.ID}}</h1>

    <section class="result-section result-info">
        <h2>Job Information</h2>
        <dl class="result-grid">

        <div class="result-item">
            <dt>Type</dt>
            <dd>{{.Job.Type}}</dd>
        </div>

        <div class="result-item">
            <dt>Status</dt>
            <dd class="result-status status-{{.Job.Status}}">
                {{.Job.Status}}
            </dd>
        </div>

        <div class="result-item">
            <dt>Created</dt>
            <dd>{{.Job.CreatedAt}}</dd>
        </div>

        {{with .Job.StartedAt}}
        <div class="result-item">
            <dt>Started</dt>
            <dd>{{.}}</dd>
        </div>
        {{end}}

        {{with .Job.CompletedAt}}
        <div class="result-item">
            <dt>Completed</dt>
            <dd>{{.}}</dd>
        </div>
        {{end}}

        </dl>
    </section>

    {{with .Job.ErrorMessage}}
    <section class="result-section result-error">
        <h2>Error</h2>
        <p>{{.}}</p>
    </section>
    {{end}}

    {{if .RouteResults}}

    <section class="result-section">
        <h2>Route Configuration</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Starting Location</dt>
            <dd>{{.RouteResults.StartLocation}}</dd>
            </div>

            <div class="result-item">
            <dt>Ending Location</dt>
            <dd>{{.RouteResults.EndLocation}}</dd>
            </div>

            <div class="result-item">
            <dt>Algorithm</dt>
            <dd>{{.RouteResults.Algorithm}}</dd>
            </div>

            <div class="result-item">
            <dt>2-opt Applied</dt>
            <dd>{{.RouteResults.TwoOptApplied}}</dd>
            </div>
        </dl>
    </section>

    <section class="result-section">
        <h2>Initial Route</h2>

        <ol class="result-list">
            {{range .RouteResults.InitialRoute}}
            <li>{{.}}</li>
            {{end}}
        </ol>

        <p>
            <strong>Initial Distance:</strong>
            {{.RouteResults.InitialDistance}}
        </p>
    </section>

    <section class="result-section">
        <h2>Optimized Route</h2>

        <ol class="result-list">
            {{range .RouteResults.OptimizedRoute}}
            <li>{{.}}</li>
            {{end}}
        </ol>

        <p>
            <strong>Optimized Distance:</strong>
            {{.RouteResults.OptimizedDistance}}
        </p>
    </section>

    <section class="result-section">
        <h2>Optimization Results</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Distance Improvement</dt>
            <dd>{{.RouteResults.DistanceImprovement}}</dd>
            </div>

            <div class="result-item">
            <dt>Improvement Percentage</dt>
            <dd>{{.RouteResults.ImprovementPercentage}}%</dd>
            </div>
        </dl>
    </section>

    <section class="result-section">
        <h2>Constraints</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Feasible</dt>
            <dd>{{.RouteResults.Feasible}}</dd>
            </div>
        </dl>
    </section>

    <section class="result-section">
        <h2>Performance</h2>

        <dl class="result-grid">
            <div class="result-item">
            <dt>Algorithm</dt>
            <dd>{{.RouteResults.Algorithm}}</dd>
            </div>

            <div class="result-item">
            <dt>2-opt Applied</dt>
            <dd>{{.RouteResults.TwoOptApplied}}</dd>
            </div>

            <div class="result-item">
            <dt>Runtime</dt>
            <dd>{{.RouteResults.RuntimeSeconds}} seconds</dd>
            </div>
        </dl>
    </section>

    {{else}}
        {{if eq .Job.Status "processing"}}
        <div class="result-status-message status-processing">
            <p>The route is still being processed.</p>
        </div>

        {{else if eq .Job.Status "queued"}}
        <div class="result-status-message status-queued">
            <p>The job is waiting to be processed.</p>
        </div>

        {{else if eq .Job.Status "failed"}}
        <div class="result-status-message status-failed">
            <p>Route processing failed.</p>
        </div>

        {{else}}
        <div class="result-status-message">
            <p>No route results are currently available.</p>
        </div>
        {{end}}

    {{end}}

    <div class="result-actions">
        <a href="/">Back to jobs</a>
    </div>
</div>
</body>
</html>`),
)

var RegisterTemplate = template.Must(template.New("register").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register</title>
    <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
    <h1>Create an Account</h1>

    <form action="/register" method="post">
        <div>
            <label for="username">Username:</label>
            <input
                type="text"
                id="username"
                name="username"
                required
            >
        </div>

        <div>
            <label for="password">Password:</label>
            <input
                type="password"
                id="password"
                name="password"
                required
            >
        </div>

        <button type="submit">Register</button>
    </form>

    <p>
        <a href="/">Back to Data Processing Platform</a>
    </p>
</body>
</html>`))

var LoginTemplate = template.Must(template.New("login").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login</title>
    <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
    <h1>Log In</h1>

    <form action="/login" method="post">
        <div>
            <label for="username">Username:</label>
            <input
                type="text"
                id="username"
                name="username"
                required
            >
        </div>

        <div>
            <label for="password">Password:</label>
            <input
                type="password"
                id="password"
                name="password"
                required
            >
        </div>

        <button type="submit">Log In</button>
    </form>

    <p>
        <a href="/register">Create an account</a>
    </p>

    <p>
        <a href="/">Back to Data Processing Platform</a>
    </p>
</body>
</html>
`))
