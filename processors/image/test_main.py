import sys
from unittest.mock import patch

import main


def test_parse_arguments():
    """Test that command-line arguments are parsed correctly."""
    test_args = [
        "main.py",
        "input.jpg",
        "results.json",
        "config.json",
    ]

    with patch.object(sys, "argv", test_args):
        args = main.parse_arguments()

    assert args.input_image == "input.jpg"
    assert args.output_json == "results.json"
    assert args.config_json == "config.json"


def test_main_success():
    """Test that main returns success when processing completes."""
    test_args = [
        "main.py",
        "input.jpg",
        "results.json",
        "config.json",
    ]

    config = {
        "resize": False,
        "compression": False,
        "output_format": "original",
        "extract_metadata": False,
    }

    with (
        patch.object(sys, "argv", test_args),
        patch(
            "main.load_config",
            return_value=config,
        ) as mock_load_config,
        patch(
            "main.validate_config"
        ) as mock_validate_config,
        patch(
            "main.process_image"
        ) as mock_process_image,
    ):
        result = main.main()

    assert result == 0

    mock_load_config.assert_called_once_with(
        "config.json"
    )
    mock_validate_config.assert_called_once_with(config)
    mock_process_image.assert_called_once_with(
        "input.jpg",
        "results.json",
        config,
    )


def test_main_handles_processing_error(capsys):
    """Test that processing errors are reported and return failure."""
    test_args = [
        "main.py",
        "input.jpg",
        "results.json",
        "config.json",
    ]

    with (
        patch.object(sys, "argv", test_args),
        patch(
            "main.load_config",
            side_effect=ValueError("invalid config"),
        ),
    ):
        result = main.main()

    assert result == 1

    captured = capsys.readouterr()

    assert (
        captured.err
        == "Error processing image: invalid config\n"
    )


def test_main_handles_validation_error(capsys):
    """Test that configuration validation errors are reported."""
    test_args = [
        "main.py",
        "input.jpg",
        "results.json",
        "config.json",
    ]

    config = {
        "resize": True,
    }

    with (
        patch.object(sys, "argv", test_args),
        patch(
            "main.load_config",
            return_value=config,
        ),
        patch(
            "main.validate_config",
            side_effect=ValueError(
                "missing required key"
            ),
        ),
    ):
        result = main.main()

    assert result == 1

    captured = capsys.readouterr()

    assert (
        captured.err
        == "Error processing image: missing required key\n"
    )


def test_main_handles_image_processing_error(capsys):
    """Test that image-processing errors are reported."""
    test_args = [
        "main.py",
        "input.jpg",
        "results.json",
        "config.json",
    ]

    config = {
        "resize": False,
        "compression": False,
        "output_format": "original",
        "extract_metadata": False,
    }

    with (
        patch.object(sys, "argv", test_args),
        patch(
            "main.load_config",
            return_value=config,
        ),
        patch(
            "main.validate_config"
        ),
        patch(
            "main.process_image",
            side_effect=ValueError(
                "invalid image"
            ),
        ),
    ):
        result = main.main()

    assert result == 1

    captured = capsys.readouterr()

    assert (
        captured.err
        == "Error processing image: invalid image\n"
    )