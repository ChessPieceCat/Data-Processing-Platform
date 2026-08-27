import argparse
import json
import os
import sys
from pathlib import Path

from PIL import Image


SUPPORTED_OUTPUT_FORMATS = {
    "jpeg": "JPEG",
    "jpg": "JPEG",
    "png": "PNG",
    "webp": "WEBP",
    "gif": "GIF",
}


def parse_arguments():
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description="Process an image."
    )

    parser.add_argument("input_image")
    parser.add_argument("output_json")
    parser.add_argument("config_json")

    return parser.parse_args()


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


def make_json_safe(value):
    """Convert metadata values into JSON-safe values."""
    if isinstance(value, dict):
        return {
            str(key): make_json_safe(item)
            for key, item in value.items()
        }

    if isinstance(value, (list, tuple)):
        return [
            make_json_safe(item)
            for item in value
        ]

    if isinstance(value, bytes):
        return value.decode(
            "utf-8",
            errors="replace",
        )

    if isinstance(value, (str, int, float, bool)) or value is None:
        return value

    return str(value)


def get_image_metadata(image):
    """Extract and normalize the complete image metadata."""
    return make_json_safe(image.info)


def build_metadata_summary(image):
    """
    Build a small human-readable metadata summary.

    Only useful, commonly displayable values are included here.
    Complete metadata is saved separately.
    """
    summary = {}

    info = image.info

    for key in (
        "dpi",
        "jfif",
        "jfif_version",
        "jfif_unit",
        "jfif_density",
        "software",
        "comment",
    ):
        if key in info:
            summary[key] = make_json_safe(info[key])

    exif = image.getexif()

    if exif:
        exif_summary = {}

        exif_names = {
            270: "image_description",
            271: "make",
            272: "model",
            305: "software",
            306: "date_time",
        }

        for tag_id, name in exif_names.items():
            if tag_id in exif:
                exif_summary[name] = make_json_safe(
                    exif[tag_id]
                )

        if exif_summary:
            summary["exif"] = exif_summary

    return summary


def get_image_info(
    image_path,
    extract_metadata=False,
):
    """Get information about an image."""
    try:
        file_size = os.path.getsize(image_path)

        with Image.open(image_path) as image:
            info = {
                "format": image.format,
                "width": image.width,
                "height": image.height,
                "file_size": file_size,
            }

            if extract_metadata:
                info["metadata"] = get_image_metadata(
                    image
                )

                info["metadata_summary"] = (
                    build_metadata_summary(image)
                )

            return info

    except Exception as error:
        raise ValueError(
            f"Error reading image file: {error}"
        ) from error


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


def prepare_image_for_format(
    image,
    output_format,
):
    """Prepare an image for saving in the requested format."""
    if output_format in ("jpeg", "webp"):
        if image.mode not in ("RGB", "L"):
            return image.convert("RGB")

    if output_format == "png":
        return image

    if output_format == "gif":
        return image.convert("P")

    return image


def build_save_kwargs(
    output_format,
    config,
):
    """Build Pillow save options for the output format."""
    save_kwargs = {}

    if output_format in ("jpeg", "webp"):
        if config["compression"]:
            save_kwargs["quality"] = (
                config["compression_quality"]
            )

    elif output_format == "png":
        if config["compression"]:
            save_kwargs["optimize"] = True

    return save_kwargs


def save_metadata(
    metadata,
    metadata_path,
):
    """Save complete image metadata to a separate JSON file."""
    with open(
        metadata_path,
        "w",
        encoding="utf-8",
    ) as file:
        json.dump(
            metadata,
            file,
            indent=2,
        )


def process_image(
    input_image_path,
    output_json_path,
    config,
):
    """Process an image according to the configuration."""

    requested_format = config["output_format"].lower()

    original_info = get_image_info(
        input_image_path,
        extract_metadata=config["extract_metadata"],
    )

    original_format = original_info["format"].lower()

    output_format = determine_output_format(
        requested_format,
        original_format,
    )

    output_extension = get_output_extension(
        output_format
    )

    output_json = Path(output_json_path)
    output_image_path = output_json.with_name(
        f"processed{output_extension}"
    )

    metadata_path = output_json.with_name(
        "metadata.json"
    )

    operations = []

    with Image.open(input_image_path) as image:
        current_image = image.copy()

    if config["resize"]:
        current_image = current_image.resize(
            (
                config["resize_width"],
                config["resize_height"],
            )
        )

        operations.append("resize")

    if output_format != original_format:
        operations.append("format_conversion")

    current_image = prepare_image_for_format(
        current_image,
        output_format,
    )

    if config["compression"]:
        operations.append("compression")

    save_kwargs = build_save_kwargs(
        output_format,
        config,
    )

    current_image.save(
        output_image_path,
        format=SUPPORTED_OUTPUT_FORMATS[
            output_format
        ],
        **save_kwargs,
    )

    result_info = {
        "original_path": input_image_path,
        "processed_path": str(output_image_path),
        "operations": operations,
        "original_format": original_info["format"],
        "original_width": original_info["width"],
        "original_height": original_info["height"],
        "result_format": output_format.upper(),
        "result_width": current_image.width,
        "result_height": current_image.height,
    }

    if config["compression"]:
        original_size = original_info["file_size"]
        result_size = os.path.getsize(
            output_image_path
        )

        compression_ratio = (
            (original_size - result_size)
            / original_size
            if original_size > 0
            else 0
        )

        result_info["compression"] = {
            "original_size": original_size,
            "result_size": result_size,
            "compression_ratio": compression_ratio,
        }

    if config["extract_metadata"]:
        metadata = original_info.get(
            "metadata",
            {},
        )

        result_info["metadata"] = original_info.get(
            "metadata_summary",
            {},
        )

        result_info["metadata_reference"] = str(
            metadata_path
        )

        save_metadata(
            metadata,
            metadata_path,
        )

    with open(
        output_json_path,
        "w",
        encoding="utf-8",
    ) as file:
        json.dump(
            result_info,
            file,
            indent=2,
        )


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