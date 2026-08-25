package jobs

import (
	"encoding/json"
	"fmt"
	"os"
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

func SaveDatasetConfig(config DatasetConfig, jobID int64) (string, error) {
	configPath := fmt.Sprintf("uploads/%d/config.json", jobID)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", err
	}

	return configPath, nil
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
