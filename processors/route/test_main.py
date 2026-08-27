import json
from unittest.mock import patch

import pytest

import main


class TestParseArguments:
    def test_parse_valid_arguments(self):
        args = [
            "main.py",
            "route.csv",
            "distances.csv",
            "results.json",
            "config.json",
        ]

        with patch("sys.argv", args):
            result = main.parse_arguments()

        assert result.route_csv == "route.csv"
        assert result.distance_csv == "distances.csv"
        assert result.output_json == "results.json"
        assert result.config_json == "config.json"

    def test_parse_requires_all_arguments(self):
        args = [
            "main.py",
            "route.csv",
            "distances.csv",
            "results.json",
        ]

        with patch("sys.argv", args):
            with pytest.raises(SystemExit):
                main.parse_arguments()


class TestValidateRouteInputs:
    class Location:
        def __init__(self, name):
            self.name = name

    def test_matching_route_and_distance_locations(self):
        locations = [
            self.Location("Warehouse"),
            self.Location("Customer A"),
            self.Location("Customer B"),
        ]

        distance_table = {
            "Warehouse": {},
            "Customer A": {},
            "Customer B": {},
        }

        main.validate_route_inputs(
            locations,
            distance_table,
        )

    def test_missing_distance_location(self):
        locations = [
            self.Location("Warehouse"),
            self.Location("Customer A"),
        ]

        distance_table = {
            "Warehouse": {},
        }

        with pytest.raises(
            ValueError,
            match="Distance table is missing locations: Customer A",
        ):
            main.validate_route_inputs(
                locations,
                distance_table,
            )

    def test_extra_distance_location(self):
        locations = [
            self.Location("Warehouse"),
        ]

        distance_table = {
            "Warehouse": {},
            "Customer A": {},
        }

        with pytest.raises(
            ValueError,
            match="Distance table contains unknown locations: Customer A",
        ):
            main.validate_route_inputs(
                locations,
                distance_table,
            )

    def test_multiple_missing_locations_are_reported(self):
        locations = [
            self.Location("Warehouse"),
            self.Location("Customer A"),
            self.Location("Customer B"),
        ]

        distance_table = {
            "Warehouse": {},
        }

        with pytest.raises(
            ValueError,
            match="Distance table is missing locations: Customer A, Customer B",
        ):
            main.validate_route_inputs(
                locations,
                distance_table,
            )


class TestMain:
    @pytest.fixture
    def locations(self):
        return [
            self.make_location("Warehouse"),
            self.make_location("Customer A"),
            self.make_location("Customer B"),
        ]

    @staticmethod
    def make_location(name):
        location = type(
            "Location",
            (),
            {},
        )()

        location.name = name

        return location

    @pytest.fixture
    def config(self):
        return {
            "start_location": "Warehouse",
            "end_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
            "constraints": {
                "max_distance": 100.0,
                "max_stops": 2,
            },
        }

    @pytest.fixture
    def route_results(self):
        return {
            "initial_route": [
                "Warehouse",
                "Customer A",
                "Customer B",
                "Warehouse",
            ],
            "optimized_route": [
                "Warehouse",
                "Customer B",
                "Customer A",
                "Warehouse",
            ],
            "initial_distance": 40.0,
            "optimized_distance": 35.0,
            "distance_improvement": 5.0,
            "improvement_percentage": 12.5,
        }

    def patch_common_dependencies(
        self,
        monkeypatch,
        config,
        locations,
        route_results,
    ):
        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "load_locations",
            lambda path: locations,
        )

        monkeypatch.setattr(
            main,
            "load_distance_table",
            lambda path: {
                "Warehouse": {},
                "Customer A": {},
                "Customer B": {},
            },
        )

        monkeypatch.setattr(
            main,
            "validate_route_inputs",
            lambda locations, distance_table: None,
        )

        monkeypatch.setattr(
            main,
            "validate_location_references",
            lambda config, locations: None,
        )

        monkeypatch.setattr(
            main,
            "build_route",
            lambda *args, **kwargs: route_results,
        )

        monkeypatch.setattr(
            main,
            "check_route_feasibility",
            lambda *args, **kwargs: True,
        )

    @staticmethod
    def make_args(output_json):
        return type(
            "Args",
            (),
            {
                "route_csv": "route.csv",
                "distance_csv": "distances.csv",
                "output_json": str(output_json),
                "config_json": "config.json",
            },
        )()

    def test_successful_orchestration(
        self,
        monkeypatch,
        tmp_path,
        config,
        locations,
        route_results,
    ):
        output_path = tmp_path / "results.json"

        self.patch_common_dependencies(
            monkeypatch,
            config,
            locations,
            route_results,
        )

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args(output_path),
        )

        result = main.main()

        assert result == 0
        assert output_path.exists()

        output = json.loads(
            output_path.read_text(
                encoding="utf-8",
            )
        )

        assert output["start_location"] == "Warehouse"
        assert output["end_location"] == "Warehouse"
        assert output["initial_route"] == (
            route_results["initial_route"]
        )
        assert output["optimized_route"] == (
            route_results["optimized_route"]
        )
        assert output["initial_distance"] == 40.0
        assert output["optimized_distance"] == 35.0
        assert output["distance_improvement"] == 5.0
        assert output["improvement_percentage"] == 12.5
        assert output["algorithm"] == (
            "nearest_neighbor_2opt"
        )
        assert output["two_opt_applied"] is True
        assert output["runtime_seconds"] >= 0
        assert output["feasible"] is True

    def test_successful_orchestration_without_end_location(
        self,
        monkeypatch,
        tmp_path,
        locations,
        route_results,
    ):
        config = {
            "start_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        output_path = tmp_path / "results.json"

        self.patch_common_dependencies(
            monkeypatch,
            config,
            locations,
            route_results,
        )

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args(output_path),
        )

        result = main.main()

        assert result == 0

        output = json.loads(
            output_path.read_text(
                encoding="utf-8",
            )
        )

        assert output["start_location"] == "Warehouse"
        assert output["end_location"] == "Warehouse"

    def test_configuration_failure_returns_one(
        self,
        monkeypatch,
    ):
        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args("results.json"),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: {},
        )

        def fail_validation(config):
            raise ValueError(
                "Invalid route configuration."
            )

        monkeypatch.setattr(
            main,
            "validate_config",
            fail_validation,
        )

        result = main.main()

        assert result == 1

    def test_constraint_configuration_failure_returns_one(
        self,
        monkeypatch,
    ):
        config = {
            "start_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args("results.json"),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        def fail_constraints(config):
            raise ValueError(
                "Invalid routing constraints."
            )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            fail_constraints,
        )

        result = main.main()

        assert result == 1

    def test_route_input_validation_failure_returns_one(
        self,
        monkeypatch,
        locations,
    ):
        config = {
            "start_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args("results.json"),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "load_locations",
            lambda path: locations,
        )

        monkeypatch.setattr(
            main,
            "load_distance_table",
            lambda path: {},
        )

        def fail_input_validation(
            route_locations,
            distance_table,
        ):
            raise ValueError(
                "Distance table is missing locations."
            )

        monkeypatch.setattr(
            main,
            "validate_route_inputs",
            fail_input_validation,
        )

        result = main.main()

        assert result == 1

    def test_location_reference_validation_failure_returns_one(
        self,
        monkeypatch,
        locations,
    ):
        config = {
            "start_location": "Missing",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args("results.json"),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "load_locations",
            lambda path: locations,
        )

        monkeypatch.setattr(
            main,
            "load_distance_table",
            lambda path: {
                "Warehouse": {},
                "Customer A": {},
                "Customer B": {},
            },
        )

        monkeypatch.setattr(
            main,
            "validate_route_inputs",
            lambda route_locations, distance_table: None,
        )

        def fail_location_references(
            config,
            locations,
        ):
            raise ValueError(
                "Start location 'Missing' does not exist."
            )

        monkeypatch.setattr(
            main,
            "validate_location_references",
            fail_location_references,
        )

        result = main.main()

        assert result == 1

    def test_processing_failure_returns_one(
        self,
        monkeypatch,
        locations,
    ):
        config = {
            "start_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args("results.json"),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "load_locations",
            lambda path: locations,
        )

        monkeypatch.setattr(
            main,
            "load_distance_table",
            lambda path: {
                "Warehouse": {},
                "Customer A": {},
                "Customer B": {},
            },
        )

        monkeypatch.setattr(
            main,
            "validate_route_inputs",
            lambda route_locations, distance_table: None,
        )

        monkeypatch.setattr(
            main,
            "validate_location_references",
            lambda config, route_locations: None,
        )

        def fail_build_route(*args, **kwargs):
            raise ValueError(
                "Unable to build route."
            )

        monkeypatch.setattr(
            main,
            "build_route",
            fail_build_route,
        )

        result = main.main()

        assert result == 1

    def test_feasibility_failure_returns_one(
        self,
        monkeypatch,
        locations,
        route_results,
    ):
        config = {
            "start_location": "Warehouse",
            "end_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args("results.json"),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "load_locations",
            lambda path: locations,
        )

        monkeypatch.setattr(
            main,
            "load_distance_table",
            lambda path: {
                "Warehouse": {},
                "Customer A": {},
                "Customer B": {},
            },
        )

        monkeypatch.setattr(
            main,
            "validate_route_inputs",
            lambda route_locations, distance_table: None,
        )

        monkeypatch.setattr(
            main,
            "validate_location_references",
            lambda config, route_locations: None,
        )

        monkeypatch.setattr(
            main,
            "build_route",
            lambda *args, **kwargs: route_results,
        )

        def fail_feasibility(*args, **kwargs):
            raise ValueError(
                "Route exceeds maximum distance."
            )

        monkeypatch.setattr(
            main,
            "check_route_feasibility",
            fail_feasibility,
        )

        result = main.main()

        assert result == 1

    def test_output_is_not_written_when_processing_fails(
        self,
        monkeypatch,
        tmp_path,
        locations,
    ):
        config = {
            "start_location": "Warehouse",
            "optimization": {
                "algorithm": "nearest_neighbor_2opt",
            },
        }

        output_path = tmp_path / "results.json"

        monkeypatch.setattr(
            main,
            "parse_arguments",
            lambda: self.make_args(output_path),
        )

        monkeypatch.setattr(
            main,
            "load_config",
            lambda path: config,
        )

        monkeypatch.setattr(
            main,
            "validate_config",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "validate_constraints",
            lambda value: None,
        )

        monkeypatch.setattr(
            main,
            "load_locations",
            lambda path: locations,
        )

        monkeypatch.setattr(
            main,
            "load_distance_table",
            lambda path: {
                "Warehouse": {},
                "Customer A": {},
                "Customer B": {},
            },
        )

        monkeypatch.setattr(
            main,
            "validate_route_inputs",
            lambda route_locations, distance_table: None,
        )

        monkeypatch.setattr(
            main,
            "validate_location_references",
            lambda config, route_locations: None,
        )

        monkeypatch.setattr(
            main,
            "build_route",
            lambda *args, **kwargs: {
                "initial_route": [
                    "Warehouse",
                    "Customer A",
                    "Customer B",
                    "Warehouse",
                ],
                "optimized_route": [
                    "Warehouse",
                    "Customer A",
                    "Customer B",
                    "Warehouse",
                ],
                "initial_distance": 40.0,
                "optimized_distance": 40.0,
                "distance_improvement": 0.0,
                "improvement_percentage": 0.0,
            },
        )

        monkeypatch.setattr(
            main,
            "check_route_feasibility",
            lambda *args, **kwargs: (
                (_ for _ in ()).throw(
                    ValueError(
                        "Route exceeds maximum distance."
                    )
                )
            ),
        )

        result = main.main()

        assert result == 1
        assert not output_path.exists()