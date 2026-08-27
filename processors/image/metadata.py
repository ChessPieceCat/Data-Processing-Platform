import json


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