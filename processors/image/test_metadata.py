import json

from PIL import Image

from metadata import (
    build_metadata_summary,
    get_image_metadata,
    make_json_safe,
    save_metadata,
)


def test_make_json_safe_dict():
    """Test that dictionary values are recursively converted."""
    value = {
        1: "one",
        "nested": {
            2: "two",
        },
    }

    result = make_json_safe(value)

    assert result == {
        "1": "one",
        "nested": {
            "2": "two",
        },
    }


def test_make_json_safe_list_and_tuple():
    """Test that lists and tuples are recursively converted."""
    value = [
        (1, 2),
        ["three", "four"],
    ]

    result = make_json_safe(value)

    assert result == [
        [1, 2],
        ["three", "four"],
    ]


def test_make_json_safe_bytes():
    """Test that byte strings are converted to strings."""
    value = b"hello"

    result = make_json_safe(value)

    assert result == "hello"


def test_make_json_safe_invalid_utf8():
    """Test that invalid UTF-8 bytes are safely decoded."""
    value = b"hello\xffworld"

    result = make_json_safe(value)

    assert result == "hello\uFFFDworld"


def test_make_json_safe_primitive_values():
    """Test that JSON-compatible primitive values are unchanged."""
    values = [
        "text",
        123,
        1.5,
        True,
        False,
        None,
    ]

    for value in values:
        assert make_json_safe(value) == value


class CustomObject:
    """Object used to test fallback string conversion."""

    def __str__(self):
        return "custom object"


def test_make_json_safe_unknown_object():
    """Test that unsupported objects are converted to strings."""
    value = CustomObject()

    result = make_json_safe(value)

    assert result == "custom object"


def test_get_image_metadata():
    """Test that image metadata is extracted and normalized."""
    image = Image.new("RGB", (10, 20))

    image.info["software"] = "Test Software"
    image.info["dpi"] = (72, 72)
    image.info["raw"] = b"test"

    result = get_image_metadata(image)

    assert result["software"] == "Test Software"
    assert result["dpi"] == [72, 72]
    assert result["raw"] == "test"


def test_build_metadata_summary():
    """Test that useful displayable metadata is included."""
    image = Image.new("RGB", (10, 20))

    image.info["dpi"] = (72, 72)
    image.info["jfif"] = 258
    image.info["jfif_version"] = (1, 2)
    image.info["jfif_unit"] = 1
    image.info["jfif_density"] = (72, 72)
    image.info["software"] = "Test Software"
    image.info["comment"] = b"Test comment"

    result = build_metadata_summary(image)

    assert result["dpi"] == [72, 72]
    assert result["jfif"] == 258
    assert result["jfif_version"] == [1, 2]
    assert result["jfif_unit"] == 1
    assert result["jfif_density"] == [72, 72]
    assert result["software"] == "Test Software"
    assert result["comment"] == "Test comment"


def test_build_metadata_summary_ignores_unlisted_metadata():
    """Test that non-summary metadata is omitted."""
    image = Image.new("RGB", (10, 20))

    image.info["software"] = "Test Software"
    image.info["unimportant"] = "hidden"

    result = build_metadata_summary(image)

    assert result["software"] == "Test Software"
    assert "unimportant" not in result


def test_build_metadata_summary_without_metadata():
    """Test that an image without relevant metadata produces an empty summary."""
    image = Image.new("RGB", (10, 20))

    result = build_metadata_summary(image)

    assert result == {}


def test_build_metadata_summary_exif():
    """Test that supported EXIF tags are included in the summary."""
    image = Image.new("RGB", (10, 20))

    exif = image.getexif()

    exif[270] = "Test description"
    exif[271] = "Test Make"
    exif[272] = "Test Model"
    exif[305] = "Test Software"
    exif[306] = "2026:08:27 12:00:00"

    result = build_metadata_summary(image)

    assert result["exif"]["image_description"] == (
        "Test description"
    )
    assert result["exif"]["make"] == "Test Make"
    assert result["exif"]["model"] == "Test Model"
    assert result["exif"]["software"] == "Test Software"
    assert result["exif"]["date_time"] == (
        "2026:08:27 12:00:00"
    )


def test_build_metadata_summary_ignores_unsupported_exif_tags():
    """Test that unsupported EXIF tags are omitted from the summary."""
    image = Image.new("RGB", (10, 20))

    exif = image.getexif()

    exif[270] = "Test description"
    exif[300] = "Unsupported value"

    result = build_metadata_summary(image)

    assert "image_description" in result["exif"]
    assert "300" not in result["exif"]


def test_save_metadata(tmp_path):
    """Test that complete metadata is written as JSON."""
    metadata_path = tmp_path / "metadata.json"

    metadata = {
        "software": "Test Software",
        "dpi": [72, 72],
        "nested": {
            "value": "test",
        },
    }

    save_metadata(
        metadata,
        metadata_path,
    )

    assert metadata_path.exists()

    with metadata_path.open(
        "r",
        encoding="utf-8",
    ) as file:
        result = json.load(file)

    assert result == metadata