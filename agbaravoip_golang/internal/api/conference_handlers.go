package api

import (
	"errors"
	"net/http"
	"strings" // For strings.Title in mapConferenceControlError

	"github.com/user/agbaravoip_golang/internal/domain"   // For domain.ErrNotFound
	"github.com/user/agbaravoip_golang/internal/services" // For service errors
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ConferenceHandler handles API requests related to conferences.
type ConferenceHandler struct {
	service     services.IConferenceService
	callService services.ICallService // For AddParticipantToConference (dial-out) if implemented
	logger      *logrus.Entry
}

// NewConferenceHandler creates a new ConferenceHandler.
func NewConferenceHandler(service services.IConferenceService, callService services.ICallService, logger *logrus.Logger) *ConferenceHandler {
	return &ConferenceHandler{
		service:     service,
		callService: callService,
		logger:      logger.WithField("handler", "conference"),
	}
}

// --- Conference Metadata Handlers ---

// ListConferences godoc
// @Summary List conferences for an account
// @Description Retrieves a list of conferences associated with the authenticated account.
// @Tags Conferences
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   status query string false "Filter by conference status (e.g., in-progress, completed)"
// @Param   friendly_name query string false "Filter by friendly name (contains)"
// @Success 200 {array} ConferenceResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences [get]
func (h *ConferenceHandler) ListConferences(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	filters := make(map[string]interface{})
	if val := c.Query("status"); val != "" { filters["status"] = val }
	if val := c.Query("friendly_name"); val != "" { filters["friendly_name"] = val }

	conferences, err := h.service.ListConferences(c.Request.Context(), accountSid, filters)
	if err != nil {
		h.logger.Errorf("ListConferences: Error for account %s: %v", accountSid, err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve conferences", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ToConferenceResponseList(conferences))
}

// GetConference godoc
// @Summary Get conference details
// @Description Retrieves details for a specific conference.
// @Tags Conferences
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   conf_sid     path   string  true  "Conference SID"
// @Success 200 {object} ConferenceResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid} [get]
func (h *ConferenceHandler) GetConference(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")

	conference, err := h.service.GetConferenceBySID(c.Request.Context(), accountSid, confSid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Conference not found"})
		} else {
			h.logger.Errorf("GetConference: Error for conf %s, account %s: %v", confSid, accountSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve conference", Details: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, ToConferenceResponse(conference))
}

// --- Participant Metadata Handlers ---

// ListParticipants godoc
// @Summary List participants in a conference
// @Description Retrieves a list of participants currently in the specified conference.
// @Tags Conferences-Participants
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   conf_sid     path   string  true  "Conference SID"
// @Success 200 {array} ParticipantResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants [get]
func (h *ConferenceHandler) ListParticipants(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")

	participants, err := h.service.ListParticipants(c.Request.Context(), accountSid, confSid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Conference not found or not authorized"})
		} else {
			h.logger.Errorf("ListParticipants: Error for conf %s, account %s: %v", confSid, accountSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve participants", Details: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, ToParticipantResponseList(participants))
}

// GetParticipant godoc
// @Summary Get a specific conference participant
// @Description Retrieves details for a specific participant in a conference.
// @Tags Conferences-Participants
// @Produce  json
// @Param   account_sid     path   string  true  "Account SID"
// @Param   conf_sid        path   string  true  "Conference SID"
// @Param   participant_sid path   string  true  "Participant SID (CPxxxx)"
// @Success 200 {object} ParticipantResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference or Participant not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants/{participant_sid} [get]
func (h *ConferenceHandler) GetParticipant(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")
	participantSid := c.Param("participant_sid")

	participant, err := h.service.GetParticipant(c.Request.Context(), accountSid, confSid, participantSid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Conference or Participant not found"})
		} else {
			h.logger.Errorf("GetParticipant: Error for participant %s in conf %s, account %s: %v", participantSid, confSid, accountSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve participant", Details: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, ToParticipantResponse(participant))
}


// --- Live Conference Control Handlers ---

// ConferencePlayAudio godoc
// @Summary Play audio in a conference
// @Description Plays an audio file to all participants in the conference.
// @Tags Conferences-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string                       true  "Account SID"
// @Param   conf_sid     path   string                       true  "Conference SID"
// @Param   play_request body   ConferenceControlPlayRequest true  "Play Audio Request"
// @Success 202 {object} CallActionResponse "Audio playback initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference not found or not in-progress"
// @Failure 500 {object} GenericErrorResponse "Internal/ESL error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/play [post]
func (h *ConferenceHandler) ConferencePlayAudio(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")
	var req ConferenceControlPlayRequest // Alias for CallPlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}
	loop := 1; if req.Loop != nil { if *req.Loop == 0 {loop = 1} else {loop = *req.Loop} }


	jobID, err := h.service.PlayAudioInConference(c.Request.Context(), accountSid, confSid, req.URL, loop)
	if err != nil {
		h.mapConferenceControlError(c, err, "play audio in conference "+confSid)
		return
	}
	// Assuming CallActionResponse DTO is updated to include ConferenceSID, or use gin.H
	c.JSON(http.StatusAccepted, CallActionResponse{Success: true, Message: "Play audio initiated.", CallSID: confSid, JobID: jobID}) // Using CallSID to hold ConfSID for now
}

// ConferenceSayText godoc
// @Summary Speak text in a conference
// @Description Speaks text to all participants using TTS.
// @Tags Conferences-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string                      true  "Account SID"
// @Param   conf_sid     path   string                      true  "Conference SID"
// @Param   say_request  body   ConferenceControlSayRequest true  "Say Text Request"
// @Success 202 {object} CallActionResponse "Say text initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference not found or not in-progress"
// @Failure 500 {object} GenericErrorResponse "Internal/ESL error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/say [post]
func (h *ConferenceHandler) ConferenceSayText(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")
	var req ConferenceControlSayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}
	jobID, err := h.service.SayTextInConference(c.Request.Context(), accountSid, confSid, req.Text, req.Language, req.Voice)
	if err != nil {
		h.mapConferenceControlError(c, err, "say text in conference "+confSid)
		return
	}
	c.JSON(http.StatusAccepted, CallActionResponse{Success: true, Message: "Say text initiated.", CallSID: confSid, JobID: jobID}) // Using CallSID to hold ConfSID for now
}

// ConferenceRecordAction godoc
// @Summary Start or stop recording a conference
// @Description Initiates or stops recording of the conference.
// @Tags Conferences-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid    path   string                         true  "Account SID"
// @Param   conf_sid       path   string                         true  "Conference SID"
// @Param   record_request body   ConferenceControlRecordRequest true  "Record Action Request"
// @Success 202 {object} CallActionResponse "Recording action initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference not found or not in-progress"
// @Failure 500 {object} GenericErrorResponse "Internal/ESL error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/record [post]
func (h *ConferenceHandler) ConferenceRecordAction(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")
	var req ConferenceControlRecordRequest // Alias for CallRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	var jobID, recordingName, message string
	var err error

	if req.Action == CallRecordActionStart {
		recordingName, jobID, err = h.service.StartRecordingConference(c.Request.Context(), accountSid, confSid, req.FileName, req.MaxDurationSeconds, req.Format, req.PlayBeep)
		if err == nil {
			message = "Conference recording started: " + recordingName
		} else {
			message = "Conference recording start failed"
		}
	} else if req.Action == CallRecordActionStop {
		var nameOrAll string
		if req.FileName != nil && *req.FileName != "" { nameOrAll = *req.FileName } else { nameOrAll = "all" }
		jobID, err = h.service.StopRecordingConference(c.Request.Context(), accountSid, confSid, nameOrAll)
		message = "Conference recording stop initiated for: " + nameOrAll
	} else {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Invalid recording action", Details: "Action must be 'start' or 'stop'"})
		return
	}

	if err != nil {
		h.mapConferenceControlError(c, err, string(req.Action) + " recording for conference "+confSid)
		return
	}
	// If CallActionResponse is updated to include RecordingName and ConferenceSID:
	// c.JSON(http.StatusAccepted, CallActionResponse{Success: true, Message: message, ConferenceSID: &confSid, RecordingName: &recordingName, JobID: jobID})
	// For now, using CallSID to hold ConfSID and message for recording name.
	c.JSON(http.StatusAccepted, CallActionResponse{Success: true, Message: message, CallSID: confSid, JobID: jobID})
}


// --- Live Participant Control Handlers ---

// ParticipantMute godoc
// @Summary Mute or unmute a conference participant
// @Description Sets the mute status for a specific participant in a conference.
// @Tags Conferences-Participants-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid     path   string                 true  "Account SID"
// @Param   conf_sid        path   string                 true  "Conference SID"
// @Param   participant_call_sid path string              true  "Call SID of the participant (used as Member ID)"
// @Param   mute_request    body   ParticipantMuteRequest true  "Mute Request"
// @Success 202 {object} CallActionResponse "Mute/unmute action initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference or Participant not found / not in-progress"
// @Failure 500 {object} GenericErrorResponse "Internal/ESL error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants/{participant_call_sid}/mute [put]
func (h *ConferenceHandler) ParticipantMute(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")
	participantCallSid := c.Param("participant_call_sid")
	var req ParticipantMuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}
	if req.Mute == nil {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: "'mute' field (true/false) is required"})
		return
	}

	jobID, err := h.service.MuteParticipantInConference(c.Request.Context(), accountSid, confSid, participantCallSid, *req.Mute)
	if err != nil {
		h.mapConferenceControlError(c, err, "mute/unmute participant "+participantCallSid)
		return
	}
	action := "Unmute"; if *req.Mute { action = "Mute" }
	c.JSON(http.StatusAccepted, CallActionResponse{Success: true, Message: action + " action initiated for participant " + participantCallSid, CallSID: confSid, JobID: jobID}) // Using CallSID for ConfSID
}

// ParticipantKick godoc
// @Summary Kick a participant from a conference
// @Description Removes a specific participant from a conference.
// @Tags Conferences-Participants-LiveControl
// @Produce  json
// @Param   account_sid     path   string  true  "Account SID"
// @Param   conf_sid        path   string  true  "Conference SID"
// @Param   participant_call_sid path string  true  "Call SID of the participant to kick (used as Member ID)"
// @Success 202 {object} CallActionResponse "Kick action initiated"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 404 {object} GenericErrorResponse "Conference or Participant not found / not in-progress"
// @Failure 500 {object} GenericErrorResponse "Internal/ESL error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants/{participant_call_sid}/kick [post]
func (h *ConferenceHandler) ParticipantKick(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	confSid := c.Param("conf_sid")
	participantCallSid := c.Param("participant_call_sid")

	jobID, err := h.service.KickParticipantFromConference(c.Request.Context(), accountSid, confSid, participantCallSid)
	if err != nil {
		h.mapConferenceControlError(c, err, "kick participant "+participantCallSid)
		return
	}
	c.JSON(http.StatusAccepted, CallActionResponse{Success: true, Message: "Kick action initiated for participant " + participantCallSid, CallSID: confSid, JobID: jobID}) // Using CallSID for ConfSID
}


// mapConferenceControlError is a helper to map service errors to HTTP responses for control actions
func (h *ConferenceHandler) mapConferenceControlError(c *gin.Context, err error, action string) {
	h.logger.Errorf("%s: Error: %v", strings.Title(action), err)
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, services.ErrConferenceInvalidState) {
		c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Conference not found or not in a valid state for action: " + action, Details: err.Error()})
	} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) {
		c.JSON(http.StatusServiceUnavailable, GenericErrorResponse{Error: "Failed to send command to media server for action: " + action, Details: err.Error()})
	} else {
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error during action: " + action, Details: err.Error()})
	}
}
