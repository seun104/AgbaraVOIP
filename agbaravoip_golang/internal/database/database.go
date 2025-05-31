package database

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // For PostgreSQL driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // For reading migrations from files
	_ "github.com/lib/pq"                                     // PostgreSQL driver for database/sql
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/logging"
)

var DB *sql.DB

// InitDB initializes the database connection and runs migrations.
func InitDB(cfg config.Config) error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		url.QueryEscape(cfg.DBUser),
		url.QueryEscape(cfg.DBPassword),
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSchema,
	)

	logging.Logger.Infof("Connecting to database: %s:%s/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logging.Logger.Info("Successfully connected to the database.")

	// Run migrations
	// The path to migrations is relative to the execution directory of the binary.
	// If running `go run cmd/agbaravoip_server/main.go` from `agbaravoip_golang` dir,
	// then "file://db/migrations" should work.
	// For a Docker container, ensure `db/migrations` is copied to an accessible path.
	// For now, this assumes migrations are in `db/migrations` relative to the project root.
	migrationsPath := "file://db/migrations"

	// For docker, the path needs to be relative to the workdir in Dockerfile, e.g. /root/db/migrations
	// This might need adjustment based on where the binary is run or how migrations are packaged.
	// A common pattern is to have migrations path configurable or relative to the binary.

	logging.Logger.Infof("Running database migrations from %s", migrationsPath)
	m, err := migrate.New(migrationsPath, connStr)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	sourceErr, dbErr := m.Close()
    if sourceErr != nil {
        logging.Logger.Warnf("Error closing migration source: %v", sourceErr)
    }
    if dbErr != nil {
        logging.Logger.Warnf("Error closing migration database connection: %v", dbErr)
    }


	logging.Logger.Info("Database migrations applied successfully (or no changes).")
	return nil
}
