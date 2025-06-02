package main

import (
	"context"
	"net/http" 
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	
	"gorm.io/gorm" 

	"github.com/user/agbaravoip_golang/internal/api"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/database"
	"github.com/user/agbaravoip_golang/internal/esl"
	"github.com/user/agbaravoip_golang/internal/logging"
	"github.com/user/agbaravoip_golang/internal/services" 
	"github.com/sirupsen/logrus"
)

var dbConn *gorm.DB 

type AppConfigProvider struct { cfg config.Config }
func (p AppConfigProvider) GetESLOutboundServerListenAddress() string { return p.cfg.Freeswitch.FSOutboundListenAddress }

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil { logrus.Fatalf("Failed to load configuration: %v", err) }

	appLogger := logging.InitLogger(cfg)
	appLogger.Info("AgbaraVOIP GoLang Server Starting...")
	appLogger.Infof("Config loaded: %+v", cfg)

	dbConn, err = database.InitDB(cfg)
	if err != nil { appLogger.Fatalf("Failed to initialize database: %v", err) }
	appLogger.Info("Database initialized and migrations applied.")

	eslInboundClient, errEsl := esl.NewFSInboundClient(cfg.Freeswitch) 
	if errEsl != nil {
		appLogger.Warnf("Failed to initialize ESL Inbound Client: %v. Call origination may fail.", errEsl)
	} else {
		appLogger.Info("ESL Inbound Client initialized.")
		// Test command removed for cleaner startup in this script
	}

	accountService := services.NewAccountService(dbConn, appLogger)
	applicationService := services.NewApplicationService(dbConn, appLogger)
	configProvider := AppConfigProvider{cfg: cfg}
	callService := services.NewCallService(dbConn, eslInboundClient, applicationService, configProvider, appLogger) 
	
	apiServer := api.NewServer(cfg, appLogger, accountService, applicationService, callService) 
	
	go func() {
		if errSrv := apiServer.Start(); errSrv != nil && errSrv != http.ErrServerClosed { 
			appLogger.Fatalf("Could not start HTTP server: %v", errSrv)
		}
	}()

	eslOutboundServer, errOutbound := esl.NewFSOutboundServer(cfg, appLogger) 
	if errOutbound != nil {
		appLogger.Warnf("Failed to initialize ESL Outbound Server: %v.", errOutbound)
	} else {
		appLogger.Info("ESL Outbound Server initialized.")
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	go func() {
		sig := <-shutdown
		appLogger.Infof("Received shutdown signal: %v. Starting graceful shutdown...", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		if apiServer != nil { wg.Add(1); go func() { defer wg.Done(); if errSrvShutdown := apiServer.Shutdown(shutdownCtx); errSrvShutdown != nil { appLogger.Errorf("HTTP server shutdown error: %v", errSrvShutdown) } else { appLogger.Info("HTTP server shut down gracefully.") } }() }
		if eslOutboundServer != nil { wg.Add(1); go func() { defer wg.Done(); eslOutboundServer.Shutdown() }() }
		if eslInboundClient != nil { wg.Add(1); go func() { defer wg.Done(); eslInboundClient.Close(); appLogger.Info("ESL Inbound Client closed.") }() }
		
		waitDone := make(chan struct{}); go func() { defer close(waitDone); wg.Wait() }()
		select {
		case <-waitDone: appLogger.Info("All components shut down gracefully.")
		case <-shutdownCtx.Done(): appLogger.Error("Shutdown timed out.")
		}
		
		if dbConn != nil { appLogger.Info("Closing database connection."); sqlDB, _ := dbConn.DB(); if sqlDB != nil { sqlDB.Close() } }
		appLogger.Info("Shutdown complete."); os.Exit(0) 
	}()
	appLogger.Info("Application started. Press Ctrl+C to exit."); select {} 
}


