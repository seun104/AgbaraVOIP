package api

import (
	"errors"
	"net/http"
	// "strconv" // Not used in this version

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AccountSMSHandler handles API requests related to account-specific SMS messages.
// This is different from the existing SMSHandler which handles inbound gateway webhooks.
type AccountSMSHandler struct {
	service services.ISMSService
	logger  *logrus.Entry
}

// NewAccountSMSHandler creates a new AccountSMSHandler.
func NewAccountSMSHandler(service services.ISMSService, logger *logrus.Logger) *AccountSMSHandler {
	return &AccountSMSHandler{
		service: service,
		logger:  logger.WithField("handler", "account_sms"),
	}
}

// SendSMS godoc
// @Summary Send an SMS message
// @Description Sends an outbound SMS message from the authenticated account.
// @Tags SMS
// @Accept  json
// @Produce  json
// @Param   account_sid  path   string          true  "Account SID"
// @Param   sms_request  body   SendSMSRequest  true  "SMS Send Request"
// @Success 201 {object} SMSMessageResponse "SMS message created and queued/sent"
// @Failure 400 {object} GenericErrorResponse "Invalid request payload or validation error"
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 500 {object} GenericErrorResponse "Internal server error or gateway error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/sms/messages [post]
func (h *AccountSMSHandler) SendSMS(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID)) // From JWT

	var req SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("SendSMS: Validation error for account %s: %v", accountSid, err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	// Basic validation (service layer should also validate, e.g., 'from' number ownership)
	if req.From == "" || req.To == "" || req.Body == "" {
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: "'from', 'to', and 'body' are required."})
		return
	}

	smsMessage, err := h.service.SendSMSViaAPI(c.Request.Context(), accountSid, req.From, req.To, req.Body, req.StatusCallbackURL)
	if err != nil {
		h.logger.Errorf("SendSMS: Error for account %s: %v", accountSid, err)
		if errors.Is(err, domain.ErrValidationFailed) { // Assuming service returns this for biz logic validation
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Failed to send SMS due to validation", Details: err.Error()})
		} else { // For gateway errors or other internal errors
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Failed to send SMS", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("SendSMS: SMS %s initiated for account %s", smsMessage.SID, accountSid)
	c.JSON(http.StatusCreated, ToSMSMessageResponse(smsMessage))
}

// GetSMSMessage godoc
// @Summary Get SMS message details
// @Description Retrieves details for a specific SMS message belonging to the authenticated account.
// @Tags SMS
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   sms_sid      path   string  true  "SMS Message SID"
// @Success 200 {object} SMSMessageResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 404 {object} GenericErrorResponse "SMS message not found"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/sms/messages/{sms_sid} [get]
func (h *AccountSMSHandler) GetSMSMessage(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))
	smsSid := c.Param("sms_sid")

	smsMessage, err := h.service.GetSMSBySID(c.Request.Context(), accountSid, smsSid)
	if err != nil {
		h.logger.Warnf("GetSMSMessage: Error for account %s, SMS SID %s: %v", accountSid, smsSid, err)
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "SMS message not found"})
		} else {
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve SMS message", Details: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, ToSMSMessageResponse(smsMessage))
}

// ListSMSMessages godoc
// @Summary List SMS messages for an account
// @Description Retrieves a list of SMS messages associated with the authenticated account. Supports filtering.
// @Tags SMS
// @Produce  json
// @Param   account_sid  path   string  true  "Account SID"
// @Param   to query string false "Filter by recipient phone number"
// @Param   from query string false "Filter by sender phone number/ID"
// @Param   status query string false "Filter by SMS status (e.g., sent, failed, delivered)"
// @Param   direction query string false "Filter by SMS direction (e.g., outbound-api, inbound)"
// @Param   date_from query string false "Filter by date from (YYYY-MM-DD)"
// @Param   date_to query string false "Filter by date to (YYYY-MM-DD)"
// @Success 200 {array} SMSMessageResponse
// @Failure 401 {object} GenericErrorResponse "Unauthorized"
// @Failure 403 {object} GenericErrorResponse "Forbidden"
// @Failure 500 {object} GenericErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/accounts/{account_sid}/sms/messages [get]
func (h *AccountSMSHandler) ListSMSMessages(c *gin.Context) {
	accountSid := c.GetString(string(ContextKeyAccountSID))

	filters := make(map[string]interface{})
	if val := c.Query("to"); val != "" { filters["to"] = val }
	if val := c.Query("from"); val != "" { filters["from"] = val }
	if val := c.Query("status"); val != "" { filters["status"] = val }
	if val := c.Query("direction"); val != "" { filters["direction"] = val }
	if val := c.Query("date_from"); val != "" { filters["date_from"] = val }
	if val := c.Query("date_to"); val != "" { filters["date_to"] = val }
	// Add limit/offset query params for pagination if desired

	smsMessages, err := h.service.ListSMSMessages(c.Request.Context(), accountSid, filters)
	if err != nil {
		h.logger.Errorf("ListSMSMessages: Error for account %s: %v", accountSid, err)
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve SMS messages", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ToSMSMessageResponseList(smsMessages))
}
