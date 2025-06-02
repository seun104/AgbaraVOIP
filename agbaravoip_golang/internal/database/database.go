package database

import (
	"fmt"
	"time"
	// Blank import for the PostgreSQL driver
	_ "github.com/lib/pq" 
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/domain" 
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"net/url" // Added for migrate DSN construction
)

// GormDB is the global GORM database instance (corrected previous comment about "DB")
var GormDB *gorm.DB 

// InitDB initializes the database connection and runs migrations.
func InitDB(cfg config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC search_path=%s", // Added search_path
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSchema,
	)

	var gormLogLevel gormlogger.LogLevel
	switch cfg.LogLevel {
	case "debug", "trace":
		gormLogLevel = gormlogger.Info 
	case "info":
		gormLogLevel = gormlogger.Warn
	case "warn":
		gormLogLevel = gormlogger.Warn
	case "error":
		gormLogLevel = gormlogger.Error
	default:
		gormLogLevel = gormlogger.Silent
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.New(
			logrus.StandardLogger(), 
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond, // Adjusted
				LogLevel:                  gormLogLevel,    
				IgnoreRecordNotFoundError: true,        
				Colorful:                  false,       
			},
		),
	}
	
	var err error
	GormDB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logrus.Info("Successfully connected to database. Running migrations...")

	username := url.QueryEscape(cfg.DBUser)
	password := url.QueryEscape(cfg.DBPassword)
	host := cfg.DBHost
	port := cfg.DBPort
	dbname := cfg.DBName
	schema := cfg.DBSchema
	
	migrationDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		username, password, host, port, dbname,
	)
	if schema != "" { // Append search_path if schema is defined
		migrationDSN = fmt.Sprintf("%s&search_path=%s", migrationDSN, schema)
	}


	m, err := migrate.New(
		"file://db/migrations", 
		migrationDSN,           
	)
	if err != nil {
		logrus.Warnf("Failed to initialize migrate instance (this might be ok in some CI environments): %v", err)
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			logrus.Errorf("Failed to apply migrations: %v", err)
			// Consider returning error here for stricter startup in dev
			// return nil, fmt.Errorf("failed to apply migrations: %w", err)
		} else if err == migrate.ErrNoChange {
			logrus.Info("No new database migrations to apply.")
		} else {
			logrus.Info("Database migrations applied successfully.")
		}
		
		version, dirty, verr := m.Version()
		if verr != nil {
			logrus.Warnf("Could not get migration version: %v", verr)
		} else {
			logrus.Infof("Current migration version: %d, Dirty: %v", version, dirty)
		}
		// Close migration source and db connections
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logrus.Warnf("Error closing migration source: %v", srcErr)
		}
		if dbErr != nil {
			logrus.Warnf("Error closing migration DB connection: %v", dbErr)
		}
	}

	err = GormDB.AutoMigrate(
		&domain.Account{}, 
		&domain.Application{},
		&domain.Call{}, 
	)
	if err != nil {
		// This might be a warning if manual migrations are primary
		logrus.Warnf("GORM AutoMigrate completed with potential issues: %v", err)
		// return nil, fmt.Errorf("gorm automigrate failed: %w", err)
	} else {
		logrus.Info("GORM AutoMigrate completed successfully.")
	}

	return GormDB, nil
}


