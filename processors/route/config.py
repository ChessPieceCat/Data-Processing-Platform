import json


SUPPORTED_ALGORITHMS = {
    "nearest_neighbor_2opt",
}


def load_config(config_path):
    """Load a route optimization configuration from JSON."""
    try:
        with open(
            config_path,
            "r",
            encoding="utf-8",
        ) as file:
            return json.load(file)
    except Exception as error:
        raise ValueError(
            f"Error reading config file: {error}"
        ) from error


def validate_config(config):
    """Validate a route optimization configuration."""
    if not isinstance(config, dict):
        raise ValueError(
            "Configuration must be a JSON object."
        )

    if "start_location" not in config:
        raise ValueError(
            "Missing required key 'start_location' in config file."
        )

    start_location = config["start_location"]

    if not isinstance(start_location, str):
        raise ValueError(
            "'start_location' must be a string."
        )

    if not start_location.strip():
        raise ValueError(
            "'start_location' cannot be empty."
        )

    if "end_location" in config:
        end_location = config["end_location"]

        if not isinstance(end_location, str):
            raise ValueError(
                "'end_location' must be a string."
            )

        if not end_location.strip():
            raise ValueError(
                "'end_location' cannot be empty."
            )

    if "optimization" not in config:
        raise ValueError(
            "Missing required key 'optimization' in config file."
        )

    optimization = config["optimization"]

    if not isinstance(optimization, dict):
        raise ValueError(
            "'optimization' must be an object."
        )

    if "algorithm" not in optimization:
        raise ValueError(
            "Missing required key 'algorithm' "
            "in optimization configuration."
        )

    algorithm = optimization["algorithm"]

    if not isinstance(algorithm, str):
        raise ValueError(
            "'algorithm' must be a string."
        )

    if algorithm not in SUPPORTED_ALGORITHMS:
        raise ValueError(
            f"Unsupported routing algorithm: {algorithm!r}. "
            f"Supported algorithms: "
            f"{sorted(SUPPORTED_ALGORITHMS)}"
        )

    if "constraints" in config:
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

def validate_location_references(config, locations):
    """Validate configured start and end locations."""
    location_names = {
        location.name
        for location in locations
    }

    start_location = config["start_location"]

    if start_location not in location_names:
        raise ValueError(
            f"Start location {start_location!r} "
            "does not exist in the route data."
        )

    end_location = config.get("end_location")

    if (
        end_location is not None
        and end_location not in location_names
    ):
        raise ValueError(
            f"End location {end_location!r} "
            "does not exist in the route data."
        )