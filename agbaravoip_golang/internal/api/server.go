package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/user/agbaravoip_golang/internal/auth"         // For InitJWTSecret & middlewares
	"github.com/user/agbaravoip_golang/internal/callcontrol" // For XMLProcessor
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/monitoring" // Added for Prometheus
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config             config.Config
	logger             *logrus.Logger // Base logger
	log                *logrus.Entry  // Server-level log entry
	router             *gin.Engine
	httpServer         *http.Server
	accountService     services.IAccountService
	applicationService services.IApplicationService
	callService        services.CallServicerForESL // Use the composite interface
	xmlProcessor       *callcontrol.XMLProcessor
	freeswitchService services.IFreeswitchServerService
	gatewayService    services.IGatewayService
	recordingService  services.IRecordingService // <-- New
}

func NewServer(
	cfg config.Config,
	logger *logrus.Logger,
	accountService services.IAccountService,
	applicationService services.IApplicationService,
	callService services.CallServicerForESL, // Use the composite interface
	xmlProcessor *callcontrol.XMLProcessor,
	freeswitchService services.IFreeswitchServerService,
	gatewayService services.IGatewayService,
	recordingService services.IRecordingService, // <-- New
	processedAdminSIDs []string,
) *Server {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize JWT Secret from config right at the start
	if cfg.Auth.JWTSecret == "" {
		logger.Fatal("CRITICAL: JWT Secret is not configured in app config!")
	}
	auth.InitJWTSecret(cfg.Auth.JWTSecret) // Call global init

	router := gin.New()
	// Gin's default logger uses time.RFC3339Nano
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP, param.TimeStamp.Format(time.RFC3339Nano), // Use RFC3339Nano for consistency
			param.Method, param.Path, param.Request.Proto,
			param.StatusCode, param.Latency, param.Request.UserAgent(), param.ErrorMessage,
		)
	}))
	router.Use(gin.Recovery())

	serverLogEntry := logger.WithField("component", "server")

	srv := &Server{
		config:             cfg,
		logger:             logger, // Store base logger
		log:                serverLogEntry, // Store server-level entry
		router:             router,
		accountService:     accountService,
		applicationService: applicationService,
		callService:        callService, // Store composite service
		xmlProcessor:       xmlProcessor,
		freeswitchService: freeswitchService,
		gatewayService:    gatewayService,
		recordingService:  recordingService, // <-- New
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	// All API routes are prefixed with /api; versioning within the group
	apiRouter := s.router.Group("/api")

	// Prometheus Metrics Endpoint (typically without /v1 prefix, but can be grouped if desired)
	// For simplicity, adding it under /api for now.
	// Could also be s.router.GET("/metrics", monitoring.PrometheusHandler()) for root level
	apiRouter.GET("/metrics", monitoring.PrometheusHandler())

	baseRouter := apiRouter.Group("/v1") // Group for v1 routes
	baseRouter.Use(monitoring.PrometheusMiddleware()) // Apply Prometheus middleware to all v1 routes

	baseRouter.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })

	// Logger entry for middlewares
	mwLogger := s.log.WithField("subcomponent", "middleware")

	// --- Auth Handler for Token Generation (Public) ---
	// AuthHandler needs CallServicerForESL which includes ValidateCredentials
	authH := NewAuthHandler(s.callService, s.logger, s.config.Auth.JWTSecret, s.config.Auth.JWTTokenDuration, processedAdminSIDs) // Pass processedAdminSIDs
	baseRouter.POST("/auth/token", authH.GenerateTokenHandler)

	// --- Account Handler (Master Account Creation - potentially public or admin only) ---
	// For master account creation, it might not be under /accounts/:account_sid or use JWT initially.
	// If it's a public signup, it won't have auth. If admin, might have different auth.
	// Assuming CreateMasterAccount might be more open or have specific controls not covered by typical JWT.
	accountHandler := NewAccountHandler(s.accountService, s.logger) // s.accountService is IAccountService
	baseRouter.POST("/accounts", accountHandler.CreateMasterAccount)

	// --- Authenticated Account-Scoped Routes ---
	jwtAuthMW := JWTMiddleware(mwLogger)
	accAccessMW := AccountAccessMiddleware(mwLogger)

	authenticatedAccountRoutes := baseRouter.Group("/accounts/:account_sid")
	authenticatedAccountRoutes.Use(jwtAuthMW)
	authenticatedAccountRoutes.Use(accAccessMW)
	{
		// Account specific (GET /, PUT /) - AccountHandler uses IAccountService
		authenticatedAccountRoutes.GET("", accountHandler.GetAccount)
		authenticatedAccountRoutes.PUT("", accountHandler.UpdateAccount)
		// Subaccounts - AccountHandler uses IAccountService
		authenticatedAccountRoutes.POST("/subaccounts", accountHandler.CreateSubAccount)
		authenticatedAccountRoutes.GET("/subaccounts", accountHandler.ListSubAccounts)

		// Applications under an account - ApplicationHandler uses IApplicationService
		appHandler := NewApplicationHandler(s.applicationService, s.logger)
		appsRoutes := authenticatedAccountRoutes.Group("/applications")
		{
			appsRoutes.POST("", appHandler.CreateApplication)
			appsRoutes.GET("", appHandler.ListApplications)
			appsRoutes.GET("/:app_sid", appHandler.GetApplication)
			appsRoutes.PUT("/:app_sid", appHandler.UpdateApplication)
			appsRoutes.DELETE("/:app_sid", appHandler.DeleteApplication)
		}

		// Calls under an account - CallHandler uses CallServicerForESL and IApplicationService
		callHandler := NewCallHandler(s.callService, s.applicationService, s.logger)
		callsRouteGroup := authenticatedAccountRoutes.Group("/calls") // Renamed for clarity
		{
			callsRouteGroup.POST("", callHandler.CreateCall)
			callsRouteGroup.GET("", callHandler.ListCalls) // List calls for the account

			// Routes for specific call SID
			callSpecificRoutes := callsRouteGroup.Group("/:call_sid")
			{
				callSpecificRoutes.GET("", callHandler.GetCall) // Get specific call details

				// Live Call Control Endpoints
				callSpecificRoutes.POST("/play", callHandler.PlayAudio)
				callSpecificRoutes.POST("/say", callHandler.SayText)
				callSpecificRoutes.POST("/dtmf", callHandler.SendDTMF)
				callSpecificRoutes.POST("/record", callHandler.RecordAction) // Handles start/stop
				callSpecificRoutes.POST("/hangup", callHandler.HangupLiveCall)
			}
		}

		// SMS resources under an account (e.g. list sent/received SMS for this account)
		// s.callService (type *services.CallService) implements ISMSService via delegation
		accountSMSHandler := NewAccountSMSHandler(s.callService, s.logger)
		smsMessagesRoutes := authenticatedAccountRoutes.Group("/sms/messages")
		{
			smsMessagesRoutes.POST("", accountSMSHandler.SendSMS)
			smsMessagesRoutes.GET("", accountSMSHandler.ListSMSMessages)
			smsMessagesRoutes.GET("/:sms_sid", accountSMSHandler.GetSMSMessage)
		}

		// Recording Management under an account
		recordingHandler := NewRecordingHandler(s.recordingService, s.logger)
		recordingsRoutes := authenticatedAccountRoutes.Group("/recordings")
		{
			recordingsRoutes.GET("", recordingHandler.ListRecordings)
			recordingsRoutes.GET("/:recording_sid", recordingHandler.GetRecording)
			recordingsRoutes.DELETE("/:recording_sid", recordingHandler.DeleteRecording)
		}
	}

	// --- Inbound SMS Endpoint (Typically from Gateway - specific auth, not user JWT) ---
	// SMSHandler needs CallServicerForESL (for GetApplicationByIncomingDID, RecordInboundSMS)
	// and XMLProcessor.
	smsH := NewSMSHandler(s.callService, s.xmlProcessor, s.logger)
	baseRouter.POST("/sms/inbound", smsH.InboundSMSEntrypoint)

	// --- Admin Routes ---
	// Protected by JWT and Admin Role check
	adminRouterGroup := baseRouter.Group("/admin")
	adminRouterGroup.Use(jwtAuthMW) // First, ensure user is authenticated via JWT
	adminRouterGroup.Use(AdminRoleAuthMiddleware(mwLogger)) // Then, ensure user has 'admin' role

	{
		// Freeswitch Server Admin Routes
		fsAdminHandler := NewAdminFreeswitchHandler(s.freeswitchService, s.logger)
		fsRoutes := adminRouterGroup.Group("/freeswitch-servers")
		{
			fsRoutes.POST("", fsAdminHandler.CreateFreeswitchServer)
			fsRoutes.GET("", fsAdminHandler.ListFreeswitchServers)
			fsRoutes.GET("/:fs_sid", fsAdminHandler.GetFreeswitchServer)
			fsRoutes.PUT("/:fs_sid", fsAdminHandler.UpdateFreeswitchServer)
			fsRoutes.DELETE("/:fs_sid", fsAdminHandler.DeleteFreeswitchServer)
		}

		// Gateway Admin Routes
		gwAdminHandler := NewAdminGatewayHandler(s.gatewayService, s.logger)
		gwRoutes := adminRouterGroup.Group("/gateways")
		{
			gwRoutes.POST("", gwAdminHandler.CreateGateway)
			gwRoutes.GET("", gwAdminHandler.ListGateways)
			gwRoutes.GET("/:gw_sid", gwAdminHandler.GetGateway)
			gwRoutes.PUT("/:gw_sid", gwAdminHandler.UpdateGateway)
			gwRoutes.DELETE("/:gw_sid", gwAdminHandler.DeleteGateway)
		}
	}

	s.log.Info("Server routes setup complete.")
}

func (s *Server) Start() error {
	addr := ":" + s.config.ServerPort
	s.logger.Infof("Starting HTTP server on %s", addr)
	s.httpServer = &http.Server{Addr: addr, Handler: s.router}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down...")
	if s.httpServer == nil {
		s.logger.Info("HTTP server was not running.")
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) GetRouter() *gin.Engine { return s.router }
