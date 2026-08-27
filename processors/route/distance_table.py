import csv


def load_distance_table(file_path):
    """Load a complete CSV distance table into a nested dictionary."""
    distance_table = {}

    with open(
        file_path,
        newline="",
        encoding="utf-8",
    ) as csv_file:
        reader = csv.reader(csv_file)

        try:
            header = next(reader)
        except StopIteration as error:
            raise ValueError(
                "Distance table is empty."
            ) from error

        if len(header) < 2:
            raise ValueError(
                "Distance table must contain at least one location."
            )

        locations = [
            location.strip()
            for location in header[1:]
        ]

        if any(not location for location in locations):
            raise ValueError(
                "Distance table contains a blank location name."
            )

        if len(locations) != len(set(locations)):
            raise ValueError(
                "Distance table contains duplicate location names."
            )

        for row_number, row in enumerate(
            reader,
            start=2,
        ):
            if not row:
                continue

            location_name = row[0].strip()

            if not location_name:
                raise ValueError(
                    f"Missing location name on row {row_number}."
                )

            if location_name in distance_table:
                raise ValueError(
                    f"Duplicate location row "
                    f"{location_name!r} on row {row_number}."
                )

            if location_name not in locations:
                raise ValueError(
                    f"Unknown location {location_name!r} "
                    f"on row {row_number}."
                )

            if len(row) < len(locations) + 1:
                raise ValueError(
                    f"Row {row_number} does not contain "
                    "a distance for every location."
                )

            distance_table[location_name] = {}

            for index, other_location in enumerate(
                locations
            ):
                cell = row[index + 1].strip()

                if cell == "":
                    raise ValueError(
                        f"Missing distance from "
                        f"{location_name!r} to "
                        f"{other_location!r} "
                        f"on row {row_number}."
                    )

                try:
                    distance = float(cell)
                except ValueError as error:
                    raise ValueError(
                        f"Invalid distance {cell!r} "
                        f"on row {row_number}."
                    ) from error

                if distance < 0:
                    raise ValueError(
                        f"Distance cannot be negative "
                        f"on row {row_number}."
                    )

                distance_table[location_name][
                    other_location
                ] = distance

        missing_locations = [
            location
            for location in locations
            if location not in distance_table
        ]

        if missing_locations:
            raise ValueError(
                "Distance table is missing rows for: "
                + ", ".join(missing_locations)
            )

    return distance_table