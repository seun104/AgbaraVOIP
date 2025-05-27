package main

import (
	"agbara-go/pkg/api"
	"agbara-go/pkg/database"
	"agbara-go/pkg/freeswitch" // <-- ADD THIS IMPORT
	"agbara-go/pkg/services"
	"fmt"
	"log"
	"net/http" // Added for health check http status codes
	"os"
	"strconv" // For parsing FS connection timeout and retries
	"time"    // For FS connection timeout

	"github.com/gin-gonic/gin"
	// _ "github.com/jackc/pgx/v5/stdlib" // Usually in database/postgres.go
)

func main() {
	// --- Configuration ---
	dbDSN := os.Getenv("AGBARA_DB_DSN")
	if dbDSN == "" {
		log.Println("Warning: AGBARA_DB_DSN environment variable not set. Using default local DSN.")
        dbDSN = "postgres://user:password@localhost:5432/agbaradb?sslmode=disable"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// FreeSWITCH ESL Configuration
	fsHost := os.Getenv("FS_HOST")
	if fsHost == "" {
		fsHost = "localhost"
		log.Printf("Warning: FS_HOST not set, defaulting to %s\n", fsHost)
	}
	fsPort := os.Getenv("FS_PORT")
	if fsPort == "" {
		fsPort = "8021"
		log.Printf("Warning: FS_PORT not set, defaulting to %s\n", fsPort)
	}
	fsPassword := os.Getenv("FS_PASSWORD")
	if fsPassword == "" {
		fsPassword = "YourESLPassword" // Change this default in production!
		log.Printf("Warning: FS_PASSWORD not set, defaulting to a placeholder. CHANGE THIS!\n")
	}
    fsTimeoutSecondsStr := os.Getenv("FS_TIMEOUT_SECONDS")
    fsTimeoutSeconds, err := strconv.Atoi(fsTimeoutSecondsStr)
    if err != nil || fsTimeoutSeconds <= 0 {
        fsTimeoutSeconds = 10 // Default to 10 seconds
        log.Printf("Warning: FS_TIMEOUT_SECONDS invalid or not set, defaulting to %d seconds\n", fsTimeoutSeconds)
    }
    fsMaxRetriesStr := os.Getenv("FS_MAX_RETRIES")
    fsMaxRetries, err := strconv.Atoi(fsMaxRetriesStr)
    if err != nil || fsMaxRetries <= 0 {
        fsMaxRetries = 3 // Default to 3 retries
        log.Printf("Warning: FS_MAX_RETRIES invalid or not set, defaulting to %d\n", fsMaxRetries)
    }


	// --- Database Connection ---
	db, err := database.ConnectDB(dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Successfully connected to the database.")

	// --- FreeSWITCH ESL Connection ---
	eslConn, err := freeswitch.NewESLConnection(fsHost, fsPort, fsPassword, time.Duration(fsTimeoutSeconds)*time.Second, fsMaxRetries)
	if err != nil {
		// Log as fatal if ESL connection is critical for startup,
		// or log as warning if application can run with limited functionality.
		// For call origination, it's likely critical.
		log.Fatalf("Failed to connect to FreeSWITCH ESL: %v. Ensure FreeSWITCH is running and configured.", err)
	}
	defer eslConn.Close() // Ensure ESL connection is closed when main exits


	// --- Initialize Services ---
	accountService := services.NewPostgresAccountService(db)
	// Pass eslConn to NewPostgresCallService
	callService := services.NewPostgresCallService(db, eslConn) // <-- MODIFIED HERE
	conferenceService := services.NewPostgresConferenceService(db) // Does not use ESL directly yet
	applicationService := services.NewPostgresApplicationService(db)

	// --- Initialize API Handlers ---
	callAPI := api.NewCallAPI(callService, accountService)
	accountAPI := api.NewAccountAPI(accountService)
	conferenceAPI := api.NewConferenceAPI(conferenceService, accountService)
	applicationAPI := api.NewApplicationAPI(applicationService, accountService)

	// --- Setup Gin Router ---
	router := gin.Default()
    router.GET("/health", func(c *gin.Context) {
        // DB Health
        dbErr := db.Ping()
        // FS Health (optional: add a simple status check if ESLConnection provides one)
        // fsHealthy := eslConn.IsConnected() // Hypothetical method
        
        healthStatus := gin.H{"status": "ok", "database": "healthy"}
        httpCode := http.StatusOK
        if dbErr != nil {
            healthStatus["database"] = "unhealthy"
            healthStatus["status"] = "error"
            healthStatus["db_details"] = dbErr.Error()
            httpCode = http.StatusServiceUnavailable
        }
        // if !fsHealthy { // If we had fsHealthy check
        //     healthStatus["freeswitch_esl"] = "unhealthy"
        //     healthStatus["status"] = "error"
        //     httpCode = http.StatusServiceUnavailable
        // }
        c.JSON(httpCode, healthStatus)
    })

	apiV1 := router.Group("/api/v1")
	accountAPI.RegisterAccountRoutes(apiV1)
	accountSpecificGroup := apiV1.Group("/Accounts/:accountSidInPath")
	{
		callAPI.RegisterCallRoutes(accountSpecificGroup)
		conferenceAPI.RegisterConferenceRoutes(accountSpecificGroup)
		applicationAPI.RegisterApplicationRoutes(accountSpecificGroup)
	}

	// --- Start HTTP Server ---
	serverAddr := fmt.Sprintf(":%s", httpPort)
	log.Printf("Starting server on %s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
