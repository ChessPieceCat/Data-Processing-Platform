import pandas as pd

from sklearn.ensemble import RandomForestRegressor
from sklearn.impute import SimpleImputer
from sklearn.metrics import mean_squared_error
from sklearn.model_selection import train_test_split
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import OneHotEncoder
from sklearn.compose import ColumnTransformer


def build_preprocessor(X):
    """Build preprocessing pipelines for categorical and numeric features."""

    categorical_columns = X.select_dtypes(
        include=["object", "string"]
    ).columns

    numeric_columns = X.select_dtypes(
        include=["number"]
    ).columns

    categorical_pipeline = Pipeline([
        (
            "imputer",
            SimpleImputer(strategy="most_frequent"),
        ),
        (
            "onehot",
            OneHotEncoder(handle_unknown="ignore"),
        ),
    ])

    numeric_pipeline = Pipeline([
        (
            "imputer",
            SimpleImputer(strategy="median"),
        ),
    ])

    preprocessor = ColumnTransformer([
        (
            "categorical",
            categorical_pipeline,
            categorical_columns.tolist(),
        ),
        (
            "numeric",
            numeric_pipeline,
            numeric_columns.tolist(),
        ),
    ])

    return (
        preprocessor,
        categorical_columns,
        numeric_columns,
    )


def build_random_forest(configuration):
    """Create the Random Forest Regressor."""
    return RandomForestRegressor(
        n_estimators=configuration.get(
            "n_estimators",
            100,
        ),
        max_depth=configuration.get(
            "max_depth",
            None,
        ),
        random_state=42,
    )


def aggregate_feature_importances(
    model,
    categorical_columns,
    numeric_columns,
):
    """Aggregate one-hot feature importance back to original columns."""

    importances = model.named_steps[
        "regressor"
    ].feature_importances_

    importance_index = 0
    feature_importance_dict = {}

    if len(categorical_columns) > 0:
        categorical_encoder = (
            model.named_steps["preprocessor"]
            .named_transformers_["categorical"]
            .named_steps["onehot"]
        )

        for column, categories in zip(
            categorical_columns,
            categorical_encoder.categories_,
        ):
            num_encoded_features = len(categories)

            feature_importance_dict[column] = float(
                sum(
                    importances[
                        importance_index:
                        importance_index + num_encoded_features
                    ]
                )
            )

            importance_index += num_encoded_features

    for column in numeric_columns:
        feature_importance_dict[column] = float(
            importances[importance_index]
        )

        importance_index += 1

    return feature_importance_dict


def train_random_forest(
    df,
    target,
    features,
    configuration,
):
    """Train and evaluate a Random Forest regression model."""

    if not pd.api.types.is_numeric_dtype(df[target]):
        raise ValueError(
            f"Target '{target}' is not numeric."
        )

    df = df[df[target].notna()]

    if len(df) < 2:
        raise ValueError(
            "Insufficient rows available for training."
        )

    X = df[features]
    y = df[target]

    (
        preprocessor,
        categorical_columns,
        numeric_columns,
    ) = build_preprocessor(X)

    model = Pipeline([
        (
            "preprocessor",
            preprocessor,
        ),
        (
            "regressor",
            build_random_forest(configuration),
        ),
    ])

    X_train, X_test, y_train, y_test = train_test_split(
        X,
        y,
        test_size=0.2,
        random_state=42,
    )

    model.fit(X_train, y_train)

    predictions = model.predict(X_test)

    r2 = model.score(X_test, y_test)
    mse = mean_squared_error(
        y_test,
        predictions,
    )

    feature_importances = aggregate_feature_importances(
        model,
        categorical_columns,
        numeric_columns,
    )

    actual_vs_predicted = [
        {
            "actual": actual,
            "predicted": predicted,
        }
        for actual, predicted in zip(
            y_test.tolist(),
            predictions.tolist(),
        )
    ]

    model_results = {
        "model": "random_forest_regressor",
        "evaluation": {
            "r2": r2,
            "mse": mse,
        },
        "feature_importances": feature_importances,
        "actual_vs_predicted": actual_vs_predicted,
        "features_used": features,
        "target": target,
        "configuration": configuration,
    }

    return (
        model_results,
        y_test,
        predictions,
    )