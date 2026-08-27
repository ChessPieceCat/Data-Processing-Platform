import argparse
import sys

from config import load_config, validate_config
from processing import process_image


def parse_arguments():
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description="Process an image."
    )

    parser.add_argument("input_image")
    parser.add_argument("output_json")
    parser.add_argument("config_json")

    return parser.parse_args()


def main():
    """Run the image-processing pipeline."""
    args = parse_arguments()

    try:
        config = load_config(args.config_json)
        validate_config(config)

        process_image(
            args.input_image,
            args.output_json,
            config,
        )

    except Exception as error:
        print(
            f"Error processing image: {error}",
            file=sys.stderr,
        )
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())