import numpy as np
import pandas as pd
import pytest

from visualizations import (
    build_output_path,
    generate_actual_vs_predicted,
    generate_correlation_heatmap,
    generate_dataset_visualizations,
    generate_feature_distributions,
)


def create_numeric_dataframe():
    return pd.DataFrame(
        {
            "temperature": [20.0, 21.0, 22.0, 23.0, 24.0],
            "humidity": [40.0, 45.0, 50.0, 55.0, 60.0],
            "pressure": [1000.0, 1005.0, 1010.0, 1015.0, 1020.0],
        }
    )


def create_mixed_dataframe():
    return pd.DataFrame(
        {
            "temperature": [20.0, 21.0, 22.0, 23.0, 24.0],
            "city": ["Chicago", "Denver", "Chicago", "Denver", "Chicago"],
        }
    )


def test_build_output_path():
    result = build_output_path(
        "/tmp/results.json",
        "feature_distributions",
    )

    assert result == "/tmp/results_feature_distributions.png"


def test_build_output_path_with_nested_directory():
    result = build_output_path(
        "/tmp/job/results.json",
        "correlation_heatmap",
    )

    assert result == "/tmp/job/results_correlation_heatmap.png"


def test_generate_feature_distributions_disabled(tmp_path):
    df = create_numeric_dataframe()

    result = generate_feature_distributions(
        df,
        {
            "histograms": False,
            "box_plots": False,
        },
        str(tmp_path / "results.json"),
    )

    assert result is None

    assert not (
        tmp_path / "results_feature_distributions.png"
    ).exists()


def test_generate_histograms(tmp_path):
    df = create_numeric_dataframe()
    output_path = tmp_path / "results.json"

    result = generate_feature_distributions(
        df,
        {
            "histograms": True,
            "box_plots": False,
        },
        str(output_path),
    )

    expected_path = (
        tmp_path / "results_feature_distributions.png"
    )

    assert result == str(expected_path)
    assert expected_path.exists()
    assert expected_path.stat().st_size > 0


def test_generate_box_plots(tmp_path):
    df = create_numeric_dataframe()
    output_path = tmp_path / "results.json"

    result = generate_feature_distributions(
        df,
        {
            "histograms": False,
            "box_plots": True,
        },
        str(output_path),
    )

    expected_path = (
        tmp_path / "results_feature_distributions.png"
    )

    assert result == str(expected_path)
    assert expected_path.exists()
    assert expected_path.stat().st_size > 0


def test_generate_histograms_and_box_plots(tmp_path):
    df = create_numeric_dataframe()
    output_path = tmp_path / "results.json"

    result = generate_feature_distributions(
        df,
        {
            "histograms": True,
            "box_plots": True,
        },
        str(output_path),
    )

    expected_path = (
        tmp_path / "results_feature_distributions.png"
    )

    assert result == str(expected_path)
    assert expected_path.exists()
    assert expected_path.stat().st_size > 0


def test_generate_feature_distributions_ignores_missing_values(
    tmp_path,
):
    df = create_numeric_dataframe()

    df.loc[0, "temperature"] = np.nan
    df.loc[1, "humidity"] = np.nan

    result = generate_feature_distributions(
        df,
        {
            "histograms": True,
            "box_plots": True,
        },
        str(tmp_path / "results.json"),
    )

    assert result is not None
    assert (
        tmp_path / "results_feature_distributions.png"
    ).exists()


def test_generate_feature_distributions_without_numeric_columns(
    tmp_path,
):
    df = create_mixed_dataframe()
    df = df[["city"]]

    with pytest.raises(
        ValueError,
        match="No numeric columns are available for feature distributions",
    ):
        generate_feature_distributions(
            df,
            {
                "histograms": True,
            },
            str(tmp_path / "results.json"),
        )


def test_generate_correlation_heatmap_disabled(tmp_path):
    df = create_numeric_dataframe()

    result = generate_correlation_heatmap(
        df,
        {
            "correlation_heatmap": False,
        },
        str(tmp_path / "results.json"),
    )

    assert result is None

    assert not (
        tmp_path / "results_correlation_heatmap.png"
    ).exists()


def test_generate_correlation_heatmap(tmp_path):
    df = create_numeric_dataframe()
    output_path = tmp_path / "results.json"

    result = generate_correlation_heatmap(
        df,
        {
            "correlation_heatmap": True,
        },
        str(output_path),
    )

    expected_path = (
        tmp_path / "results_correlation_heatmap.png"
    )

    assert result == str(expected_path)
    assert expected_path.exists()
    assert expected_path.stat().st_size > 0


def test_generate_correlation_heatmap_without_numeric_columns(
    tmp_path,
):
    df = create_mixed_dataframe()
    df = df[["city"]]

    with pytest.raises(
        ValueError,
        match="No numeric columns are available for correlation heatmap",
    ):
        generate_correlation_heatmap(
            df,
            {
                "correlation_heatmap": True,
            },
            str(tmp_path / "results.json"),
        )


def test_generate_actual_vs_predicted(tmp_path):
    y_test = np.array([10.0, 20.0, 30.0, 40.0])
    predictions = np.array([11.0, 19.0, 31.0, 39.0])

    output_path = tmp_path / "results.json"

    result = generate_actual_vs_predicted(
        y_test,
        predictions,
        str(output_path),
    )

    expected_path = (
        tmp_path / "results_actual_vs_predicted.png"
    )

    assert result == str(expected_path)
    assert expected_path.exists()
    assert expected_path.stat().st_size > 0


def test_generate_dataset_visualizations_none_requested(tmp_path):
    df = create_numeric_dataframe()

    result = generate_dataset_visualizations(
        df,
        {},
        str(tmp_path / "results.json"),
    )

    assert result == {}


def test_generate_dataset_visualizations_feature_distributions(
    tmp_path,
):
    df = create_numeric_dataframe()

    result = generate_dataset_visualizations(
        df,
        {
            "histograms": True,
            "box_plots": True,
            "correlation_heatmap": False,
        },
        str(tmp_path / "results.json"),
    )

    assert "feature_distributions" in result
    assert "correlation_heatmap" not in result

    assert (
        tmp_path / "results_feature_distributions.png"
    ).exists()


def test_generate_dataset_visualizations_all(
    tmp_path,
):
    df = create_numeric_dataframe()

    result = generate_dataset_visualizations(
        df,
        {
            "histograms": True,
            "box_plots": True,
            "correlation_heatmap": True,
        },
        str(tmp_path / "results.json"),
    )

    assert set(result.keys()) == {
        "feature_distributions",
        "correlation_heatmap",
    }

    assert (
        tmp_path / "results_feature_distributions.png"
    ).exists()

    assert (
        tmp_path / "results_correlation_heatmap.png"
    ).exists()