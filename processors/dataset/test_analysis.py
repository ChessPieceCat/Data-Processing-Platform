import pandas as pd

from analysis import (
    analyze_dataset,
    calculate_categorical_summary,
    calculate_correlation_matrix,
    calculate_feature_distributions,
    calculate_numeric_summary,
    detect_outliers,
    determine_invalid_entries,
    get_numeric_columns,
    get_string_columns,
)


def create_basic_dataframe():
    return pd.DataFrame(
        {
            "temperature": [20.0, 22.0, 24.0, 26.0, 28.0],
            "humidity": [40.0, 45.0, 50.0, 55.0, 60.0],
            "city": ["Chicago", "Chicago", "Denver", "Denver", "Chicago"],
        }
    )


def test_get_string_columns():
    df = create_basic_dataframe()

    columns = get_string_columns(df)

    assert list(columns) == ["city"]


def test_get_numeric_columns():
    df = create_basic_dataframe()

    columns = get_numeric_columns(df)

    assert list(columns) == ["temperature", "humidity"]


def test_get_string_columns_supports_pandas_string_dtype():
    df = pd.DataFrame(
        {
            "category": pd.Series(
                ["A", "B", "A"],
                dtype="string",
            ),
            "value": [1, 2, 3],
        }
    )

    columns = get_string_columns(df)

    assert list(columns) == ["category"]


def test_determine_invalid_entries():
    df = pd.DataFrame(
        {
            "name": ["Alice", "", "   ", "Bob"],
            "value": [1, 2, 3, 4],
        }
    )

    string_columns = get_string_columns(df)

    result = determine_invalid_entries(
        df,
        string_columns,
    )

    assert result["name"] == 2
    assert result["value"] == 0


def test_determine_invalid_entries_with_no_string_columns():
    df = pd.DataFrame(
        {
            "a": [1, 2, 3],
            "b": [4.0, 5.0, 6.0],
        }
    )

    string_columns = get_string_columns(df)

    result = determine_invalid_entries(
        df,
        string_columns,
    )

    assert result == {
        "a": 0,
        "b": 0,
    }


def test_calculate_numeric_summary():
    df = pd.DataFrame(
        {
            "value": [1.0, 2.0, 3.0, 4.0, 5.0],
        }
    )

    result = calculate_numeric_summary(
        df,
        get_numeric_columns(df),
    )

    assert result["value"]["mean"] == 3.0
    assert result["value"]["median"] == 3.0
    assert result["value"]["min"] == 1.0
    assert result["value"]["max"] == 5.0

    # pandas uses sample standard deviation by default.
    assert result["value"]["std_dev"] == 1.5811388300841898


def test_calculate_categorical_summary():
    df = pd.DataFrame(
        {
            "city": [
                "Chicago",
                "Chicago",
                "Denver",
                "Chicago",
            ]
        }
    )

    result = calculate_categorical_summary(
        df,
        get_string_columns(df),
    )

    assert result["city"]["mode"] == "Chicago"
    assert result["city"]["unique_values"] == 2
    assert result["city"]["value_counts"]["Chicago"] == 3
    assert result["city"]["value_counts"]["Denver"] == 1


def test_calculate_categorical_summary_with_no_mode():
    df = pd.DataFrame(
        {
            "category": pd.Series(
                [None, None, None],
                dtype="string",
            )
        }
    )

    result = calculate_categorical_summary(
        df,
        get_string_columns(df),
    )

    assert result["category"]["mode"] is None
    assert result["category"]["unique_values"] == 0


def test_detect_outliers():
    df = pd.DataFrame(
        {
            "value": [
                10.0,
                11.0,
                12.0,
                13.0,
                100.0,
            ]
        }
    )

    result = detect_outliers(
        df,
        get_numeric_columns(df),
    )

    assert result["value"]["num_outliers"] == 1
    assert result["value"]["outlier_indices"] == [4]


def test_detect_outliers_without_outliers():
    df = pd.DataFrame(
        {
            "value": [10.0, 11.0, 12.0, 13.0, 14.0],
        }
    )

    result = detect_outliers(
        df,
        get_numeric_columns(df),
    )

    assert result["value"]["num_outliers"] == 0
    assert result["value"]["outlier_indices"] == []


def test_calculate_feature_distributions():
    df = pd.DataFrame(
        {
            "number": [1, 1, 2, 2],
            "category": ["A", "A", "B", "B"],
        }
    )

    numeric_columns = get_numeric_columns(df)
    string_columns = get_string_columns(df)

    result = calculate_feature_distributions(
        df,
        numeric_columns,
        string_columns,
    )

    assert result["number"][1] == 0.5
    assert result["number"][2] == 0.5
    assert result["category"]["A"] == 0.5
    assert result["category"]["B"] == 0.5


def test_calculate_correlation_matrix():
    df = pd.DataFrame(
        {
            "x": [1.0, 2.0, 3.0, 4.0],
            "y": [2.0, 4.0, 6.0, 8.0],
            "category": ["A", "B", "A", "B"],
        }
    )

    result = calculate_correlation_matrix(
        df,
        get_numeric_columns(df),
    )

    assert result["x"]["x"] == 1.0
    assert result["y"]["y"] == 1.0
    assert result["x"]["y"] == 1.0
    assert result["y"]["x"] == 1.0
    assert "category" not in result


def test_analyze_dataset():
    df = pd.DataFrame(
        {
            "temperature": [20.0, 21.0, 22.0, 23.0],
            "city": ["Chicago", "Chicago", "Denver", "Denver"],
        }
    )

    result = analyze_dataset(df)

    assert result["num_rows"] == 4
    assert result["num_columns"] == 2

    assert result["column_names"] == [
        "temperature",
        "city",
    ]

    assert result["data_types"]["temperature"] == "float64"

    assert result["missing_values"]["temperature"] == 0
    assert result["missing_values"]["city"] == 0

    assert result["duplicate_rows"] == 0

    assert result["unique_values"]["temperature"] == 4
    assert result["unique_values"]["city"] == 2

    assert "temperature" in result["numeric_summary"]
    assert "city" in result["categorical_summary"]

    assert "temperature" in result["outlier_summary"]

    assert "temperature" in result["feature_distributions"]
    assert "city" in result["feature_distributions"]

    assert "temperature" in result["correlation_matrix"]

    assert "temperature" in result["missing_percentage"]
    assert result["missing_percentage"]["temperature"] == 0.0


def test_analyze_dataset_with_missing_values_and_duplicates():
    df = pd.DataFrame(
        {
            "value": [1.0, None, 1.0, 4.0],
            "category": ["A", "B", "A", "A"],
        }
    )

    result = analyze_dataset(df)

    assert result["missing_values"]["value"] == 1
    assert result["missing_percentage"]["value"] == 25.0

    assert result["duplicate_rows"] == 1

    assert result["numeric_summary"]["value"]["mean"] == 2.0


def test_analyze_dataset_counts_duplicate_rows():
    df = pd.DataFrame(
        {
            "value": [1, 1, 2],
            "category": ["A", "A", "B"],
        }
    )

    result = analyze_dataset(df)

    assert result["duplicate_rows"] == 1