package main

import (
	"agbara-go/pkg/api"
	"agbara-go/pkg/database"
	"agbara-go/pkg/freeswitch" 
	"agbara-go/pkg/services"
	"fmt"
	"log"
	"net/http" // Added for health check status codes
	"os"
	"strconv" 
	"time"    

	"github.com/gin-gonic/gin"
	// _ "github.com/jackc/pgx/v5/stdlib" 
)

func main() {
	// --- Configuration --- (existing)
	dbDSN := os.Getenv("AGBARA_DB_DSN")
	if dbDSN == "" {
		log.Println("Warning: AGBARA_DB_DSN environment variable not set. Using default local DSN.")
        dbDSN = "postgres://user:password@localhost:5432/agbaradb?sslmode=disable"
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" { httpPort = "8080" }
	fsHost := os.Getenv("FS_HOST")
	if fsHost == "" { fsHost = "localhost"; log.Printf("Warning: FS_HOST not set, defaulting to %s\n", fsHost) } // Corrected newline
	fsPort := os.Getenv("FS_PORT")
	if fsPort == "" { fsPort = "8021"; log.Printf("Warning: FS_PORT not set, defaulting to %s\n", fsPort) } // Corrected newline
	fsPassword := os.Getenv("FS_PASSWORD")
	if fsPassword == "" { fsPassword = "YourESLPassword"; log.Printf("Warning: FS_PASSWORD not set, defaulting to a placeholder. CHANGE THIS!\n") } // Corrected newline
    fsTimeoutSecondsStr := os.Getenv("FS_TIMEOUT_SECONDS")
    fsTimeoutSeconds, err := strconv.Atoi(fsTimeoutSecondsStr)
    if err != nil || fsTimeoutSeconds <= 0 { fsTimeoutSeconds = 10; log.Printf("Warning: FS_TIMEOUT_SECONDS invalid or not set, defaulting to %d seconds\n", fsTimeoutSeconds) } // Corrected newline
    fsMaxRetriesStr := os.Getenv("FS_MAX_RETRIES")
    fsMaxRetries, err := strconv.Atoi(fsMaxRetriesStr)
    if err != nil || fsMaxRetries <= 0 { fsMaxRetries = 3; log.Printf("Warning: FS_MAX_RETRIES invalid or not set, defaulting to %d\n", fsMaxRetries) } // Corrected newline

	// --- Database Connection --- (existing)
	db, err := database.ConnectDB(dbDSN)
	if err != nil { log.Fatalf("Failed to connect to database: %v", err) }
	defer db.Close()
	log.Println("Successfully connected to the database.")

	// --- FreeSWITCH ESL Connection --- (existing)
	eslConn, err := freeswitch.NewESLConnection(fsHost, fsPort, fsPassword, time.Duration(fsTimeoutSeconds)*time.Second, fsMaxRetries)
	if err != nil { log.Fatalf("Failed to connect to FreeSWITCH ESL: %v. Ensure FreeSWITCH is running and configured.", err) }
	defer eslConn.Close()

	// --- Initialize Services ---
	accountService := services.NewPostgresAccountService(db)
	applicationService := services.NewPostgresApplicationService(db) // <-- Initialized before CallService
	callService := services.NewPostgresCallService(db, eslConn, applicationService) // Pass appService
	conferenceService := services.NewPostgresConferenceService(db) 
	
	// --- Initialize API Handlers ---
	accountAPI := api.NewAccountAPI(accountService)
	callAPI := api.NewCallAPI(callService, accountService) 
	conferenceAPI := api.NewConferenceAPI(conferenceService, accountService)
	applicationAPI := api.NewApplicationAPI(applicationService, accountService)
	voiceControlAPI := api.NewVoiceControlAPI(callService, applicationService) // <-- ADD THIS

	// --- Setup Gin Router --- (existing)
	router := gin.Default()
    router.GET("/health", func(c *gin.Context) {
        dbErr := db.Ping()
        healthStatus := gin.H{"status": "ok", "database": "healthy"}
        httpCode := http.StatusOK
        if dbErr != nil {
            healthStatus["database"] = "unhealthy"; healthStatus["status"] = "error"; healthStatus["db_details"] = dbErr.Error()
            httpCode = http.StatusServiceUnavailable
        }
        c.JSON(httpCode, healthStatus)
    })

	apiV1 := router.Group("/api/v1")
	
	accountAPI.RegisterAccountRoutes(apiV1) // For /api/v1/Accounts, /api/v1/Accounts/Master etc.

	// Account-specific resource routes (Calls, Conferences, Applications)
	accountSpecificGroup := apiV1.Group("/Accounts/:accountSidInPath")
	{
		callAPI.RegisterCallRoutes(accountSpecificGroup)
		conferenceAPI.RegisterConferenceRoutes(accountSpecificGroup)
		applicationAPI.RegisterApplicationRoutes(accountSpecificGroup)
	}

    // Voice Control routes (e.g., /api/v1/voice/control - for FreeSWITCH to call into)
    voiceControlAPI.RegisterVoiceControlRoutes(apiV1) // <-- ADD THIS (registers under /api/v1/voice)


	// --- Start HTTP Server --- (existing)
	serverAddr := fmt.Sprintf(":%s", httpPort)
	log.Printf("Starting server on %s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
