package jobs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSaveDatasetConfig(t *testing.T) {
	jobID := int64(12345)

	config := DatasetConfig{
		Model:             "random_forest_regressor",
		Target:            "co2",
		FeatureSelection:  "manual",
		Features:          []string{"temperature", "humidity", "pressure"},
		ConfigurationType: "manual",
		Configuration: RandomForestConfig{
			NEstimators: 25,
			MaxDepth:    10,
		},
		Visualizations: VisualizationConfig{
			Histograms:         true,
			BoxPlots:           true,
			CorrelationHeatmap: true,
			ActualVsPredicted:  true,
		},
	}

	jobDirectory := filepath.Join("uploads", "12345")

	if err := os.MkdirAll(jobDirectory, 0755); err != nil {
		t.Fatalf("failed to create test job directory: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(jobDirectory)
	})

	path, err := SaveDatasetConfig(config, jobID)
	if err != nil {
		t.Fatalf("SaveDatasetConfig failed: %v", err)
	}

	expectedPath := filepath.Join(jobDirectory, "config.json")

	if path != expectedPath {
		t.Fatalf(
			"expected config path %q, got %q",
			expectedPath,
			path,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var saved DatasetConfig

	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("saved config contains invalid JSON: %v", err)
	}

	if !reflect.DeepEqual(saved, config) {
		t.Fatalf(
			"saved configuration does not match original\nexpected: %#v\ngot: %#v",
			config,
			saved,
		)
	}
}

func TestLoadDatasetConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	expected := DatasetConfig{
		Model:             "random_forest_regressor",
		Target:            "co2",
		FeatureSelection:  "automatic",
		ConfigurationType: "automatic",
		Configuration: RandomForestConfig{
			NEstimators: 100,
			MaxDepth:    0,
		},
		Visualizations: VisualizationConfig{
			Histograms:         true,
			BoxPlots:           false,
			CorrelationHeatmap: true,
			ActualVsPredicted:  true,
		},
	}

	data, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test configuration: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write test configuration: %v", err)
	}

	actual, err := LoadDatasetConfig(configPath)
	if err != nil {
		t.Fatalf("LoadDatasetConfig failed: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf(
			"loaded configuration does not match expected\nexpected: %#v\ngot: %#v",
			expected,
			actual,
		)
	}
}

func TestSaveAndLoadDatasetConfig(t *testing.T) {
	jobID := int64(12346)

	original := DatasetConfig{
		Model:             "random_forest_regressor",
		Target:            "co2",
		FeatureSelection:  "manual",
		Features:          []string{"temperature", "humidity"},
		ConfigurationType: "manual",
		Configuration: RandomForestConfig{
			NEstimators: 12,
			MaxDepth:    10,
		},
		Visualizations: VisualizationConfig{
			Histograms:         true,
			BoxPlots:           true,
			CorrelationHeatmap: false,
			ActualVsPredicted:  true,
		},
	}

	jobDirectory := filepath.Join("uploads", "12346")

	if err := os.MkdirAll(jobDirectory, 0755); err != nil {
		t.Fatalf("failed to create test job directory: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(jobDirectory)
	})

	configPath, err := SaveDatasetConfig(original, jobID)
	if err != nil {
		t.Fatalf("SaveDatasetConfig failed: %v", err)
	}

	loaded, err := LoadDatasetConfig(configPath)
	if err != nil {
		t.Fatalf("LoadDatasetConfig failed: %v", err)
	}

	if !reflect.DeepEqual(loaded, original) {
		t.Fatalf(
			"configuration changed during save/load round trip\nexpected: %#v\ngot: %#v",
			original,
			loaded,
		)
	}
}

func TestLoadDatasetConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, err := LoadDatasetConfig(path)
	if err == nil {
		t.Fatal("expected error when loading missing configuration file")
	}
}

func TestLoadDatasetConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.json")

	invalidJSON := `{
		"model": "random_forest_regressor",
		"target": "co2",
		"configuration": {
			"n_estimators": 100,
		}
	}`

	if err := os.WriteFile(path, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	_, err := LoadDatasetConfig(path)
	if err == nil {
		t.Fatal("expected error when loading invalid JSON")
	}
}

func TestDatasetConfigJSONTags(t *testing.T) {
	config := DatasetConfig{
		Model:             "random_forest_regressor",
		Target:            "co2",
		FeatureSelection:  "automatic",
		ConfigurationType: "automatic",
		Configuration: RandomForestConfig{
			NEstimators: 100,
			MaxDepth:    10,
		},
		Visualizations: VisualizationConfig{
			Histograms:         true,
			CorrelationHeatmap: true,
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal configuration: %v", err)
	}

	jsonString := string(data)

	expectedKeys := []string{
		`"model"`,
		`"target"`,
		`"feature_selection"`,
		`"configuration_type"`,
		`"configuration"`,
		`"n_estimators"`,
		`"max_depth"`,
		`"visualizations"`,
		`"histograms"`,
		`"box_plots"`,
		`"correlation_heatmap"`,
		`"actual_vs_predicted"`,
	}

	for _, key := range expectedKeys {
		if !strings.Contains(jsonString, key) {
			t.Errorf("expected JSON to contain key %s", key)
		}
	}
}

func TestDatasetConfigOptionalFeatures(t *testing.T) {
	config := DatasetConfig{
		Model:             "random_forest_regressor",
		Target:            "co2",
		FeatureSelection:  "automatic",
		ConfigurationType: "automatic",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal configuration: %v", err)
	}

	// Features has omitempty, so it should not be emitted when nil.
	if strings.Contains(string(data), `"features"`) {
		t.Error("expected nil features to be omitted from JSON")
	}
}
