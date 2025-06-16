package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// --- Setup for AdminRoleAuthMiddleware Tests ---

func setupAdminRoleAuthTestRouter(logger *logrus.Logger, roleToSet interface{}, setRole bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	if logger == nil {
		logger = logrus.New()
	}
	logger.SetOutput(io.Discard) // Suppress logs during testing

	router := gin.New() // Use gin.New() for more control, not gin.Default() which has logger/recovery

	// Mock JWTMiddleware part: just set the context key that AdminRoleAuthMiddleware expects
	router.Use(func(c *gin.Context) {
		if setRole {
			if roleToSet != nil {
				c.Set(string(ContextKeyUserRole), roleToSet)
			}
			// If roleToSet is nil but setRole is true, it means the key itself is missing.
            // AdminRoleAuthMiddleware should handle c.Get() returning nil, false.
		}
		c.Next()
	})

	// Apply the middleware to be tested
	router.Use(AdminRoleAuthMiddleware(logrus.NewEntry(logger))) // Pass a logrus.Entry

	router.GET("/admin/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})
	return router
}

// --- Tests for AdminRoleAuthMiddleware ---

func TestAdminRoleAuthMiddleware_Success_AdminRole(t *testing.T) {
	router := setupAdminRoleAuthTestRouter(nil, "admin", true) // Set "admin" role

	req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "admin access granted")
}

func TestAdminRoleAuthMiddleware_Success_AdminRole_CaseInsensitive(t *testing.T) {
	router := setupAdminRoleAuthTestRouter(nil, "AdMiN", true) // Set "AdMiN" role

	req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "admin access granted")
}


func TestAdminRoleAuthMiddleware_Forbidden_UserRole(t *testing.T) {
	router := setupAdminRoleAuthTestRouter(nil, "user", true) // Set "user" role

	req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "admin role required")
}

func TestAdminRoleAuthMiddleware_Forbidden_EmptyRole(t *testing.T) {
	router := setupAdminRoleAuthTestRouter(nil, "", true) // Set empty string role

	req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid role claims")
}

func TestAdminRoleAuthMiddleware_Forbidden_RoleKeyMissing(t *testing.T) {
	// Simulate ContextKeyUserRole not being set at all by JWTMiddleware
	router := setupAdminRoleAuthTestRouter(nil, nil, false) // setRole = false

	req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing role claims")
}

func TestAdminRoleAuthMiddleware_Forbidden_RoleNotString(t *testing.T) {
	router := setupAdminRoleAuthTestRouter(nil, 123, true) // Set role as int

	req, _ := http.NewRequest(http.MethodGet, "/admin/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid role claims")
}

// --- Placeholder for existing tests in auth_middleware_test.go ---
// If this file is new, these would not be present.
// If appending, ensure existing tests for other middlewares are preserved.
// func TestJWTMiddleware_Success(t *testing.T) { ... }
// func TestAccountAccessMiddleware_Success(t *testing.T) { ... }
