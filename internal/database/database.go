package database

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
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

type rdsSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func getDatabasePassword() string {
	secretARN := os.Getenv("DB_SECRET_ARN")

	// Local development continues to use DB_PASSWORD.
	if secretARN == "" {
		return getEnv(
			"DB_PASSWORD",
			"dev_password",
		)
	}

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(
			"Error loading AWS configuration:",
			err,
		)
	}

	client := secretsmanager.NewFromConfig(cfg)

	result, err := client.GetSecretValue(
		ctx,
		&secretsmanager.GetSecretValueInput{
			SecretId: &secretARN,
		},
	)
	if err != nil {
		log.Fatal(
			"Error retrieving database secret:",
			err,
		)
	}

	var secret rdsSecret

	if err := json.Unmarshal(
		[]byte(*result.SecretString),
		&secret,
	); err != nil {
		log.Fatal(
			"Error parsing database secret:",
			err,
		)
	}

	if secret.Password == "" {
		log.Fatal(
			"Database secret does not contain a password",
		)
	}

	return secret.Password
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

	password := getDatabasePassword()

	name := getEnv(
		"DB_NAME",
		"data_platform",
	)

	sslMode := getEnv(
		"DB_SSLMODE",
		"verify-full",
	)

	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		url.PathEscape(user),
		url.PathEscape(password),
		host,
		port,
		name,
		sslMode,
	)
}
