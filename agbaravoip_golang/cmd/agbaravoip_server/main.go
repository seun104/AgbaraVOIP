package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"time"

	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/database"
	"github.com/user/agbaravoip_golang/internal/esl"
	"github.com/user/agbaravoip_golang/internal/logging"
	"github.com/sirupsen/logrus"
)

// var (
// 	appLogger *logrus.Logger // No longer needed as a global here, InitLogger returns it.
// )

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		// Use a basic logrus logger if config loading itself fails
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger and assign it to appLogger
	appLogger := logging.InitLogger(cfg)
	appLogger.Info("AgbaraVOIP GoLang Server Starting...")
	appLogger.Debugf("Config loaded: ServerPort[%s], LogLevel[%s], DBHost[%s], FSAddress[%s], FSOutboundListen[%s]",
		cfg.ServerPort, cfg.LogLevel, cfg.DBHost, cfg.Freeswitch.FSAddress, cfg.Freeswitch.FSOutboundListenAddress)


	// Initialize Database
	// InitDB now only returns an error. The DB connection is available via database.DB
	errDb := database.InitDB(cfg)
	if errDb != nil {
		appLogger.Fatalf("Failed to initialize database: %v", errDb)
	}
	// defer database.DB.Close() // Defer close until after shutdown sequence
	appLogger.Info("Database initialized and migrations applied.")

	// Initialize ESL Inbound Client
	// Pass only the Freeswitch part of the config and the logger
	eslInboundClient, errEsl := esl.NewFSInboundClient(cfg.Freeswitch)
	if errEsl != nil {
		// Allow to continue if FS is not available during dev/test for other parts
		appLogger.Warnf("Failed to initialize ESL Inbound Client: %v. Server will continue without Inbound ESL.", errEsl)
		eslInboundClient = nil // Ensure it's nil if connection failed
	} else {
		appLogger.Info("ESL Inbound Client initialized.")
		// Test Inbound Connection
		fsStatus, errFsStatus := eslInboundClient.GetFSStatus()
		if errFsStatus != nil {
			appLogger.Warnf("Could not get Freeswitch status via Inbound ESL: %v", errFsStatus)
		} else {
			appLogger.Infof("Freeswitch Status (via Inbound ESL): %s", fsStatus)
		}
	}


	// Initialize ESL Outbound Server
	// Pass the full config (as FSOutboundServer uses it) and the logger
	eslOutboundServer, errOutbound := esl.NewFSOutboundServer(cfg, appLogger)
	if errOutbound != nil {
		appLogger.Fatalf("Failed to initialize ESL Outbound Server: %v", errOutbound)
	}
	appLogger.Info("ESL Outbound Server initialized.")


	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	go func() {
		sig := <-shutdown
		appLogger.Infof("Received shutdown signal: %v. Starting graceful shutdown...", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if eslOutboundServer != nil {
				eslOutboundServer.Shutdown()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if eslInboundClient != nil {
				eslInboundClient.Close()
			}
		}()

		waitDone := make(chan struct{})
		go func() {
			defer close(waitDone)
			wg.Wait()
		}()

		select {
		case <-waitDone:
			appLogger.Info("All components shut down gracefully.")
		case <-shutdownCtx.Done():
			appLogger.Error("Shutdown timed out.")
		}

		if database.DB != nil { // Check if DB was successfully initialized
			appLogger.Info("Closing database connection.")
			database.DB.Close()
		}
		appLogger.Info("Shutdown complete.")
		os.Exit(0)
	}()

	appLogger.Info("Application started. Press Ctrl+C to exit.")
	select {}
}
