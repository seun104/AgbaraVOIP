package database

import (
	"context" // Added import for context
	"database/sql"
	"fmt" // For error wrapping
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"time"
)

// ConnectDB establishes a connection to the PostgreSQL database.
// DSN (Data Source Name) example: "postgres://user:password@host:port/dbname?sslmode=disable"
func ConnectDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool properties
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection with a ping
	ctxPing, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Renamed ctx to ctxPing
	defer cancel()
	if err = db.PingContext(ctxPing); err != nil { // Used ctxPing
		db.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
