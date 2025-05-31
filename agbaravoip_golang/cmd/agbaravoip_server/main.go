package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"time"

	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/database"
	"github.com/user/agbaravoip_golang/internal/esl"
	"github.com/user/agbaravoip_golang/internal/logging"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/user/agbaravoip_golang/internal/api"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var _ *gorm.DB

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	appLogger := logging.InitLogger(cfg)
	appLogger.Info("AgbaraVOIP GoLang Server Starting...")
	appLogger.Debugf("Config loaded: ServerPort[%s], LogLevel[%s], DBHost[%s], FSAddress[%s], FSOutboundListen[%s]",
		cfg.ServerPort, cfg.LogLevel, cfg.DBHost, cfg.Freeswitch.FSAddress, cfg.Freeswitch.FSOutboundListenAddress)

	gormDb, errDb := database.InitDB(cfg)
	if errDb != nil {
		appLogger.Fatalf("Failed to initialize database: %v", errDb)
	}
	appLogger.Info("Database initialized and migrations applied.")

	accountService := services.NewAccountService(gormDb, appLogger)
	appLogger.Info("AccountService initialized.")

	eslInboundClient, errEsl := esl.NewFSInboundClient(cfg.Freeswitch)
	if errEsl != nil {
		appLogger.Warnf("Failed to initialize ESL Inbound Client: %v. Server will continue without Inbound ESL.", errEsl)
		eslInboundClient = nil
	} else {
		appLogger.Info("ESL Inbound Client initialized.")
	}

	eslOutboundServer, errOutbound := esl.NewFSOutboundServer(cfg, appLogger)
	if errOutbound != nil {
		appLogger.Fatalf("Failed to initialize ESL Outbound Server: %v", errOutbound)
	}
	appLogger.Info("ESL Outbound Server initialized.")

	httpServer := api.NewServer(cfg, appLogger, accountService)
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: httpServer.GetRouter(), // Corrected: Use GetRouter()
	}

	go func() {
		appLogger.Infof("HTTP server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	go func() {
		sig := <-shutdown
		appLogger.Infof("Received shutdown signal: %v. Starting graceful shutdown...", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer shutdownCancel()

		wg.Add(1)
		go func() {
			defer wg.Done()
			appLogger.Info("Shutting down HTTP server...")
			if err := srv.Shutdown(shutdownCtx); err != nil {
				appLogger.Errorf("HTTP server shutdown error: %v", err)
			} else {
				appLogger.Info("HTTP server shut down gracefully.")
			}
		}()

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

		if gormDb != nil {
			appLogger.Info("Closing database connection.")
			sqlDB, _ := gormDb.DB()
            if sqlDB != nil {
                sqlDB.Close()
            }
		}
		appLogger.Info("Shutdown complete.")
		os.Exit(0)
	}()

	appLogger.Info("Application started. Press Ctrl+C to exit.")
	select {}
}
