import csv
from dataclasses import dataclass


@dataclass(frozen=True)
class Location:
    """Represents a location in a route optimization problem."""

    id: str
    name: str
    demand: int = 0
    priority: int = 0
    window_start: int = 0
    window_end: int = 0


def parse_demand(demand_str, row_number):
    """Parse and validate a non-negative location demand."""
    try:
        demand = int(demand_str)
    except ValueError as error:
        raise ValueError(
            f"Invalid demand {demand_str!r} on row {row_number}."
        ) from error

    if demand < 0:
        raise ValueError(
            f"Demand cannot be negative on row {row_number}."
        )

    return demand


def parse_priority(priority_str, row_number):
    """Parse and validate a non-negative location priority."""
    try:
        priority = int(priority_str)
    except ValueError as error:
        raise ValueError(
            f"Invalid priority {priority_str!r} on row {row_number}."
        ) from error

    if priority < 0:
        raise ValueError(
            f"Priority cannot be negative on row {row_number}."
        )

    return priority

def parse_time_value(value, field_name, row_number):
    """Parse HH:MM into minutes from midnight."""
    value = value.strip()

    if (
        len(value) != 5
        or value[2] != ":"
    ):
        raise ValueError(
            f"Invalid {field_name} {value!r} "
            f"on row {row_number}."
        )

    try:
        hours = int(value[:2])
        minutes = int(value[3:])
    except ValueError as error:
        raise ValueError(
            f"Invalid {field_name} {value!r} "
            f"on row {row_number}."
        ) from error

    if not 0 <= hours <= 23 or not 0 <= minutes <= 59:
        raise ValueError(
            f"Invalid {field_name} {value!r} "
            f"on row {row_number}."
        )

    return hours * 60 + minutes

def parse_time_window(
    window_start_str,
    window_end_str,
    row_number,
):
    """Parse and validate a location time window."""
    window_start = parse_time_value(
        window_start_str,
        "window_start",
        row_number,
    )

    window_end = parse_time_value(
        window_end_str,
        "window_end",
        row_number,
    )

    if window_start > window_end:
        raise ValueError(
            f"Window start cannot be greater than "
            f"window end on row {row_number}."
        )

    return window_start, window_end


def validate_unique_locations(locations):
    """Validate that location IDs and names are unique."""
    seen_ids = set()
    seen_names = set()

    for row_number, location in enumerate(
        locations,
        start=2,
    ):
        if location.id in seen_ids:
            raise ValueError(
                f"Duplicate location ID "
                f"{location.id!r} on row {row_number}."
            )

        seen_ids.add(location.id)

        if location.name in seen_names:
            raise ValueError(
                f"Duplicate location name "
                f"{location.name!r} on row {row_number}."
            )

        seen_names.add(location.name)


def load_locations(file_path):
    """Load and validate route locations from a CSV file."""
    locations = []

    with open(
        file_path,
        newline="",
        encoding="utf-8",
    ) as csv_file:
        reader = csv.DictReader(csv_file)

        required_columns = {
            "id",
            "name",
            "demand",
            "priority",
            "window_start",
            "window_end",
        }

        if not required_columns.issubset(
            reader.fieldnames or []
        ):
            raise ValueError(
                "Route CSV is missing required columns."
            )

        for row_number, row in enumerate(
            reader,
            start=2,
        ):
            location_id = row["id"].strip()
            name = row["name"].strip()

            if not location_id:
                raise ValueError(
                    f"Missing location ID on row {row_number}."
                )

            if not name:
                raise ValueError(
                    f"Missing location name on row {row_number}."
                )

            demand = parse_demand(
                row["demand"],
                row_number,
            )

            priority = parse_priority(
                row["priority"],
                row_number,
            )

            window_start, window_end = parse_time_window(
                row["window_start"],
                row["window_end"],
                row_number,
            )

            locations.append(
                Location(
                    id=location_id,
                    name=name,
                    demand=demand,
                    priority=priority,
                    window_start=window_start,
                    window_end=window_end,
                )
            )

    if not locations:
        raise ValueError(
            "Route CSV contains no locations."
        )

    validate_unique_locations(locations)

    return locations