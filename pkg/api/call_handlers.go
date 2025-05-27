package api

import (
	"agbara-go/pkg/models"
	"agbara-go/pkg/services"
	"database/sql" 
	"errors"       
	"fmt"          
	"net/http"
	"time" 

	"github.com/gin-gonic/gin"
)

// CallAPI struct and NewCallAPI (ensure it's the version that accepts AccountService)
// ... (ensure CallAPI and NewCallAPI are the version from Turn 36/38 - with accountService)
type CallAPI struct {
	callService    services.CallService
	accountService services.AccountService 
}

func NewCallAPI(callService services.CallService, accountService services.AccountService) *CallAPI {
	return &CallAPI{
		callService:    callService,
		accountService: accountService,
	}
}


// getAuthenticatedAccountSidForCall and authAndAccountCheck (ensure these are present and correct)
// ... (ensure these helpers are the version from Turn 36/38)
func getAuthenticatedAccountSidForCall(c *gin.Context) (string, error) {
	authUserSid := c.GetHeader("X-Auth-User-Sid")
	if authUserSid == "" {
		return "", errors.New("authentication required: X-Auth-User-Sid header missing")
	}
	return authUserSid, nil
}

func (api *CallAPI) authAndAccountCheck(c *gin.Context) (string, bool) {
	authenticatedAccountSid, err := getAuthenticatedAccountSidForCall(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return "", false
	}

	pathAccountSid := c.Param("accountSidInPath") 
	if pathAccountSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account SID missing in path"})
		return "", false
	}

	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access to this account's calls is forbidden."})
		return "", false
	}

	_, err = api.accountService.GetAccount(c.Request.Context(), pathAccountSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") { 
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account", "details": err.Error()})
		}
		return "", false
	}
	return pathAccountSid, true 
}


// RegisterCallRoutes (ensure this is present and correct)
// ... (ensure this is the version from Turn 36/38)
func (api *CallAPI) RegisterCallRoutes(router *gin.RouterGroup) {
	callsRoutes := router.Group("/Calls")
	{
		callsRoutes.GET("", api.ListCallsHandler)
		callsRoutes.POST("/Call", api.MakeCallHandler) 
		callsRoutes.POST("/:callSid", api.ModifyCallHandler)
	}
}


// ListCallsHandler (ensure this is present and correct)
// ... (ensure this is the version from Turn 36/38)
func (api *CallAPI) ListCallsHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}

	calls, err := api.callService.ListCalls(c.Request.Context(), accountSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve calls", "details": err.Error()})
		return
	}
	if calls == nil {
		calls = []*models.Call{}
	}
	c.JSON(http.StatusOK, calls)
}

// Modify MakeCallHandler
func (api *CallAPI) MakeCallHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}

	var req models.CallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Validation: Ensure either ApplicationSid or AnswerUrl is provided for call handling.
	if req.ApplicationSid == "" && req.AnswerUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either applicationSid or answerUrl must be provided to handle the call."})
		return
	}
    if req.ApplicationSid != "" && req.AnswerUrl != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide either applicationSid or answerUrl, not both."})
		return
	}


	// Populate the call model for the service
	call := &models.Call{
		AccountSid:     accountSid,
		CallerId:       req.From, // CallerId from request 'From' field
		CallTo:         req.To,
		ApplicationSid: req.ApplicationSid, // Pass ApplicationSid
		AnswerUrl:      req.AnswerUrl,      // Pass AnswerUrl (service will prioritize AppSid if present)
		Direction:      "outbound-api",
		Status:         models.CallStatusQueued, // Initial status, service might change to Initiating quickly
		Timeout:        req.TimeLimit,          // Pass TimeLimit as Timeout string
		// Price and Duration will use defaults or be set by service/later actions.
		// Timestamps (DateCreated, DateUpdated, StartTime, EndTime) handled by service.
	}
    if call.StartTime.IsZero() { 
        call.StartTime = time.Now().UTC()
    }
    
	createdCall, err := api.callService.CreateCall(c.Request.Context(), call)
	if err != nil {
		// Check if the error is due to FS origination failure, which might have specific error messages
        // The service layer now returns the call object even on FS failure, with status 'failed'.
        if createdCall != nil && createdCall.Status == models.CallStatusFailed {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Failed to originate call via FreeSWITCH", 
                "details": err.Error(),
                "call_record": createdCall, // Return the created call record which includes the SID
            })
        } else {
		    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create call", "details": err.Error()})
        }
		return
	}
	// If call origination was attempted and successful, status might be Ringing or InProgress.
	// If ESL connection was nil, service returns error and call status might be Queued or Failed.
	if createdCall.Status == models.CallStatusFailed {
        c.JSON(http.StatusInternalServerError, gin.H{ // Or perhaps a 201 with a warning if record created but FS failed
            "message": "Call record created but FreeSWITCH origination failed.",
            "call": createdCall,
            "error_details": err.Error(), // This might be nil if service handled the error message itself
        })
    } else {
	    c.JSON(http.StatusCreated, createdCall)
    }
}

// ModifyCallHandler (ensure this is present and correct, including auth check)
// ... (ensure this is the version from Turn 36/38)
func (api *CallAPI) ModifyCallHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	callSid := c.Param("callSid")
	if callSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Call SID missing in path"})
		return
	}

	var req models.CallRequest 
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload for modify", "details": err.Error()})
		return
	}

	existingCall, err := api.callService.GetCall(c.Request.Context(), callSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") { 
			c.JSON(http.StatusNotFound, gin.H{"error": "Call not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve call for modification", "details": err.Error()})
		}
		return
	}

	if existingCall.AccountSid != accountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access to modify this call is forbidden."})
		return
	}

    // For Modify, primarily AnswerUrl was considered updatable via this generic request.
    // ApplicationSid is usually set at creation. If it needs to change, it's a more complex operation.
	if req.AnswerUrl != "" {
		existingCall.AnswerUrl = req.AnswerUrl
	}
    if req.StatusCallbackUrl != "" {
        // If Call model had StatusCallbackUrl, it would be updated here.
        // existingCall.StatusCallbackUrl = req.StatusCallbackUrl
    }
    // Potentially update other fields if they are part of CallRequest and deemed modifiable here.
    // e.g., if req.TimeLimit is provided, update existingCall.Timeout = req.TimeLimit

	updatedCall, err := api.callService.UpdateCall(c.Request.Context(), existingCall)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to modify call", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedCall)
}


// containsSubstring (ensure this is present and correct)
// ... (ensure this is the version from Turn 36/38, or the one from Turn 42/44 if standardized)
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
