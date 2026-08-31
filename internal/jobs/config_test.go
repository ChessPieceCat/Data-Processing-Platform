package jobs

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"
)

func TestSaveDatasetConfig(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())
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

	path, err := SaveDatasetConfig(
		config,
		jobID,
		store,
	)
	if err != nil {
		t.Fatalf(
			"SaveDatasetConfig failed: %v",
			err,
		)
	}

	expectedPath := "jobs/12345/config.json"

	if path != expectedPath {
		t.Fatalf(
			"expected config path %q, got %q",
			expectedPath,
			path,
		)
	}

	reader, err := store.Get(
		context.Background(),
		path,
	)
	if err != nil {
		t.Fatalf(
			"failed to get saved config: %v",
			err,
		)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf(
			"failed to read saved config: %v",
			err,
		)
	}

	var saved DatasetConfig

	if err := json.Unmarshal(
		data,
		&saved,
	); err != nil {
		t.Fatalf(
			"saved config contains invalid JSON: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		saved,
		config,
	) {
		t.Fatalf(
			"saved configuration does not match original\nexpected: %#v\ngot: %#v",
			config,
			saved,
		)
	}
}

func TestLoadDatasetConfig(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(
		tempDir,
		"config.json",
	)

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

	data, err := json.MarshalIndent(
		expected,
		"",
		"  ",
	)
	if err != nil {
		t.Fatalf(
			"failed to marshal test configuration: %v",
			err,
		)
	}

	if err := os.WriteFile(
		configPath,
		data,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write test configuration: %v",
			err,
		)
	}

	actual, err := LoadDatasetConfig(
		configPath,
	)
	if err != nil {
		t.Fatalf(
			"LoadDatasetConfig failed: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		actual,
		expected,
	) {
		t.Fatalf(
			"loaded configuration does not match expected\nexpected: %#v\ngot: %#v",
			expected,
			actual,
		)
	}
}

func TestSaveAndLoadDatasetConfig(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())
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

	configKey, err := SaveDatasetConfig(
		original,
		jobID,
		store,
	)
	if err != nil {
		t.Fatalf(
			"SaveDatasetConfig failed: %v",
			err,
		)
	}

	reader, err := store.Get(
		context.Background(),
		configKey,
	)
	if err != nil {
		t.Fatalf(
			"failed to get saved config: %v",
			err,
		)
	}

	data, err := io.ReadAll(reader)
	reader.Close()

	if err != nil {
		t.Fatalf(
			"failed to read saved config: %v",
			err,
		)
	}

	tempDir := t.TempDir()

	configPath := filepath.Join(
		tempDir,
		"config.json",
	)

	if err := os.WriteFile(
		configPath,
		data,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to create temporary config file: %v",
			err,
		)
	}

	loaded, err := LoadDatasetConfig(
		configPath,
	)
	if err != nil {
		t.Fatalf(
			"LoadDatasetConfig failed: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		loaded,
		original,
	) {
		t.Fatalf(
			"configuration changed during save/load round trip\nexpected: %#v\ngot: %#v",
			original,
			loaded,
		)
	}
}

func TestLoadDatasetConfigMissingFile(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"missing.json",
	)

	_, err := LoadDatasetConfig(path)

	if err == nil {
		t.Fatal(
			"expected error when loading missing configuration file",
		)
	}
}

func TestLoadDatasetConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"invalid.json",
	)

	invalidJSON := `{
		"model": "random_forest_regressor",
		"target": "co2",
		"configuration": {
			"n_estimators": 100,
		}
	}`

	if err := os.WriteFile(
		path,
		[]byte(invalidJSON),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write invalid JSON: %v",
			err,
		)
	}

	_, err := LoadDatasetConfig(path)

	if err == nil {
		t.Fatal(
			"expected error when loading invalid JSON",
		)
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
		t.Fatalf(
			"failed to marshal configuration: %v",
			err,
		)
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
		if !strings.Contains(
			jsonString,
			key,
		) {
			t.Errorf(
				"expected JSON to contain key %s",
				key,
			)
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
		t.Fatalf(
			"failed to marshal configuration: %v",
			err,
		)
	}

	// Features has omitempty, so it should not be emitted when nil.
	if strings.Contains(
		string(data),
		`"features"`,
	) {
		t.Error(
			"expected nil features to be omitted from JSON",
		)
	}
}

func TestSaveRouteConfig(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())
	jobID := int64(12347)

	maxDistance := 140.0
	maxStops := 40

	config := RouteConfig{
		StartLocation: "Warehouse",
		EndLocation:   "Warehouse",
		Optimization: RouteOptimizationConfig{
			Algorithm: "nearest_neighbor_2opt",
		},
		Constraints: RouteConstraintConfig{
			MaxDistance: &maxDistance,
			MaxStops:    &maxStops,
		},
	}

	path, err := SaveRouteConfig(
		config,
		jobID,
		store,
	)
	if err != nil {
		t.Fatalf(
			"SaveRouteConfig failed: %v",
			err,
		)
	}

	expectedPath := "jobs/12347/config.json"

	if path != expectedPath {
		t.Fatalf(
			"expected config path %q, got %q",
			expectedPath,
			path,
		)
	}

	reader, err := store.Get(
		context.Background(),
		path,
	)
	if err != nil {
		t.Fatalf(
			"failed to get saved config: %v",
			err,
		)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf(
			"failed to read saved config: %v",
			err,
		)
	}

	var saved RouteConfig

	if err := json.Unmarshal(
		data,
		&saved,
	); err != nil {
		t.Fatalf(
			"saved route config contains invalid JSON: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		saved,
		config,
	) {
		t.Fatalf(
			"saved route configuration does not match original\nexpected: %#v\ngot: %#v",
			config,
			saved,
		)
	}
}

func TestLoadRouteConfig(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(
		tempDir,
		"config.json",
	)

	maxDistance := 200.5
	maxStops := 25

	expected := RouteConfig{
		StartLocation: "Depot",
		EndLocation:   "Warehouse",
		Optimization: RouteOptimizationConfig{
			Algorithm: "nearest_neighbor_2opt",
		},
		Constraints: RouteConstraintConfig{
			MaxDistance: &maxDistance,
			MaxStops:    &maxStops,
		},
	}

	data, err := json.MarshalIndent(
		expected,
		"",
		"  ",
	)
	if err != nil {
		t.Fatalf(
			"failed to marshal route configuration: %v",
			err,
		)
	}

	if err := os.WriteFile(
		configPath,
		data,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write route configuration: %v",
			err,
		)
	}

	actual, err := LoadRouteConfig(
		configPath,
	)
	if err != nil {
		t.Fatalf(
			"LoadRouteConfig failed: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		actual,
		expected,
	) {
		t.Fatalf(
			"loaded route configuration does not match expected\nexpected: %#v\ngot: %#v",
			expected,
			actual,
		)
	}
}

func TestSaveAndLoadRouteConfig(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())
	jobID := int64(12348)

	maxDistance := 140.0
	maxStops := 40

	original := RouteConfig{
		StartLocation: "Warehouse",
		EndLocation:   "Warehouse",
		Optimization: RouteOptimizationConfig{
			Algorithm: "nearest_neighbor_2opt",
		},
		Constraints: RouteConstraintConfig{
			MaxDistance: &maxDistance,
			MaxStops:    &maxStops,
		},
	}

	configKey, err := SaveRouteConfig(
		original,
		jobID,
		store,
	)
	if err != nil {
		t.Fatalf(
			"SaveRouteConfig failed: %v",
			err,
		)
	}

	reader, err := store.Get(
		context.Background(),
		configKey,
	)
	if err != nil {
		t.Fatalf(
			"failed to get saved config: %v",
			err,
		)
	}

	data, err := io.ReadAll(reader)
	reader.Close()

	if err != nil {
		t.Fatalf(
			"failed to read saved config: %v",
			err,
		)
	}

	tempDir := t.TempDir()

	configPath := filepath.Join(
		tempDir,
		"config.json",
	)

	if err := os.WriteFile(
		configPath,
		data,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to create temporary config file: %v",
			err,
		)
	}

	loaded, err := LoadRouteConfig(
		configPath,
	)
	if err != nil {
		t.Fatalf(
			"LoadRouteConfig failed: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		loaded,
		original,
	) {
		t.Fatalf(
			"route configuration changed during save/load round trip\nexpected: %#v\ngot: %#v",
			original,
			loaded,
		)
	}
}

func TestSaveAndLoadRouteConfigWithoutOptionalFields(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())
	jobID := int64(12349)

	original := RouteConfig{
		StartLocation: "Warehouse",
		Optimization: RouteOptimizationConfig{
			Algorithm: "nearest_neighbor_2opt",
		},
	}

	configKey, err := SaveRouteConfig(
		original,
		jobID,
		store,
	)
	if err != nil {
		t.Fatalf(
			"SaveRouteConfig failed: %v",
			err,
		)
	}

	reader, err := store.Get(
		context.Background(),
		configKey,
	)
	if err != nil {
		t.Fatalf(
			"failed to get saved config: %v",
			err,
		)
	}

	data, err := io.ReadAll(reader)
	reader.Close()

	if err != nil {
		t.Fatalf(
			"failed to read saved config: %v",
			err,
		)
	}

	jsonString := string(data)

	if strings.Contains(
		jsonString,
		`"end_location"`,
	) {
		t.Error(
			"expected omitted end_location field",
		)
	}

	if strings.Contains(
		jsonString,
		`"max_distance"`,
	) {
		t.Error(
			"expected omitted max_distance field",
		)
	}

	if strings.Contains(
		jsonString,
		`"max_stops"`,
	) {
		t.Error(
			"expected omitted max_stops field",
		)
	}

	tempDir := t.TempDir()

	configPath := filepath.Join(
		tempDir,
		"config.json",
	)

	if err := os.WriteFile(
		configPath,
		data,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to create temporary config file: %v",
			err,
		)
	}

	loaded, err := LoadRouteConfig(
		configPath,
	)
	if err != nil {
		t.Fatalf(
			"LoadRouteConfig failed: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		loaded,
		original,
	) {
		t.Fatalf(
			"route configuration changed during save/load round trip\nexpected: %#v\ngot: %#v",
			original,
			loaded,
		)
	}
}

func TestLoadRouteConfigMissingFile(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"missing.json",
	)

	_, err := LoadRouteConfig(path)

	if err == nil {
		t.Fatal(
			"expected error when loading missing route configuration file",
		)
	}
}

func TestLoadRouteConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"invalid.json",
	)

	invalidJSON := `{
		"start_location": "Warehouse",
		"optimization": {
			"algorithm": "nearest_neighbor_2opt",
		}
	}`

	if err := os.WriteFile(
		path,
		[]byte(invalidJSON),
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write invalid route JSON: %v",
			err,
		)
	}

	_, err := LoadRouteConfig(path)

	if err == nil {
		t.Fatal(
			"expected error when loading invalid route JSON",
		)
	}
}

func TestRouteConfigJSONTags(t *testing.T) {
	maxDistance := 140.0
	maxStops := 40

	config := RouteConfig{
		StartLocation: "Warehouse",
		EndLocation:   "Customer C",
		Optimization: RouteOptimizationConfig{
			Algorithm: "nearest_neighbor_2opt",
		},
		Constraints: RouteConstraintConfig{
			MaxDistance: &maxDistance,
			MaxStops:    &maxStops,
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf(
			"failed to marshal route configuration: %v",
			err,
		)
	}

	jsonString := string(data)

	expectedKeys := []string{
		`"start_location"`,
		`"end_location"`,
		`"optimization"`,
		`"algorithm"`,
		`"constraints"`,
		`"max_distance"`,
		`"max_stops"`,
	}

	for _, key := range expectedKeys {
		if !strings.Contains(
			jsonString,
			key,
		) {
			t.Errorf(
				"expected JSON to contain key %s",
				key,
			)
		}
	}
}
