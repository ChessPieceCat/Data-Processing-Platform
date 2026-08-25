package server

import "github.com/ChessPieceCat/Data-Processing-Platform/internal/jobs"

type DatasetResults struct {
	NumRows              int                           `json:"num_rows"`
	NumColumns           int                           `json:"num_columns"`
	ColumnNames          []string                      `json:"column_names"`
	DataTypes            map[string]string             `json:"data_types"`
	MissingValues        map[string]int                `json:"missing_values"`
	DuplicateRows        int                           `json:"duplicate_rows"`
	InvalidEntries       map[string]int                `json:"invalid_entries"`
	UniqueValues         map[string]int                `json:"unique_values"`
	DescriptiveStats     map[string]DescriptiveStats   `json:"descriptive_stats"`
	NumericSummary       map[string]NumericSummary     `json:"numeric_summary"`
	CategoricalSummary   map[string]CategoricalSummary `json:"categorical_summary"`
	OutlierSummary       map[string]OutlierSummary     `json:"outlier_summary"`
	FeatureDistributions map[string]map[string]float64 `json:"feature_distributions"`
	CorrelationMatrix    map[string]map[string]float64 `json:"correlation_matrix"`
	Visualizations       *VisualizationResults         `json:"visualizations"`
}

type DescriptiveStats struct {
	Count  float64 `json:"count"`
	Mean   float64 `json:"mean"`
	Std    float64 `json:"std"`
	Min    float64 `json:"min"`
	Q25    float64 `json:"25%"`
	Median float64 `json:"50%"`
	Q75    float64 `json:"75%"`
	Max    float64 `json:"max"`
}

type NumericSummary struct {
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	StdDev float64 `json:"std_dev"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

type CategoricalSummary struct {
	Mode         string         `json:"mode"`
	UniqueValues int            `json:"unique_values"`
	ValueCounts  map[string]int `json:"value_counts"`
}

type OutlierSummary struct {
	NumOutliers    int   `json:"num_outliers"`
	OutlierIndices []int `json:"outlier_indices"`
}

type ResultsPage struct {
	Job                  *jobs.Job
	Results              *DatasetResults
	ModelResults         *ModelResults
	VisualizationResults *VisualizationResults
}

type ModelResults struct {
	Model              string                 `json:"model"`
	Evaluation         ModelEvaluation        `json:"evaluation"`
	FeatureImportances map[string]float64     `json:"feature_importances"`
	ActualVsPredicted  []ActualVsPredicted    `json:"actual_vs_predicted"`
	FeaturesUsed       []string               `json:"features_used"`
	Target             string                 `json:"target"`
	Configuration      map[string]interface{} `json:"configuration"`
}

type ModelEvaluation struct {
	R2  float64 `json:"r2"`
	MSE float64 `json:"mse"`
}

type ActualVsPredicted struct {
	Actual    float64 `json:"actual"`
	Predicted float64 `json:"predicted"`
}

type VisualizationResults struct {
	FeatureDistributions string `json:"feature_distributions"`
	CorrelationHeatmap   string `json:"correlation_heatmap"`
	ActualVsPredicted    string `json:"actual_vs_predicted"`
}
