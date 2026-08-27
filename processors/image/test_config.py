import json

import pytest

from config import (
    SUPPORTED_OUTPUT_FORMATS,
    determine_output_format,
    get_output_extension,
    load_config,
    validate_config,
)


def valid_config():
    """Return a valid base image configuration."""
    return {
        "resize": False,
        "compression": False,
        "output_format": "original",
        "extract_metadata": False,
    }


def test_load_config(tmp_path):
    """Test that a valid JSON configuration is loaded."""
    config_path = tmp_path / "config.json"

    config = valid_config()

    config_path.write_text(
        json.dumps(config),
        encoding="utf-8",
    )

    result = load_config(str(config_path))

    assert result == config


def test_load_config_invalid_json(tmp_path):
    """Test that invalid JSON produces a ValueError."""
    config_path = tmp_path / "config.json"

    config_path.write_text(
        "{invalid json",
        encoding="utf-8",
    )

    with pytest.raises(
        ValueError,
        match="Error reading config file",
    ):
        load_config(str(config_path))


def test_load_config_missing_file(tmp_path):
    """Test that a missing configuration file produces a ValueError."""
    config_path = tmp_path / "missing.json"

    with pytest.raises(
        ValueError,
        match="Error reading config file",
    ):
        load_config(str(config_path))


def test_validate_config_valid():
    """Test that a valid configuration passes validation."""
    validate_config(valid_config())


@pytest.mark.parametrize(
    "missing_key",
    [
        "resize",
        "compression",
        "output_format",
        "extract_metadata",
    ],
)
def test_validate_config_missing_required_key(missing_key):
    """Test that required configuration keys cannot be omitted."""
    config = valid_config()
    del config[missing_key]

    with pytest.raises(
        ValueError,
        match=f"Missing required key '{missing_key}'",
    ):
        validate_config(config)


@pytest.mark.parametrize(
    "key",
    [
        "resize",
        "compression",
        "extract_metadata",
    ],
)
def test_validate_config_boolean_values(key):
    """Test that boolean configuration values must actually be booleans."""
    config = valid_config()
    config[key] = "true"

    with pytest.raises(
        ValueError,
        match=f"'{key}' must be a boolean",
    ):
        validate_config(config)


@pytest.mark.parametrize(
    "width,height",
    [
        (0, 100),
        (-1, 100),
        (100, 0),
        (100, -1),
        (None, 100),
        (100, None),
    ],
)
def test_validate_config_invalid_resize_dimensions(
    width,
    height,
):
    """Test that invalid resize dimensions are rejected."""
    config = valid_config()

    config["resize"] = True
    config["resize_width"] = width
    config["resize_height"] = height

    with pytest.raises(ValueError):
        validate_config(config)


def test_validate_config_missing_resize_width():
    """Test that resize width is required when resizing is enabled."""
    config = valid_config()

    config["resize"] = True
    config["resize_height"] = 100

    with pytest.raises(
        ValueError,
        match="resize_width",
    ):
        validate_config(config)


def test_validate_config_missing_resize_height():
    """Test that resize height is required when resizing is enabled."""
    config = valid_config()

    config["resize"] = True
    config["resize_width"] = 100

    with pytest.raises(
        ValueError,
        match="resize_height",
    ):
        validate_config(config)


@pytest.mark.parametrize(
    "quality",
    [
        0,
        -1,
        101,
        None,
        "85",
    ],
)
def test_validate_config_invalid_compression_quality(
    quality,
):
    """Test that invalid compression quality values are rejected."""
    config = valid_config()

    config["compression"] = True
    config["compression_quality"] = quality

    with pytest.raises(
        ValueError,
        match="compression_quality",
    ):
        validate_config(config)


def test_validate_config_missing_compression_quality():
    """Test that compression quality is required when compression is enabled."""
    config = valid_config()

    config["compression"] = True

    with pytest.raises(
        ValueError,
        match="compression_quality",
    ):
        validate_config(config)


@pytest.mark.parametrize(
    "output_format",
    [
        "",
        "bmp",
        "tiff",
        "avif",
        "invalid",
    ],
)
def test_validate_config_invalid_output_format(
    output_format,
):
    """Test that unsupported output formats are rejected."""
    config = valid_config()
    config["output_format"] = output_format

    with pytest.raises(
        ValueError,
        match="Unsupported 'output_format'",
    ):
        validate_config(config)


@pytest.mark.parametrize(
    "output_format",
    [
        "jpeg",
        "jpg",
        "png",
        "webp",
        "gif",
        "original",
    ],
)
def test_validate_config_supported_output_formats(
    output_format,
):
    """Test that supported output formats pass validation."""
    config = valid_config()
    config["output_format"] = output_format

    validate_config(config)


def test_supported_output_formats():
    """Test the supported Pillow format mappings."""
    assert SUPPORTED_OUTPUT_FORMATS["jpeg"] == "JPEG"
    assert SUPPORTED_OUTPUT_FORMATS["jpg"] == "JPEG"
    assert SUPPORTED_OUTPUT_FORMATS["png"] == "PNG"
    assert SUPPORTED_OUTPUT_FORMATS["webp"] == "WEBP"
    assert SUPPORTED_OUTPUT_FORMATS["gif"] == "GIF"


@pytest.mark.parametrize(
    "requested,original,expected",
    [
        ("original", "JPEG", "jpeg"),
        ("original", "jpeg", "jpeg"),
        ("original", "PNG", "png"),
        ("original", "WEBP", "webp"),
        ("jpeg", "png", "jpeg"),
        ("png", "jpeg", "png"),
        ("webp", "jpeg", "webp"),
        ("gif", "png", "gif"),
        ("jpg", "png", "jpeg"),
    ],
)
def test_determine_output_format(
    requested,
    original,
    expected,
):
    """Test final output-format determination."""
    assert (
        determine_output_format(
            requested,
            original,
        )
        == expected
    )


def test_determine_output_format_unsupported_format():
    """Test that unsupported output formats are rejected."""
    with pytest.raises(
        ValueError,
        match="Unsupported output format",
    ):
        determine_output_format(
            "bmp",
            "jpeg",
        )


@pytest.mark.parametrize(
    "output_format,expected",
    [
        ("jpeg", ".jpg"),
        ("png", ".png"),
        ("webp", ".webp"),
        ("gif", ".gif"),
    ],
)
def test_get_output_extension(
    output_format,
    expected,
):
    """Test output filename extensions."""
    assert (
        get_output_extension(output_format)
        == expected
    )


def test_get_output_extension_unsupported_format():
    """Test that unsupported output formats have no output extension."""
    with pytest.raises(
        ValueError,
        match="Unsupported output format",
    ):
        get_output_extension("bmp")