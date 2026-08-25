import json
import sys

import pandas as pd

import main


def write_config(path, config):
    with open(path, "w", encoding="utf-8") as file:
        json.dump(config, file)


def write_csv(path):
    df = pd.DataFrame(
        {
            "temperature": [20, 21, 22, 23, 24, 25, 26, 27, 28, 29],
            "humidity": [40, 42, 41, 45, 43, 47, 46, 48, 49, 50],
            "co2": [400, 410, 420, 430, 440, 450, 460, 470, 480, 490],
        }
    )

    df.to_csv(path, index=False)


def test_parse_arguments(monkeypatch):
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            "input.csv",
            "results.json",
            "config.json",
        ],
    )

    args = main.parse_arguments()

    assert args.input_csv == "input.csv"
    assert args.output_json == "results.json"
    assert args.config_json == "config.json"


def test_main_without_model(tmp_path, monkeypatch):
    input_path = tmp_path / "input.csv"
    output_path = tmp_path / "results.json"
    config_path = tmp_path / "config.json"

    write_csv(input_path)

    write_config(
        config_path,
        {
            "model": "none",
            "visualizations": {},
        },
    )

    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            str(input_path),
            str(output_path),
            str(config_path),
        ],
    )

    result = main.main()

    assert result == 0
    assert output_path.exists()

    with open(output_path, "r", encoding="utf-8") as file:
        results = json.load(file)

    assert results["num_rows"] == 10
    assert results["num_columns"] == 3
    assert "column_names" in results
    assert "model_results" not in results


def test_main_with_random_forest(tmp_path, monkeypatch):
    input_path = tmp_path / "input.csv"
    output_path = tmp_path / "results.json"
    config_path = tmp_path / "config.json"

    write_csv(input_path)

    write_config(
        config_path,
        {
            "model": "random_forest_regressor",
            "target": "co2",
            "feature_selection": "automatic",
            "configuration_type": "automatic",
            "configuration": {},
            "visualizations": {},
        },
    )

    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            str(input_path),
            str(output_path),
            str(config_path),
        ],
    )

    result = main.main()

    assert result == 0
    assert output_path.exists()

    model_results_path = tmp_path / "results_model_results.json"

    assert model_results_path.exists()

    with open(model_results_path, "r", encoding="utf-8") as file:
        model_results = json.load(file)

    assert model_results["model"] == "random_forest_regressor"
    assert "evaluation" in model_results
    assert "feature_importances" in model_results
    assert "actual_vs_predicted" in model_results


def test_main_generates_visualizations(tmp_path, monkeypatch):
    input_path = tmp_path / "input.csv"
    output_path = tmp_path / "results.json"
    config_path = tmp_path / "config.json"

    write_csv(input_path)

    write_config(
        config_path,
        {
            "model": "none",
            "visualizations": {
                "histograms": True,
                "box_plots": True,
                "correlation_heatmap": True,
            },
        },
    )

    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            str(input_path),
            str(output_path),
            str(config_path),
        ],
    )

    result = main.main()

    assert result == 0

    feature_distribution_path = (
        tmp_path / "results_feature_distributions.png"
    )
    correlation_heatmap_path = (
        tmp_path / "results_correlation_heatmap.png"
    )

    assert feature_distribution_path.exists()
    assert correlation_heatmap_path.exists()

    with open(output_path, "r", encoding="utf-8") as file:
        results = json.load(file)

    assert "visualizations" in results
    assert "feature_distributions" in results["visualizations"]
    assert "correlation_heatmap" in results["visualizations"]


def test_main_generates_actual_vs_predicted(tmp_path, monkeypatch):
    input_path = tmp_path / "input.csv"
    output_path = tmp_path / "results.json"
    config_path = tmp_path / "config.json"

    write_csv(input_path)

    write_config(
        config_path,
        {
            "model": "random_forest_regressor",
            "target": "co2",
            "feature_selection": "automatic",
            "configuration_type": "automatic",
            "configuration": {},
            "visualizations": {
                "actual_vs_predicted": True,
            },
        },
    )

    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            str(input_path),
            str(output_path),
            str(config_path),
        ],
    )

    result = main.main()

    assert result == 0

    plot_path = tmp_path / "results_actual_vs_predicted.png"

    assert plot_path.exists()

    with open(output_path, "r", encoding="utf-8") as file:
        results = json.load(file)

    assert (
        "actual_vs_predicted"
        in results["visualizations"]
    )


def test_main_returns_one_when_csv_cannot_be_read(
    tmp_path,
    monkeypatch,
):
    input_path = tmp_path / "missing.csv"
    output_path = tmp_path / "results.json"
    config_path = tmp_path / "config.json"

    write_config(
        config_path,
        {
            "model": "none",
            "visualizations": {},
        },
    )

    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            str(input_path),
            str(output_path),
            str(config_path),
        ],
    )

    result = main.main()

    assert result == 1
    assert not output_path.exists()


def test_main_returns_one_for_invalid_configuration(
    tmp_path,
    monkeypatch,
):
    input_path = tmp_path / "input.csv"
    output_path = tmp_path / "results.json"
    config_path = tmp_path / "config.json"

    write_csv(input_path)

    write_config(
        config_path,
        {
            "model": "random_forest_regressor",
            "target": "does_not_exist",
            "feature_selection": "automatic",
            "configuration_type": "automatic",
            "configuration": {},
            "visualizations": {},
        },
    )

    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            str(input_path),
            str(output_path),
            str(config_path),
        ],
    )

    result = main.main()

    assert result == 1
    assert not output_path.exists()