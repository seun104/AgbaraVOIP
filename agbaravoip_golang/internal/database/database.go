package database

import (
	"database/sql" // Ensure this is considered used
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/logging"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/user/agbaravoip_golang/internal/domain"
)

// DB is now a GORM DB instance
var GormDB *gorm.DB
var _ *sql.DB // Explicitly use sql.DB type to satisfy compiler if direct GormDB.DB() usage isn't enough

// InitDB initializes the GORM database connection and runs migrations.
// It returns the *gorm.DB instance for use in services.
func InitDB(cfg config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC search_path=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSchema,
	)

	logging.Logger.Infof("Connecting to database (GORM): %s:%s/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)
	var err error
	GormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open GORM database connection: %w", err)
	}

	// Assign to the package-level var _ *sql.DB to ensure sql package is used.
	// This also retrieves the underlying *sql.DB for pinging.
	underlyingSqlDB, err := GormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB from GORM: %w", err)
	}
	_ = underlyingSqlDB // Use it to avoid "declared and not used" for underlyingSqlDB itself.


	err = underlyingSqlDB.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database via GORM: %w", err)
	}
	logging.Logger.Info("Successfully connected to the database (GORM).")

	username := url.QueryEscape(cfg.DBUser)
	password := url.QueryEscape(cfg.DBPassword)
	host := cfg.DBHost
	port := cfg.DBPort
	dbname := cfg.DBName
	schema := cfg.DBSchema

	migrateDSN := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		username, password, host, port, dbname, schema)

	logging.Logger.Infof("Running database migrations from file://db/migrations")
	m, err := migrate.New("file://db/migrations", migrateDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logging.Logger.Errorf("Failed to apply migrations: %v", err)
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}
	sourceErr, dbErr := m.Close()
    if sourceErr != nil {
        logging.Logger.Warnf("Error closing migration source: %v", sourceErr)
    }
    if dbErr != nil {
        logging.Logger.Warnf("Error closing migration database connection used by migrate: %v", dbErr)
    }
	logging.Logger.Info("Database migrations applied successfully (or no changes).")

	logging.Logger.Info("Running GORM AutoMigrate for Account and Application structs...")
	err = GormDB.AutoMigrate(&domain.Account{}, &domain.Application{})
	if err != nil {
		logging.Logger.Warnf("GORM AutoMigrate finished with potential warnings/errors (often ignorable if manual migrations are correct): %v", err)
	} else {
		logging.Logger.Info("GORM AutoMigrate completed.")
	}

	return GormDB, nil
}
