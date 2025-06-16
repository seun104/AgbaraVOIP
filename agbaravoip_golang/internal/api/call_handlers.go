package api

import (
	"errors"
	"net/http"
	// "strings" // Not directly used now

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var _ *domain.Call 

type CallHandler struct {
	callService services.ICallService 
	appService  services.IApplicationService 
	logger      *logrus.Entry
}

func NewCallHandler(cs services.ICallService, as services.IApplicationService, logger *logrus.Logger) *CallHandler {
	return &CallHandler{
		callService: cs,
		appService:  as,
		logger:      logger.WithField("handler", "call"),
	}
}

func (h *CallHandler) CreateCall(c *gin.Context) {
	authAccountSid := c.GetString(ContextAuthAccountKey) 
	var req CreateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for CreateCall: %v", err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}
	if req.From == "" || req.To == "" { // Gin "required" binding handles this, but defense in depth
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: "from and to fields are required"})
		return
	}
	
	finalAnswerURL := req.AnswerURL
	var appSidForCallRecord *string = req.ApplicationSID

	if req.ApplicationSID != nil && *req.ApplicationSID != "" {
		if finalAnswerURL == "" { 
			app, err := h.appService.GetApplicationBySID(authAccountSid, *req.ApplicationSID)
			if err != nil {
				if errors.Is(err, services.ErrApplicationNotFound) {
					c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Invalid Application SID", Details: "Application not found."})
				} else { c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve application details"}) }
				return
			}
			finalAnswerURL = app.VoiceURL 
		}
	} else { appSidForCallRecord = nil }

	if finalAnswerURL == "" {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: "answer_url or valid application_sid is required"})
		return
	}
	
	call, err := h.callService.OriginateCall(authAccountSid, appSidForCallRecord, req.From, req.To, finalAnswerURL, req.TimeoutSeconds)
	if err != nil {
		// Use specific errors from call_service for better client feedback if needed
		if errors.Is(err, services.ErrCallValidationFailed_CS) || errors.Is(err, services.ErrAppLogicError_CS) { 
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Call request invalid", Details: err.Error()})
		} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) { 
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Call origination failed at telephony level", Details: err.Error()})
		} else { 
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not initiate call", Details: err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, ToCallResponse(call))
}

func (h *CallHandler) GetCall(c *gin.Context) {
    authAccountSid := c.GetString(ContextAuthAccountKey)
    callSid := c.Param("call_sid")
    call, err := h.callService.GetCallBySID(authAccountSid, callSid)
    if err != nil {
        if errors.Is(err, services.ErrCallNotFound_CS) { // Use specific error
            c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Call not found"})
        } else { c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve call"}) }
        return
    }
    c.JSON(http.StatusOK, ToCallResponse(call))
}

func (h *CallHandler) ListCalls(c *gin.Context) {
    authAccountSid := c.GetString(ContextAuthAccountKey)
    filters := make(map[string]interface{}) 
    if val := c.Query("status"); val != "" { filters["status"] = val }
    if val := c.Query("from"); val != "" { filters["from_num"] = val } 
    if val := c.Query("to"); val != "" { filters["to_num"] = val }     

    calls, err := h.callService.ListCalls(authAccountSid, filters) // This call should now match
    if err != nil {
        c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve calls"})
        return
    }
    c.JSON(http.StatusOK, ToCallResponseList(calls))
}

// --- Live Call Control Handlers ---

// PlayAudio godoc
// @Summary Play audio on a live call
// @Description Initiates playback of an audio file on the specified call leg.
// @Tags Calls-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   call_sid     path   string  true  "Call SID of the live call"
// @Param   play_request body   CallPlayRequest true "Play Audio Request"
// @Success 202 {object} CallActionResponse "Audio playback initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request payload"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden (call does not belong to account)"
// @Failure 404 {object} GenericErrorResponse "Call not found or not in a playable state"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Failure 503 {object} GenericErrorResponse "Media server command failure"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/calls/{call_sid}/play [post]
func (h *CallHandler) PlayAudio(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID)) // Ensure using the correct context key constant
	callSid := c.Param("call_sid")

	var req CallPlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("PlayAudio: Validation error for call %s: %v", callSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	legs := "aleg" // Default
	if req.Legs != "" {
		legs = req.Legs
	}
	loop := 1 // Default to play once, as 0 might mean loop indefinitely in some FS apps, but 1 is safer for single play.
	if req.Loop != nil {
		if *req.Loop == 0 { // Treat 0 as play once for this API, underlying FS might differ.
			loop = 1
		} else {
			loop = *req.Loop
		}
	}

	jobID, err := h.callService.PlayAudioOnCall(c.Request.Context(), accountSid, callSid, req.URL, loop, legs)
	if err != nil {
		h.logger.Errorf("PlayAudio: Error for call %s: %v", callSid, err)
		if errors.Is(err, services.ErrCallNotFound_CS) || errors.Is(err, services.ErrCallInvalidState_CS) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Call not found or not in a playable state", Details: err.Error()})
		} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) {
			c.JSON(http.StatusServiceUnavailable, GenericErrorResponse{Error: "Failed to send command to media server", Details: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error", Details: err.Error()})
		}
		return
	}

	c.JSON(http.StatusAccepted, CallActionResponse{
		CallSID: callSid,
		Success: true,
		Message: "Audio playback initiated.",
		JobID:   jobID,
	})
}

// SayText godoc
// @Summary Speak text on a live call
// @Description Initiates text-to-speech on the specified call leg.
// @Tags Calls-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   call_sid     path   string  true  "Call SID of the live call"
// @Param   say_request  body   CallSayRequest true "Say Text Request"
// @Success 202 {object} CallActionResponse "Text-to-speech initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request payload"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "Call not found or not in a speakable state"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Failure 503 {object} GenericErrorResponse "Media server command failure"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/calls/{call_sid}/say [post]
func (h *CallHandler) SayText(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	callSid := c.Param("call_sid")

	var req CallSayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("SayText: Validation error for call %s: %v", callSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	legs := "aleg" // Default
	if req.Legs != "" {
		legs = req.Legs
	}

	jobID, err := h.callService.SayTextOnCall(c.Request.Context(), accountSid, callSid, req.Text, req.Language, req.Voice, legs)
	if err != nil {
		h.logger.Errorf("SayText: Error for call %s: %v", callSid, err)
		if errors.Is(err, services.ErrCallNotFound_CS) || errors.Is(err, services.ErrCallInvalidState_CS) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Call not found or not in a speakable state", Details: err.Error()})
		} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) {
			c.JSON(http.StatusServiceUnavailable, GenericErrorResponse{Error: "Failed to send command to media server", Details: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error", Details: err.Error()})
		}
		return
	}

	c.JSON(http.StatusAccepted, CallActionResponse{
		CallSID: callSid,
		Success: true,
		Message: "Text-to-speech initiated.",
		JobID:   jobID,
	})
}

// SendDTMF godoc
// @Summary Send DTMF tones on a live call
// @Description Sends a sequence of DTMF tones on the specified call leg.
// @Tags Calls-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   call_sid     path   string  true  "Call SID of the live call"
// @Param   dtmf_request body   CallDTMFRequest true "Send DTMF Request"
// @Success 202 {object} CallActionResponse "DTMF send initiated"
// @Failure 400 {object} GenericErrorResponse "Invalid request payload"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "Call not found or not in a valid state for DTMF"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Failure 503 {object} GenericErrorResponse "Media server command failure"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/calls/{call_sid}/dtmf [post]
func (h *CallHandler) SendDTMF(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	callSid := c.Param("call_sid")

	var req CallDTMFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("SendDTMF: Validation error for call %s: %v", callSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	legs := "aleg" // Default
	if req.Legs != "" {
		legs = req.Legs
	}

	jobID, err := h.callService.SendDTMFOnCall(c.Request.Context(), accountSid, callSid, req.Digits, req.DurationMs, legs)
	if err != nil {
		h.logger.Errorf("SendDTMF: Error for call %s: %v", callSid, err)
		if errors.Is(err, services.ErrCallNotFound_CS) || errors.Is(err, services.ErrCallInvalidState_CS) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Call not found or not in a valid state for DTMF", Details: err.Error()})
		} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) {
			c.JSON(http.StatusServiceUnavailable, GenericErrorResponse{Error: "Failed to send command to media server", Details: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error", Details: err.Error()})
		}
		return
	}

	c.JSON(http.StatusAccepted, CallActionResponse{
		CallSID: callSid,
		Success: true,
		Message: "DTMF send initiated.",
		JobID:   jobID,
	})
}

// RecordAction godoc
// @Summary Start or stop recording on a live call
// @Description Manages recording for a live call. 'start' initiates recording, 'stop' terminates it.
// @Tags Calls-LiveControl
// @Accept  json
// @Produce  json
// @Param   account_sid    path   string  true  "Account SID"
// @Param   call_sid       path   string  true  "Call SID of the live call"
// @Param   record_request body   CallRecordRequest true "Record Action Request"
// @Success 202 {object} CallActionResponse "Recording action processed"
// @Failure 400 {object} GenericErrorResponse "Invalid request payload"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "Call not found or not in a valid state for recording action"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Failure 503 {object} GenericErrorResponse "Media server command failure"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/calls/{call_sid}/record [post]
func (h *CallHandler) RecordAction(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	callSid := c.Param("call_sid")

	var req CallRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("RecordAction: Validation error for call %s: %v", callSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	var jobID, message, recordingName string
	var err error

	switch req.Action {
	case CallRecordActionStart:
		message = "Call recording started."
		recordingName, jobID, err = h.callService.StartRecordingCall(c.Request.Context(), accountSid, callSid, req.FileName, req.MaxDurationSeconds, req.Format, req.PlayBeep)
		if err == nil && recordingName != "" {
			message = "Call recording started: " + recordingName
		}
	case CallRecordActionStop:
		// For stop, FileName from request is interpreted as the recording to stop (if specific, else 'all' or similar).
		// The service layer's StopRecordingCall expects `recordingNameOrUUID` which could be the file path or a special value.
		// Here, we assume if FileName is provided in stop request, it's the specific recording to stop.
		// If not provided, it's a general stop (e.g. "all" if service supports it, or a specific one if only one active).
		// For simplicity, let's assume the service handles `*req.FileName` being nil for a general stop if applicable.
		// The `CallRecordRequest.FileName` can be used as the `recordingNameOrUUID` argument for stop.
		// If `req.FileName` is nil, we might pass a default like "all" or an empty string if service handles it.
		// For now, let's pass `*req.FileName` if not nil, otherwise a sensible default like the CallSID itself or "current".
        // The service method `StopRecordingCall` expects `recordingNameOrUUID string`.
        var recNameToStop string
        if req.FileName != nil && *req.FileName != "" {
            recNameToStop = *req.FileName
        } else {
            // Default to stopping a recording associated with the call if not specified.
            // This might mean stopping based on callSid or a default naming convention.
            // For simplicity, using callSid as a potential identifier if no specific name given.
            // The service layer should ideally resolve this.
            recNameToStop = callSid // Or a more specific identifier if known, like 'all'
            h.logger.Infof("RecordAction Stop: No specific FileName provided, attempting to stop recording identified by: %s", recNameToStop)
        }
		message = "Call recording stop initiated for: " + recNameToStop
		jobID, err = h.callService.StopRecordingCall(c.Request.Context(), accountSid, callSid, recNameToStop)
	default:
		h.logger.Warnf("RecordAction: Invalid action '%s' for call %s", req.Action, callSid)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Invalid recording action", Details: "Action must be 'start' or 'stop'"})
		return
	}

	if err != nil {
		h.logger.Errorf("RecordAction: Error for call %s, action %s: %v", callSid, req.Action, err)
		if errors.Is(err, services.ErrCallNotFound_CS) || errors.Is(err, services.ErrCallInvalidState_CS) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Call not found or not in a valid state for recording", Details: err.Error()})
		} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) {
			c.JSON(http.StatusServiceUnavailable, GenericErrorResponse{Error: "Failed to send recording command to media server", Details: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error during recording action", Details: err.Error()})
		}
		return
	}

	response := CallActionResponse{
		CallSID: callSid,
		Success: true,
		Message: message,
		JobID:   jobID,
	}
	// Include recordingName in message or a specific field if desired for start action.
	// For now, message includes it.

	c.JSON(http.StatusAccepted, response)
}

// HangupLiveCall godoc
// @Summary Hangup a live call
// @Description Terminates an active call.
// @Tags Calls-LiveControl
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   call_sid     path   string  true  "Call SID of the live call to hangup"
// @Param   cause        query  string  false "Optional hangup cause (e.g., NORMAL_CLEARING)"
// @Success 202 {object} CallActionResponse "Hangup command accepted"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "Call not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Failure 503 {object} GenericErrorResponse "Media server command failure"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/calls/{call_sid}/hangup [post] // POST is safer for actions, can also be DELETE
func (h *CallHandler) HangupLiveCall(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	callSid := c.Param("call_sid")
	cause := c.Query("cause") // Optional query parameter for hangup cause

	jobID, err := h.callService.HangupCall(c.Request.Context(), accountSid, callSid, cause)
	if err != nil {
		h.logger.Errorf("HangupLiveCall: Error for call %s: %v", callSid, err)
		if errors.Is(err, services.ErrCallNotFound_CS) { // No ErrCallInvalidState_CS for hangup generally
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Call not found", Details: err.Error()})
		} else if errors.Is(err, services.ErrESLCommandFailed_CS) || errors.Is(err, services.ErrESLClientNotAvailable_CS) {
			c.JSON(http.StatusServiceUnavailable, GenericErrorResponse{Error: "Failed to send hangup command to media server", Details: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error during hangup", Details: err.Error()})
		}
		return
	}

	c.JSON(http.StatusAccepted, CallActionResponse{
		CallSID: callSid,
		Success: true,
		Message: "Hangup command accepted.",
		JobID:   jobID,
	})
}

