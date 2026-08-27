import pytest

from distance_table import load_distance_table


class TestLoadDistanceTable:
    def write_csv(self, tmp_path, content):
        file_path = tmp_path / "distances.csv"
        file_path.write_text(
            content,
            encoding="utf-8",
        )
        return file_path

    def test_load_valid_distance_table(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A,Customer B
Warehouse,0,10.5,20
Customer A,10.5,0,7.5
Customer B,20,7.5,0
""",
        )

        result = load_distance_table(file_path)

        assert result == {
            "Warehouse": {
                "Warehouse": 0.0,
                "Customer A": 10.5,
                "Customer B": 20.0,
            },
            "Customer A": {
                "Warehouse": 10.5,
                "Customer A": 0.0,
                "Customer B": 7.5,
            },
            "Customer B": {
                "Warehouse": 20.0,
                "Customer A": 7.5,
                "Customer B": 0.0,
            },
        }

    def test_empty_table(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            "",
        )

        with pytest.raises(
            ValueError,
            match="Distance table is empty",
        ):
            load_distance_table(file_path)

    def test_table_without_locations(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            "location\n",
        )

        with pytest.raises(
            ValueError,
            match="Distance table must contain at least one location",
        ):
            load_distance_table(file_path)

    def test_blank_location_in_header(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,,Customer A
Warehouse,0,10,20
,10,0,5
Customer A,20,5,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Distance table contains a blank location name",
        ):
            load_distance_table(file_path)

    def test_duplicate_location_in_header(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A,Customer A
Warehouse,0,10,20
Customer A,10,0,5
Customer B,20,5,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Distance table contains duplicate location names",
        ):
            load_distance_table(file_path)

    def test_missing_location_name_in_row(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A
Warehouse,0,10
,10,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Missing location name on row 3",
        ):
            load_distance_table(file_path)

    def test_duplicate_location_row(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A
Warehouse,0,10
Warehouse,10,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Duplicate location row 'Warehouse' on row 3",
        ):
            load_distance_table(file_path)

    def test_unknown_location_row(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A
Warehouse,0,10
Customer B,10,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Unknown location 'Customer B' on row 3",
        ):
            load_distance_table(file_path)

    def test_row_missing_distance_columns(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A,Customer B
Warehouse,0,10
Customer A,10,0,5
Customer B,20,5,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Row 2 does not contain a distance for every location",
        ):
            load_distance_table(file_path)

    def test_missing_distance_value(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A
Warehouse,0,
Customer A,10,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Missing distance from 'Warehouse' to 'Customer A'",
        ):
            load_distance_table(file_path)

    def test_malformed_distance(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A
Warehouse,0,abc
Customer A,10,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Invalid distance 'abc' on row 2",
        ):
            load_distance_table(file_path)

    def test_negative_distance(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A
Warehouse,0,-10
Customer A,10,0
""",
        )

        with pytest.raises(
            ValueError,
            match="Distance cannot be negative on row 2",
        ):
            load_distance_table(file_path)

    def test_missing_location_row(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location,Warehouse,Customer A,Customer B
Warehouse,0,10,20
Customer A,10,0,5
""",
        )

        with pytest.raises(
            ValueError,
            match="Distance table is missing rows for: Customer B",
        ):
            load_distance_table(file_path)

    def test_whitespace_is_stripped(self, tmp_path):
        file_path = self.write_csv(
            tmp_path,
            """location, Warehouse , Customer A
Warehouse,0,10
Customer A,10,0
""",
        )

        result = load_distance_table(file_path)

        assert "Warehouse" in result
        assert "Customer A" in result
        assert result["Warehouse"]["Customer A"] == 10.0