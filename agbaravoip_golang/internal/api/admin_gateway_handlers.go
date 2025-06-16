package api

import (
	"errors"
	"net/http"
	"strconv" // For parsing boolean query params

	"github.com/user/agbaravoip_golang/internal/domain" // Required for CreateGatewayRequest which has domain.GatewayRoutes
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AdminGatewayHandler struct {
	service services.IGatewayService
	logger  *logrus.Entry
}

func NewAdminGatewayHandler(service services.IGatewayService, logger *logrus.Logger) *AdminGatewayHandler {
	return &AdminGatewayHandler{
		service: service,
		logger:  logger.WithField("handler", "admin_gateway"),
	}
}

// CreateGateway godoc
// @Summary Create a new Gateway configuration
// @Description (Admin) Adds a new Gateway configuration to the system.
// @Tags Admin-Gateway
// @Accept  json
// @Produce  json
// @Param   gateway_config  body   CreateGatewayRequest  true  "Gateway Configuration"
// @Success 201 {object} GatewayResponse
// @Failure 400 {object} GenericErrorResponse "Validation error"
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/gateways [post]
func (h *AdminGatewayHandler) CreateGateway(c *gin.Context) {
	var req CreateGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for CreateGateway: %v", err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	// Admins specify the AccountSID the gateway belongs to.
	if req.AccountSID == "" {
		h.logger.Warn("AccountSID is required when admin creates a gateway.")
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: "account_sid is required for gateway creation"})
		return
	}

	gateway, err := h.service.CreateGateway(
		req.AccountSID,
		req.FreeswitchServerSID,
		req.FriendlyName,
		req.GatewayString,
		req.Codecs,
		req.RetryCount,
		req.TimeoutSeconds,
		req.Routes,
		req.IsEnabled,
	)

	if err != nil {
		if errors.Is(err, services.ErrGatewayValidation) {
			h.logger.Warnf("Failed to create gateway due to validation: %v", err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Failed to create gateway", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error creating gateway: %v", err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not create gateway", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Gateway created successfully: %s", gateway.SID)
	c.JSON(http.StatusCreated, ToGatewayResponse(gateway))
}

// GetGateway godoc
// @Summary Get Gateway configuration details by SID
// @Description (Admin) Retrieves details for a specific Gateway configuration.
// @Tags Admin-Gateway
// @Produce  json
// @Param   gw_sid  path   string  true  "Gateway SID"
// @Success 200 {object} GatewayResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 404 {object} GenericErrorResponse "Gateway not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/gateways/{gw_sid} [get]
func (h *AdminGatewayHandler) GetGateway(c *gin.Context) {
	gwSid := c.Param("gw_sid")
	// Use the global getter for admin access
	gateway, err := h.service.GetGlobalGatewayBySID(gwSid)
	if err != nil {
		if errors.Is(err, services.ErrGatewayNotFound) {
			h.logger.Warnf("Gateway %s not found", gwSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Gateway not found"})
		} else {
			h.logger.Errorf("Error getting gateway %s: %v", gwSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve gateway", Details: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, ToGatewayResponse(gateway))
}

// ListGateways godoc
// @Summary List Gateway configurations
// @Description (Admin) Retrieves a list of all Gateway configurations. Supports filtering.
// @Tags Admin-Gateway
// @Produce  json
// @Param account_sid query string false "Filter by Account SID"
// @Param friendly_name query string false "Filter by friendly name (contains)"
// @Param is_enabled query bool false "Filter by enabled status"
// @Success 200 {array} GatewayResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/gateways [get]
func (h *AdminGatewayHandler) ListGateways(c *gin.Context) {
	filters := make(map[string]interface{})
	if accountSID := c.Query("account_sid"); accountSID != "" {
		filters["account_sid"] = accountSID
	}
	if friendlyName := c.Query("friendly_name"); friendlyName != "" {
		filters["friendly_name"] = friendlyName
	}
	if isEnabledStr, ok := c.GetQuery("is_enabled"); ok {
		if isEnabled, err := strconv.ParseBool(isEnabledStr); err == nil {
			filters["is_enabled"] = isEnabled
		} else {
			h.logger.Warnf("Invalid boolean value for is_enabled filter: %s", isEnabledStr)
		}
	}

	// Use the global lister for admin access
	gateways, err := h.service.ListGlobalGateways(filters)
	if err != nil {
		h.logger.Errorf("Error listing gateways: %v", err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve gateways", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ToGatewayResponseList(gateways))
}

// UpdateGateway godoc
// @Summary Update a Gateway configuration
// @Description (Admin) Updates details for a specific Gateway configuration.
// @Tags Admin-Gateway
// @Accept  json
// @Produce  json
// @Param   gw_sid         path   string               true  "Gateway SID"
// @Param   gateway_update body   UpdateGatewayRequest true  "Gateway Update Data"
// @Success 200 {object} GatewayResponse
// @Failure 400 {object} GenericErrorResponse "Validation error or no fields to update"
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 404 {object} GenericErrorResponse "Gateway not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/gateways/{gw_sid} [put]
func (h *AdminGatewayHandler) UpdateGateway(c *gin.Context) {
	gwSid := c.Param("gw_sid")
	var req UpdateGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for UpdateGateway (SID: %s): %v", gwSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.FreeswitchServerSID != nil { updates["freeswitch_server_sid"] = req.FreeswitchServerSID } // Handles null via pointer
	if req.FriendlyName != nil { updates["friendly_name"] = *req.FriendlyName }
	if req.GatewayString != nil { updates["gateway_string"] = *req.GatewayString }
	if req.Codecs != nil { updates["codecs"] = req.Codecs } // Assuming full replacement for codecs
	if req.RetryCount != nil { updates["retry_count"] = *req.RetryCount }
	if req.TimeoutSeconds != nil { updates["timeout_seconds"] = *req.TimeoutSeconds }
	if req.Routes != nil { updates["routes"] = *req.Routes }
	if req.IsEnabled != nil { updates["is_enabled"] = *req.IsEnabled }
	// Note: AccountSID of a gateway is not updatable through this admin endpoint to prevent accidental changes.
	// It should be managed via deletion and recreation if an account change is needed.

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "No fields to update"})
		return
	}

	// Use the global updater for admin access
	updatedGateway, err := h.service.UpdateGlobalGateway(gwSid, updates)
	if err != nil {
		if errors.Is(err, services.ErrGatewayNotFound) {
			h.logger.Warnf("Gateway %s not found for update", gwSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Gateway not found"})
		} else if errors.Is(err, services.ErrGatewayValidation) {
			h.logger.Warnf("Validation error updating gateway %s: %v", gwSid, err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Update failed", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error updating gateway %s: %v", gwSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not update gateway", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Gateway %s updated successfully", updatedGateway.SID)
	c.JSON(http.StatusOK, ToGatewayResponse(updatedGateway))
}

// DeleteGateway godoc
// @Summary Delete a Gateway configuration
// @Description (Admin) Deletes a specific Gateway configuration.
// @Tags Admin-Gateway
// @Param   gw_sid  path   string  true  "Gateway SID"
// @Success 204 "No Content"
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 404 {object} GenericErrorResponse "Gateway not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/gateways/{gw_sid} [delete]
func (h *AdminGatewayHandler) DeleteGateway(c *gin.Context) {
	gwSid := c.Param("gw_sid")

	// Use the global deleter for admin access
	err := h.service.DeleteGlobalGateway(gwSid)
	if err != nil {
		if errors.Is(err, services.ErrGatewayNotFound) {
			h.logger.Warnf("Gateway %s not found for delete", gwSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Gateway not found"})
		} else {
			h.logger.Errorf("Internal error deleting gateway %s: %v", gwSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not delete gateway", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Gateway %s deleted successfully", gwSid)
	c.Status(http.StatusNoContent)
}
