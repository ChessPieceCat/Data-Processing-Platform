import json

import pandas as pd
import pytest

from config import load_config, validate_model_config


def create_numeric_dataframe():
    return pd.DataFrame(
        {
            "temperature": [20.0, 21.0, 22.0, 23.0],
            "humidity": [40.0, 45.0, 50.0, 55.0],
            "co2": [400.0, 410.0, 420.0, 430.0],
        }
    )


def create_mixed_dataframe():
    return pd.DataFrame(
        {
            "temperature": [20.0, 21.0, 22.0, 23.0],
            "city": ["Chicago", "Denver", "Chicago", "Denver"],
            "co2": [400.0, 410.0, 420.0, 430.0],
        }
    )


def test_load_config(tmp_path):
    config_path = tmp_path / "config.json"

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "automatic",
        "configuration_type": "automatic",
    }

    with open(config_path, "w", encoding="utf-8") as file:
        json.dump(config, file)

    result = load_config(config_path)

    assert result == config


def test_load_config_invalid_json(tmp_path):
    config_path = tmp_path / "config.json"

    config_path.write_text(
        '{"model": "random_forest_regressor",',
        encoding="utf-8",
    )

    with pytest.raises(ValueError, match="Error reading config file"):
        load_config(config_path)


def test_load_config_missing_file(tmp_path):
    config_path = tmp_path / "missing.json"

    with pytest.raises(ValueError, match="Error reading config file"):
        load_config(config_path)


def test_validate_model_config_none():
    df = create_numeric_dataframe()

    config = {
        "model": "none",
    }

    result = validate_model_config(config, df)

    assert result == {
        "model": "none",
        "target": None,
        "features": [],
        "configuration": {},
    }


def test_validate_model_config_automatic_features():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "automatic",
        "configuration_type": "automatic",
    }

    result = validate_model_config(config, df)

    assert result["model"] == "random_forest_regressor"
    assert result["target"] == "co2"
    assert result["features"] == [
        "temperature",
        "humidity",
    ]
    assert result["configuration"] == {
        "n_estimators": 100,
        "max_depth": None,
    }


def test_validate_model_config_manual_features():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": [
            "temperature",
            "humidity",
        ],
        "configuration_type": "manual",
        "configuration": {
            "n_estimators": 25,
            "max_depth": 10,
        },
    }

    result = validate_model_config(config, df)

    assert result["features"] == [
        "temperature",
        "humidity",
    ]

    assert result["configuration"] == {
        "n_estimators": 25,
        "max_depth": 10,
    }


def test_validate_model_config_accepts_mixed_features():
    df = create_mixed_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": [
            "temperature",
            "city",
        ],
        "configuration_type": "manual",
        "configuration": {
            "n_estimators": 25,
            "max_depth": 10,
        },
    }

    result = validate_model_config(config, df)

    assert result["features"] == [
        "temperature",
        "city",
    ]


def test_validate_model_config_missing_required_key():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "automatic",
        # configuration_type intentionally missing
    }

    with pytest.raises(
        ValueError,
        match="Missing required key 'configuration_type'",
    ):
        validate_model_config(config, df)


def test_validate_model_config_invalid_target():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "does_not_exist",
        "feature_selection": "automatic",
        "configuration_type": "automatic",
    }

    with pytest.raises(
        ValueError,
        match="Target 'does_not_exist' not found in the dataset",
    ):
        validate_model_config(config, df)


@pytest.mark.parametrize("feature_selection", ["invalid", "manual123", ""])
def test_validate_model_config_invalid_feature_selection(
    feature_selection,
):
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": feature_selection,
        "configuration_type": "automatic",
    }

    with pytest.raises(
        ValueError,
        match="feature_selection must be either",
    ):
        validate_model_config(config, df)


def test_validate_model_config_manual_feature_does_not_exist():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": ["does_not_exist"],
        "configuration_type": "manual",
        "configuration": {
            "n_estimators": 25,
            "max_depth": 10,
        },
    }

    with pytest.raises(
        ValueError,
        match="Feature 'does_not_exist' not found in the dataset",
    ):
        validate_model_config(config, df)


def test_validate_model_config_empty_manual_features():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": [],
        "configuration_type": "manual",
        "configuration": {
            "n_estimators": 25,
            "max_depth": 10,
        },
    }

    with pytest.raises(
        ValueError,
        match="Features list is empty",
    ):
        validate_model_config(config, df)


def test_validate_model_config_target_in_features():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": [
            "temperature",
            "co2",
        ],
        "configuration_type": "manual",
        "configuration": {
            "n_estimators": 25,
            "max_depth": 10,
        },
    }

    with pytest.raises(
        ValueError,
        match="Target 'co2' should not be included in features",
    ):
        validate_model_config(config, df)


def test_validate_model_config_no_automatic_features():
    df = pd.DataFrame(
        {
            "co2": [400.0, 410.0, 420.0],
        }
    )

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "automatic",
        "configuration_type": "automatic",
    }

    with pytest.raises(
        ValueError,
        match="No features available for automatic feature selection",
    ):
        validate_model_config(config, df)


@pytest.mark.parametrize("configuration_type", ["invalid", "", "manual123"])
def test_validate_model_config_invalid_configuration_type(
    configuration_type,
):
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "automatic",
        "configuration_type": configuration_type,
    }

    with pytest.raises(
        ValueError,
        match="configuration_type must be either",
    ):
        validate_model_config(config, df)


def test_validate_model_config_manual_configuration_must_be_dict():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "automatic",
        "configuration_type": "manual",
        "configuration": [],
    }

    with pytest.raises(
        ValueError,
        match="Configuration should be a dictionary",
    ):
        validate_model_config(config, df)


def test_validate_model_config_unsupported_model():
    df = create_numeric_dataframe()

    config = {
        "model": "unsupported_model",
        "target": "co2",
        "feature_selection": "automatic",
        "configuration_type": "automatic",
    }

    with pytest.raises(
        ValueError,
        match="Model 'unsupported_model' is not supported",
    ):
        validate_model_config(config, df)


def test_validate_model_config_nonnumeric_target():
    df = pd.DataFrame(
        {
            "temperature": [20.0, 21.0, 22.0],
            "co2_label": ["low", "medium", "high"],
        }
    )

    config = {
        "model": "random_forest_regressor",
        "target": "co2_label",
        "feature_selection": "automatic",
        "configuration_type": "automatic",
    }

    with pytest.raises(
        ValueError,
        match="Target 'co2_label' is not numeric",
    ):
        validate_model_config(config, df)


def test_validate_model_config_manual_missing_n_estimators():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": ["temperature"],
        "configuration_type": "manual",
        "configuration": {
            "max_depth": 10,
        },
    }

    with pytest.raises(
        ValueError,
        match="Missing 'n_estimators' or 'max_depth'",
    ):
        validate_model_config(config, df)


def test_validate_model_config_manual_missing_max_depth():
    df = create_numeric_dataframe()

    config = {
        "model": "random_forest_regressor",
        "target": "co2",
        "feature_selection": "manual",
        "features": ["temperature"],
        "configuration_type": "manual",
        "configuration": {
            "n_estimators": 25,
        },
    }

    with pytest.raises(
        ValueError,
        match="Missing 'n_estimators' or 'max_depth'",
    ):
        validate_model_config(config, df)


def test_validate_model_config_case_insensitive_values():
    df = create_numeric_dataframe()

    config = {
        "model": "RANDOM_FOREST_REGRESSOR",
        "target": "co2",
        "feature_selection": "AUTOMATIC",
        "configuration_type": "AUTOMATIC",
    }

    result = validate_model_config(config, df)

    assert result["model"] == "random_forest_regressor"
    assert result["target"] == "co2"
    assert result["features"] == [
        "temperature",
        "humidity",
    ]