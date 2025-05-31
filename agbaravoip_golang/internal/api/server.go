package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config         config.Config
	logger         *logrus.Logger
	router         *gin.Engine
	httpServer     *http.Server
	accountService services.IAccountService
}

func NewServer(cfg config.Config, logger *logrus.Logger, accountService services.IAccountService) *Server {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	router.Use(gin.Recovery())

	srv := &Server{
		config:         cfg,
		logger:         logger,
		router:         router,
		accountService: accountService,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	baseRouter := s.router.Group("/v1")

	baseRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	accountHandler := NewAccountHandler(s.accountService, s.logger)
	authMiddleware := BasicAuthMiddleware(s.accountService, s.logger) // Create auth middleware instance

	// Public route for creating a master account
	baseRouter.POST("/accounts", accountHandler.CreateMasterAccount)

	// Authenticated routes for specific accounts
	// All routes under /accounts/:account_sid will use the authMiddleware
	// The middleware will ensure that c.Param("account_sid") is the authenticated user
	// or deny access.
	accountSpecificRoutes := baseRouter.Group("/accounts/:account_sid")
	accountSpecificRoutes.Use(authMiddleware)
	{
		// GET /v1/accounts/:account_sid
		// The GetAccount handler will use c.GetString(ContextAuthAccountKey) which is set by the middleware.
		// It should also verify that c.Param("account_sid") matches this context key.
		accountSpecificRoutes.GET("", accountHandler.GetAccount)
		// TODO: Add PUT handler: accountSpecificRoutes.PUT("", accountHandler.UpdateAccount)
		// TODO: Add POST handler for subaccounts: accountSpecificRoutes.POST("/subaccounts", accountHandler.CreateSubAccount)
		// TODO: Add GET handler for subaccounts: accountSpecificRoutes.GET("/subaccounts", accountHandler.GetSubAccounts)
	}
}

func (s *Server) Start(serverReady chan<- bool) error {
	addr := ":" + s.config.ServerPort
	s.logger.Infof("Starting HTTP server on %s", addr)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	if serverReady != nil {
		serverReady <- true
	}
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

func (s *Server) GetRouter() *gin.Engine {
    return s.router
}
