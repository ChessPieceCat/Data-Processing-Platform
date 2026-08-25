import json

import numpy as np


def json_serializer(value):
    """Convert NumPy scalar values into native Python values."""
    if isinstance(value, np.generic):
        return value.item()

    raise TypeError(
        f"Object of type {type(value).__name__} "
        "is not JSON serializable"
    )


def save_json(data, output_path):
    """Write a Python object to a formatted JSON file."""
    try:
        with open(output_path, "w", encoding="utf-8") as file:
            json.dump(
                data,
                file,
                indent=2,
                default=json_serializer,
            )
    except Exception as error:
        raise ValueError(
            f"Error writing JSON file: {error}"
        ) from error