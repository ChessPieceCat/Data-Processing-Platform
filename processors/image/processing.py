import json
import os
from pathlib import Path

from PIL import Image

from config import (
    SUPPORTED_OUTPUT_FORMATS,
    determine_output_format,
    get_output_extension,
)
from metadata import (
    build_metadata_summary,
    get_image_metadata,
    save_metadata,
)


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
                info["metadata"] = get_image_metadata(image)
                info["metadata_summary"] = (
                    build_metadata_summary(image)
                )

            return info

    except Exception as error:
        raise ValueError(
            f"Error reading image file: {error}"
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