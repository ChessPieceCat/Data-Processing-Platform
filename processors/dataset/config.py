import json
import pandas as pd

def load_config(config_path):
    """Load a JSON processing configuration."""
    try:
        with open(config_path, "r", encoding="utf-8") as file:
            return json.load(file)
    except Exception as error:
        raise ValueError(
            f"Error reading config file: {error}"
        ) from error


def validate_model_config(config, df):
    """Validate and normalize model configuration."""

    model = config.get("model", "none").lower()

    if model == "none":
        return {
            "model": "none",
            "target": None,
            "features": [],
            "configuration": {},
        }

    required_keys = [
        "model",
        "target",
        "feature_selection",
        "configuration_type",
    ]

    for key in required_keys:
        if key not in config:
            raise ValueError(
                f"Missing required key '{key}' in config file."
            )

    target = config.get("target", "")

    if target not in df.columns:
        raise ValueError(
            f"Target '{target}' not found in the dataset."
        )

    feature_selection = config.get(
        "feature_selection",
        "none",
    ).lower()

    if feature_selection not in ["manual", "automatic"]:
        raise ValueError(
            'feature_selection must be either "manual" or "automatic".'
        )

    if feature_selection == "manual":
        features = config.get("features", [])

        for feature in features:
            if feature not in df.columns:
                raise ValueError(
                    f"Feature '{feature}' not found in the dataset."
                )

        if not features:
            raise ValueError(
                "Features list is empty for manual feature selection."
            )

    else:
        features = [
            column
            for column in df.columns
            if column != target
        ]

        if not features:
            raise ValueError(
                "No features available for automatic feature selection."
            )

    if target in features:
        raise ValueError(
            f"Target '{target}' should not be included in features."
        )

    configuration_type = config.get(
        "configuration_type",
        "none",
    ).lower()

    if configuration_type not in ["manual", "automatic"]:
        raise ValueError(
            'configuration_type must be either "manual" or "automatic".'
        )

    if configuration_type == "manual":
        configuration = config.get("configuration", {})

        if not isinstance(configuration, dict):
            raise ValueError(
                "Configuration should be a dictionary."
            )

    else:
        configuration = {
            "n_estimators": 100,
            "max_depth": None,
        }

    if model != "random_forest_regressor":
        raise ValueError(
            f"Model '{config.get('model')}' is not supported."
        )

    if not pd.api.types.is_numeric_dtype(df[target]):
        raise ValueError(
            f"Target '{target}' is not numeric."
        )

    if configuration_type == "manual":
        if (
            "n_estimators" not in configuration
            or "max_depth" not in configuration
        ):
            raise ValueError(
                "Missing 'n_estimators' or 'max_depth' "
                "in configuration for random_forest_regressor."
            )

    return {
        "model": model,
        "target": target,
        "features": features,
        "configuration": configuration,
    }