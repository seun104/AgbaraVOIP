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


