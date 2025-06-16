package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/user/agbaravoip_golang/internal/auth" // For JWT
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4" // For JWT error types
	"github.com/sirupsen/logrus"
)

// ContextKey type for setting values in Gin context to avoid collisions
type ContextKey string

const (
	// For Basic Auth
	ContextAuthAccountKey ContextKey = "authenticated_account_sid"
	// ContextAuthAccountObjectKey = "authenticated_account_object"

	// For JWT Auth
	ContextKeyAccountSID ContextKey = "jwt_account_sid" // Renamed to avoid conflict if both used (though unlikely)
	ContextKeyUserRole   ContextKey = "jwt_user_role"
	ContextKeyClaims     ContextKey = "jwt_claims"
)

// AccountAccessMiddleware ensures that the AccountSID from the JWT token
// matches the :account_sid path parameter.
// It should be placed AFTER JWTMiddleware in the middleware chain.
func AccountAccessMiddleware(logger *logrus.Entry) gin.HandlerFunc {
    var log *logrus.Entry
	if logger != nil {
		log = logger.WithField("middleware", "account_access_check")
	} else {
		defaultLogger := logrus.New()
		log = logrus.NewEntry(defaultLogger).WithField("middleware", "account_access_check")
		log.Warn("AccountAccessMiddleware initialized with no logger provided, using default logrus instance.")
	}

    return func(c *gin.Context) {
        tokenAccountSID, exists := c.Get(string(ContextKeyAccountSID)) // Using the JWT specific key
        if !exists {
            log.Error("AccountAccessMiddleware: AccountSID not found in JWT claims (JWTMiddleware should run first)")
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access forbidden: missing token claims"})
            return
        }

        tokenAccountSIDStr, ok := tokenAccountSID.(string)
        if !ok || tokenAccountSIDStr == "" {
            log.Error("AccountAccessMiddleware: AccountSID in token is invalid or empty")
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access forbidden: invalid token claims"})
            return
        }

        pathAccountSID := c.Param("account_sid")
        if pathAccountSID == "" {
            log.Warn("AccountAccessMiddleware: :account_sid path parameter missing from route definition. Skipping check.")
            c.Next()
            return
        }

        if tokenAccountSIDStr != pathAccountSID {
            log.Warnf("Forbidden access attempt: Token AccountSID '%s' does not match Path AccountSID '%s'", tokenAccountSIDStr, pathAccountSID)
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access forbidden: you cannot access resources of another account."})
            return
        }

        log.Debugf("AccountAccessMiddleware: Access granted for AccountSID '%s' to path '%s'", tokenAccountSIDStr, c.Request.URL.Path)
        c.Next()
    }
}


// JWTMiddleware creates a Gin middleware for JWT authentication.
func JWTMiddleware(logger *logrus.Entry) gin.HandlerFunc {
	if !auth.IsJWTSecretInitialized() {
		if logger != nil {
			logger.Fatal("CRITICAL: JWT Authentication Middleware loaded but JWT secret key is not initialized!")
		} else {
			logrus.Fatal("CRITICAL: JWT Authentication Middleware loaded but JWT secret key is not initialized!")
		}
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "JWT system not configured"})
		}
	}

	var log *logrus.Entry
	if logger != nil {
		log = logger.WithField("middleware", "jwt_auth")
	} else {
		defaultLogger := logrus.New()
		log = logrus.NewEntry(defaultLogger).WithField("middleware", "jwt_auth")
		log.Warn("JWTMiddleware initialized with no logger provided, using default logrus instance.")
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Debug("Authorization header missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && strings.ToLower(parts[0]) == "bearer") {
			log.Debug("Authorization header format must be Bearer {token}")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			log.Debug("Bearer token missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token missing"})
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Warnf("Token validation failed: %v", err)
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Malformed token"})
					return
				} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token is expired"})
					return
				} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token not active yet"})
					return
				}
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set(string(ContextKeyClaims), claims)
		c.Set(string(ContextKeyAccountSID), claims.AccountSID) // Uses the new jwt_account_sid
		if claims.Role != "" {
			c.Set(string(ContextKeyUserRole), claims.Role)
		}

		log.Debugf("JWT authentication successful for AccountSID: %s, Role: %s", claims.AccountSID, claims.Role)
		c.Next()
	}
}

// AdminRoleAuthMiddleware ensures that the JWT token contains an 'admin' role.
// It should be placed AFTER JWTMiddleware in the middleware chain.
func AdminRoleAuthMiddleware(logger *logrus.Entry) gin.HandlerFunc {
	var log *logrus.Entry
	if logger != nil {
		log = logger.WithField("middleware", "admin_role_auth")
	} else {
		defaultLogger := logrus.New()
		log = logrus.NewEntry(defaultLogger).WithField("middleware", "admin_role_auth")
		log.Warn("AdminRoleAuthMiddleware initialized with no logger provided, using default logrus instance.")
	}

	return func(c *gin.Context) {
		userRoleVal, exists := c.Get(string(ContextKeyUserRole))
		if !exists {
			log.Error("AdminRoleAuthMiddleware: UserRole not found in JWT claims (JWTMiddleware should run first and set this)")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access forbidden: missing role claims"})
			return
		}

		userRole, ok := userRoleVal.(string)
		if !ok || userRole == "" {
			log.Error("AdminRoleAuthMiddleware: UserRole in token is invalid or empty")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access forbidden: invalid role claims"})
			return
		}

		if strings.ToLower(userRole) != "admin" {
			log.Warnf("Forbidden access attempt: User with role '%s' tried to access admin-only route '%s'", userRole, c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access forbidden: admin role required."})
			return
		}

		log.Debugf("AdminRoleAuthMiddleware: Access granted for admin user to path '%s'", c.Request.URL.Path)
		c.Next()
	}
}


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
