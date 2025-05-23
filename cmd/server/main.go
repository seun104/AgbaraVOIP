package main

import (
	"agbara-go/pkg/api"       // Already here
	"agbara-go/pkg/database"  // Already here
	"agbara-go/pkg/services"  // Already here
	"fmt"                     // Already here
	"log"                     // Already here
	"os"                      // Already here

	"github.com/gin-gonic/gin" // Already here
	// _ "github.com/jackc/pgx/v5/stdlib" // Should be in database/postgres.go or here
)

func main() {
	// Configuration (existing)
	dbDSN := os.Getenv("AGBARA_DB_DSN")
	if dbDSN == "" {
		log.Println("Warning: AGBARA_DB_DSN environment variable not set. Using default local DSN.")
        dbDSN = "postgres://user:password@localhost:5432/agbaradb?sslmode=disable"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Database Connection (existing)
	db, err := database.ConnectDB(dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Successfully connected to the database.")

	// Initialize Services
	callService := services.NewPostgresCallService(db)
	accountService := services.NewPostgresAccountService(db) // <-- ADD THIS

	// Initialize API Handlers
	callAPI := api.NewCallAPI(callService)
	accountAPI := api.NewAccountAPI(accountService) // <-- ADD THIS

	// Setup Gin Router (existing)
	router := gin.Default()

    // Health check endpoint (existing)
    router.GET("/health", func(c *gin.Context) {
        err := db.Ping()
        if err != nil {
            c.JSON(503, gin.H{"status": "error", "db_status": "unhealthy", "details": err.Error()})
            return
        }
        c.JSON(200, gin.H{"status": "ok", "db_status": "healthy"})
    })

	// Register API routes under a group, e.g., /api/v1
	apiV1Group := router.Group("/api/v1") 
	{ // Use a block for clarity if registering multiple API groups to apiV1Group
		callAPI.RegisterCallRoutes(apiV1Group)
		accountAPI.RegisterAccountRoutes(apiV1Group) // <-- ADD THIS
	}

	// Start HTTP Server (existing)
	serverAddr := fmt.Sprintf(":%s", httpPort)
	log.Printf("Starting server on %s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
