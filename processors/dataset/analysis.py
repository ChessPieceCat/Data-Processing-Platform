import pandas as pd


def get_string_columns(df):
    """Return columns containing string/categorical data."""
    return df.select_dtypes(include=["object", "string"]).columns


def get_numeric_columns(df):
    """Return columns containing numeric data."""
    return df.select_dtypes(include=["number"]).columns


def determine_invalid_entries(df, string_columns):
    """Count blank string entries for string columns."""
    invalid_entries = {}

    for column in df.columns:
        if column in string_columns:
            invalid_entries[column] = df[column].apply(
                lambda value: isinstance(value, str) and not value.strip()
            ).sum()
        else:
            invalid_entries[column] = 0

    return invalid_entries


def calculate_numeric_summary(df, numeric_columns):
    """Calculate summary statistics for numeric columns."""
    summary = {}

    for column in numeric_columns:
        summary[column] = {
            "mean": df[column].mean(),
            "median": df[column].median(),
            "std_dev": df[column].std(),
            "min": df[column].min(),
            "max": df[column].max(),
        }

    return summary


def calculate_categorical_summary(df, string_columns):
    """Calculate summary statistics for categorical columns."""
    summary = {}

    for column in string_columns:
        mode = df[column].mode()

        summary[column] = {
            "mode": mode.iloc[0] if not mode.empty else None,
            "unique_values": df[column].nunique(),
            "value_counts": df[column].value_counts().to_dict(),
        }

    return summary


def detect_outliers(df, numeric_columns):
    """Detect numeric outliers using the IQR method."""
    outlier_summary = {}

    for column in numeric_columns:
        q1 = df[column].quantile(0.25)
        q3 = df[column].quantile(0.75)

        iqr = q3 - q1

        lower_bound = q1 - 1.5 * iqr
        upper_bound = q3 + 1.5 * iqr

        outliers = df[
            (df[column] < lower_bound)
            | (df[column] > upper_bound)
        ]

        outlier_summary[column] = {
            "num_outliers": len(outliers),
            "outlier_indices": outliers.index.tolist(),
        }

    return outlier_summary


def calculate_feature_distributions(df, numeric_columns, string_columns):
    """Calculate normalized feature distributions."""
    distributions = {}

    for column in numeric_columns:
        distributions[column] = (
            df[column].value_counts(normalize=True).to_dict()
        )

    for column in string_columns:
        distributions[column] = (
            df[column].value_counts(normalize=True).to_dict()
        )

    return distributions


def calculate_correlation_matrix(df, numeric_columns):
    """Calculate the numeric correlation matrix."""
    return df[numeric_columns].corr().to_dict()


def analyze_dataset(df):
    """Perform all dataset-level analysis."""
    string_columns = get_string_columns(df)
    numeric_columns = get_numeric_columns(df)

    data_types = df.dtypes.apply(str).to_dict()

    missing_values = df.isnull().sum().to_dict()

    missing_percentage = (
        df.isnull().sum() / len(df) * 100
    ).to_dict()

    invalid_entries = determine_invalid_entries(
        df,
        string_columns,
    )

    numeric_summary = calculate_numeric_summary(
        df,
        numeric_columns,
    )

    categorical_summary = calculate_categorical_summary(
        df,
        string_columns,
    )

    outlier_summary = detect_outliers(
        df,
        numeric_columns,
    )

    feature_distributions = calculate_feature_distributions(
        df,
        numeric_columns,
        string_columns,
    )

    correlation_matrix = calculate_correlation_matrix(
        df,
        numeric_columns,
    )

    return {
        "num_rows": df.shape[0],
        "num_columns": df.shape[1],
        "column_names": df.columns.tolist(),
        "data_types": data_types,
        "missing_values": missing_values,
        "duplicate_rows": df.duplicated().sum(),
        "invalid_entries": invalid_entries,
        "unique_values": df.nunique().to_dict(),
        "descriptive_stats": df.describe().to_dict(),
        "numeric_summary": numeric_summary,
        "categorical_summary": categorical_summary,
        "outlier_summary": outlier_summary,
        "feature_distributions": feature_distributions,
        "correlation_matrix": correlation_matrix,
        "missing_percentage": missing_percentage,
    }