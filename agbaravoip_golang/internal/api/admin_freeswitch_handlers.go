package api

import (
	"errors"
	"net/http"
	"strconv" // Added for ParseBool

	"github.com/user/agbaravoip_golang/internal/services"
	// "github.com/user/agbaravoip_golang/internal/utils" // For ParseBool, using strconv for now
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AdminFreeswitchHandler struct {
	service services.IFreeswitchServerService
	logger  *logrus.Entry
}

func NewAdminFreeswitchHandler(service services.IFreeswitchServerService, logger *logrus.Logger) *AdminFreeswitchHandler {
	return &AdminFreeswitchHandler{
		service: service,
		logger:  logger.WithField("handler", "admin_freeswitch"),
	}
}

// CreateFreeswitchServer godoc
// @Summary Create a new Freeswitch server configuration
// @Description (Admin) Adds a new Freeswitch server that can be used by the system.
// @Tags Admin-Freeswitch
// @Accept  json
// @Produce  json
// @Param   server_config  body   CreateFreeswitchServerRequest  true  "Freeswitch Server Configuration"
// @Success 201 {object} FreeswitchServerResponse
// @Failure 400 {object} GenericErrorResponse "Validation error"
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement // Placeholder for actual security definition
// @Router /api/v1/admin/freeswitch-servers [post]
func (h *AdminFreeswitchHandler) CreateFreeswitchServer(c *gin.Context) {
	var req CreateFreeswitchServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for CreateFreeswitchServer: %v", err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	fsServer, err := h.service.CreateFreeswitchServer(req.Host, req.Port, req.Password, req.OutboundAddress, req.IsActive)
	if err != nil {
		if errors.Is(err, services.ErrFreeswitchServerValidation) {
			h.logger.Warnf("Failed to create Freeswitch server due to validation: %v", err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Failed to create server", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error creating Freeswitch server: %v", err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not create Freeswitch server", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Freeswitch server created successfully: %s", fsServer.SID)
	c.JSON(http.StatusCreated, ToFreeswitchServerResponse(fsServer))
}

// GetFreeswitchServer godoc
// @Summary Get Freeswitch server details by SID
// @Description (Admin) Retrieves details for a specific Freeswitch server configuration.
// @Tags Admin-Freeswitch
// @Produce  json
// @Param   fs_sid  path   string  true  "Freeswitch Server SID"
// @Success 200 {object} FreeswitchServerResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 404 {object} GenericErrorResponse "Server not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/freeswitch-servers/{fs_sid} [get]
func (h *AdminFreeswitchHandler) GetFreeswitchServer(c *gin.Context) {
	fsSid := c.Param("fs_sid")
	fsServer, err := h.service.GetFreeswitchServerBySID(fsSid)
	if err != nil {
		if errors.Is(err, services.ErrFreeswitchServerNotFound) {
			h.logger.Warnf("Freeswitch server %s not found", fsSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Freeswitch server not found"})
		} else {
			h.logger.Errorf("Error getting Freeswitch server %s: %v", fsSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve Freeswitch server", Details: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, ToFreeswitchServerResponse(fsServer))
}

// ListFreeswitchServers godoc
// @Summary List Freeswitch server configurations
// @Description (Admin) Retrieves a list of all Freeswitch server configurations. Supports filtering.
// @Tags Admin-Freeswitch
// @Produce  json
// @Param is_active query bool false "Filter by active status"
// @Success 200 {array} FreeswitchServerResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/freeswitch-servers [get]
func (h *AdminFreeswitchHandler) ListFreeswitchServers(c *gin.Context) {
	filters := make(map[string]interface{})
	if isActiveStr, ok := c.GetQuery("is_active"); ok {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filters["is_active"] = isActive
		} else {
			h.logger.Warnf("Invalid boolean value for is_active filter: %s", isActiveStr)
            // Optionally return bad request, or just ignore filter
		}
	}

	servers, err := h.service.ListFreeswitchServers(filters)
	if err != nil {
		h.logger.Errorf("Error listing Freeswitch servers: %v", err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve Freeswitch servers", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ToFreeswitchServerResponseList(servers))
}

// UpdateFreeswitchServer godoc
// @Summary Update a Freeswitch server configuration
// @Description (Admin) Updates details for a specific Freeswitch server configuration.
// @Tags Admin-Freeswitch
// @Accept  json
// @Produce  json
// @Param   fs_sid         path   string                         true  "Freeswitch Server SID"
// @Param   server_update  body   UpdateFreeswitchServerRequest  true  "Freeswitch Server Update Data"
// @Success 200 {object} FreeswitchServerResponse
// @Failure 400 {object} GenericErrorResponse "Validation error or no fields to update"
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 404 {object} GenericErrorResponse "Server not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/freeswitch-servers/{fs_sid} [put]
func (h *AdminFreeswitchHandler) UpdateFreeswitchServer(c *gin.Context) {
	fsSid := c.Param("fs_sid")
	var req UpdateFreeswitchServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for UpdateFreeswitchServer (SID: %s): %v", fsSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Host != nil { updates["host"] = *req.Host }
	if req.Port != nil { updates["port"] = *req.Port }
	if req.Password != nil && *req.Password != "" { updates["password"] = *req.Password } // Service handles hashing
	if req.OutboundAddress != nil { updates["outbound_address"] = *req.OutboundAddress }
	if req.IsActive != nil { updates["is_active"] = *req.IsActive }

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "No fields to update"})
		return
	}

	updatedServer, err := h.service.UpdateFreeswitchServer(fsSid, updates)
	if err != nil {
		if errors.Is(err, services.ErrFreeswitchServerNotFound) {
			h.logger.Warnf("Freeswitch server %s not found for update", fsSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Freeswitch server not found"})
		} else if errors.Is(err, services.ErrFreeswitchServerValidation) {
			h.logger.Warnf("Validation error updating Freeswitch server %s: %v", fsSid, err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Update failed", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error updating Freeswitch server %s: %v", fsSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not update Freeswitch server", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Freeswitch server %s updated successfully", updatedServer.SID)
	c.JSON(http.StatusOK, ToFreeswitchServerResponse(updatedServer))
}

// DeleteFreeswitchServer godoc
// @Summary Delete a Freeswitch server configuration
// @Description (Admin) Deletes a specific Freeswitch server configuration.
// @Tags Admin-Freeswitch
// @Param   fs_sid  path   string  true  "Freeswitch Server SID"
// @Success 204 "No Content"
// @Failure 401 {object} GenericErrorResponse "Unauthorized (Admin role required)"
// @Failure 404 {object} GenericErrorResponse "Server not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security AdminAuthRequirement
// @Router /api/v1/admin/freeswitch-servers/{fs_sid} [delete]
func (h *AdminFreeswitchHandler) DeleteFreeswitchServer(c *gin.Context) {
	fsSid := c.Param("fs_sid")

	err := h.service.DeleteFreeswitchServer(fsSid)
	if err != nil {
		if errors.Is(err, services.ErrFreeswitchServerNotFound) {
			h.logger.Warnf("Freeswitch server %s not found for delete", fsSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Freeswitch server not found"})
		} else {
			h.logger.Errorf("Internal error deleting Freeswitch server %s: %v", fsSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not delete Freeswitch server", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Freeswitch server %s deleted successfully", fsSid)
	c.Status(http.StatusNoContent)
}
