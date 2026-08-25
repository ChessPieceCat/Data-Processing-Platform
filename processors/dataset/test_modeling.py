import numpy as np
import pandas as pd
import pytest

from modeling import (
    aggregate_feature_importances,
    build_preprocessor,
    build_random_forest,
    train_random_forest,
)


def create_numeric_dataframe():
    return pd.DataFrame(
        {
            "temperature": np.arange(20, 40, dtype=float),
            "humidity": np.arange(40, 60, dtype=float),
            "pressure": np.arange(1000, 1020, dtype=float),
            "co2": np.arange(400, 440, 2, dtype=float),
        }
    )


def create_mixed_dataframe():
    return pd.DataFrame(
        {
            "temperature": np.arange(20, 40, dtype=float),
            "city": [
                "Chicago",
                "Denver",
                "Chicago",
                "Denver",
            ] * 5,
            "humidity": np.arange(40, 60, dtype=float),
            "co2": np.arange(400, 440, 2, dtype=float),
        }
    )


def test_build_preprocessor_numeric_columns():
    df = create_numeric_dataframe()

    X = df[["temperature", "humidity", "pressure"]]

    preprocessor, categorical_columns, numeric_columns = (
        build_preprocessor(X)
    )

    assert list(categorical_columns) == []
    assert list(numeric_columns) == [
        "temperature",
        "humidity",
        "pressure",
    ]

    assert list(categorical_columns) == []
    assert list(numeric_columns) == [
        "temperature",
        "humidity",
        "pressure",
    ]


def test_build_preprocessor_mixed_columns():
    df = create_mixed_dataframe()

    X = df[["temperature", "city", "humidity"]]

    preprocessor, categorical_columns, numeric_columns = (
        build_preprocessor(X)
    )

    assert list(categorical_columns) == ["city"]
    assert list(numeric_columns) == [
        "temperature",
        "humidity",
    ]


def test_build_random_forest_defaults():
    model = build_random_forest({})

    assert isinstance(model.n_estimators, int)
    assert model.n_estimators == 100
    assert model.max_depth is None
    assert model.random_state == 42


def test_build_random_forest_custom_configuration():
    model = build_random_forest(
        {
            "n_estimators": 25,
            "max_depth": 10,
        }
    )

    assert model.n_estimators == 25
    assert model.max_depth == 10
    assert model.random_state == 42


def test_train_random_forest_all_numeric():
    df = create_numeric_dataframe()

    model_results, y_test, predictions = train_random_forest(
        df=df,
        target="co2",
        features=[
            "temperature",
            "humidity",
            "pressure",
        ],
        configuration={
            "n_estimators": 10,
            "max_depth": 5,
        },
    )

    assert model_results["model"] == "random_forest_regressor"

    assert "r2" in model_results["evaluation"]
    assert "mse" in model_results["evaluation"]

    assert isinstance(
        model_results["evaluation"]["r2"],
        float,
    )

    assert isinstance(
        model_results["evaluation"]["mse"],
        float,
    )

    assert set(model_results["feature_importances"]) == {
        "temperature",
        "humidity",
        "pressure",
    }

    assert len(y_test) == len(predictions)
    assert len(model_results["actual_vs_predicted"]) == len(y_test)


def test_train_random_forest_mixed_features():
    df = create_mixed_dataframe()

    model_results, y_test, predictions = train_random_forest(
        df=df,
        target="co2",
        features=[
            "temperature",
            "city",
            "humidity",
        ],
        configuration={
            "n_estimators": 10,
            "max_depth": 5,
        },
    )

    assert model_results["model"] == "random_forest_regressor"

    assert set(model_results["feature_importances"]) == {
        "temperature",
        "city",
        "humidity",
    }

    assert len(y_test) == len(predictions)


def test_train_random_forest_handles_missing_feature_values():
    df = create_mixed_dataframe()

    df.loc[0, "temperature"] = np.nan
    df.loc[1, "humidity"] = np.nan
    df.loc[2, "city"] = None

    model_results, y_test, predictions = train_random_forest(
        df=df,
        target="co2",
        features=[
            "temperature",
            "city",
            "humidity",
        ],
        configuration={
            "n_estimators": 10,
            "max_depth": 5,
        },
    )

    assert model_results["model"] == "random_forest_regressor"
    assert len(y_test) == len(predictions)


def test_train_random_forest_drops_missing_target_rows():
    df = create_numeric_dataframe()

    df.loc[0, "co2"] = np.nan
    df.loc[1, "co2"] = np.nan

    model_results, y_test, predictions = train_random_forest(
        df=df,
        target="co2",
        features=[
            "temperature",
            "humidity",
            "pressure",
        ],
        configuration={
            "n_estimators": 10,
            "max_depth": 5,
        },
    )

    # There were originally 20 rows and two target rows were removed.
    # With test_size=0.2, 18 usable rows should remain, and 3 of those should be in the test set.
    assert len(y_test) == len(predictions)
    assert 0 < len(y_test) < 18

    assert model_results["model"] == "random_forest_regressor"


def test_train_random_forest_rejects_nonnumeric_target():
    df = pd.DataFrame(
        {
            "temperature": [20, 21, 22, 23],
            "category": ["low", "medium", "high", "medium"],
        }
    )

    with pytest.raises(
        ValueError,
        match="Target 'category' is not numeric",
    ):
        train_random_forest(
            df=df,
            target="category",
            features=["temperature"],
            configuration={
                "n_estimators": 10,
                "max_depth": 5,
            },
        )


def test_train_random_forest_rejects_insufficient_rows():
    df = pd.DataFrame(
        {
            "temperature": [20.0],
            "co2": [400.0],
        }
    )

    with pytest.raises(
        ValueError,
        match="Insufficient rows available for training",
    ):
        train_random_forest(
            df=df,
            target="co2",
            features=["temperature"],
            configuration={
                "n_estimators": 10,
                "max_depth": 5,
            },
        )


def test_train_random_forest_records_configuration():
    df = create_numeric_dataframe()

    configuration = {
        "n_estimators": 12,
        "max_depth": 8,
    }

    model_results, _, _ = train_random_forest(
        df=df,
        target="co2",
        features=["temperature", "humidity"],
        configuration=configuration,
    )

    assert model_results["configuration"] == configuration


def test_train_random_forest_records_features_and_target():
    df = create_numeric_dataframe()

    features = [
        "temperature",
        "humidity",
    ]

    model_results, _, _ = train_random_forest(
        df=df,
        target="co2",
        features=features,
        configuration={
            "n_estimators": 10,
            "max_depth": 5,
        },
    )

    assert model_results["features_used"] == features
    assert model_results["target"] == "co2"


def test_train_random_forest_actual_vs_predicted_structure():
    df = create_numeric_dataframe()

    model_results, _, _ = train_random_forest(
        df=df,
        target="co2",
        features=[
            "temperature",
            "humidity",
        ],
        configuration={
            "n_estimators": 10,
            "max_depth": 5,
        },
    )

    for result in model_results["actual_vs_predicted"]:
        assert "actual" in result
        assert "predicted" in result
        assert isinstance(result["actual"], float)
        assert isinstance(result["predicted"], float)


def test_aggregate_feature_importances_all_numeric():
    df = create_numeric_dataframe()

    X = df[
        [
            "temperature",
            "humidity",
            "pressure",
        ]
    ]

    preprocessor, categorical_columns, numeric_columns = (
        build_preprocessor(X)
    )

    model = (
        build_random_forest(
            {
                "n_estimators": 10,
                "max_depth": 5,
            }
        )
    )

    from sklearn.pipeline import Pipeline

    pipeline = Pipeline(
        [
            ("preprocessor", preprocessor),
            ("regressor", model),
        ]
    )

    pipeline.fit(X, df["co2"])

    importances = aggregate_feature_importances(
        pipeline,
        categorical_columns,
        numeric_columns,
    )

    assert set(importances) == {
        "temperature",
        "humidity",
        "pressure",
    }

    assert all(
        isinstance(value, float)
        for value in importances.values()
    )

    assert all(
        value >= 0
        for value in importances.values()
    )


def test_aggregate_feature_importances_mixed_features():
    df = create_mixed_dataframe()

    X = df[
        [
            "temperature",
            "city",
            "humidity",
        ]
    ]

    preprocessor, categorical_columns, numeric_columns = (
        build_preprocessor(X)
    )

    model = build_random_forest(
        {
            "n_estimators": 10,
            "max_depth": 5,
        }
    )

    from sklearn.pipeline import Pipeline

    pipeline = Pipeline(
        [
            ("preprocessor", preprocessor),
            ("regressor", model),
        ]
    )

    pipeline.fit(X, df["co2"])

    importances = aggregate_feature_importances(
        pipeline,
        categorical_columns,
        numeric_columns,
    )

    assert set(importances) == {
        "temperature",
        "city",
        "humidity",
    }

    assert all(
        isinstance(value, float)
        for value in importances.values()
    )