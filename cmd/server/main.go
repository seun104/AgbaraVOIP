package main

import (
	"agbara-go/pkg/api"
	"agbara-go/pkg/database"
	"agbara-go/pkg/freeswitch" 
	"agbara-go/pkg/freeswitch_events" // <-- ADDED IMPORT
	"agbara-go/pkg/services"
	"fmt"
	"log"
	"net/http" 
	"os"
	"os/signal" // For graceful shutdown
	"strconv" 
	"syscall"   // For graceful shutdown
	"time"    

	"github.com/gin-gonic/gin"
	"github.com/percipia/eslgo" // For eslgo.Message channel type
	// _ "github.com/jackc/pgx/v5/stdlib" 
)

func main() {
	// --- Configuration ---
	dbDSN := os.Getenv("AGBARA_DB_DSN")
	if dbDSN == "" { log.Println("Warning: AGBARA_DB_DSN not set. Using default."); dbDSN = "postgres://user:password@localhost:5432/agbaradb?sslmode=disable" }
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" { httpPort = "8080" }
	fsHost := os.Getenv("FS_HOST")
	if fsHost == "" { fsHost = "localhost"; log.Printf("Warning: FS_HOST not set, defaulting to %s\n", fsHost) }
	fsPort := os.Getenv("FS_PORT")
	if fsPort == "" { fsPort = "8021"; log.Printf("Warning: FS_PORT not set, defaulting to %s\n", fsPort) }
	fsPassword := os.Getenv("FS_PASSWORD")
	if fsPassword == "" { fsPassword = "YourESLPassword"; log.Printf("Warning: FS_PASSWORD not set, using default 'YourESLPassword'. CHANGE THIS!\n") }
    fsTimeoutSecondsStr := os.Getenv("FS_TIMEOUT_SECONDS")
    fsTimeoutSeconds, err := strconv.Atoi(fsTimeoutSecondsStr)
    if err != nil || fsTimeoutSeconds <= 0 { fsTimeoutSeconds = 10; log.Printf("Warning: FS_TIMEOUT_SECONDS invalid/not set, defaulting to %d s\n", fsTimeoutSeconds) }
    fsMaxRetriesStr := os.Getenv("FS_MAX_RETRIES")
    fsMaxRetries, err := strconv.Atoi(fsMaxRetriesStr)
    if err != nil || fsMaxRetries <= 0 { fsMaxRetries = 3; log.Printf("Warning: FS_MAX_RETRIES invalid/not set, defaulting to %d\n", fsMaxRetries) }
	fsEventSubscriptions := os.Getenv("FS_EVENT_SUBSCRIPTIONS")
	if fsEventSubscriptions == "" {
		fsEventSubscriptions = "CHANNEL_CREATE CHANNEL_ANSWER CHANNEL_HANGUP_COMPLETE CHANNEL_PROGRESS_MEDIA CUSTOM conference::maintenance RECORD_STOP"
		log.Printf("Warning: FS_EVENT_SUBSCRIPTIONS not set, defaulting to: %s\n", fsEventSubscriptions)
	}


	// --- Database Connection ---
	db, err := database.ConnectDB(dbDSN)
	if err != nil { log.Fatalf("Failed to connect to database: %v", err) }
	defer db.Close()
	log.Println("Successfully connected to the database.")

	// --- FreeSWITCH ESL Connection & Event Handling ---
	eslEventChannel := make(chan *eslgo.Message, 256) // Buffered channel

	eslConn, err := freeswitch.NewESLConnection(
		fsHost, fsPort, fsPassword, 
		time.Duration(fsTimeoutSeconds)*time.Second, fsMaxRetries,
		fsEventSubscriptions, eslEventChannel, 
	)
	if err != nil { 
		log.Fatalf("Failed to connect to FreeSWITCH ESL for events/commands: %v. Ensure FreeSWITCH is running and configured.", err)
	}
	// eslConn.Close() will be called by the graceful shutdown mechanism

	eventDispatcher := freeswitch_events.NewEventDispatcher(eslEventChannel)
	eventDispatcher.Start() // Starts its own goroutine
	// eventDispatcher.Stop() will be called by graceful shutdown


	// --- Initialize Services ---
	accountService := services.NewPostgresAccountService(db) 
	applicationService := services.NewPostgresApplicationService(db) 
	callService := services.NewPostgresCallService(db, eslConn, applicationService, accountService, eventDispatcher) 
	conferenceService := services.NewPostgresConferenceService(db, eventDispatcher) 
	
	// --- Initialize API Handlers ---
	accountAPI := api.NewAccountAPI(accountService)
	callAPI := api.NewCallAPI(callService, accountService) 
	conferenceAPI := api.NewConferenceAPI(conferenceService, accountService)
	applicationAPI := api.NewApplicationAPI(applicationService, accountService)
	voiceControlAPI := api.NewVoiceControlAPI(callService, applicationService) 

	// --- Setup Gin Router ---
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
        dbErr := db.Ping()
        // TODO: Add a check for ESL connection status if ESLConnection provides one
        healthStatus := gin.H{"status": "ok", "database": "healthy"} // "freeswitch_esl": "healthy" (pending check)
        httpCode := http.StatusOK
        if dbErr != nil {
            healthStatus["database"] = "unhealthy"; healthStatus["status"] = "error"; healthStatus["db_details"] = dbErr.Error()
            httpCode = http.StatusServiceUnavailable
        }
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
    voiceControlAPI.RegisterVoiceControlRoutes(apiV1)


	// --- Start HTTP Server & Graceful Shutdown ---
	serverAddr := fmt.Sprintf(":%s", httpPort)
	log.Printf("Starting HTTP server on %s", serverAddr)
	
    go func() {
        if err := router.Run(serverAddr); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start HTTP server: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down server...")

    if eslConn != nil {
        log.Println("Closing FreeSWITCH ESL connection...")
        eslConn.Close() 
    }

    if eventDispatcher != nil {
        log.Println("Stopping event dispatcher...")
        eventDispatcher.Stop() 
    }
    
	log.Println("Server exiting.")
}
