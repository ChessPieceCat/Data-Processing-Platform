import pytest

from locations import (
    Location,
    load_locations,
    parse_demand,
    parse_priority,
    parse_time_value,
    parse_time_window,
    validate_unique_locations,
)


class TestLocation:
    def test_location_defaults(self):
        location = Location(
            id="1",
            name="Warehouse",
        )

        assert location.id == "1"
        assert location.name == "Warehouse"
        assert location.demand == 0
        assert location.priority == 0
        assert location.window_start == 0
        assert location.window_end == 0

    def test_location_values(self):
        location = Location(
            id="1",
            name="Customer A",
            demand=25,
            priority=2,
            window_start=480,
            window_end=1020,
        )

        assert location.id == "1"
        assert location.name == "Customer A"
        assert location.demand == 25
        assert location.priority == 2
        assert location.window_start == 480
        assert location.window_end == 1020

    def test_location_is_immutable(self):
        location = Location(
            id="1",
            name="Warehouse",
        )

        with pytest.raises(
            AttributeError,
        ):
            location.name = "Customer A"


class TestParseDemand:
    @pytest.mark.parametrize(
        "value, expected",
        [
            ("0", 0),
            ("1", 1),
            ("25", 25),
            ("1000", 1000),
        ],
    )
    def test_valid_demand(self, value, expected):
        assert parse_demand(value, 2) == expected

    def test_negative_demand(self):
        with pytest.raises(
            ValueError,
            match="Demand cannot be negative on row 2",
        ):
            parse_demand("-1", 2)

    @pytest.mark.parametrize(
        "value",
        [
            "abc",
            "10.5",
            "",
            "1a",
        ],
    )
    def test_invalid_demand(self, value):
        with pytest.raises(
            ValueError,
            match="Invalid demand",
        ):
            parse_demand(value, 2)


class TestParsePriority:
    @pytest.mark.parametrize(
        "value, expected",
        [
            ("0", 0),
            ("1", 1),
            ("5", 5),
            ("100", 100),
        ],
    )
    def test_valid_priority(self, value, expected):
        assert parse_priority(value, 2) == expected

    def test_negative_priority(self):
        with pytest.raises(
            ValueError,
            match="Priority cannot be negative on row 2",
        ):
            parse_priority("-1", 2)

    @pytest.mark.parametrize(
        "value",
        [
            "abc",
            "1.5",
            "",
            "1a",
        ],
    )
    def test_invalid_priority(self, value):
        with pytest.raises(
            ValueError,
            match="Invalid priority",
        ):
            parse_priority(value, 2)


class TestParseTimeValue:
    @pytest.mark.parametrize(
        "value, expected",
        [
            ("00:00", 0),
            ("08:00", 480),
            ("08:30", 510),
            ("12:00", 720),
            ("17:45", 1065),
            ("23:59", 1439),
        ],
    )
    def test_valid_time(self, value, expected):
        assert parse_time_value(
            value,
            "window_start",
            2,
        ) == expected

    def test_time_with_whitespace(self):
        assert parse_time_value(
            " 08:30 ",
            "window_start",
            2,
        ) == 510

    @pytest.mark.parametrize(
        "value",
        [
            "",
            "8",
            "08",
            "8:3",
            "8:30:00",
            "abc",
            "08-30",
        ],
    )
    def test_malformed_time(self, value):
        with pytest.raises(
            ValueError,
            match="Invalid window_start",
        ):
            parse_time_value(
                value,
                "window_start",
                2,
            )

    @pytest.mark.parametrize(
        "value",
        [
            "24:00",
            "25:30",
            "08:60",
            "99:99",
            "-1:00",
        ],
    )
    def test_out_of_range_time(self, value):
        with pytest.raises(
            ValueError,
            match="Invalid window_start",
        ):
            parse_time_value(
                value,
                "window_start",
                2,
            )


class TestParseTimeWindow:
    def test_valid_time_window(self):
        assert parse_time_window(
            "08:00",
            "17:00",
            2,
        ) == (480, 1020)

    def test_same_start_and_end(self):
        assert parse_time_window(
            "12:00",
            "12:00",
            2,
        ) == (720, 720)

    def test_rejects_start_after_end(self):
        with pytest.raises(
            ValueError,
            match="Window start cannot be greater than window end",
        ):
            parse_time_window(
                "17:00",
                "08:00",
                2,
            )

    def test_invalid_start_time(self):
        with pytest.raises(
            ValueError,
            match="Invalid window_start",
        ):
            parse_time_window(
                "25:00",
                "17:00",
                2,
            )

    def test_invalid_end_time(self):
        with pytest.raises(
            ValueError,
            match="Invalid window_end",
        ):
            parse_time_window(
                "08:00",
                "25:00",
                2,
            )


class TestValidateUniqueLocations:
    def test_unique_locations(self):
        locations = [
            Location(
                id="1",
                name="Warehouse",
            ),
            Location(
                id="2",
                name="Customer A",
            ),
            Location(
                id="3",
                name="Customer B",
            ),
        ]

        validate_unique_locations(locations)

    def test_duplicate_id(self):
        locations = [
            Location(
                id="1",
                name="Warehouse",
            ),
            Location(
                id="1",
                name="Customer A",
            ),
        ]

        with pytest.raises(
            ValueError,
            match="Duplicate location ID '1' on row 3",
        ):
            validate_unique_locations(locations)

    def test_duplicate_name(self):
        locations = [
            Location(
                id="1",
                name="Warehouse",
            ),
            Location(
                id="2",
                name="Warehouse",
            ),
        ]

        with pytest.raises(
            ValueError,
            match="Duplicate location name 'Warehouse' on row 3",
        ):
            validate_unique_locations(locations)


class TestLoadLocations:
    def write_csv(self, tmp_path, content):
        file_path = tmp_path / "route.csv"
        file_path.write_text(
            content,
            encoding="utf-8",
        )
        return file_path

    def test_load_valid_csv(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,08:00,17:00
2,Customer A,10,1,09:00,12:00
3,Customer B,5,2,10:00,14:00
""",
        )

        result = load_locations(file_path)

        assert result == [
            Location(
                id="1",
                name="Warehouse",
                demand=0,
                priority=0,
                window_start=480,
                window_end=1020,
            ),
            Location(
                id="2",
                name="Customer A",
                demand=10,
                priority=1,
                window_start=540,
                window_end=720,
            ),
            Location(
                id="3",
                name="Customer B",
                demand=5,
                priority=2,
                window_start=600,
                window_end=840,
            ),
        ]

    def test_missing_required_columns(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority
1,Warehouse,0,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Route CSV is missing required columns",
        ):
            load_locations(file_path)

    def test_empty_csv(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
""",
        )

        with pytest.raises(
            ValueError,
            match="Route CSV contains no locations",
        ):
            load_locations(file_path)

    def test_missing_location_id(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
,Warehouse,0,0,08:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Missing location ID on row 2",
        ):
            load_locations(file_path)

    def test_missing_location_name(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,,0,0,08:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Missing location name on row 2",
        ):
            load_locations(file_path)

    def test_malformed_demand(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,invalid,0,08:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Invalid demand",
        ):
            load_locations(file_path)

    def test_negative_demand(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,-5,0,08:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Demand cannot be negative",
        ):
            load_locations(file_path)

    def test_malformed_priority(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,invalid,08:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Invalid priority",
        ):
            load_locations(file_path)

    def test_negative_priority(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,-1,08:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Priority cannot be negative",
        ):
            load_locations(file_path)

    def test_invalid_start_time(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,25:00,17:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Invalid window_start",
        ):
            load_locations(file_path)

    def test_invalid_end_time(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,08:00,25:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Invalid window_end",
        ):
            load_locations(file_path)

    def test_start_after_end(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,17:00,08:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Window start cannot be greater than window end",
        ):
            load_locations(file_path)

    def test_duplicate_ids(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,08:00,17:00
1,Customer A,10,1,09:00,12:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Duplicate location ID '1' on row 3",
        ):
            load_locations(file_path)

    def test_duplicate_names(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
1,Warehouse,0,0,08:00,17:00
2,Warehouse,10,1,09:00,12:00
""",
        )

        with pytest.raises(
            ValueError,
            match="Duplicate location name 'Warehouse' on row 3",
        ):
            load_locations(file_path)

    def test_whitespace_is_stripped(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """id,name,demand,priority,window_start,window_end
 1 , Warehouse , 10 , 2 , 08:00 , 17:00
""",
        )

        result = load_locations(file_path)

        assert result == [
            Location(
                id="1",
                name="Warehouse",
                demand=10,
                priority=2,
                window_start=480,
                window_end=1020,
            )
        ]