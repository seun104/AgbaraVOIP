package api

import (
	"agbara-go/pkg/models"
	"agbara-go/pkg/services"
	"database/sql" // For errors.Is(err, sql.ErrNoRows)
	"errors"
	"fmt"
	"net/http"
	"strings" // For strings.Contains

	"github.com/gin-gonic/gin"
)

// AccountAPI holds handlers for account-related API endpoints.
type AccountAPI struct {
	accountService services.AccountService
}

// NewAccountAPI creates a new AccountAPI instance.
func NewAccountAPI(accountService services.AccountService) *AccountAPI {
	return &AccountAPI{accountService: accountService}
}

// Helper to simulate getting authenticated account SID from context (set by a middleware)
func getAuthenticatedAccountSid(c *gin.Context) (string, error) {
	// In a real app, this would be set by an auth middleware after token validation.
	// Example: sid, exists := c.Get("authenticatedAccountSid")
	// if !exists { return "", errors.New("authentication required") }
	// return sid.(string), nil

	// For this exercise, let's assume an "X-Auth-User-Sid" header for simplicity.
	// THIS IS NOT PRODUCTION-READY AUTH.
	authUserSid := c.GetHeader("X-Auth-User-Sid")
	if authUserSid == "" {
		// For /Accounts/Master, auth might not be needed or is different.
		// For other routes, this would be an error.
		// Let's allow it to be empty and handlers will decide.
		// return "", errors.New("missing X-Auth-User-Sid header for authenticated operations")
	}
	return authUserSid, nil
}


// RegisterAccountRoutes sets up the routes for the Account API.
func (api *AccountAPI) RegisterAccountRoutes(router *gin.RouterGroup) {
	accountRoutes := router.Group("/Accounts")
	{
		accountRoutes.POST("/Master", api.CreateMasterAccountHandler)
		accountRoutes.GET("", api.ListSubAccountsHandler)      // List authenticated user's sub-accounts
		accountRoutes.POST("", api.CreateSubAccountHandler)    // Create sub-account for authenticated user
		
		accountRoutes.GET("/:accountSid", api.GetAccountHandler) // Get own account details
		accountRoutes.POST("/:accountSid", api.ModifyAccountStatusHandler) // Modify own account status
        accountRoutes.POST("/:accountSid/AuthToken", api.RegenerateAuthTokenHandler) // Regenerate auth token
        // Add route for ChangeAccountType if desired:
        // accountRoutes.POST("/:accountSid/Type", api.ModifyAccountTypeHandler) 
	}
}

// CreateMasterAccountHandler handles POST /Accounts/Master
func (api *AccountAPI) CreateMasterAccountHandler(c *gin.Context) {
	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	account, err := api.accountService.CreateMasterAccount(c.Request.Context(), req.FriendlyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create master account", "details": err.Error()})
		return
	}
	// Important: The raw token is NOT in the account model for JSON.
	// The C# API returns the Account object. If a token needs to be returned upon creation,
	// it should be handled explicitly here. For now, just return the account.
	// The service method CreateMasterAccount *does not* return the raw token with the account object.
	// If the token needs to be returned here, a separate call to GenerateAuthToken would be needed,
	// or the service method would need to be changed.  The current Account model also has AuthToken as json:"-".
	// Let's assume the requirement is to return the account object *without* the token,
	// and the client must use the "regenerate" endpoint if they need a new token value.
	c.JSON(http.StatusCreated, account)
}

// ListSubAccountsHandler handles GET /Accounts (lists sub-accounts of authenticated user)
func (api *AccountAPI) ListSubAccountsHandler(c *gin.Context) {
	authenticatedAccountSid, err := getAuthenticatedAccountSid(c)
	if err != nil || authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (X-Auth-User-Sid header missing or invalid)"})
		return
	}

	accounts, err := api.accountService.ListSubAccounts(c.Request.Context(), authenticatedAccountSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sub-accounts", "details": err.Error()})
		return
	}
	if accounts == nil { // Should be handled by service returning empty slice
		accounts = []*models.Account{}
	}
	c.JSON(http.StatusOK, accounts)
}

// CreateSubAccountHandler handles POST /Accounts (creates sub-account for authenticated user)
func (api *AccountAPI) CreateSubAccountHandler(c *gin.Context) {
	authenticatedAccountSid, err := getAuthenticatedAccountSid(c)
	if err != nil || authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (X-Auth-User-Sid header missing or invalid)"})
		return
	}

	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	account, err := api.accountService.CreateSubAccount(c.Request.Context(), req.FriendlyName, authenticatedAccountSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sub-account", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, account)
}

// GetAccountHandler handles GET /Accounts/{accountSid}
// This should only allow a user to fetch their own account details.
func (api *AccountAPI) GetAccountHandler(c *gin.Context) {
	pathAccountSid := c.Param("accountSid")
	authenticatedAccountSid, err := getAuthenticatedAccountSid(c)
	if err != nil || authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (X-Auth-User-Sid header missing or invalid)"})
		return
	}

	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access to this account is forbidden."})
		return
	}

	account, err := api.accountService.GetAccount(c.Request.Context(), pathAccountSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, account)
}

// ModifyAccountStatusHandler handles POST /Accounts/{accountSid}
// This should only allow a user to modify their own account.
func (api *AccountAPI) ModifyAccountStatusHandler(c *gin.Context) {
	pathAccountSid := c.Param("accountSid")
	authenticatedAccountSid, err := getAuthenticatedAccountSid(c)
	if err != nil || authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (X-Auth-User-Sid header missing or invalid)"})
		return
	}

	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Modification of this account is forbidden."})
		return
	}

	var req models.ChangeAccountStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
    // Additional validation for status value if needed (e.g. against known constants)
    // For now, relying on the AccountStatus type and service layer if it does validation.

	account, err := api.accountService.ChangeAccountStatus(c.Request.Context(), pathAccountSid, req.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found for status update"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change account status", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, account)
}

// RegenerateAuthTokenHandler handles POST /Accounts/{accountSid}/AuthToken
func (api *AccountAPI) RegenerateAuthTokenHandler(c *gin.Context) {
    pathAccountSid := c.Param("accountSid")
    authenticatedAccountSid, err := getAuthenticatedAccountSid(c) // Assuming only self can regenerate token
    if err != nil || authenticatedAccountSid == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
        return
    }
    if pathAccountSid != authenticatedAccountSid {
        c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden to regenerate token for this account"})
        return
    }

    plainToken, err := api.accountService.GenerateAuthToken(c.Request.Context(), pathAccountSid)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
            c.JSON(http.StatusNotFound, gin.H{"error": "Account not found for token regeneration"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate auth token", "details": err.Error()})
        }
        return
    }
    // Return the new plain text token. This is sensitive and should be shown only once.
    c.JSON(http.StatusOK, gin.H{"accountSid": pathAccountSid, "authToken": plainToken})
}


// containsSubstring is a helper (can be moved to a common place)
// Replaced with strings.Contains for simplicity in this context, assuming case-insensitivity is not critical for "not found".
// For more robust error handling, specific error types or checking wrapped errors is preferred.
// func containsSubstring(s, substr string) bool {
//     return errors.Is(fmt.Errorf(s), fmt.Errorf(substr)) || // Basic check
//            (len(s) >= len(substr) && s[len(s)-len(substr):] == substr) || // Suffix
//            (len(s) >= len(substr) && s[:len(substr)] == substr) || // Prefix
//            (strings.Contains(s, substr)); // General substring
// }
// A better way for error checking specific wrapped errors is errors.As or custom error types.
// For "not found" from service: errors.Is(err, sql.ErrNoRows) is good if the service wraps sql.ErrNoRows.
// The service implementation fmt.Errorf("...: %w", err) does wrap it.

// Note: The ModifyAccountTypeHandler is commented out in RegisterAccountRoutes.
// If it were to be implemented, it would look similar to ModifyAccountStatusHandler:
/*
func (api *AccountAPI) ModifyAccountTypeHandler(c *gin.Context) {
	pathAccountSid := c.Param("accountSid")
	authenticatedAccountSid, err := getAuthenticatedAccountSid(c)
	if err != nil || authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Modification of this account is forbidden."})
		return
	}

	var req struct { // Define an inline struct or a new model in pkg/models
		Type models.AccountType `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	account, err := api.accountService.ChangeAccountType(c.Request.Context(), pathAccountSid, req.Type)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found for type update"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change account type", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, account)
}
*/
