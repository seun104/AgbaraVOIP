package api

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/user/agbaravoip_golang/internal/auth" // For JWTClaims and GenerateToken
	"github.com/user/agbaravoip_golang/internal/domain" // For domain.Account
	"github.com/user/agbaravoip_golang/internal/services" // For services.CallServicerForESL
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v4" // Corrected import alias if needed, or direct use
	"github.com/sirupsen/logrus"
)

// AuthHandler handles token generation.
type AuthHandler struct {
	credentialValidator services.CallServicerForESL // Using CallServicerForESL as passed from server.go
	logger              *logrus.Entry
	jwtDuration         time.Duration
	adminSIDs           map[string]bool // Loaded from config: map of AccountSIDs that are admins
}

// NewAuthHandler creates a new AuthHandler.
// server.go passes `s.callService` which is `services.CallServicerForESL`. This interface
// needs to expose `ValidateCredentials`.
func NewAuthHandler(validator services.CallServicerForESL, baseLogger *logrus.Logger, jwtSecret string, jwtDuration time.Duration, adminSIDs []string) *AuthHandler {
	// JWT secret is initialized globally via auth.InitJWTSecret(jwtSecret) in server.go

	adminMap := make(map[string]bool)
	for _, sid := range adminSIDs {
		adminMap[sid] = true
	}

	return &AuthHandler{
		credentialValidator: validator,
		logger:              baseLogger.WithField("handler", "auth"),
		jwtDuration:         jwtDuration,
		adminSIDs:           adminMap,
	}
}

type GenerateTokenResponse struct {
	Token        string `json:"token"`
	ExpiresAt    string `json:"expires_at"` // RFC3339 format
	TokenType    string `json:"token_type"`
}

// GenerateTokenHandler godoc
// @Summary Generate a JWT token
// @Description Authenticates using Basic Auth (AccountSID:AuthToken) and returns a JWT token if valid.
// @Tags Authentication
// @Produce  json
// @Success 200 {object} GenerateTokenResponse "Successfully generated token"
// @Failure 400 {object} GenericErrorResponse "Invalid request format (e.g., missing auth header)"
// @Failure 401 {object} GenericErrorResponse "Invalid credentials"
// @Failure 500 {object} GenericErrorResponse "Internal server error (e.g., token generation failed)"
// @Header 401 {string} WWW-Authenticate "Basic realm="Restricted""
// @Router /api/v1/auth/token [post]
func (h *AuthHandler) GenerateTokenHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		h.logger.Warn("Authorization header missing for token generation")
		c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Authorization header required"})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "basic" {
		h.logger.Warnf("Invalid Authorization header format for token generation: %s", authHeader)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Invalid authorization format, expected Basic auth"})
		return
	}

	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		h.logger.Warnf("Failed to decode base64 auth payload for token generation: %v", err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Invalid base64 encoding in auth header"})
		return
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		h.logger.Warn("Invalid auth payload format (expected account_sid:token) for token generation")
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Invalid auth payload format"})
		return
	}

	accountSid := pair[0]
	token := pair[1]

	if accountSid == "" || token == "" {
		h.logger.Warn("Account SID or token missing in auth payload for token generation")
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Account SID and token are required"})
		return
	}

	// Validate credentials
	// The ValidateCredentials method needs to be available on the CallServicerForESL interface.
	// This might mean CallServicerForESL embeds IAccountService, or has this method directly.
	validatedAccount, err := h.credentialValidator.ValidateCredentials(c.Request.Context(), accountSid, token)
	if err != nil {
		h.logger.Warnf("Failed to validate credentials for SID %s during token generation: %v", accountSid, err)
		c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
		c.JSON(http.StatusUnauthorized, GenericErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Determine role
	role := "user" // Default role
	if _, isAdmin := h.adminSIDs[validatedAccount.SID]; isAdmin {
		role = "admin"
	}

	// Prepare JWT claims
	expirationTime := time.Now().Add(h.jwtDuration)
	claims := &auth.JWTClaims{
		AccountSID:       validatedAccount.SID,
		ParentAccountSID: "", // Populate if subaccount logic is fully integrated and relevant here
		Role:             role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)), // Allow for slight clock skew
			Subject:   validatedAccount.SID,
			Issuer:    "AgbaraVOIP", // Example issuer
			// JTI (JWT ID) can be added for unique token identification if needed for revocation list
		},
	}
    if validatedAccount.ParentSID != nil && *validatedAccount.ParentSID != "" {
        claims.ParentAccountSID = *validatedAccount.ParentSID
    }


	jwtToken, err := auth.GenerateToken(claims) // Removed expirationTime, as GenerateToken should derive it from claims
	if err != nil {
		h.logger.Errorf("Failed to generate JWT token for SID %s: %v", validatedAccount.SID, err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Token generation failed", Details: err.Error()})
		return
	}

	h.logger.Infof("JWT token generated successfully for Account SID: %s with role: %s", validatedAccount.SID, role)
	c.JSON(http.StatusOK, GenerateTokenResponse{
		Token:     jwtToken,
		ExpiresAt: expirationTime.Format(time.RFC3339),
		TokenType: "Bearer",
	})
}
