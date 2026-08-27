import argparse
import json
import sys
import time

from config import load_config, validate_config, validate_location_references
from constraints import (
    check_route_feasibility,
    validate_constraints,
)
from distance_table import load_distance_table
from locations import load_locations
from routing import build_route


def parse_arguments():
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description="Optimize a route."
    )

    parser.add_argument("route_csv")
    parser.add_argument("distance_csv")
    parser.add_argument("output_json")
    parser.add_argument("config_json")

    return parser.parse_args()

def validate_route_inputs(
    locations,
    distance_table,
):
    """Validate consistency between route and distance data."""
    location_names = {
        location.name
        for location in locations
    }

    distance_locations = set(
        distance_table.keys()
    )

    missing_distances = (
        location_names - distance_locations
    )

    if missing_distances:
        raise ValueError(
            "Distance table is missing locations: "
            + ", ".join(sorted(missing_distances))
        )

    extra_distances = (
        distance_locations - location_names
    )

    if extra_distances:
        raise ValueError(
            "Distance table contains unknown locations: "
            + ", ".join(sorted(extra_distances))
        )


def main():
    """Run the route-optimization pipeline."""
    args = parse_arguments()

    try:
        # Load and validate configuration.
        config = load_config(args.config_json)
        validate_config(config)
        validate_constraints(config)

        # Load route data.
        locations = load_locations(args.route_csv)

        # Load the distance table.
        distance_table = load_distance_table(
            args.distance_csv
        )

        # Validate consistency between all inputs.
        validate_route_inputs(
            locations,
            distance_table,
        )

        validate_location_references(
            config,
            locations,
        )

        # Get configured start and end locations.
        start_location = config["start_location"]

        end_location = config.get(
            "end_location",
            start_location,
        )

        # Build and optimize the route.
        algorithm = config["optimization"]["algorithm"]

        use_two_opt = algorithm == "nearest_neighbor_2opt"

        start_time = time.perf_counter()

        route_results = build_route(
            [location.name for location in locations],
            distance_table,
            start_location,
            end_location,
            use_two_opt=use_two_opt,
        )

        runtime_seconds = time.perf_counter() - start_time

        # Check whether the optimized route satisfies
        # the configured constraints.
        constraints = config.get(
            "constraints",
            {},
        )

        check_route_feasibility(
            route_results["optimized_route"],
            locations,
            distance_table,
            constraints,
            start_location,
            end_location,
        )

        # Build final results.
        result = {
            "start_location": start_location,
            "end_location": end_location,
            **route_results,
            "algorithm": algorithm,
            "two_opt_applied": use_two_opt,
            "runtime_seconds": runtime_seconds,
            "feasible": True,
        }

        # Write results.
        with open(
            args.output_json,
            "w",
            encoding="utf-8",
        ) as file:
            json.dump(
                result,
                file,
                indent=2,
            )

    except Exception as error:
        print(
            f"Error processing route: {error}",
            file=sys.stderr,
        )
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())