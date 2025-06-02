package api

import (
	"context"
	"fmt" // For Gin logger
	"net/http"
	"time"

	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/services" 
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ContextAuthAccountKey is defined in auth_middleware.go
// const ContextAuthAccountKey = "authenticated_account_sid" 

type Server struct {
	config             config.Config
	logger             *logrus.Logger
	router             *gin.Engine
	httpServer         *http.Server
	accountService     services.IAccountService     
	applicationService services.IApplicationService 
	callService        services.ICallService        
}

func NewServer(
	cfg config.Config, 
	logger *logrus.Logger, 
	accountService services.IAccountService,
	applicationService services.IApplicationService,
	callService services.ICallService, 
) *Server {
	if cfg.LogLevel != "debug" { 
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	
	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n", 
			param.ClientIP, param.TimeStamp.Format(time.RFC1123), param.Method, param.Path,
			param.Request.Proto, param.StatusCode, param.Latency, param.Request.UserAgent(), param.ErrorMessage,
		)
	}))
	// router.Use(LoggingMiddleware(logger)) // Re-enable if preferred over Gin default
	router.Use(gin.Recovery())

	srv := &Server{
		config:             cfg, logger:             logger, router:             router,
		accountService:     accountService, applicationService: applicationService, callService:        callService,        
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	baseRouter := s.router.Group("/v1")
	baseRouter.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "UP"}) })

	accountHandler := NewAccountHandler(s.accountService, s.logger)
	authMiddleware := BasicAuthMiddleware(s.accountService, s.logger) 
	baseRouter.POST("/accounts", accountHandler.CreateMasterAccount)
	
	authenticatedAccountRoutes := baseRouter.Group("/accounts/:account_sid")
	authenticatedAccountRoutes.Use(authMiddleware) 
	{
		authenticatedAccountRoutes.GET("", accountHandler.GetAccount)
		
		appHandler := NewApplicationHandler(s.applicationService, s.logger)
		appsRoutes := authenticatedAccountRoutes.Group("/applications")
		{
			appsRoutes.POST("", appHandler.CreateApplication)
			appsRoutes.GET("", appHandler.ListApplications)
			appsRoutes.GET("/:app_sid", appHandler.GetApplication)
			appsRoutes.PUT("/:app_sid", appHandler.UpdateApplication)
			appsRoutes.DELETE("/:app_sid", appHandler.DeleteApplication)
		}

		callHandler := NewCallHandler(s.callService, s.applicationService, s.logger) 
		callsRoutes := authenticatedAccountRoutes.Group("/calls")
		{
			callsRoutes.POST("", callHandler.CreateCall)
			callsRoutes.GET("/:call_sid", callHandler.GetCall)
			callsRoutes.GET("", callHandler.ListCalls)
		}
	}
}

func (s *Server) Start() error { 
	addr := ":" + s.config.ServerPort
	s.logger.Infof("Starting HTTP server on %s", addr)
	s.httpServer = &http.Server{ Addr: addr, Handler: s.router }
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down...")
	if s.httpServer == nil { s.logger.Info("HTTP server was not running."); return nil }
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) GetRouter() *gin.Engine { return s.router }


