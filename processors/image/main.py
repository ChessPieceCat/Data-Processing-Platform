import json
import shutil
import sys
from pathlib import Path


def main():
    if len(sys.argv) != 4:
        print(
            "Usage: main.py <input_path> <output_json> <config_path>",
            file=sys.stderr,
        )
        return 1

    input_path = Path(sys.argv[1])
    output_json = Path(sys.argv[2])
    config_path = Path(sys.argv[3])

    if not input_path.is_file():
        print(
            f"Input image not found: {input_path}",
            file=sys.stderr,
        )
        return 1

    if not config_path.is_file():
        print(
            f"Config file not found: {config_path}",
            file=sys.stderr,
        )
        return 1

    # Create the processed image by copying the input image.
    processed_path = output_json.with_name(
        f"processed{input_path.suffix}"
    )

    shutil.copy2(input_path, processed_path)

    # Dummy image-processing results.
    results = {
        "original_path": str(input_path),
        "processed_path": str(processed_path),
        "operations": [
            "dummy_processing",
        ],
        "original_format": input_path.suffix.lstrip(".").upper(),
        "original_width": 100,
        "original_height": 100,
        "result_format": input_path.suffix.lstrip(".").upper(),
        "result_width": 100,
        "result_height": 100,
        "compression": {
            "original_size": 1000,
            "result_size": 800,
            "compression_ratio": 0.2,
        },
        "metadata": {
            "dummy": "true",
        },
    }

    with output_json.open("w", encoding="utf-8") as file:
        json.dump(
            results,
            file,
            indent=2,
        )

    return 0


if __name__ == "__main__":
    sys.exit(main())