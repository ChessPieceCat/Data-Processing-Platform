import pytest

from routing import (
    build_route,
    calculate_total_distance,
    get_distance,
    nearest_neighbor,
    two_opt,
)


class TestGetDistance:
    @pytest.fixture
    def distance_table(self):
        return {
            "A": {
                "A": 0.0,
                "B": 10.0,
                "C": 20.0,
            },
            "B": {
                "A": 10.0,
                "B": 0.0,
            },
            "C": {
                "A": 20.0,
                "C": 0.0,
            },
        }

    def test_get_direct_distance(self, distance_table):
        assert get_distance(
            "A",
            "B",
            distance_table,
        ) == 10.0

    def test_get_reverse_distance(self, distance_table):
        assert get_distance(
            "B",
            "A",
            distance_table,
        ) == 10.0

    def test_unknown_first_location(self, distance_table):
        with pytest.raises(
            ValueError,
            match="Unknown location: 'X'",
        ):
            get_distance(
                "X",
                "A",
                distance_table,
            )

    def test_unknown_second_location(self, distance_table):
        with pytest.raises(
            ValueError,
            match="Unknown location: 'X'",
        ):
            get_distance(
                "A",
                "X",
                distance_table,
            )

    def test_missing_distance_pair(self):
        distance_table = {
            "A": {
                "A": 0.0,
            },
            "B": {
                "B": 0.0,
            },
        }

        with pytest.raises(
            ValueError,
            match="No distance available between",
        ):
            get_distance(
                "A",
                "B",
                distance_table,
            )

    def test_distance_is_numeric(self, distance_table):
        result = get_distance(
            "A",
            "B",
            distance_table,
        )

        assert isinstance(result, (int, float))


class TestNearestNeighbor:
    @pytest.fixture
    def distance_table(self):
        return {
            "Warehouse": {
                "Warehouse": 0.0,
                "A": 2.0,
                "B": 8.0,
                "C": 5.0,
            },
            "A": {
                "Warehouse": 2.0,
                "B": 3.0,
                "C": 6.0,
            },
            "B": {
                "Warehouse": 8.0,
                "A": 3.0,
                "C": 1.0,
            },
            "C": {
                "Warehouse": 5.0,
                "A": 6.0,
                "B": 1.0,
            },
        }

    def test_nearest_neighbor_closed_route(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
            "C",
        ]

        route = nearest_neighbor(
            locations,
            distance_table,
            "Warehouse",
        )

        assert route[0] == "Warehouse"
        assert route[-1] == "Warehouse"
        assert set(route) == {
            "Warehouse",
            "A",
            "B",
            "C",
        }

        assert len(route) == 5

    def test_nearest_neighbor_open_route(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
            "C",
        ]

        route = nearest_neighbor(
            locations,
            distance_table,
            "Warehouse",
            "C",
        )

        assert route[0] == "Warehouse"
        assert route[-1] == "C"

        assert route[1:-1] == [
            "A",
            "B",
        ]

        assert len(route) == 4

    def test_nearest_neighbor_selects_closest_location(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
            "C",
        ]

        route = nearest_neighbor(
            locations,
            distance_table,
            "Warehouse",
        )

        assert route[1] == "A"

    def test_start_location_must_exist(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
        ]

        with pytest.raises(
            ValueError,
            match="Start location 'Missing'",
        ):
            nearest_neighbor(
                locations,
                distance_table,
                "Missing",
            )

    def test_end_location_must_exist(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
        ]

        with pytest.raises(
            ValueError,
            match="End location 'Missing'",
        ):
            nearest_neighbor(
                locations,
                distance_table,
                "Warehouse",
                "Missing",
            )

    def test_start_is_not_selected_as_intermediate_stop(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
        ]

        route = nearest_neighbor(
            locations,
            distance_table,
            "Warehouse",
        )

        assert route.count("Warehouse") == 2

    def test_end_is_not_selected_as_intermediate_stop(
        self,
        distance_table,
    ):
        locations = [
            "Warehouse",
            "A",
            "B",
            "C",
        ]

        route = nearest_neighbor(
            locations,
            distance_table,
            "Warehouse",
            "C",
        )

        assert route.count("C") == 1


class TestCalculateTotalDistance:
    @pytest.fixture
    def distance_table(self):
        return {
            "Warehouse": {
                "Warehouse": 0.0,
                "A": 10.0,
                "B": 20.0,
                "C": 30.0,
            },
            "A": {
                "Warehouse": 10.0,
                "B": 5.0,
                "C": 15.0,
            },
            "B": {
                "Warehouse": 20.0,
                "A": 5.0,
                "C": 7.0,
            },
            "C": {
                "Warehouse": 30.0,
                "A": 15.0,
                "B": 7.0,
            },
        }

    def test_empty_route(self, distance_table):
        assert calculate_total_distance(
            [],
            distance_table,
        ) == 0.0

    def test_single_location(self, distance_table):
        assert calculate_total_distance(
            ["Warehouse"],
            distance_table,
        ) == 0.0

    def test_closed_route(self, distance_table):
        route = [
            "Warehouse",
            "A",
            "B",
            "C",
            "Warehouse",
        ]

        distance = calculate_total_distance(
            route,
            distance_table,
        )

        assert distance == 10.0 + 5.0 + 7.0 + 30.0
        assert distance == 52.0

    def test_open_route(self, distance_table):
        route = [
            "Warehouse",
            "A",
            "B",
            "C",
        ]

        distance = calculate_total_distance(
            route,
            distance_table,
        )

        assert distance == 10.0 + 5.0 + 7.0
        assert distance == 22.0

    def test_reverse_distance_lookup(self):
        distance_table = {
            "A": {
                "A": 0.0,
                "B": 10.0,
            },
            "B": {
                "B": 0.0,
            },
        }

        route = [
            "A",
            "B",
            "A",
        ]

        assert calculate_total_distance(
            route,
            distance_table,
        ) == 20.0


class TestTwoOpt:
    @pytest.fixture
    def improvement_distance_table(self):
        return {
            "Warehouse": {
                "Warehouse": 0.0,
                "A": 1.0,
                "B": 1.41421356237,
                "C": 1.0,
                "D": 2.2360679775,
            },
            "A": {
                "Warehouse": 1.0,
                "A": 0.0,
                "B": 1.0,
                "C": 1.41421356237,
                "D": 1.41421356237,
            },
            "B": {
                "Warehouse": 1.41421356237,
                "A": 1.0,
                "B": 0.0,
                "C": 1.0,
                "D": 1.0,
            },
            "C": {
                "Warehouse": 1.0,
                "A": 1.41421356237,
                "B": 1.0,
                "C": 0.0,
                "D": 2.0,
            },
            "D": {
                "Warehouse": 2.2360679775,
                "A": 1.41421356237,
                "B": 1.0,
                "C": 2.0,
                "D": 0.0,
            },
        }

    def test_two_opt_improves_route(
        self,
        improvement_distance_table,
    ):
        route = [
            "Warehouse",
            "A",
            "C",
            "B",
            "D",
            "Warehouse",
        ]

        initial_distance = calculate_total_distance(
            route,
            improvement_distance_table,
        )

        optimized_route = two_opt(
            route,
            improvement_distance_table,
        )

        optimized_distance = calculate_total_distance(
            optimized_route,
            improvement_distance_table,
        )

        assert optimized_distance < initial_distance

    def test_two_opt_does_not_return_worse_route(
        self,
        improvement_distance_table,
    ):
        route = [
            "Warehouse",
            "A",
            "C",
            "B",
            "Warehouse",
        ]

        initial_distance = calculate_total_distance(
            route,
            improvement_distance_table,
        )

        optimized_route = two_opt(
            route,
            improvement_distance_table,
        )

        optimized_distance = calculate_total_distance(
            optimized_route,
            improvement_distance_table,
        )

        assert optimized_distance <= initial_distance

    def test_two_opt_preserves_start_and_end(
        self,
        improvement_distance_table,
    ):
        route = [
            "Warehouse",
            "A",
            "C",
            "B",
            "Warehouse",
        ]

        optimized_route = two_opt(
            route,
            improvement_distance_table,
        )

        assert optimized_route[0] == "Warehouse"
        assert optimized_route[-1] == "Warehouse"

    def test_two_opt_does_not_mutate_original_route(
        self,
        improvement_distance_table,
    ):
        route = [
            "Warehouse",
            "A",
            "C",
            "B",
            "Warehouse",
        ]

        original_route = route.copy()

        two_opt(
            route,
            improvement_distance_table,
        )

        assert route == original_route

    def test_two_opt_no_improvement_case(self):
        distance_table = {
            "Warehouse": {
                "Warehouse": 0.0,
                "A": 1.0,
                "B": 2.0,
                "C": 3.0,
            },
            "A": {
                "Warehouse": 1.0,
                "A": 0.0,
                "B": 1.0,
                "C": 2.0,
            },
            "B": {
                "Warehouse": 2.0,
                "A": 1.0,
                "B": 0.0,
                "C": 1.0,
            },
            "C": {
                "Warehouse": 3.0,
                "A": 2.0,
                "B": 1.0,
                "C": 0.0,
            },
        }

        route = [
            "Warehouse",
            "A",
            "B",
            "C",
            "Warehouse",
        ]

        optimized_route = two_opt(
            route,
            distance_table,
        )

        assert optimized_route == route

    def test_two_opt_preserves_open_route_endpoints(self):
        distance_table = {
            "Warehouse": {
                "A": 1.0,
                "B": 5.0,
                "C": 1.0,
            },
            "A": {
                "Warehouse": 1.0,
                "B": 1.0,
                "C": 10.0,
            },
            "B": {
                "Warehouse": 5.0,
                "A": 1.0,
                "C": 1.0,
            },
            "C": {
                "Warehouse": 1.0,
                "A": 10.0,
                "B": 1.0,
            },
        }

        route = [
            "Warehouse",
            "A",
            "C",
            "B",
        ]

        optimized_route = two_opt(
            route,
            distance_table,
        )

        assert optimized_route[0] == "Warehouse"
        assert optimized_route[-1] == "B"


class TestBuildRoute:
    @pytest.fixture
    def distance_table(self):
        return {
            "Warehouse": {
                "Warehouse": 0.0,
                "A": 1.0,
                "B": 5.0,
                "C": 1.0,
            },
            "A": {
                "Warehouse": 1.0,
                "A": 0.0,
                "B": 1.0,
                "C": 10.0,
            },
            "B": {
                "Warehouse": 5.0,
                "A": 1.0,
                "B": 0.0,
                "C": 1.0,
            },
            "C": {
                "Warehouse": 1.0,
                "A": 10.0,
                "B": 1.0,
                "C": 0.0,
            },
        }

    @pytest.fixture
    def locations(self):
        return [
            "Warehouse",
            "A",
            "B",
            "C",
        ]

    def test_build_route_with_two_opt(
        self,
        locations,
        distance_table,
    ):
        result = build_route(
            locations,
            distance_table,
            "Warehouse",
            "Warehouse",
            use_two_opt=True,
        )

        assert set(result) == {
            "initial_route",
            "optimized_route",
            "initial_distance",
            "optimized_distance",
            "distance_improvement",
            "improvement_percentage",
        }

        assert result["initial_route"][0] == "Warehouse"
        assert result["initial_route"][-1] == "Warehouse"

        assert result["optimized_route"][0] == "Warehouse"
        assert result["optimized_route"][-1] == "Warehouse"

        assert (
            result["optimized_distance"]
            <= result["initial_distance"]
        )

    def test_build_route_without_two_opt(
        self,
        locations,
        distance_table,
    ):
        result = build_route(
            locations,
            distance_table,
            "Warehouse",
            "Warehouse",
            use_two_opt=False,
        )

        assert result["optimized_route"] == (
            result["initial_route"]
        )

        assert result["optimized_distance"] == (
            result["initial_distance"]
        )

        assert result["distance_improvement"] == 0.0
        assert result["improvement_percentage"] == 0.0

    def test_build_route_calculates_improvement(
        self,
        locations,
        distance_table,
    ):
        result = build_route(
            locations,
            distance_table,
            "Warehouse",
            "Warehouse",
            use_two_opt=True,
        )

        expected_improvement = (
            result["initial_distance"]
            - result["optimized_distance"]
        )

        assert (
            result["distance_improvement"]
            == pytest.approx(expected_improvement)
        )

        if result["initial_distance"] > 0:
            expected_percentage = (
                expected_improvement
                / result["initial_distance"]
                * 100
            )

            assert (
                result["improvement_percentage"]
                == pytest.approx(expected_percentage)
            )

    def test_build_route_supports_open_route(
        self,
        locations,
        distance_table,
    ):
        result = build_route(
            locations,
            distance_table,
            "Warehouse",
            "C",
            use_two_opt=True,
        )

        assert result["initial_route"][0] == "Warehouse"
        assert result["initial_route"][-1] == "C"

        assert result["optimized_route"][0] == "Warehouse"
        assert result["optimized_route"][-1] == "C"

    def test_build_route_with_zero_initial_distance(self):
        locations = [
            "Warehouse",
        ]

        distance_table = {
            "Warehouse": {
                "Warehouse": 0.0,
            },
        }

        result = build_route(
            locations,
            distance_table,
            "Warehouse",
            "Warehouse",
            use_two_opt=True,
        )

        assert result["initial_distance"] == 0.0
        assert result["optimized_distance"] == 0.0
        assert result["distance_improvement"] == 0.0
        assert result["improvement_percentage"] == 0.0