package server

import "text/template"

var DatasetResultsTemplate = template.Must(template.New("dataset_results").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Job #{{.Job.ID}} Results</title>
</head>
<body>
    <h1>Job #{{.Job.ID}} Results</h1>

    <section>
        <h2>Job Information</h2>
        <dl>
            <dt>Type</dt>
            <dd>{{.Job.Type}}</dd>

            <dt>Status</dt>
            <dd>{{.Job.Status}}</dd>

            <dt>Created</dt>
            <dd>{{.Job.CreatedAt}}</dd>

            {{with .Job.StartedAt}}
            <dt>Started</dt>
            <dd>{{.}}</dd>
            {{end}}

            {{with .Job.CompletedAt}}
            <dt>Completed</dt>
            <dd>{{.}}</dd>
            {{end}}
        </dl>
    </section>

    {{with .Job.ErrorMessage}}
    <section>
        <h2>Error</h2>
        <p>{{.}}</p>
    </section>
    {{end}}

    {{if .Results}}
    <section>
        <h2>Dataset Overview</h2>

        <dl>
            <dt>Rows</dt>
            <dd>{{.Results.NumRows}}</dd>

            <dt>Columns</dt>
            <dd>{{.Results.NumColumns}}</dd>
        </dl>

        <h3>Columns</h3>

        <ul>
            {{range .Results.ColumnNames}}
                <li>{{.}}</li>
            {{end}}
        </ul>
    </section>

    <section>
        <h2>Data Types</h2>

        <table>
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

    <section>
        <h2>Missing Values</h2>

        <table>
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

    <section>
        <h2>Duplicates and Unique Values</h2>

        <p>Duplicate rows: {{.Results.DuplicateRows}}</p>

        <table>
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

    <section>
        <h2>Descriptive Statistics</h2>

        {{range $column, $stats := .Results.DescriptiveStats}}
        <h3>{{$column}}</h3>

        <table>
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

    <section>
        <h2>Numeric Summary</h2>

        {{range $column, $summary := .Results.NumericSummary}}
        <h3>{{$column}}</h3>

        <dl>
            <dt>Mean</dt>
            <dd>{{$summary.Mean}}</dd>

            <dt>Median</dt>
            <dd>{{$summary.Median}}</dd>

            <dt>Standard Deviation</dt>
            <dd>{{$summary.StdDev}}</dd>

            <dt>Minimum</dt>
            <dd>{{$summary.Min}}</dd>

            <dt>Maximum</dt>
            <dd>{{$summary.Max}}</dd>
        </dl>
        {{end}}
    </section>

    <section>
        <h2>Categorical Summary</h2>

        {{range $column, $summary := .Results.CategoricalSummary}}
        <h3>{{$column}}</h3>

        <p>Mode: {{$summary.Mode}}</p>
        <p>Unique values: {{$summary.UniqueValues}}</p>

        <table>
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

    <section>
        <h2>Outliers</h2>

        {{range $column, $summary := .Results.OutlierSummary}}
        <h3>{{$column}}</h3>

        <p>Number of outliers: {{$summary.NumOutliers}}</p>
        {{end}}
    </section>

    <section>
        <h2>Correlation Matrix</h2>

        {{range $column, $correlations := .Results.CorrelationMatrix}}
        <h3>{{$column}}</h3>

        <ul>
            {{range $otherColumn, $value := $correlations}}
                <li>{{$otherColumn}}: {{$value}}</li>
            {{end}}
        </ul>
        {{end}}
    </section>

    <p>
        <a href="/results/download?id={{.Job.ID}}">
            Download results JSON
        </a>
    </p>

    {{if .ModelResults}}
    <section>
        <h2>Model Results</h2>

        <p>Model Type: {{.ModelResults.Model}}</p>
        <p>R²: {{.ModelResults.Evaluation.R2}}</p>
        <p>Mean Squared Error: {{.ModelResults.Evaluation.MSE}}</p>
        <p>Features:</p>
        <ul>
            {{range .ModelResults.FeaturesUsed}}
                <li>{{.}}</li>
            {{end}}
        </ul>
        <p>Feature Importances:</p>
        <ul>
            {{range $feature, $importance := .ModelResults.FeatureImportances}}
                <li>{{$feature}}: {{$importance}}</li>
            {{end}}
        </ul>
        <p>Actual vs Predicted Values:</p>
        <ul>
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
            <p>The dataset is still being processed.</p>
        {{else if eq .Job.Status "queued"}}
            <p>The job is waiting to be processed.</p>
        {{else}}
            <p>No dataset results are currently available.</p>
        {{end}}
    {{end}}

    {{if .VisualizationResults}}
    <section>
        <h2>Visualizations</h2>

        {{if .VisualizationResults.FeatureDistributions}}
        <h3>Feature Distributions</h3>
        <img src="/results/visualization?id={{.Job.ID}}&type=feature_distributions" alt="Feature Distributions">
        {{end}}

        {{if .VisualizationResults.CorrelationHeatmap}}
        <h3>Correlation Heatmap</h3>
        <img src="/results/visualization?id={{.Job.ID}}&type=correlation_heatmap" alt="Correlation Heatmap">
        {{end}}

        {{if .VisualizationResults.ActualVsPredicted}}
        <h3>Actual vs Predicted</h3>
        <img src="/results/visualization?id={{.Job.ID}}&type=actual_vs_predicted" alt="Actual vs Predicted">
        {{end}}
    </section>
    {{end}}

    <p><a href="/">Back to jobs</a></p>
</body>
</html>`))

var ImageResultsTemplate = template.Must(
	template.New("image_results").Parse(`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Results for Job #{{.Job.ID}}</title>
</head>

<body>
    <h1>Image Results for Job #{{.Job.ID}}</h1>

    <section>
        <h2>Job Information</h2>

        <dl>
            <dt>Type</dt>
            <dd>{{.Job.Type}}</dd>

            <dt>Status</dt>
            <dd>{{.Job.Status}}</dd>

            <dt>Created</dt>
            <dd>{{.Job.CreatedAt}}</dd>

            {{with .Job.StartedAt}}
            <dt>Started</dt>
            <dd>{{.}}</dd>
            {{end}}

            {{with .Job.CompletedAt}}
            <dt>Completed</dt>
            <dd>{{.}}</dd>
            {{end}}
        </dl>
    </section>

    {{with .Job.ErrorMessage}}
    <section>
        <h2>Error</h2>
        <p>{{.}}</p>
    </section>
    {{end}}

    {{if .ImageResults}}

    <section>
        <h2>Images</h2>

        <div>
            <h3>Original Image</h3>
            <img
                src="/results/image?id={{.Job.ID}}&type=original"
                alt="Original image"
                style="max-width: 100%;"
            >
        </div>

        <div>
            <h3>Processed Image</h3>
            <img
                src="/results/image?id={{.Job.ID}}&type=processed"
                alt="Processed image"
                style="max-width: 100%;"
            >
        </div>
    </section>

    <section>
        <h2>Processing Operations</h2>

        {{if .ImageResults.Operations}}
        <ul>
            {{range .ImageResults.Operations}}
            <li>{{.}}</li>
            {{end}}
        </ul>
        {{else}}
        <p>No image processing operations were performed.</p>
        {{end}}
    </section>

    <section>
        <h2>Original Image</h2>

        <dl>
            <dt>Format</dt>
            <dd>{{.ImageResults.OriginalFormat}}</dd>

            <dt>Dimensions</dt>
            <dd>
                {{.ImageResults.OriginalWidth}}
                ×
                {{.ImageResults.OriginalHeight}}
            </dd>
        </dl>
    </section>

    <section>
        <h2>Processed Image</h2>

        <dl>
            <dt>Format</dt>
            <dd>{{.ImageResults.ResultFormat}}</dd>

            <dt>Dimensions</dt>
            <dd>
                {{.ImageResults.ResultWidth}}
                ×
                {{.ImageResults.ResultHeight}}
            </dd>
        </dl>
    </section>

    {{with .ImageResults.Compression}}
    <section>
        <h2>Compression</h2>

        <dl>
            <dt>Original Size</dt>
            <dd>{{.OriginalSize}} bytes</dd>

            <dt>Result Size</dt>
            <dd>{{.ResultSize}} bytes</dd>

            <dt>Compression Ratio</dt>
            <dd>{{.CompressionRatio}}</dd>
        </dl>
    </section>
    {{end}}

    {{if .ImageResults.Metadata}}
    <section>
        <h2>Metadata</h2>

        <dl>
            {{range $key, $value := .ImageResults.Metadata}}
            <dt>{{$key}}</dt>
            <dd>{{$value}}</dd>
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
        <section>
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
        <p>The image is still being processed.</p>

        {{else if eq .Job.Status "queued"}}
        <p>The job is waiting to be processed.</p>

        {{else if eq .Job.Status "failed"}}
        <p>Image processing failed.</p>

        {{else}}
        <p>No image results are currently available.</p>
        {{end}}
    {{end}}

    <p>
        <a href="/">Back to jobs</a>
    </p>

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
</head>

<body>

    <h1>Route Results for Job #{{.Job.ID}}</h1>

    <section>
        <h2>Job Information</h2>

        <dl>
            <dt>Type</dt>
            <dd>{{.Job.Type}}</dd>

            <dt>Status</dt>
            <dd>{{.Job.Status}}</dd>

            <dt>Created</dt>
            <dd>{{.Job.CreatedAt}}</dd>

            {{with .Job.StartedAt}}
            <dt>Started</dt>
            <dd>{{.}}</dd>
            {{end}}

            {{with .Job.CompletedAt}}
            <dt>Completed</dt>
            <dd>{{.}}</dd>
            {{end}}
        </dl>
    </section>

    {{with .Job.ErrorMessage}}
    <section>
        <h2>Error</h2>
        <p>{{.}}</p>
    </section>
    {{end}}

    {{if .RouteResults}}

    <section>
        <h2>Route Configuration</h2>

        <dl>
            <dt>Starting Location</dt>
            <dd>{{.RouteResults.StartLocation}}</dd>

            <dt>Ending Location</dt>
            <dd>{{.RouteResults.EndLocation}}</dd>

            <dt>Algorithm</dt>
            <dd>{{.RouteResults.Algorithm}}</dd>

            <dt>2-opt Applied</dt>
            <dd>{{.RouteResults.TwoOptApplied}}</dd>
        </dl>
    </section>

    <section>
        <h2>Initial Route</h2>

        <ol>
            {{range .RouteResults.InitialRoute}}
            <li>{{.}}</li>
            {{end}}
        </ol>

        <p>
            <strong>Initial Distance:</strong>
            {{.RouteResults.InitialDistance}}
        </p>
    </section>

    <section>
        <h2>Optimized Route</h2>

        <ol>
            {{range .RouteResults.OptimizedRoute}}
            <li>{{.}}</li>
            {{end}}
        </ol>

        <p>
            <strong>Optimized Distance:</strong>
            {{.RouteResults.OptimizedDistance}}
        </p>
    </section>

    <section>
        <h2>Optimization Results</h2>

        <dl>
            <dt>Distance Improvement</dt>
            <dd>{{.RouteResults.DistanceImprovement}}</dd>

            <dt>Improvement Percentage</dt>
            <dd>{{.RouteResults.ImprovementPercentage}}%</dd>
        </dl>
    </section>

    <section>
        <h2>Constraints</h2>

        <dl>
            <dt>Feasible</dt>
            <dd>{{.RouteResults.Feasible}}</dd>
        </dl>
    </section>

    <section>
        <h2>Performance</h2>

        <dl>
            <dt>Algorithm</dt>
            <dd>{{.RouteResults.Algorithm}}</dd>

            <dt>2-opt Applied</dt>
            <dd>{{.RouteResults.TwoOptApplied}}</dd>

            <dt>Runtime</dt>
            <dd>{{.RouteResults.RuntimeSeconds}} seconds</dd>
        </dl>
    </section>

    {{else}}

        {{if eq .Job.Status "processing"}}
        <p>The route is still being processed.</p>

        {{else if eq .Job.Status "queued"}}
        <p>The job is waiting to be processed.</p>

        {{else if eq .Job.Status "failed"}}
        <p>Route processing failed.</p>

        {{else}}
        <p>No route results are currently available.</p>
        {{end}}

    {{end}}

    <p>
        <a href="/">Back to jobs</a>
    </p>

</body>
</html>`),
)

var RegisterTemplate = template.Must(template.New("register").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register</title>
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
