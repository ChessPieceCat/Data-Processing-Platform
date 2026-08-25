import json

import numpy as np
import pytest

from io_utils import json_serializer, save_json


def test_json_serializer_numpy_integer():
    value = np.int64(42)

    result = json_serializer(value)

    assert result == 42
    assert isinstance(result, int)


def test_json_serializer_numpy_float():
    value = np.float64(3.14)

    result = json_serializer(value)

    assert result == pytest.approx(3.14)
    assert isinstance(result, float)


def test_json_serializer_rejects_unsupported_type():
    with pytest.raises(
        TypeError,
        match="Object of type",
    ):
        json_serializer(object())


def test_save_json_creates_file(tmp_path):
    output_path = tmp_path / "results.json"

    data = {
        "rows": 10,
        "name": "test",
    }

    save_json(data, output_path)

    assert output_path.exists()

    with open(output_path, "r", encoding="utf-8") as file:
        result = json.load(file)

    assert result == data


def test_save_json_handles_numpy_values(tmp_path):
    output_path = tmp_path / "results.json"

    data = {
        "count": np.int64(10),
        "mean": np.float64(4.5),
    }

    save_json(data, output_path)

    with open(output_path, "r", encoding="utf-8") as file:
        result = json.load(file)

    assert result["count"] == 10
    assert result["mean"] == pytest.approx(4.5)


def test_save_json_is_indented(tmp_path):
    output_path = tmp_path / "results.json"

    data = {
        "outer": {
            "value": 10,
        }
    }

    save_json(data, output_path)

    content = output_path.read_text(encoding="utf-8")

    assert content == (
        '{\n'
        '  "outer": {\n'
        '    "value": 10\n'
        '  }\n'
        '}'
    )


def test_save_json_invalid_output_path():
    with pytest.raises(
        ValueError,
        match="Error writing JSON file",
    ):
        save_json(
            {"value": 1},
            "/path/that/does/not/exist/results.json",
        )