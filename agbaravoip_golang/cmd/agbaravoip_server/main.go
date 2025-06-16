package main
import ( "context"; "net/http"; "os"; "os/signal"; "strings"; "sync"; "syscall"; "time"; "gorm.io/gorm";
	"github.com/user/agbaravoip_golang/internal/api"; "github.com/user/agbaravoip_golang/internal/config";
	"github.com/user/agbaravoip_golang/internal/database"; "github.com/user/agbaravoip_golang/internal/esl";
	"github.com/user/agbaravoip_golang/internal/logging"; "github.com/user/agbaravoip_golang/internal/services";
	"github.com/user/agbaravoip_golang/internal/callcontrol"; "github.com/sirupsen/logrus" )
var dbConn *gorm.DB 
type AppConfigProvider struct { cfg config.Config }
func (p AppConfigProvider) GetESLOutboundServerListenAddress() string { 
	addr := p.cfg.Freeswitch.FSOutboundListenAddress; if addr == "" { addr = ":8084" } 
	if strings.HasPrefix(addr, ":") { logrus.Warnf("FSOutboundListenAddress %s starts with :, use resolvable host for FS in Docker.", addr); return "127.0.0.1" + addr }
	return addr
}
func main() {
	cfg, err := config.LoadConfig("."); if err != nil { logrus.Fatalf("Failed config: %v", err) }
	appLogger := logging.InitLogger(cfg); appLogger.Info("AgbaraVOIP Server Starting...")

	// Process AdminSIDs from config
	var adminSIDsList []string
	if len(cfg.Auth.AdminSIDs) == 1 && strings.Contains(cfg.Auth.AdminSIDs[0], ",") {
		// Handle case where Viper might load a single comma-separated string into the first element from ENV
		adminSIDsList = strings.Split(cfg.Auth.AdminSIDs[0], ",")
	} else if len(cfg.Auth.AdminSIDs) > 0 && cfg.Auth.AdminSIDs[0] != "" {
		// Handle case where Viper loaded a list directly (e.g. from YAML) or a single non-empty SID
		adminSIDsList = cfg.Auth.AdminSIDs
	} else {
		adminSIDsList = []string{} // Ensure it's an empty slice, not nil, if no admins are configured
	}

	// Trim whitespace from each SID in the list
	finalAdminSIDs := []string{} // Initialize as empty slice
	for _, sid := range adminSIDsList {
		trimmedSid := strings.TrimSpace(sid)
		if trimmedSid != "" {
			finalAdminSIDs = append(finalAdminSIDs, trimmedSid)
		}
	}
	appLogger.Infof("Processed Admin SIDs: %v", finalAdminSIDs)
	// Note: finalAdminSIDs will be passed to api.NewServer in a subsequent step.

	dbConn, err = database.InitDB(cfg); if err != nil { appLogger.Fatalf("Failed DB: %v", err) }
	appLogger.Info("DB initialized.")
	httpClient := &http.Client{Timeout: 10 * time.Second}
	xmlProcessor := callcontrol.NewXMLProcessor(appLogger, httpClient)
	eslInboundClient, err := esl.NewFSInboundClient(cfg, appLogger)
	if err != nil { appLogger.Warnf("Failed Inbound ESL: %v.", err) } else { appLogger.Info("Inbound ESL Client initialized.") }
	accountService := services.NewAccountService(dbConn, appLogger)
	applicationService := services.NewApplicationService(dbConn, appLogger)
	configProvider := AppConfigProvider{cfg: cfg}
	callService := services.NewCallService(dbConn, eslInboundClient, applicationService, accountService, configProvider, appLogger) // Corrected: Added accountService

	// Initialize new admin services
	freeswitchService := services.NewFreeswitchServerService(dbConn, appLogger)
	gatewayService := services.NewGatewayService(dbConn, appLogger)
	recordingService := services.NewRecordingService(dbConn, appLogger) // Initialize RecordingService

	apiServer := api.NewServer(cfg, appLogger, accountService, applicationService, callService, xmlProcessor, freeswitchService, gatewayService, recordingService, finalAdminSIDs)
	go func() { if err := apiServer.Start(); err != nil && err != http.ErrServerClosed { appLogger.Fatalf("HTTP server error: %v", err) } }()
	eslOutboundServer, err := esl.NewFSOutboundServer(cfg, appLogger, callService, xmlProcessor) // Corrected: Pass callService
	if err != nil { appLogger.Warnf("Failed Outbound ESL: %v.", err) } else { appLogger.Info("Outbound ESL Server initialized.") }
	shutdown := make(chan os.Signal, 1); signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	go func() {
		sig := <-shutdown; appLogger.Infof("Shutdown signal: %v...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second); defer cancel()
		if apiServer != nil { wg.Add(1); go func() { defer wg.Done(); if err := apiServer.Shutdown(ctx); err != nil { appLogger.Errorf("HTTP shutdown error: %v", err) } else { appLogger.Info("HTTP server down.") } }() }
		if eslOutboundServer != nil { wg.Add(1); go func() { defer wg.Done(); eslOutboundServer.Shutdown() }() }
		if eslInboundClient != nil { wg.Add(1); go func() { defer wg.Done(); eslInboundClient.Close(); appLogger.Info("ESL Inbound Client closed.") }() }
		done := make(chan struct{}); go func() { defer close(done); wg.Wait() }()
		select { case <-done: appLogger.Info("Graceful shutdown.") case <-ctx.Done(): appLogger.Error("Shutdown timed out.") }
		if dbConn != nil { appLogger.Info("Closing DB."); sqlDB, _ := dbConn.DB(); if sqlDB != nil { sqlDB.Close() } }
		appLogger.Info("Application shutdown."); os.Exit(0) 
	}();
	appLogger.Info("Application started. Ctrl+C to exit."); select {} 
}

