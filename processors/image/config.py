import json


SUPPORTED_OUTPUT_FORMATS = {
    "jpeg": "JPEG",
    "jpg": "JPEG",
    "png": "PNG",
    "webp": "WEBP",
    "gif": "GIF",
}


def load_config(config_path):
    """Load an image processing configuration."""
    try:
        with open(config_path, "r", encoding="utf-8") as file:
            return json.load(file)
    except Exception as error:
        raise ValueError(
            f"Error reading config file: {error}"
        ) from error


def validate_config(config):
    """Validate image processing configuration."""
    required_keys = [
        "resize",
        "compression",
        "output_format",
        "extract_metadata",
    ]

    for key in required_keys:
        if key not in config:
            raise ValueError(
                f"Missing required key '{key}' in config file."
            )

    if not isinstance(config["resize"], bool):
        raise ValueError(
            "'resize' must be a boolean."
        )

    if not isinstance(config["compression"], bool):
        raise ValueError(
            "'compression' must be a boolean."
        )

    if not isinstance(config["extract_metadata"], bool):
        raise ValueError(
            "'extract_metadata' must be a boolean."
        )

    if config["resize"]:
        resize_width = config.get("resize_width")
        resize_height = config.get("resize_height")

        if (
            not isinstance(resize_width, int)
            or resize_width <= 0
        ):
            raise ValueError(
                "Invalid or missing 'resize_width' in config file."
            )

        if (
            not isinstance(resize_height, int)
            or resize_height <= 0
        ):
            raise ValueError(
                "Invalid or missing 'resize_height' in config file."
            )

    if config["compression"]:
        compression_quality = config.get(
            "compression_quality"
        )

        if (
            not isinstance(compression_quality, int)
            or not 1 <= compression_quality <= 100
        ):
            raise ValueError(
                "Invalid or missing "
                "'compression_quality' in config file."
            )

    output_format = config["output_format"].lower()

    if (
        output_format != "original"
        and output_format not in SUPPORTED_OUTPUT_FORMATS
    ):
        supported_formats = [
            "jpeg",
            "png",
            "webp",
            "gif",
            "original",
        ]

        raise ValueError(
            f"Unsupported 'output_format': {output_format}. "
            f"Supported formats: {supported_formats}"
        )


def determine_output_format(
    requested_format,
    original_format,
):
    """Determine the final image format."""
    if requested_format == "original":
        output_format = original_format.lower()
    else:
        output_format = requested_format

    if output_format == "jpg":
        output_format = "jpeg"

    if output_format not in SUPPORTED_OUTPUT_FORMATS:
        raise ValueError(
            f"Unsupported output format: {output_format}"
        )

    return output_format


def get_output_extension(output_format):
    """Return the file extension for an output format."""
    extensions = {
        "jpeg": ".jpg",
        "png": ".png",
        "webp": ".webp",
        "gif": ".gif",
    }

    try:
        return extensions[output_format]
    except KeyError as error:
        raise ValueError(
            f"Unsupported output format: {output_format}"
        ) from error