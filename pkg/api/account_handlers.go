package api

import (
	"agbara-go/pkg/models"
	"agbara-go/pkg/services"
	"database/sql" 
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AccountAPI struct and NewAccountAPI
type AccountAPI struct {
	accountService services.AccountService
}
func NewAccountAPI(accountService services.AccountService) *AccountAPI {
	return &AccountAPI{accountService: accountService}
}
func getAuthenticatedAccountSid(c *gin.Context) (string, error) {
	authUserSid := c.GetHeader("X-Auth-User-Sid")
	if authUserSid == "" {
		return "", errors.New("authentication required: X-Auth-User-Sid header missing or invalid")
	}
	return authUserSid, nil
}


// Update RegisterAccountRoutes to include the new settings endpoint.
func (api *AccountAPI) RegisterAccountRoutes(router *gin.RouterGroup) {
	accountRoutes := router.Group("/Accounts") 
	{
		accountRoutes.POST("/Master", api.CreateMasterAccountHandler) 
		
		accountRoutes.GET("", api.ListSubAccountsHandler)      
		accountRoutes.POST("", api.CreateSubAccountHandler)    
		
		accountSidRoutes := accountRoutes.Group("/:accountSid")
		{
			accountSidRoutes.GET("", api.GetAccountHandler) 
			accountSidRoutes.POST("", api.ModifyAccountStatusHandler) 
            accountSidRoutes.PATCH("/settings", api.UpdateAccountSettingsHandler) // <-- NEW ENDPOINT
			accountSidRoutes.POST("/AuthToken", api.RegenerateAuthTokenHandler) 
		}
	}
}

// NEW HANDLER: UpdateAccountSettingsHandler
func (api *AccountAPI) UpdateAccountSettingsHandler(c *gin.Context) {
	pathAccountSid := c.Param("accountSid")
	authenticatedAccountSid, err := getAuthenticatedAccountSid(c)
	if err != nil || authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (X-Auth-User-Sid header missing or invalid)"})
		return
	}

	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own account settings."})
		return
	}

	var req models.UpdateAccountSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

    if req.Type != nil {
        if !(*req.Type == models.AccountTypeTrial || *req.Type == models.AccountTypeFull || *req.Type == "") {
             c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid account type: %s. Must be '%s' or '%s' or empty.", 
                *req.Type, models.AccountTypeTrial, models.AccountTypeFull)})
            return
        }
    }
    

	account, err := api.accountService.UpdateAccountSettings(c.Request.Context(), pathAccountSid, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found for update"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account settings", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, account)
}

// containsSubstring helper
func containsSubstring(s, substr string) bool {
    var errStr string
    if err, ok := s.(error); ok { 
        errStr = err.Error()
    } else if str, ok := s.(string); ok { 
        errStr = str
    } else {
        return false 
    }
    for i := 0; i <= len(errStr)-len(substr); i++ {
        if errStr[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}

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
	c.JSON(http.StatusCreated, account)
}
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
	if accounts == nil { accounts = []*models.Account{} }
	c.JSON(http.StatusOK, accounts)
}
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
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, account)
}
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
	account, err := api.accountService.ChangeAccountStatus(c.Request.Context(), pathAccountSid, req.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found for status update"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change account status", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, account)
}
func (api *AccountAPI) RegenerateAuthTokenHandler(c *gin.Context) {
    pathAccountSid := c.Param("accountSid")
    authenticatedAccountSid, err := getAuthenticatedAccountSid(c) 
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
        if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
            c.JSON(http.StatusNotFound, gin.H{"error": "Account not found for token regeneration"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate auth token", "details": err.Error()})
        }
        return
    }
    c.JSON(http.StatusOK, gin.H{"accountSid": pathAccountSid, "authToken": plainToken})
}
