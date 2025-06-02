package api

import (
	"errors"
	"net/http"
	// "reflect" // Removed: not used after switching to manual map
	// "strings" // Removed: not used after switching to manual map

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// var _ *domain.Account // This was for account_handlers.go, not strictly needed here if domain types are used.
// However, domain.Application is used in CreateApplication, so domain import is needed.

type ApplicationHandler struct {
	service services.IApplicationService
	logger  *logrus.Entry
}

func NewApplicationHandler(service services.IApplicationService, logger *logrus.Logger) *ApplicationHandler {
	return &ApplicationHandler{
		service: service,
		logger:  logger.WithField("handler", "application"),
	}
}

// CreateApplication godoc
// @Summary Create a new application for an account
// @Description Creates a new voice/SMS application under the authenticated account.
// @Tags applications
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID owning the application"
// @Param   application  body   CreateApplicationRequest   true  "Application Creation Request"
// @Success 201 {object} ApplicationResponse
// @Failure 400 {object} GenericErrorResponse "Validation error"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /v1/accounts/{account_sid}/applications [post]
func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	authAccountSid := c.GetString(ContextAuthAccountKey) 

	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for CreateApplication: %v", err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	app := &domain.Application{ // domain.Application is used here
		FriendlyName:         req.FriendlyName,
		VoiceURL:             req.VoiceURL,
		VoiceMethod:          req.VoiceMethod, 
		VoiceFallbackURL:     req.VoiceFallbackURL,
		VoiceFallbackMethod:  req.VoiceFallbackMethod,
		SmsURL:               req.SmsURL,
		SmsMethod:            req.SmsMethod,
		SmsFallbackURL:       req.SmsFallbackURL,
		SmsFallbackMethod:    req.SmsFallbackMethod,
		StatusCallbackURL:    req.StatusCallbackURL,
		StatusCallbackMethod: req.StatusCallbackMethod,
	}

	createdApp, err := h.service.CreateApplication(authAccountSid, app)
	if err != nil {
		if errors.Is(err, services.ErrAppValidationFailed) {
			h.logger.Warnf("Failed to create application due to validation: %v", err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Failed to create application", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error creating application: %v", err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not create application", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Application %s created successfully for account %s", createdApp.SID, authAccountSid)
	c.JSON(http.StatusCreated, ToApplicationResponse(createdApp))
}


// GetApplication godoc
// @Summary Get application details by SID
// @Description Retrieves details for a specific application owned by the authenticated account.
// @Tags applications
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   app_sid      path   string  true  "Application SID"
// @Success 200 {object} ApplicationResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Application not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /v1/accounts/{account_sid}/applications/{app_sid} [get]
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	authAccountSid := c.GetString(ContextAuthAccountKey)
	appSid := c.Param("app_sid")

	app, err := h.service.GetApplicationBySID(authAccountSid, appSid)
	if err != nil {
		if errors.Is(err, services.ErrApplicationNotFound) {
			h.logger.Warnf("Application %s not found for account %s", appSid, authAccountSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Application not found"})
		} else {
			h.logger.Errorf("Error getting application %s: %v", appSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve application", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Application %s retrieved for account %s", app.SID, authAccountSid)
	c.JSON(http.StatusOK, ToApplicationResponse(app))
}

// ListApplications godoc
// @Summary List applications for an account
// @Description Retrieves all applications owned by the authenticated account.
// @Tags applications
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Success 200 {array} ApplicationResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /v1/accounts/{account_sid}/applications [get]
func (h *ApplicationHandler) ListApplications(c *gin.Context) {
	authAccountSid := c.GetString(ContextAuthAccountKey)

	apps, err := h.service.ListApplications(authAccountSid)
	if err != nil {
		h.logger.Errorf("Error listing applications for account %s: %v", authAccountSid, err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve applications", Details: err.Error()})
		return
	}

	h.logger.Infof("Retrieved %d applications for account %s", len(apps), authAccountSid)
	c.JSON(http.StatusOK, ToApplicationResponseList(apps))
}

// UpdateApplication godoc
// @Summary Update an application
// @Description Updates details for a specific application owned by the authenticated account.
// @Tags applications
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   app_sid      path   string  true  "Application SID"
// @Param   application  body   UpdateApplicationRequest   true  "Application Update Request (partial updates allowed)"
// @Success 200 {object} ApplicationResponse
// @Failure 400 {object} GenericErrorResponse "Validation error"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Application not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /v1/accounts/{account_sid}/applications/{app_sid} [put]
func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	authAccountSid := c.GetString(ContextAuthAccountKey)
	appSid := c.Param("app_sid")

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for UpdateApplication (SID: %s): %v", appSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	updatesManual := make(map[string]interface{})
	if req.FriendlyName != nil { updatesManual["friendly_name"] = *req.FriendlyName }
	if req.VoiceURL != nil { updatesManual["voice_url"] = *req.VoiceURL }
	if req.VoiceMethod != nil { updatesManual["voice_method"] = *req.VoiceMethod }
	if req.VoiceFallbackURL != nil { updatesManual["voice_fallback_url"] = *req.VoiceFallbackURL }
	if req.VoiceFallbackMethod != nil { updatesManual["voice_fallback_method"] = *req.VoiceFallbackMethod }
	if req.SmsURL != nil { updatesManual["sms_url"] = *req.SmsURL }
	if req.SmsMethod != nil { updatesManual["sms_method"] = *req.SmsMethod }
	if req.SmsFallbackURL != nil { updatesManual["sms_fallback_url"] = *req.SmsFallbackURL }
	if req.SmsFallbackMethod != nil { updatesManual["sms_fallback_method"] = *req.SmsFallbackMethod }
	if req.StatusCallbackURL != nil { updatesManual["status_callback_url"] = *req.StatusCallbackURL }
	if req.StatusCallbackMethod != nil { updatesManual["status_callback_method"] = *req.StatusCallbackMethod }

	if len(updatesManual) == 0 {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "No fields to update"})
		return
	}

	updatedApp, err := h.service.UpdateApplication(authAccountSid, appSid, updatesManual)
	if err != nil {
		if errors.Is(err, services.ErrApplicationNotFound) {
			h.logger.Warnf("App %s not found for update, account %s", appSid, authAccountSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Application not found"})
		} else if errors.Is(err, services.ErrAppValidationFailed) {
			h.logger.Warnf("Validation error updating app %s: %v", appSid, err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Update failed", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error updating app %s: %v", appSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not update application", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Application %s updated for account %s", updatedApp.SID, authAccountSid)
	c.JSON(http.StatusOK, ToApplicationResponse(updatedApp))
}

// DeleteApplication godoc
// @Summary Delete an application
// @Description Deletes a specific application owned by the authenticated account.
// @Tags applications
// @Param   account_sid  path   string  true  "Account SID"
// @Param   app_sid      path   string  true  "Application SID"
// @Success 204 "No Content"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Application not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /v1/accounts/{account_sid}/applications/{app_sid} [delete]
func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	authAccountSid := c.GetString(ContextAuthAccountKey)
	appSid := c.Param("app_sid")

	err := h.service.DeleteApplication(authAccountSid, appSid)
	if err != nil {
		if errors.Is(err, services.ErrApplicationNotFound) {
			h.logger.Warnf("App %s not found for delete, account %s", appSid, authAccountSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Application not found"})
		} else {
			h.logger.Errorf("Internal error deleting app %s: %v", appSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not delete application", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Application %s deleted for account %s", appSid, authAccountSid)
	c.Status(http.StatusNoContent)
}
