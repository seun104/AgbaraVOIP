package api

import (
	"errors"
	"net/http"
	// "strconv" // Not currently used, but could be for pagination filters

	"github.com/user/agbaravoip_golang/internal/domain" // For domain.ErrNotFound if used, though services errors are preferred
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RecordingHandler handles API requests related to recording metadata.
type RecordingHandler struct {
	service services.IRecordingService
	logger  *logrus.Entry
}

// NewRecordingHandler creates a new RecordingHandler.
func NewRecordingHandler(service services.IRecordingService, logger *logrus.Logger) *RecordingHandler {
	return &RecordingHandler{
		service: service,
		logger:  logger.WithField("handler", "recording"),
	}
}

// GetRecording godoc
// @Summary Get recording details
// @Description Retrieves details for a specific recording belonging to the authenticated account.
// @Tags Recordings
// @Produce  json
// @Param   account_sid    path   string  true  "Account SID"
// @Param   recording_sid  path   string  true  "Recording SID"
// @Success 200 {object} RecordingResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "Recording not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/recordings/{recording_sid} [get]
func (h *RecordingHandler) GetRecording(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID)) // From JWT
	recordingSid := c.Param("recording_sid")

	recording, err := h.service.GetRecordingBySID(c.Request.Context(), accountSid, recordingSid)
	if err != nil {
		h.logger.Warnf("GetRecording: Error for account %s, recording SID %s: %v", accountSid, recordingSid, err)
		if errors.Is(err, services.ErrRecordingNotFound) { // Use service specific error
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Recording not found"})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve recording", Details: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, ToRecordingResponse(recording))
}

// ListRecordings godoc
// @Summary List recordings for an account
// @Description Retrieves a list of recordings associated with the authenticated account. Supports filtering.
// @Tags Recordings
// @Produce  json
// @Param   account_sid    path   string  true  "Account SID"
// @Param   call_sid       query  string  false "Filter by Call SID"
// @Param   conference_sid query  string  false "Filter by Conference SID"
// @Param   format         query  string  false "Filter by recording format (e.g., wav, mp3)"
// @Param   date_from      query  string  false "Filter by date from (YYYY-MM-DD)" // Service needs to implement parsing
// @Param   date_to        query  string  false "Filter by date to (YYYY-MM-DD)"   // Service needs to implement parsing
// @Success 200 {array} RecordingResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/recordings [get]
func (h *RecordingHandler) ListRecordings(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))

	filters := make(map[string]interface{})
	if val := c.Query("call_sid"); val != "" { filters["call_sid"] = val }
	if val := c.Query("conference_sid"); val != "" { filters["conference_sid"] = val }
	if val := c.Query("format"); val != "" { filters["format"] = val }
	if val := c.Query("date_from"); val != "" { filters["date_from"] = val } // Pass to service for parsing
	if val := c.Query("date_to"); val != "" { filters["date_to"] = val }     // Pass to service for parsing
	// Add limit/offset query params for pagination if desired, e.g.
	// if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && limit > 0 { filters["limit"] = limit }
	// if offset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && offset >= 0 { filters["offset"] = offset }


	recordings, err := h.service.ListRecordings(c.Request.Context(), accountSid, filters)
	if err != nil {
		h.logger.Errorf("ListRecordings: Error for account %s: %v", accountSid, err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve recordings", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ToRecordingResponseList(recordings))
}

// DeleteRecording godoc
// @Summary Delete recording metadata
// @Description Deletes the metadata for a specific recording. Does not delete the actual recording file from storage.
// @Tags Recordings
// @Param   account_sid    path   string  true  "Account SID"
// @Param   recording_sid  path   string  true  "Recording SID to delete"
// @Success 204 "No Content"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "Recording not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error or deletion failed"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/recordings/{recording_sid} [delete]
func (h *RecordingHandler) DeleteRecording(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	recordingSid := c.Param("recording_sid")

	err := h.service.DeleteRecording(c.Request.Context(), accountSid, recordingSid)
	if err != nil {
		h.logger.Errorf("DeleteRecording: Error for account %s, recording SID %s: %v", accountSid, recordingSid, err)
		if errors.Is(err, services.ErrRecordingNotFound) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Recording not found"})
		} else if errors.Is(err, services.ErrRecordingDeletionFailed) {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Failed to delete recording metadata", Details: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not delete recording metadata", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("DeleteRecording: Recording metadata %s deleted for account %s", recordingSid, accountSid)
	c.Status(http.StatusNoContent)
}
