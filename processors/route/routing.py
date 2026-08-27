def get_distance(location_one, location_two, distance_table):
    """Get the distance between two locations."""
    if location_one not in distance_table:
        raise ValueError(
            f"Unknown location: {location_one!r}."
        )

    if location_two not in distance_table:
        raise ValueError(
            f"Unknown location: {location_two!r}."
        )

    if location_two in distance_table[location_one]:
        return distance_table[location_one][location_two]

    if location_one in distance_table[location_two]:
        return distance_table[location_two][location_one]

    raise ValueError(
        f"No distance available between "
        f"{location_one!r} and {location_two!r}."
    )


def nearest_neighbor(
    locations,
    distance_table,
    start_location,
    end_location=None,
):
    """Build an initial route using nearest neighbor."""
    if end_location is None:
        end_location = start_location

    if start_location not in locations:
        raise ValueError(
            f"Start location {start_location!r} "
            "does not exist in the route data."
        )

    if end_location not in locations:
        raise ValueError(
            f"End location {end_location!r} "
            "does not exist in the route data."
        )

    unvisited = [
        location
        for location in locations
        if location not in {
            start_location,
            end_location,
        }
    ]

    route = [start_location]
    current_location = start_location

    while unvisited:
        nearest_location = None
        shortest_distance = float("inf")

        for location in unvisited:
            distance = get_distance(
                current_location,
                location,
                distance_table,
            )

            if distance < shortest_distance:
                shortest_distance = distance
                nearest_location = location

        if nearest_location is None:
            raise ValueError(
                "Unable to find the next location in the route."
            )

        route.append(nearest_location)
        unvisited.remove(nearest_location)
        current_location = nearest_location

    route.append(end_location)

    return route


def calculate_total_distance(
    route,
    distance_table,
):
    """Calculate the total distance of a complete route."""
    if len(route) < 2:
        return 0.0

    total = 0.0

    for index in range(len(route) - 1):
        total += get_distance(
            route[index],
            route[index + 1],
            distance_table,
        )

    return total


def two_opt(
    route,
    distance_table,
):
    """Improve a route using the 2-opt heuristic."""
    best_route = route.copy()

    best_distance = calculate_total_distance(
        best_route,
        distance_table,
    )

    improved = True

    while improved:
        improved = False

        for i in range(1, len(best_route) - 2):
            for j in range(
                i + 2,
                len(best_route) - 1,
            ):
                old_cost = (
                    get_distance(
                        best_route[i - 1],
                        best_route[i],
                        distance_table,
                    )
                    + get_distance(
                        best_route[j],
                        best_route[j + 1],
                        distance_table,
                    )
                )

                new_cost = (
                    get_distance(
                        best_route[i - 1],
                        best_route[j],
                        distance_table,
                    )
                    + get_distance(
                        best_route[i],
                        best_route[j + 1],
                        distance_table,
                    )
                )

                if new_cost < old_cost:
                    new_route = best_route.copy()

                    new_route[i : j + 1] = reversed(
                        new_route[i : j + 1]
                    )

                    new_distance = (
                        best_distance
                        - old_cost
                        + new_cost
                    )

                    if new_distance < best_distance:
                        best_route = new_route
                        best_distance = new_distance
                        improved = True

    return best_route


def build_route(
    locations,
    distance_table,
    start_location,
    end_location=None,
    use_two_opt=True,
):
    """Build an initial route and optionally optimize it."""
    initial_route = nearest_neighbor(
        locations,
        distance_table,
        start_location,
        end_location,
    )

    initial_distance = calculate_total_distance(
        initial_route,
        distance_table,
    )

    optimized_route = initial_route

    if use_two_opt:
        optimized_route = two_opt(
            initial_route,
            distance_table,
        )

    optimized_distance = calculate_total_distance(
        optimized_route,
        distance_table,
    )

    distance_improvement = (
        initial_distance - optimized_distance
    )

    improvement_percentage = (
        distance_improvement / initial_distance * 100
        if initial_distance > 0
        else 0.0
    )

    return {
        "initial_route": initial_route,
        "optimized_route": optimized_route,
        "initial_distance": initial_distance,
        "optimized_distance": optimized_distance,
        "distance_improvement": distance_improvement,
        "improvement_percentage": improvement_percentage,
    }