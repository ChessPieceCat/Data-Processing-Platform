import matplotlib.pyplot as plt
import seaborn as sns

from pathlib import Path

import pandas as pd


def build_output_path(output_path, suffix):
    """Build a visualization path from the main results path."""
    output = Path(output_path)

    return str(
        output.with_name(
            f"{output.stem}_{suffix}.png"
        )
    )


def generate_feature_distributions(
    df,
    visualization_config,
    output_path,
):
    """Generate combined histogram and box-plot visualization."""

    histograms_enabled = visualization_config.get(
        "histograms",
        False,
    )

    box_plots_enabled = visualization_config.get(
        "box_plots",
        False,
    )

    if not histograms_enabled and not box_plots_enabled:
        return None

    numeric_columns = df.select_dtypes(
        include=["number"]
    ).columns

    if len(numeric_columns) == 0:
        raise ValueError(
            "No numeric columns are available for feature distributions."
        )

    num_plots = (
        int(histograms_enabled)
        + int(box_plots_enabled)
    )

    fig, axes = plt.subplots(
        len(numeric_columns),
        num_plots,
        figsize=(12, 4 * len(numeric_columns)),
        squeeze=False,
    )

    for row, column in enumerate(numeric_columns):
        plot_column = 0
        values = df[column].dropna()

        if histograms_enabled:
            axes[row, plot_column].hist(
                values,
                bins=30,
                color="blue",
                alpha=0.7,
            )

            axes[row, plot_column].set_title(
                f"Histogram of {column}"
            )

            axes[row, plot_column].set_xlabel(column)
            axes[row, plot_column].set_ylabel("Frequency")

            plot_column += 1

        if box_plots_enabled:
            axes[row, plot_column].boxplot(values)

            axes[row, plot_column].set_title(
                f"Box Plot of {column}"
            )

            axes[row, plot_column].set_ylabel(column)

    fig.tight_layout()

    feature_distributions_path = build_output_path(
        output_path,
        "feature_distributions",
    )

    try:
        fig.savefig(feature_distributions_path)
    finally:
        plt.close(fig)

    return feature_distributions_path


def generate_correlation_heatmap(
    df,
    visualization_config,
    output_path,
):
    """Generate a numeric correlation heatmap."""

    if not visualization_config.get(
        "correlation_heatmap",
        False,
    ):
        return None

    correlation_data = df.select_dtypes(
        include=["number"]
    ).corr()

    if correlation_data.empty:
        raise ValueError(
            "No numeric columns are available for correlation heatmap."
        )

    fig, ax = plt.subplots(
        figsize=(10, 8)
    )

    sns.heatmap(
        correlation_data,
        annot=True,
        fmt=".2f",
        cmap="coolwarm",
        ax=ax,
    )

    ax.set_title("Correlation Heatmap")

    correlation_heatmap_path = build_output_path(
        output_path,
        "correlation_heatmap",
    )

    try:
        fig.savefig(correlation_heatmap_path)
    finally:
        plt.close(fig)

    return correlation_heatmap_path


def generate_actual_vs_predicted(
    y_test,
    predictions,
    output_path,
):
    """Generate an actual-vs-predicted scatter plot."""

    fig, ax = plt.subplots(
        figsize=(8, 6)
    )

    ax.scatter(
        y_test,
        predictions,
        alpha=0.5,
    )

    minimum = y_test.min()
    maximum = y_test.max()

    ax.plot(
        [minimum, maximum],
        [minimum, maximum],
        "r--",
        linewidth=2,
    )

    ax.set_xlabel("Actual")
    ax.set_ylabel("Predicted")
    ax.set_title("Actual vs Predicted")

    actual_vs_predicted_path = build_output_path(
        output_path,
        "actual_vs_predicted",
    )

    try:
        fig.savefig(actual_vs_predicted_path)
    finally:
        plt.close(fig)

    return actual_vs_predicted_path


def generate_dataset_visualizations(
    df,
    visualization_config,
    output_path,
):
    """Generate all dataset-level visualizations."""

    visualizations = {}

    feature_distributions_path = (
        generate_feature_distributions(
            df,
            visualization_config,
            output_path,
        )
    )

    if feature_distributions_path:
        visualizations[
            "feature_distributions"
        ] = feature_distributions_path

    correlation_heatmap_path = (
        generate_correlation_heatmap(
            df,
            visualization_config,
            output_path,
        )
    )

    if correlation_heatmap_path:
        visualizations[
            "correlation_heatmap"
        ] = correlation_heatmap_path

    return visualizations