package server

import (
	"encoding/json"
	"testing"
)

func TestDatasetResultsJSON(t *testing.T) {
	input := []byte(`{
		"num_rows": 100,
		"num_columns": 3,
		"column_names": [
			"temperature",
			"humidity",
			"co2"
		],
		"data_types": {
			"temperature": "float64",
			"humidity": "float64",
			"co2": "float64"
		},
		"missing_values": {
			"temperature": 2,
			"humidity": 1,
			"co2": 0
		},
		"duplicate_rows": 4,
		"invalid_entries": {
			"temperature": 0,
			"humidity": 0,
			"co2": 0
		},
		"unique_values": {
			"temperature": 95,
			"humidity": 80,
			"co2": 100
		},
		"descriptive_stats": {
			"temperature": {
				"count": 100,
				"mean": 22.5,
				"std": 3.2,
				"min": 10,
				"25%": 20,
				"50%": 22,
				"75%": 25,
				"max": 35
			}
		},
		"numeric_summary": {
			"temperature": {
				"mean": 22.5,
				"median": 22,
				"std_dev": 3.2,
				"min": 10,
				"max": 35
			}
		},
		"categorical_summary": {},
		"outlier_summary": {
			"temperature": {
				"num_outliers": 3,
				"outlier_indices": [5, 18, 92]
			}
		},
		"feature_distributions": {
			"temperature": {
				"20": 0.10,
				"22": 0.15
			}
		},
		"correlation_matrix": {
			"temperature": {
				"temperature": 1.0,
				"humidity": -0.25
			}
		},
		"visualizations": {
			"feature_distributions": "uploads/1/results_feature_distributions.png",
			"correlation_heatmap": "uploads/1/results_correlation_heatmap.png",
			"actual_vs_predicted": "uploads/1/results_actual_vs_predicted.png"
		}
	}`)

	var results DatasetResults

	if err := json.Unmarshal(input, &results); err != nil {
		t.Fatalf("failed to unmarshal DatasetResults: %v", err)
	}

	if results.NumRows != 100 {
		t.Fatalf("expected 100 rows, got %d", results.NumRows)
	}

	if results.NumColumns != 3 {
		t.Fatalf("expected 3 columns, got %d", results.NumColumns)
	}

	if len(results.ColumnNames) != 3 {
		t.Fatalf("expected 3 column names, got %d", len(results.ColumnNames))
	}

	if results.MissingValues["temperature"] != 2 {
		t.Fatalf(
			"expected 2 missing temperature values, got %d",
			results.MissingValues["temperature"],
		)
	}

	if results.DuplicateRows != 4 {
		t.Fatalf(
			"expected 4 duplicate rows, got %d",
			results.DuplicateRows,
		)
	}

	stats := results.DescriptiveStats["temperature"]

	if stats.Count != 100 {
		t.Fatalf("expected count 100, got %f", stats.Count)
	}

	if stats.Mean != 22.5 {
		t.Fatalf("expected mean 22.5, got %f", stats.Mean)
	}

	numeric := results.NumericSummary["temperature"]

	if numeric.Median != 22 {
		t.Fatalf("expected median 22, got %f", numeric.Median)
	}

	outliers := results.OutlierSummary["temperature"]

	if outliers.NumOutliers != 3 {
		t.Fatalf(
			"expected 3 outliers, got %d",
			outliers.NumOutliers,
		)
	}

	if results.Visualizations == nil {
		t.Fatal("expected visualization results")
	}

	if results.Visualizations.FeatureDistributions == "" {
		t.Error("expected feature distributions path")
	}

	if results.Visualizations.CorrelationHeatmap == "" {
		t.Error("expected correlation heatmap path")
	}

	if results.Visualizations.ActualVsPredicted == "" {
		t.Error("expected actual vs predicted path")
	}
}

func TestModelResultsJSON(t *testing.T) {
	input := []byte(`{
		"model": "random_forest_regressor",
		"evaluation": {
			"r2": 0.82,
			"mse": 12.5
		},
		"feature_importances": {
			"temperature": 0.45,
			"humidity": 0.30,
			"pressure": 0.25
		},
		"actual_vs_predicted": [
			{
				"actual": 10.0,
				"predicted": 9.5
			},
			{
				"actual": 20.0,
				"predicted": 21.2
			}
		],
		"features_used": [
			"temperature",
			"humidity",
			"pressure"
		],
		"target": "co2",
		"configuration": {
			"n_estimators": 100,
			"max_depth": 10
		}
	}`)

	var results ModelResults

	if err := json.Unmarshal(input, &results); err != nil {
		t.Fatalf("failed to unmarshal ModelResults: %v", err)
	}

	if results.Model != "random_forest_regressor" {
		t.Fatalf("unexpected model: %q", results.Model)
	}

	if results.Evaluation.R2 != 0.82 {
		t.Fatalf("expected R2 0.82, got %f", results.Evaluation.R2)
	}

	if results.Evaluation.MSE != 12.5 {
		t.Fatalf("expected MSE 12.5, got %f", results.Evaluation.MSE)
	}

	if len(results.FeaturesUsed) != 3 {
		t.Fatalf(
			"expected 3 features, got %d",
			len(results.FeaturesUsed),
		)
	}

	if results.Target != "co2" {
		t.Fatalf("expected target co2, got %q", results.Target)
	}

	if results.FeatureImportances["temperature"] != 0.45 {
		t.Fatalf(
			"unexpected temperature importance: %f",
			results.FeatureImportances["temperature"],
		)
	}

	if len(results.ActualVsPredicted) != 2 {
		t.Fatalf(
			"expected 2 actual-vs-predicted values, got %d",
			len(results.ActualVsPredicted),
		)
	}

	if results.ActualVsPredicted[0].Actual != 10 {
		t.Fatalf(
			"expected first actual value 10, got %f",
			results.ActualVsPredicted[0].Actual,
		)
	}

	if results.ActualVsPredicted[0].Predicted != 9.5 {
		t.Fatalf(
			"expected first predicted value 9.5, got %f",
			results.ActualVsPredicted[0].Predicted,
		)
	}

	if results.Configuration["n_estimators"] != float64(100) {
		t.Fatalf(
			"unexpected n_estimators value: %v",
			results.Configuration["n_estimators"],
		)
	}
}

func TestVisualizationResultsJSON(t *testing.T) {
	input := []byte(`{
		"feature_distributions": "uploads/10/results_feature_distributions.png",
		"correlation_heatmap": "uploads/10/results_correlation_heatmap.png",
		"actual_vs_predicted": "uploads/10/results_actual_vs_predicted.png"
	}`)

	var results VisualizationResults

	if err := json.Unmarshal(input, &results); err != nil {
		t.Fatalf("failed to unmarshal VisualizationResults: %v", err)
	}

	if results.FeatureDistributions == "" {
		t.Error("expected feature distributions path")
	}

	if results.CorrelationHeatmap == "" {
		t.Error("expected correlation heatmap path")
	}

	if results.ActualVsPredicted == "" {
		t.Error("expected actual vs predicted path")
	}
}

func TestDatasetResultsJSONWithoutVisualizations(t *testing.T) {
	input := []byte(`{
		"num_rows": 10,
		"num_columns": 2,
		"column_names": ["x", "y"],
		"data_types": {
			"x": "float64",
			"y": "float64"
		}
	}`)

	var results DatasetResults

	if err := json.Unmarshal(input, &results); err != nil {
		t.Fatalf("failed to unmarshal DatasetResults: %v", err)
	}

	if results.Visualizations != nil {
		t.Fatal("expected nil visualizations when field is absent")
	}
}

func TestActualVsPredictedJSON(t *testing.T) {
	input := []byte(`{
		"actual": 25.5,
		"predicted": 24.75
	}`)

	var result ActualVsPredicted

	if err := json.Unmarshal(input, &result); err != nil {
		t.Fatalf("failed to unmarshal ActualVsPredicted: %v", err)
	}

	if result.Actual != 25.5 {
		t.Fatalf("expected actual 25.5, got %f", result.Actual)
	}

	if result.Predicted != 24.75 {
		t.Fatalf(
			"expected predicted 24.75, got %f",
			result.Predicted,
		)
	}
}
