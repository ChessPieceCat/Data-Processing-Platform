package database

import (
	"database/sql"
	"embed"
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

const databaseURL = "postgresql://data_platform:dev_password@localhost:5432/data_platform?sslmode=disable"

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations runs the database migrations.
func RunMigrations() *migrate.Migrate {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		log.Fatal("Error loading database migrations:", err)
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		databaseURL,
	)
	if err != nil {
		log.Fatal("Error creating migration instance:", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal("Error running database migrations:", err)
	}

	return m
}

// CloseMigrations ensures migration resources are closed.
func CloseMigrations(m *migrate.Migrate) {
	sourceErr, dbErr := m.Close()

	if sourceErr != nil {
		log.Println("Migration source close error:", sourceErr)
	}

	if dbErr != nil {
		log.Println("Migration database close error:", dbErr)
	}
}

// OpenDatabase opens the application database connection.
func OpenDatabase() *sql.DB {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	return db
}
