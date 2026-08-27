import json

import pytest

from config import (
    SUPPORTED_ALGORITHMS,
    load_config,
    validate_config,
    validate_location_references,
)


class TestLoadConfig:
    def test_load_valid_config(self, tmp_path):
        config_path = tmp_path / "config.json"

        config = {
            "start_location": "Warehouse",
            "end_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt"
            },
            "constraints": {
                "max_distance": 140.0,
                "max_stops": 40,
            },
        }

        config_path.write_text(
            json.dumps(config),
            encoding="utf-8",
        )

        result = load_config(config_path)

        assert result == config

    def test_load_config_invalid_json(self, tmp_path):
        config_path = tmp_path / "config.json"

        config_path.write_text(
            "{invalid json",
            encoding="utf-8",
        )

        with pytest.raises(
            ValueError,
            match="Error reading config file",
        ):
            load_config(config_path)

    def test_load_config_missing_file(self, tmp_path):
        config_path = tmp_path / "missing.json"

        with pytest.raises(
            ValueError,
            match="Error reading config file",
        ):
            load_config(config_path)


class TestValidateConfig:
    def valid_config(self):
        return {
            "start_location": "Warehouse",
            "end_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt"
            },
            "constraints": {
                "max_distance": 140.0,
                "max_stops": 40,
            },
        }

    def test_valid_config(self):
        config = self.valid_config()

        validate_config(config)

    def test_config_must_be_object(self):
        with pytest.raises(
            ValueError,
            match="Configuration must be a JSON object",
        ):
            validate_config([])

    def test_missing_start_location(self):
        config = self.valid_config()
        del config["start_location"]

        with pytest.raises(
            ValueError,
            match="Missing required key 'start_location'",
        ):
            validate_config(config)

    def test_start_location_must_be_string(self):
        config = self.valid_config()
        config["start_location"] = 123

        with pytest.raises(
            ValueError,
            match="'start_location' must be a string",
        ):
            validate_config(config)

    def test_start_location_cannot_be_empty(self):
        config = self.valid_config()
        config["start_location"] = "   "

        with pytest.raises(
            ValueError,
            match="'start_location' cannot be empty",
        ):
            validate_config(config)

    def test_end_location_is_optional(self):
        config = self.valid_config()
        del config["end_location"]

        validate_config(config)

    def test_end_location_must_be_string(self):
        config = self.valid_config()
        config["end_location"] = 123

        with pytest.raises(
            ValueError,
            match="'end_location' must be a string",
        ):
            validate_config(config)

    def test_end_location_cannot_be_empty(self):
        config = self.valid_config()
        config["end_location"] = ""

        with pytest.raises(
            ValueError,
            match="'end_location' cannot be empty",
        ):
            validate_config(config)

    def test_missing_optimization(self):
        config = self.valid_config()
        del config["optimization"]

        with pytest.raises(
            ValueError,
            match="Missing required key 'optimization'",
        ):
            validate_config(config)

    def test_optimization_must_be_object(self):
        config = self.valid_config()
        config["optimization"] = "nearest_neighbor_2opt"

        with pytest.raises(
            ValueError,
            match="'optimization' must be an object",
        ):
            validate_config(config)

    def test_missing_algorithm(self):
        config = self.valid_config()
        del config["optimization"]["algorithm"]

        with pytest.raises(
            ValueError,
            match="Missing required key 'algorithm'",
        ):
            validate_config(config)

    def test_algorithm_must_be_string(self):
        config = self.valid_config()
        config["optimization"]["algorithm"] = 123

        with pytest.raises(
            ValueError,
            match="'algorithm' must be a string",
        ):
            validate_config(config)

    def test_unsupported_algorithm(self):
        config = self.valid_config()
        config["optimization"]["algorithm"] = "nearest_neighbor"

        with pytest.raises(
            ValueError,
            match="Unsupported routing algorithm",
        ):
            validate_config(config)

    def test_all_supported_algorithms_are_accepted(self):
        for algorithm in SUPPORTED_ALGORITHMS:
            config = self.valid_config()
            config["optimization"]["algorithm"] = algorithm

            validate_config(config)

    def test_constraints_are_optional(self):
        config = self.valid_config()
        del config["constraints"]

        validate_config(config)

    def test_constraints_must_be_object(self):
        config = self.valid_config()
        config["constraints"] = []

        with pytest.raises(
            ValueError,
            match="'constraints' must be an object",
        ):
            validate_config(config)

    @pytest.mark.parametrize(
        "value",
        [0, -1, True, False, "140"],
    )
    def test_invalid_max_distance(self, value):
        config = self.valid_config()
        config["constraints"]["max_distance"] = value

        with pytest.raises(
            ValueError,
            match="'max_distance' must be a positive number",
        ):
            validate_config(config)

    @pytest.mark.parametrize(
        "value",
        [0, -1, 1.5, True, False, "40"],
    )
    def test_invalid_max_stops(self, value):
        config = self.valid_config()
        config["constraints"]["max_stops"] = value

        with pytest.raises(
            ValueError,
            match="'max_stops' must be a positive integer",
        ):
            validate_config(config)

    def test_valid_max_distance(self):
        config = self.valid_config()
        config["constraints"]["max_distance"] = 0.1

        validate_config(config)

    def test_valid_max_stops(self):
        config = self.valid_config()
        config["constraints"]["max_stops"] = 1

        validate_config(config)


class TestValidateLocationReferences:
    class Location:
        def __init__(self, name):
            self.name = name

    @pytest.fixture
    def locations(self):
        return [
            self.Location("Warehouse"),
            self.Location("Customer A"),
            self.Location("Customer B"),
        ]

    def test_valid_start_and_end_locations(self, locations):
        config = {
            "start_location": "Warehouse",
            "end_location": "Customer B",
        }

        validate_location_references(
            config,
            locations,
        )

    def test_valid_start_with_no_end_location(self, locations):
        config = {
            "start_location": "Warehouse",
        }

        validate_location_references(
            config,
            locations,
        )

    def test_invalid_start_location(self, locations):
        config = {
            "start_location": "Missing",
        }

        with pytest.raises(
            ValueError,
            match="Start location 'Missing' does not exist",
        ):
            validate_location_references(
                config,
                locations,
            )

    def test_invalid_end_location(self, locations):
        config = {
            "start_location": "Warehouse",
            "end_location": "Missing",
        }

        with pytest.raises(
            ValueError,
            match="End location 'Missing' does not exist",
        ):
            validate_location_references(
                config,
                locations,
            )