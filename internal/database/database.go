package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations runs the database migrations.
func RunMigrations() *migrate.Migrate {
	sourceDriver, err := iofs.New(
		migrationsFS,
		"migrations",
	)
	if err != nil {
		log.Fatal(
			"Error loading database migrations:",
			err,
		)
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		getDatabaseURL(),
	)
	if err != nil {
		log.Fatal(
			"Error creating migration instance:",
			err,
		)
	}

	if err := m.Up(); err != nil &&
		!errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(
			"Error running database migrations:",
			err,
		)
	}

	return m
}

// CloseMigrations ensures migration resources are closed.
func CloseMigrations(m *migrate.Migrate) {
	sourceErr, dbErr := m.Close()

	if sourceErr != nil {
		log.Println(
			"Migration source close error:",
			sourceErr,
		)
	}

	if dbErr != nil {
		log.Println(
			"Migration database close error:",
			dbErr,
		)
	}
}

// OpenDatabase opens the application database connection.
func OpenDatabase() *sql.DB {
	db, err := sql.Open(
		"postgres",
		getDatabaseURL(),
	)
	if err != nil {
		panic(err)
	}

	return db
}

func getEnv(
	key string,
	defaultValue string,
) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getDatabaseURL() string {
	host := getEnv(
		"DB_HOST",
		"localhost",
	)

	port := getEnv(
		"DB_PORT",
		"5432",
	)

	user := getEnv(
		"DB_USER",
		"data_platform",
	)

	password := getEnv(
		"DB_PASSWORD",
		"dev_password",
	)

	name := getEnv(
		"DB_NAME",
		"data_platform",
	)

	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		name,
	)
}
