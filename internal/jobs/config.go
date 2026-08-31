package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/storage"
)

type DatasetConfig struct {
	Model             string              `json:"model"`
	Target            string              `json:"target"`
	FeatureSelection  string              `json:"feature_selection"`
	Features          []string            `json:"features,omitempty"`
	ConfigurationType string              `json:"configuration_type"`
	Configuration     RandomForestConfig  `json:"configuration,omitempty"`
	Visualizations    VisualizationConfig `json:"visualizations"`
}

type VisualizationConfig struct {
	Histograms         bool `json:"histograms"`
	BoxPlots           bool `json:"box_plots"`
	CorrelationHeatmap bool `json:"correlation_heatmap"`
	ActualVsPredicted  bool `json:"actual_vs_predicted"`
}

type RandomForestConfig struct {
	NEstimators int `json:"n_estimators,omitempty"`
	MaxDepth    int `json:"max_depth,omitempty"`
}

func SaveDatasetConfig(config DatasetConfig, jobID int64, store storage.Storage) (string, error) {
	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	data, err := json.MarshalIndent(
		config,
		"",
		"  ",
	)
	if err != nil {
		return "", err
	}

	if err := store.Put(
		context.Background(),
		configKey,
		bytes.NewReader(data),
	); err != nil {
		return "", fmt.Errorf(
			"failed to save dataset config: %w",
			err,
		)
	}

	return configKey, nil
}

func LoadDatasetConfig(configPath string) (DatasetConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DatasetConfig{}, err
	}

	var config DatasetConfig

	if err := json.Unmarshal(data, &config); err != nil {
		return DatasetConfig{}, err
	}

	return config, nil
}

type ImageConfig struct {
	Resize             bool   `json:"resize"`
	ResizeWidth        int    `json:"resize_width,omitempty"`
	ResizeHeight       int    `json:"resize_height,omitempty"`
	Compression        bool   `json:"compression"`
	CompressionQuality int    `json:"compression_quality,omitempty"`
	FormatConversion   bool   `json:"format_conversion"`
	OutputFormat       string `json:"output_format,omitempty"`
	ExtractMetadata    bool   `json:"extract_metadata"`
}

func LoadImageConfig(
	configPath string,
) (ImageConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ImageConfig{}, err
	}

	var config ImageConfig

	if err := json.Unmarshal(
		data,
		&config,
	); err != nil {
		return ImageConfig{}, err
	}

	return config, nil
}

type RouteConfig struct {
	StartLocation string                  `json:"start_location"`
	EndLocation   string                  `json:"end_location,omitempty"`
	Optimization  RouteOptimizationConfig `json:"optimization"`
	Constraints   RouteConstraintConfig   `json:"constraints,omitempty"`
}

type RouteOptimizationConfig struct {
	Algorithm string `json:"algorithm"`
}

type RouteConstraintConfig struct {
	MaxDistance *float64 `json:"max_distance,omitempty"`
	MaxStops    *int     `json:"max_stops,omitempty"`
}

func SaveImageConfig(
	config ImageConfig,
	jobID int64,
	store storage.Storage,
) (string, error) {
	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	data, err := json.MarshalIndent(
		config,
		"",
		"  ",
	)
	if err != nil {
		return "", err
	}

	if err := store.Put(
		context.Background(),
		configKey,
		bytes.NewReader(data),
	); err != nil {
		return "", fmt.Errorf(
			"failed to save image config: %w",
			err,
		)
	}

	return configKey, nil
}

func SaveRouteConfig(
	config RouteConfig,
	jobID int64,
	store storage.Storage,
) (string, error) {
	configKey := fmt.Sprintf(
		"jobs/%d/config.json",
		jobID,
	)

	data, err := json.MarshalIndent(
		config,
		"",
		"  ",
	)
	if err != nil {
		return "", err
	}

	if err := store.Put(
		context.Background(),
		configKey,
		bytes.NewReader(data),
	); err != nil {
		return "", fmt.Errorf(
			"failed to save route config: %w",
			err,
		)
	}

	return configKey, nil
}

func LoadRouteConfig(
	configPath string,
) (RouteConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return RouteConfig{}, err
	}

	var config RouteConfig

	if err := json.Unmarshal(
		data,
		&config,
	); err != nil {
		return RouteConfig{}, err
	}

	return config, nil
}
