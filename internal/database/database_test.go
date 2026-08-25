package database

import "testing"

// TestOpenDatabase verifies that OpenDatabase returns a database connection
// that can successfully connect to PostgreSQL.
func TestOpenDatabase(t *testing.T) {
	db := OpenDatabase()
	if db == nil {
		t.Fatal("OpenDatabase returned nil")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}
}

// TestRunMigrations verifies that the database migrations can be applied
// successfully.
func TestRunMigrations(t *testing.T) {
	m := RunMigrations()
	if m == nil {
		t.Fatal("RunMigrations returned nil")
	}
	defer CloseMigrations(m)

	version, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("failed to retrieve migration version: %v", err)
	}

	if dirty {
		t.Fatal("database migration is marked as dirty")
	}

	if version == 0 {
		t.Fatal("expected at least one migration to be applied")
	}
}

// TestCloseMigrations verifies that migration resources can be closed safely.
func TestCloseMigrations(t *testing.T) {
	m := RunMigrations()
	if m == nil {
		t.Fatal("RunMigrations returned nil")
	}

	// CloseMigrations logs close errors rather than returning them,
	// so this test primarily verifies that it does not panic.
	CloseMigrations(m)
}

// TestRunMigrationsTwice verifies that calling RunMigrations again after
// migrations have already been applied succeeds.
func TestRunMigrationsTwice(t *testing.T) {
	m1 := RunMigrations()
	if m1 == nil {
		t.Fatal("first RunMigrations returned nil")
	}
	CloseMigrations(m1)

	m2 := RunMigrations()
	if m2 == nil {
		t.Fatal("second RunMigrations returned nil")
	}
	defer CloseMigrations(m2)

	_, dirty, err := m2.Version()
	if err != nil {
		t.Fatalf("failed to retrieve migration version: %v", err)
	}

	if dirty {
		t.Fatal("database migration is marked as dirty after second run")
	}
}
