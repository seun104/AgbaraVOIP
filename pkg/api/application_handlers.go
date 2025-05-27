package api

import (
	"agbara-go/pkg/models"
	"agbara-go/pkg/services"
	"database/sql"
	"errors"
	"fmt" // Ensure fmt is imported for containsSubstring
	"net/http"

	"github.com/gin-gonic/gin"
)

// ApplicationAPI holds handlers for application-related API endpoints.
type ApplicationAPI struct {
	applicationService services.ApplicationService
	accountService     services.AccountService // To verify account existence
}

// NewApplicationAPI creates a new ApplicationAPI instance.
func NewApplicationAPI(applicationService services.ApplicationService, accountService services.AccountService) *ApplicationAPI {
	return &ApplicationAPI{
		applicationService: applicationService,
		accountService:     accountService,
	}
}

// authAndAccountCheckForApplication is consistent with other handlers.
func (api *ApplicationAPI) authAndAccountCheck(c *gin.Context) (string, bool) {
	authenticatedAccountSid := c.GetHeader("X-Auth-User-Sid")
	if authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required: X-Auth-User-Sid header missing"})
		return "", false
	}

	pathAccountSid := c.Param("accountSidInPath")
	if pathAccountSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account SID missing in path"})
		return "", false
	}

	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access to this account's applications is forbidden."})
		return "", false
	}

	_, err := api.accountService.GetAccount(c.Request.Context(), pathAccountSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account", "details": err.Error()})
		}
		return "", false
	}
	return pathAccountSid, true
}

// RegisterApplicationRoutes sets up the routes for the Application API.
// Expects to be registered on a group like router.Group("/Accounts/:accountSidInPath")
func (api *ApplicationAPI) RegisterApplicationRoutes(router *gin.RouterGroup) {
	appRoutes := router.Group("/Applications")
	{
		appRoutes.POST("", api.CreateApplicationHandler)
		appRoutes.GET("", api.ListApplicationsHandler)
		appRoutes.GET("/:applicationSid", api.GetApplicationHandler)
		appRoutes.POST("/:applicationSid", api.UpdateApplicationHandler) // Using POST for update as per C#
		appRoutes.DELETE("/:applicationSid", api.DeleteApplicationHandler)
	}
}

// --- Application Handlers ---

func (api *ApplicationAPI) CreateApplicationHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}

	var req models.ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Validate HTTPMethods in request before passing to service
	// (Service also does this, but good to catch early)
	methodsToValidate := []models.HTTPMethod{
        req.VoiceMethod, req.VoiceFallbackMethod, req.StatusCallbackMethod,
        req.SmsMethod, req.SmsFallbackMethod, req.SmsStatusCallbackMethod,
    }
    for _, method := range methodsToValidate {
        if !method.Validate() && method != "" { // Allow empty, but if not empty, must be valid
            c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid HTTP method provided: %s", method)})
            return
        }
    }
	
	application, err := api.applicationService.CreateApplication(c.Request.Context(), accountSid, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, application)
}

func (api *ApplicationAPI) ListApplicationsHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	applications, err := api.applicationService.ListApplications(c.Request.Context(), accountSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list applications", "details": err.Error()})
		return
	}
	if applications == nil {
		applications = []*models.Application{}
	}
	c.JSON(http.StatusOK, applications)
}

func (api *ApplicationAPI) GetApplicationHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c) // Validates :accountSidInPath and auth
	if !ok {
		return
	}
	applicationSid := c.Param("applicationSid")
	if applicationSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application SID missing in path"})
		return
	}

	application, err := api.applicationService.GetApplication(c.Request.Context(), applicationSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application", "details": err.Error()})
		}
		return
	}
    // Authorization: Ensure the fetched application belongs to the authenticated account
    if application.AccountSid != accountSid {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access to this application is forbidden."})
        return
    }
	c.JSON(http.StatusOK, application)
}

func (api *ApplicationAPI) UpdateApplicationHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c) // Validates :accountSidInPath and auth
	if !ok {
		return
	}
	applicationSid := c.Param("applicationSid")
	if applicationSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application SID missing in path"})
		return
	}

	var req models.ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
    
    // Optional: Validate HTTPMethods in request here as well
    methodsToValidate := []models.HTTPMethod{
        req.VoiceMethod, req.VoiceFallbackMethod, req.StatusCallbackMethod,
        req.SmsMethod, req.SmsFallbackMethod, req.SmsStatusCallbackMethod,
    }
    for _, method := range methodsToValidate {
        if !method.Validate() && method != "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid HTTP method provided in update: %s", method)})
            return
        }
    }

	// Authorization: First, get the application to ensure it belongs to this account
	existingApp, err := api.applicationService.GetApplication(c.Request.Context(), applicationSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found for update"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application for update", "details": err.Error()})
		}
		return
	}
	if existingApp.AccountSid != accountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update application not belonging to your account."})
		return
	}

	application, err := api.applicationService.UpdateApplication(c.Request.Context(), applicationSid, &req)
	if err != nil {
		// The service's UpdateApplication already checks for sql.ErrNoRows if it tries to RETURNING from a non-existent row
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found for update") { // Match service error
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found during update"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update application", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, application)
}

func (api *ApplicationAPI) DeleteApplicationHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c) // Validates :accountSidInPath and auth
	if !ok {
		return
	}
	applicationSid := c.Param("applicationSid")
	if applicationSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application SID missing in path"})
		return
	}

	// Authorization: First, get the application to ensure it belongs to this account
	existingApp, err := api.applicationService.GetApplication(c.Request.Context(), applicationSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found for deletion"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application for deletion", "details": err.Error()})
		}
		return
	}
	if existingApp.AccountSid != accountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete application not belonging to your account."})
		return
	}

	err = api.applicationService.DeleteApplication(c.Request.Context(), applicationSid)
	if err != nil {
		// The service's DeleteApplication already checks for "not found (no rows affected)"
		if containsSubstring(err.Error(), "not found for deletion") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Application not found during deletion attempt"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete application", "details": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// Re-include containsSubstring if not in a shared util package yet.
// (Using the version from conference_handlers.go for consistency)
func containsSubstring(s, substr string) bool {
    var errStr string
    if err, ok := s.(error); ok { // Check if s is error type
        errStr = err.Error()
    } else if str, ok := s.(string); ok { // Check if s is string type
        errStr = str
    } else {
        return false // Not a type we can check
    }
    // Basic substring check
    for i := 0; i <= len(errStr)-len(substr); i++ {
        if errStr[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
