import argparse
import sys

import pandas as pd

from analysis import analyze_dataset
from config import load_config, validate_model_config
from io_utils import save_json
from modeling import train_random_forest
from visualizations import (
    generate_actual_vs_predicted,
    generate_dataset_visualizations,
)


def parse_arguments():
    parser = argparse.ArgumentParser(
        description="Process and analyze a dataset."
    )

    parser.add_argument("input_csv")
    parser.add_argument("output_json")
    parser.add_argument("config_json")

    return parser.parse_args()


def main():
    args = parse_arguments()

    try:
        df = pd.read_csv(args.input_csv)
    except Exception as error:
        print(f"Error reading CSV file: {error}", file=sys.stderr)
        return 1

    try:
        config = load_config(args.config_json)

        # Validate and prepare model configuration before generating
        # any model-related artifacts.
        model_config = validate_model_config(config, df)

        result = analyze_dataset(df)

        visualization_config = config.get("visualizations", {})

        visualizations = generate_dataset_visualizations(
            df,
            visualization_config,
            args.output_json,
        )

        result["visualizations"] = visualizations

        if model_config["model"] == "random_forest_regressor":
            model_results, y_test, predictions = train_random_forest(
                df=df,
                target=model_config["target"],
                features=model_config["features"],
                configuration=model_config["configuration"],
            )

            if visualization_config.get("actual_vs_predicted", False):
                plot_path = generate_actual_vs_predicted(
                    y_test,
                    predictions,
                    args.output_json,
                )

                result["visualizations"]["actual_vs_predicted"] = plot_path

            model_results_path = args.output_json.replace(
                ".json",
                "_model_results.json",
            )

            save_json(model_results, model_results_path)

        save_json(result, args.output_json)

    except Exception as error:
        print(f"Error processing dataset: {error}", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())