from routing import calculate_total_distance


def validate_constraints(config):
    """Validate routing constraints in the configuration."""
    if "constraints" not in config:
        return

    constraints = config["constraints"]

    if not isinstance(constraints, dict):
        raise ValueError(
            "'constraints' must be an object."
        )

    if "max_distance" in constraints:
        max_distance = constraints["max_distance"]

        if (
            not isinstance(max_distance, (int, float))
            or isinstance(max_distance, bool)
            or max_distance <= 0
        ):
            raise ValueError(
                "'max_distance' must be a positive number."
            )

    if "max_stops" in constraints:
        max_stops = constraints["max_stops"]

        if (
            not isinstance(max_stops, int)
            or isinstance(max_stops, bool)
            or max_stops <= 0
        ):
            raise ValueError(
                "'max_stops' must be a positive integer."
            )


def check_route_feasibility(
    route,
    locations,
    distance_table,
    constraints,
    start_location,
    end_location,
):
    """Check whether a route satisfies all supported constraints."""

    if not route or len(route) < 2:
        raise ValueError(
            "Route must contain at least two locations."
        )

    if route[0] != start_location:
        raise ValueError(
            "Route does not start at the configured "
            "start location."
        )

    if route[-1] != end_location:
        raise ValueError(
            "Route does not end at the configured "
            "end location."
        )

    required_locations = {
        location.name
        for location in locations
        if location.name not in {
            start_location,
            end_location,
        }
    }

    visited_locations = set(route[1:-1])

    unexpected_locations = (
        visited_locations - required_locations
    )

    if unexpected_locations:
        raise ValueError(
            "Route contains unknown locations: "
            + ", ".join(sorted(unexpected_locations))
        )

    missing_locations = (
        required_locations - visited_locations
    )

    if missing_locations:
        raise ValueError(
            "Route is missing required locations: "
            + ", ".join(sorted(missing_locations))
        )

    unexpected_locations = (
        visited_locations - required_locations
    )

    if unexpected_locations:
        raise ValueError(
            "Route contains unknown locations: "
            + ", ".join(sorted(unexpected_locations))
        )

    if len(visited_locations) != len(route[1:-1]):
        raise ValueError(
            "Route contains duplicate locations."
        )

    total_distance = calculate_total_distance(
        route,
        distance_table,
    )

    if "max_distance" in constraints:
        max_distance = constraints["max_distance"]

        if total_distance > max_distance:
            raise ValueError(
                f"Route exceeds maximum distance "
                f"of {max_distance}."
            )

    if "max_stops" in constraints:
        max_stops = constraints["max_stops"]

        if len(visited_locations) > max_stops:
            raise ValueError(
                f"Route exceeds maximum stops "
                f"of {max_stops}."
            )

    return True