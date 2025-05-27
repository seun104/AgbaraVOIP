package main

import (
	"agbara-go/pkg/api"
	"agbara-go/pkg/database"
	"agbara-go/pkg/services"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	// _ "github.com/jackc/pgx/v5/stdlib" // Usually in database/postgres.go
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
	accountService := services.NewPostgresAccountService(db) 
	callService := services.NewPostgresCallService(db)
	conferenceService := services.NewPostgresConferenceService(db)
	applicationService := services.NewPostgresApplicationService(db) // <-- ADD THIS

	// Initialize API Handlers
	callAPI := api.NewCallAPI(callService, accountService)
	accountAPI := api.NewAccountAPI(accountService)
	conferenceAPI := api.NewConferenceAPI(conferenceService, accountService)
	applicationAPI := api.NewApplicationAPI(applicationService, accountService) // <-- ADD THIS

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

	// Register API routes
	apiV1 := router.Group("/api/v1")
	
	// Account routes (e.g., /api/v1/Accounts, /api/v1/Accounts/:accountSid, etc.)
	accountAPI.RegisterAccountRoutes(apiV1)

	// Account-specific resource routes (Calls, Conferences, Applications)
	// These are nested under /Accounts/:accountSidInPath
	accountSpecificGroup := apiV1.Group("/Accounts/:accountSidInPath")
	{
		callAPI.RegisterCallRoutes(accountSpecificGroup)
		conferenceAPI.RegisterConferenceRoutes(accountSpecificGroup)
		applicationAPI.RegisterApplicationRoutes(accountSpecificGroup) // <-- ADD THIS
	}

	// Start HTTP Server (existing)
	serverAddr := fmt.Sprintf(":%s", httpPort)
	log.Printf("Starting server on %s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
