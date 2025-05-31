package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	ContextAuthAccountKey = "authenticated_account_sid"
	// ContextAuthAccountObjectKey = "authenticated_account_object" // Optional: to store the full account object
)

// BasicAuthMiddleware creates a middleware for HTTP Basic Authentication.
func BasicAuthMiddleware(accountService services.IAccountService, logger *logrus.Logger) gin.HandlerFunc {
	logEntry := logger.WithField("middleware", "basic_auth")

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logEntry.Warn("Authorization header missing")
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "basic" {
			logEntry.Warnf("Invalid Authorization header format: %s", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Invalid authorization format, expected Basic auth"})
			return
		}

		payload, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			logEntry.Warnf("Failed to decode base64 auth payload: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Invalid base64 encoding in auth header"})
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			logEntry.Warn("Invalid auth payload format (expected account_sid:token)")
			c.AbortWithStatusJSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Invalid auth payload format"})
			return
		}

		accountSid := pair[0]
		token := pair[1]

		if accountSid == "" || token == "" {
			logEntry.Warn("Account SID or token missing in auth payload")
			c.AbortWithStatusJSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Account SID and token are required"})
			return
		}

		account, err := accountService.ValidateCredentials(accountSid, token)
		if err != nil {
			logEntry.Warnf("Failed to validate credentials for SID %s: %v", accountSid, err)
			// To prevent account enumeration, always return a generic unauthorized error.
			// Specific errors (like account not active) are logged but not exposed to client.
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Invalid credentials"})
			return
		}

		logEntry.Infof("Successfully authenticated account SID: %s", account.SID)
		c.Set(ContextAuthAccountKey, account.SID)
		// c.Set(ContextAuthAccountObjectKey, account) // Optionally store the whole account object

		c.Next()
	}
}
