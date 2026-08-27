import pytest

from constraints import (
    check_route_feasibility,
    validate_constraints,
)


class Location:
    """Minimal location object for constraint tests."""

    def __init__(self, name):
        self.name = name


class TestValidateConstraints:
    def test_constraints_are_optional(self):
        config = {}

        validate_constraints(config)

    def test_valid_constraints(self):
        config = {
            "constraints": {
                "max_distance": 100.0,
                "max_stops": 5,
            }
        }

        validate_constraints(config)

    def test_constraints_must_be_object(self):
        config = {
            "constraints": []
        }

        with pytest.raises(
            ValueError,
            match="'constraints' must be an object",
        ):
            validate_constraints(config)

    @pytest.mark.parametrize(
        "value",
        [0, -1, True, False, "100", None],
    )
    def test_invalid_max_distance(self, value):
        config = {
            "constraints": {
                "max_distance": value,
            }
        }

        with pytest.raises(
            ValueError,
            match="'max_distance' must be a positive number",
        ):
            validate_constraints(config)

    @pytest.mark.parametrize(
        "value",
        [0, -1, 1.5, True, False, "5", None],
    )
    def test_invalid_max_stops(self, value):
        config = {
            "constraints": {
                "max_stops": value,
            }
        }

        with pytest.raises(
            ValueError,
            match="'max_stops' must be a positive integer",
        ):
            validate_constraints(config)

    def test_valid_max_distance_at_small_positive_value(self):
        config = {
            "constraints": {
                "max_distance": 0.1,
            }
        }

        validate_constraints(config)

    def test_valid_max_stops_at_one(self):
        config = {
            "constraints": {
                "max_stops": 1,
            }
        }

        validate_constraints(config)


class TestCheckRouteFeasibility:
    @pytest.fixture
    def locations(self):
        return [
            Location("Warehouse"),
            Location("Customer A"),
            Location("Customer B"),
            Location("Customer C"),
        ]

    @pytest.fixture
    def distance_table(self):
        return {
            "Warehouse": {
                "Warehouse": 0.0,
                "Customer A": 10.0,
                "Customer B": 20.0,
                "Customer C": 30.0,
            },
            "Customer A": {
                "Warehouse": 10.0,
                "Customer A": 0.0,
                "Customer B": 10.0,
                "Customer C": 20.0,
            },
            "Customer B": {
                "Warehouse": 20.0,
                "Customer A": 10.0,
                "Customer B": 0.0,
                "Customer C": 10.0,
            },
            "Customer C": {
                "Warehouse": 30.0,
                "Customer A": 20.0,
                "Customer B": 10.0,
                "Customer C": 0.0,
            },
        }

    def test_valid_closed_route(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
            "Warehouse",
        ]

        constraints = {
            "max_distance": 100.0,
            "max_stops": 3,
        }

        assert check_route_feasibility(
            route,
            locations,
            distance_table,
            constraints,
            "Warehouse",
            "Warehouse",
        ) is True

    def test_valid_open_route(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
        ]

        constraints = {
            "max_distance": 100.0,
            "max_stops": 3,
        }

        assert check_route_feasibility(
            route,
            locations,
            distance_table,
            constraints,
            "Warehouse",
            "Customer C",
        ) is True

    def test_empty_route(self, locations, distance_table):
        with pytest.raises(
            ValueError,
            match="Route must contain at least two locations",
        ):
            check_route_feasibility(
                [],
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_single_location_route(
        self,
        locations,
        distance_table,
    ):
        with pytest.raises(
            ValueError,
            match="Route must contain at least two locations",
        ):
            check_route_feasibility(
                ["Warehouse"],
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_invalid_start_location(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Customer A",
            "Customer B",
            "Customer C",
            "Warehouse",
        ]

        with pytest.raises(
            ValueError,
            match="Route does not start at the configured start location",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_invalid_end_location(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
        ]

        with pytest.raises(
            ValueError,
            match="Route does not end at the configured end location",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_missing_required_location(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Warehouse",
        ]

        with pytest.raises(
            ValueError,
            match="Route is missing required locations",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_unknown_location_in_route(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Unknown",
            "Customer C",
            "Warehouse",
        ]

        with pytest.raises(
            ValueError,
            match="Route contains unknown locations",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_duplicate_location(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer A",
            "Customer C",
            "Warehouse",
        ]

        with pytest.raises(
            ValueError,
            match="Route contains duplicate locations",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )

    def test_route_exceeds_max_stops(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
            "Warehouse",
        ]

        constraints = {
            "max_stops": 2,
        }

        with pytest.raises(
            ValueError,
            match="Route exceeds maximum stops of 2",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                constraints,
                "Warehouse",
                "Warehouse",
            )

    def test_route_exceeds_max_distance(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
            "Warehouse",
        ]

        constraints = {
            "max_distance": 50.0,
        }

        with pytest.raises(
            ValueError,
            match="Route exceeds maximum distance of 50.0",
        ):
            check_route_feasibility(
                route,
                locations,
                distance_table,
                constraints,
                "Warehouse",
                "Warehouse",
            )

    def test_route_without_constraints_is_feasible(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
            "Warehouse",
        ]

        assert check_route_feasibility(
            route,
            locations,
            distance_table,
            {},
            "Warehouse",
            "Warehouse",
        ) is True

    def test_missing_distance_relationship(
        self,
        locations,
        distance_table,
    ):
        route = [
            "Warehouse",
            "Customer A",
            "Customer B",
            "Customer C",
            "Warehouse",
        ]

        incomplete_distance_table = {
            "Warehouse": {
                "Warehouse": 0.0,
                "Customer A": 10.0,
                "Customer B": 20.0,
                "Customer C": 30.0,
            },
            "Customer A": {
                "Warehouse": 10.0,
                "Customer A": 0.0,
                "Customer B": 10.0,
                "Customer C": 20.0,
            },
            "Customer B": {
                "Warehouse": 20.0,
                "Customer A": 10.0,
                "Customer B": 0.0,
                # Customer C intentionally omitted
            },
            "Customer C": {
                "Warehouse": 30.0,
                "Customer A": 20.0,
                "Customer C": 0.0,
                # Customer B intentionally omitted
            },
        }

        with pytest.raises(
            ValueError,
            match="No distance available between",
        ):
            check_route_feasibility(
                route,
                locations,
                incomplete_distance_table,
                {},
                "Warehouse",
                "Warehouse",
            )