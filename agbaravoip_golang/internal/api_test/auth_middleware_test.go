package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock" // Not strictly needed if not mocking services here

	"github.com/user/agbaravoip_golang/internal/api"   // Package containing the middleware
	"github.com/user/agbaravoip_golang/internal/auth"  // For JWT utils & Init
)

const mwTestSecret = "middlewaretestsecret1234567890123"

// Helper to setup Gin router with middleware for testing
func setupMiddlewareTestRouter(handler gin.HandlerFunc, mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if mw != nil { // Allow testing handler without middleware too
		router.Use(mw)
	}
	router.GET("/testroute", handler)
	router.GET("/testroute/:account_sid", handler) // For AccountAccessMiddleware
	return router
}

// Dummy handler to check if middleware passed
var dummyHandler = func(c *gin.Context) {
	// Check for values set by middleware if needed
	claims, exists := c.Get(string(api.ContextKeyClaims))
	if exists {
		c.JSON(http.StatusOK, gin.H{"status": "passed", "claims": claims})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "passed"})
	}
}

func TestJWTMiddleware_Success(t *testing.T) {
	auth.InitJWTSecret(mwTestSecret)
	defer auth.InitJWTSecret("") // Clean up

	logger := logrus.New(); logger.SetOutput(io.Discard)
	logEntry := logrus.NewEntry(logger)

	jwtMW := api.JWTMiddleware(logEntry)
	router := setupMiddlewareTestRouter(dummyHandler, jwtMW)

	claims := &auth.JWTClaims{
		AccountSID: "ACsuccess", Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute))},
	}
	token, _ := auth.GenerateToken(claims, time.Now().Add(15*time.Minute))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var respBody map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &respBody)
	assert.Equal(t, "passed", respBody["status"])

	respClaimsMap, ok := respBody["claims"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "ACsuccess", respClaimsMap["acc_sid"])
	assert.Equal(t, "user", respClaimsMap["role"])
}

func TestJWTMiddleware_NoAuthHeader(t *testing.T) {
	auth.InitJWTSecret(mwTestSecret); defer auth.InitJWTSecret("")
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	jwtMW := api.JWTMiddleware(logEntry)
	router := setupMiddlewareTestRouter(dummyHandler, jwtMW)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute", nil) // No Auth Header
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header required")
}

func TestJWTMiddleware_InvalidHeaderFormat_NoBearer(t *testing.T) {
	auth.InitJWTSecret(mwTestSecret); defer auth.InitJWTSecret("")
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	jwtMW := api.JWTMiddleware(logEntry)
	router := setupMiddlewareTestRouter(dummyHandler, jwtMW)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute", nil)
	req.Header.Set("Authorization", "Basic somecredentials") // Wrong scheme
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header format must be Bearer {token}")
}


func TestJWTMiddleware_InvalidToken_Expired(t *testing.T) {
	auth.InitJWTSecret(mwTestSecret); defer auth.InitJWTSecret("")
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	jwtMW := api.JWTMiddleware(logEntry)
	router := setupMiddlewareTestRouter(dummyHandler, jwtMW)

	claims := &auth.JWTClaims{ AccountSID: "ACexpired", RegisteredClaims: jwt.RegisteredClaims{ ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)) } }
	expiredToken, _ := auth.GenerateToken(claims, time.Now().Add(-1*time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Token is expired")
}

// --- AccountAccessMiddleware Tests ---

func TestAccountAccessMiddleware_Success(t *testing.T) {
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	accAccessMW := api.AccountAccessMiddleware(logEntry)

	// Simulate JWTMiddleware having run first and set the context key
	handlerWithContextSet := func(c *gin.Context) {
		c.Set(string(api.ContextKeyAccountSID), "AC123") // AccountSID from JWT
		accAccessMW(c) // Call the middleware being tested
		if !c.IsAborted() { // Check if middleware passed
			dummyHandler(c)
		}
	}
	router := gin.New(); router.GET("/testroute/:account_sid", handlerWithContextSet)


	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute/AC123", nil) // Path SID matches token SID
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "passed")
}

func TestAccountAccessMiddleware_Mismatch(t *testing.T) {
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	accAccessMW := api.AccountAccessMiddleware(logEntry)
	handlerWithContextSet := func(c *gin.Context) {
		c.Set(string(api.ContextKeyAccountSID), "AC123") // Token for AC123
		accAccessMW(c)
		if !c.IsAborted() { dummyHandler(c) }
	}
	router := gin.New(); router.GET("/testroute/:account_sid", handlerWithContextSet)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute/AC456", nil) // Path for AC456
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Access forbidden: you cannot access resources of another account.")
}

func TestAccountAccessMiddleware_NoTokenAccountSIDInContext(t *testing.T) {
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	accAccessMW := api.AccountAccessMiddleware(logEntry)
	handlerWithContextSet := func(c *gin.Context) {
		// No AccountSID set in context
		accAccessMW(c)
		if !c.IsAborted() { dummyHandler(c) }
	}
	router := gin.New(); router.GET("/testroute/:account_sid", handlerWithContextSet)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute/AC123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Access forbidden: missing token claims")
}

func TestAccountAccessMiddleware_NoPathParam(t *testing.T) {
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	accAccessMW := api.AccountAccessMiddleware(logEntry)
	handlerWithContextSet := func(c *gin.Context) {
		c.Set(string(api.ContextKeyAccountSID), "AC123")
		accAccessMW(c) // Applied to a route without :account_sid
		if !c.IsAborted() { dummyHandler(c) }
	}
	// Route without :account_sid param
	router := gin.New(); router.GET("/testroute_no_param", handlerWithContextSet)


	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testroute_no_param", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Middleware should skip check and pass
	assert.Contains(t, w.Body.String(), "passed")
}

// TestMain can be added if global setup/teardown for tests in this package is needed
// func TestMain(m *testing.M) {
// 	gin.SetMode(gin.TestMode)
// 	m.Run()
// }
