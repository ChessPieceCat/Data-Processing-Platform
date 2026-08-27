import json
from pathlib import Path

import pytest
from PIL import Image

from processing import (
    build_save_kwargs,
    get_image_info,
    prepare_image_for_format,
    process_image,
)


def create_test_image(
    path,
    image_format="PNG",
    size=(10, 20),
    mode="RGB",
):
    """Create a simple valid image for testing."""
    image = Image.new(
        mode,
        size,
        color=(255, 0, 0),
    )
    image.save(
        path,
        format=image_format,
    )


def load_json(path):
    """Load JSON data from a file."""
    with open(
        path,
        "r",
        encoding="utf-8",
    ) as file:
        return json.load(file)


def base_config():
    """Return a valid base image-processing configuration."""
    return {
        "resize": False,
        "compression": False,
        "output_format": "original",
        "extract_metadata": False,
    }


def test_get_image_info(tmp_path):
    """Test that basic image information is returned."""
    image_path = tmp_path / "test.png"

    create_test_image(
        image_path,
        image_format="PNG",
        size=(10, 20),
    )

    info = get_image_info(str(image_path))

    assert info["format"] == "PNG"
    assert info["width"] == 10
    assert info["height"] == 20
    assert info["file_size"] > 0
    assert "metadata" not in info
    assert "metadata_summary" not in info


def test_get_image_info_with_metadata(tmp_path):
    """Test that metadata is included when requested."""
    image_path = tmp_path / "test.png"

    create_test_image(
        image_path,
        image_format="PNG",
    )

    info = get_image_info(
        str(image_path),
        extract_metadata=True,
    )

    assert "metadata" in info
    assert "metadata_summary" in info
    assert isinstance(
        info["metadata"],
        dict,
    )
    assert isinstance(
        info["metadata_summary"],
        dict,
    )


def test_get_image_info_missing_file(tmp_path):
    """Test that a missing image raises a ValueError."""
    image_path = tmp_path / "missing.png"

    with pytest.raises(
        ValueError,
        match="Error reading image file",
    ):
        get_image_info(str(image_path))


def test_get_image_info_invalid_image(tmp_path):
    """Test that an invalid image raises a ValueError."""
    image_path = tmp_path / "invalid.png"

    image_path.write_bytes(
        b"not an actual image"
    )

    with pytest.raises(
        ValueError,
        match="Error reading image file",
    ):
        get_image_info(str(image_path))


@pytest.mark.parametrize(
    "output_format,mode,expected_mode",
    [
        ("jpeg", "RGBA", "RGB"),
        ("jpeg", "RGB", "RGB"),
        ("jpeg", "L", "L"),
        ("webp", "RGBA", "RGB"),
        ("webp", "RGB", "RGB"),
        ("png", "RGBA", "RGBA"),
        ("gif", "RGB", "P"),
    ],
)
def test_prepare_image_for_format(
    output_format,
    mode,
    expected_mode,
):
    """Test image mode preparation for each output format."""
    image = Image.new(
        mode,
        (10, 10),
        color=(255, 0, 0)
        if mode != "L"
        else 255,
    )

    result = prepare_image_for_format(
        image,
        output_format,
    )

    assert result.mode == expected_mode


def test_prepare_image_for_format_unknown_format():
    """Test that unknown formats leave the image unchanged."""
    image = Image.new(
        "RGB",
        (10, 10),
    )

    result = prepare_image_for_format(
        image,
        "unknown",
    )

    assert result is image


def test_build_save_kwargs_jpeg_with_compression():
    """Test JPEG quality settings when compression is enabled."""
    config = {
        "compression": True,
        "compression_quality": 85,
    }

    result = build_save_kwargs(
        "jpeg",
        config,
    )

    assert result == {
        "quality": 85,
    }


def test_build_save_kwargs_webp_with_compression():
    """Test WebP quality settings when compression is enabled."""
    config = {
        "compression": True,
        "compression_quality": 70,
    }

    result = build_save_kwargs(
        "webp",
        config,
    )

    assert result == {
        "quality": 70,
    }


def test_build_save_kwargs_png_with_compression():
    """Test PNG optimization when compression is enabled."""
    config = {
        "compression": True,
        "compression_quality": 50,
    }

    result = build_save_kwargs(
        "png",
        config,
    )

    assert result == {
        "optimize": True,
    }


@pytest.mark.parametrize(
    "output_format",
    [
        "jpeg",
        "webp",
        "png",
    ],
)
def test_build_save_kwargs_without_compression(
    output_format,
):
    """Test that compression options are omitted when disabled."""
    config = {
        "compression": False,
        "compression_quality": 85,
    }

    result = build_save_kwargs(
        output_format,
        config,
    )

    assert result == {}


def test_process_image_no_operations(tmp_path):
    """Test processing with all optional operations disabled."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
        size=(10, 20),
    )

    config = base_config()

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    assert output_json.exists()

    output_image = tmp_path / "processed.png"

    assert output_image.exists()

    result = load_json(output_json)

    assert result["original_path"] == str(input_path)
    assert result["processed_path"] == str(
        output_image
    )
    assert result["operations"] == []
    assert result["original_format"] == "PNG"
    assert result["original_width"] == 10
    assert result["original_height"] == 20
    assert result["result_format"] == "PNG"
    assert result["result_width"] == 10
    assert result["result_height"] == 20

    with Image.open(output_image) as image:
        assert image.format == "PNG"
        assert image.size == (10, 20)


def test_process_image_resize(tmp_path):
    """Test image resizing."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
        size=(100, 200),
    )

    config = base_config()
    config.update(
        {
            "resize": True,
            "resize_width": 25,
            "resize_height": 30,
        }
    )

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    result = load_json(output_json)

    assert "resize" in result["operations"]
    assert result["result_width"] == 25
    assert result["result_height"] == 30

    output_image = tmp_path / "processed.png"

    with Image.open(output_image) as image:
        assert image.size == (25, 30)


@pytest.mark.parametrize(
    "output_format,expected_extension,expected_format",
    [
        ("jpeg", ".jpg", "JPEG"),
        ("webp", ".webp", "WEBP"),
        ("gif", ".gif", "GIF"),
    ],
)
def test_process_image_format_conversion(
    tmp_path,
    output_format,
    expected_extension,
    expected_format,
):
    """Test conversion to supported output formats."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
        size=(10, 20),
    )

    config = base_config()
    config["output_format"] = output_format

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    output_image = (
        tmp_path / f"processed{expected_extension}"
    )

    assert output_image.exists()

    result = load_json(output_json)

    assert "format_conversion" in result["operations"]
    assert result["result_format"] == expected_format

    with Image.open(output_image) as image:
        assert image.format == expected_format


def test_process_image_original_format(tmp_path):
    """Test that original format is preserved."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
    )

    config = base_config()
    config["output_format"] = "original"

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    output_image = tmp_path / "processed.png"

    assert output_image.exists()

    result = load_json(output_json)

    assert "format_conversion" not in result[
        "operations"
    ]
    assert result["result_format"] == "PNG"

    with Image.open(output_image) as image:
        assert image.format == "PNG"


@pytest.mark.parametrize(
    "output_format",
    [
        "jpeg",
        "webp",
    ],
)
def test_process_image_compression(
    tmp_path,
    output_format,
):
    """Test compression for formats that support quality."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
        size=(100, 100),
    )

    config = base_config()
    config.update(
        {
            "compression": True,
            "compression_quality": 50,
            "output_format": output_format,
        }
    )

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    result = load_json(output_json)

    assert "compression" in result["operations"]
    assert "compression" in result

    compression = result["compression"]

    assert compression["original_size"] > 0
    assert compression["result_size"] > 0
    assert isinstance(
        compression["compression_ratio"],
        float,
    )


def test_process_image_png_compression(tmp_path):
    """Test PNG optimization when compression is enabled."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
        size=(100, 100),
    )

    config = base_config()
    config.update(
        {
            "compression": True,
            "compression_quality": 85,
            "output_format": "png",
        }
    )

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    result = load_json(output_json)

    assert "compression" in result["operations"]
    assert "compression" in result

    output_image = tmp_path / "processed.png"

    assert output_image.exists()

    with Image.open(output_image) as image:
        assert image.format == "PNG"


def test_process_image_metadata(tmp_path):
    """Test metadata extraction and metadata artifact creation."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
    )

    config = base_config()
    config["extract_metadata"] = True

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    result = load_json(output_json)

    assert "metadata" in result
    assert "metadata_reference" in result

    metadata_path = Path(
        result["metadata_reference"]
    )

    assert metadata_path.exists()

    metadata = load_json(metadata_path)

    assert isinstance(metadata, dict)

    output_image = tmp_path / "processed.png"

    assert output_image.exists()


def test_process_image_all_operations(tmp_path):
    """Test resize, format conversion, compression, and metadata together."""
    input_path = tmp_path / "input.png"
    output_json = tmp_path / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
        size=(100, 200),
    )

    config = {
        "resize": True,
        "resize_width": 25,
        "resize_height": 30,
        "compression": True,
        "compression_quality": 80,
        "output_format": "webp",
        "extract_metadata": True,
    }

    process_image(
        str(input_path),
        str(output_json),
        config,
    )

    result = load_json(output_json)

    assert "resize" in result["operations"]
    assert "format_conversion" in result["operations"]
    assert "compression" in result["operations"]

    assert result["original_format"] == "PNG"
    assert result["original_width"] == 100
    assert result["original_height"] == 200

    assert result["result_format"] == "WEBP"
    assert result["result_width"] == 25
    assert result["result_height"] == 30

    assert "compression" in result
    assert "metadata" in result
    assert "metadata_reference" in result

    output_image = tmp_path / "processed.webp"

    assert output_image.exists()

    metadata_path = tmp_path / "metadata.json"

    assert metadata_path.exists()


def test_process_image_result_json_uses_requested_path(
    tmp_path,
):
    """Test that results are written to the supplied JSON path."""
    input_path = tmp_path / "input.png"

    output_dir = tmp_path / "results"
    output_dir.mkdir()

    output_json = output_dir / "image-results.json"

    create_test_image(
        input_path,
        image_format="PNG",
    )

    process_image(
        str(input_path),
        str(output_json),
        base_config(),
    )

    assert output_json.exists()

    result = load_json(output_json)

    expected_output_image = (
        output_dir / "processed.png"
    )

    assert result["processed_path"] == str(
        expected_output_image
    )
    assert expected_output_image.exists()


def test_process_image_paths_are_job_relative_when_used_that_way(
    tmp_path,
):
    """Test that result paths reflect the supplied input and output paths."""
    job_dir = tmp_path / "uploads" / "123"
    job_dir.mkdir(parents=True)

    input_path = job_dir / "input.png"
    output_json = job_dir / "results.json"

    create_test_image(
        input_path,
        image_format="PNG",
    )

    process_image(
        str(input_path),
        str(output_json),
        base_config(),
    )

    result = load_json(output_json)

    assert result["original_path"] == str(
        input_path
    )
    assert result["processed_path"] == str(
        job_dir / "processed.png"
    )