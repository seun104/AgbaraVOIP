package api

import (
	"errors"
	"net/http"

	"github.com/user/agbaravoip_golang/internal/domain" 
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var _ *domain.Account 

type AccountHandler struct {
	service services.IAccountService 
	logger  *logrus.Entry
}

func NewAccountHandler(service services.IAccountService, logger *logrus.Logger) *AccountHandler {
	return &AccountHandler{
		service: service,
		logger:  logger.WithField("handler", "account"),
	}
}

func (h *AccountHandler) CreateMasterAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Validation error for CreateMasterAccount: %v", err)
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: err.Error()})
		return
	}

	if req.AuthToken == "" {
		h.logger.Error("AuthToken is required for CreateMasterAccount but was empty")
		c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Validation failed", Details: "auth_token is required"})
		return
	}

	account, err := h.service.CreateMasterAccount(req.FriendlyName, req.AuthToken)
	if err != nil {
		if errors.Is(err, services.ErrValidationFailed) || errors.Is(err, services.ErrDuplicateSID) {
			h.logger.Warnf("Failed to create master account due to validation/duplicate: %v", err)
			c.JSON(http.StatusBadRequest, GenericErrorResponse{Error: "Failed to create account", Details: err.Error()})
		} else {
			h.logger.Errorf("Internal error creating master account: %v", err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not create account", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Master account created successfully: %s", account.SID)
	c.JSON(http.StatusCreated, ToAccountResponse(account))
}

func (h *AccountHandler) GetAccount(c *gin.Context) {
	// Get account SID from path parameter
	requestedAccountSid := c.Param("account_sid")

	// Retrieve authenticated account SID from context (set by auth middleware)
	authAccountSidVal, exists := c.Get(ContextAuthAccountKey)
	if !exists {
		h.logger.Error("Critical: Authenticated account_sid not found in context for GetAccount. Auth middleware not effective.")
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error", Details: "Authentication context missing."})
		return
	}
	
	authAccountSid, ok := authAccountSidVal.(string)
	if !ok {
		h.logger.Error("Critical: Authenticated account_sid in context is not a string.")
		c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Internal server error", Details: "Invalid authentication context."})
		return
	}

	// Authorization: User can only get their own account details via this endpoint.
	// Sub-account access should be through a different endpoint like /accounts/{parent_sid}/subaccounts/{sub_sid}
	if requestedAccountSid != authAccountSid {
		h.logger.Warnf("Authorization failed: Authenticated SID %s tried to access SID %s via GetAccount endpoint.", authAccountSid, requestedAccountSid)
		c.JSON(http.StatusForbidden, GenericErrorResponse{Error: "Forbidden", Details: "You can only retrieve your own account details."})
		return
	}

	// Now, proceed to fetch the account using the authenticated SID (which matches the path param)
	account, err := h.service.GetAccountBySID(authAccountSid)
	if err != nil {
		if errors.Is(err, services.ErrAccountNotFound) {
			// This case should ideally not happen if auth succeeded for this SID, but good for defense.
			h.logger.Warnf("Account %s not found in GetAccount, though it was authenticated.", authAccountSid)
			c.JSON(http.StatusNotFound, GenericErrorResponse{Error: "Account not found"})
		} else {
			h.logger.Errorf("Error getting account %s: %v", authAccountSid, err)
			c.JSON(http.StatusInternalServerError, GenericErrorResponse{Error: "Could not retrieve account", Details: err.Error()})
		}
		return
	}

	h.logger.Infof("Account %s retrieved successfully by authenticated user", account.SID)
	c.JSON(http.StatusOK, ToAccountResponse(account))
}
