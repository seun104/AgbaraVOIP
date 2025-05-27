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
type CallAPI struct {
	callService    services.CallService
	accountService services.AccountService 
}

func NewCallAPI(callService services.CallService, accountService services.AccountService) *CallAPI {
	return &CallAPI{callService: callService, accountService: accountService}
}


// getAuthenticatedAccountSidForCall and authAndAccountCheck (ensure these are present and correct)
func getAuthenticatedAccountSidForCall(c *gin.Context) (string, error) { 
	authUserSid := c.GetHeader("X-Auth-User-Sid"); if authUserSid == "" { return "", errors.New("authentication required: X-Auth-User-Sid header missing") }; return authUserSid, nil
}
func (api *CallAPI) authAndAccountCheck(c *gin.Context) (string, bool) { 
	authenticatedAccountSid, err := getAuthenticatedAccountSidForCall(c)
	if err != nil { c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()}); return "", false }
	pathAccountSid := c.Param("accountSidInPath") 
	if pathAccountSid == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Account SID missing in path"}); return "", false }
	if pathAccountSid != authenticatedAccountSid { c.JSON(http.StatusForbidden, gin.H{"error": "Access to this account's calls is forbidden."}); return "", false }
	_, err = api.accountService.GetAccount(c.Request.Context(), pathAccountSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") { 
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account", "details": err.Error()}) }
		return "", false
	}
	return pathAccountSid, true 
}


// RegisterCallRoutes (ensure this is present and correct)
func (api *CallAPI) RegisterCallRoutes(router *gin.RouterGroup) { 
	callsRoutes := router.Group("/Calls")
	{	callsRoutes.GET("", api.ListCallsHandler); callsRoutes.POST("/Call", api.MakeCallHandler); callsRoutes.POST("/:callSid", api.ModifyCallHandler) }
}

// ListCallsHandler (ensure this is present and correct)
func (api *CallAPI) ListCallsHandler(c *gin.Context) { 
	accountSid, ok := api.authAndAccountCheck(c); if !ok { return }
	calls, err := api.callService.ListCalls(c.Request.Context(), accountSid)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve calls", "details": err.Error()}); return }
	if calls == nil { calls = []*models.Call{} }
	c.JSON(http.StatusOK, calls)
}


// Modify MakeCallHandler to pass the full callRequest DTO
func (api *CallAPI) MakeCallHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}

	var callReq models.CallRequest // Changed from req to callReq for clarity
	if err := c.ShouldBindJSON(&callReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Validation: Ensure either ApplicationSid or AnswerUrl is provided for call handling.
	if callReq.ApplicationSid == "" && callReq.AnswerUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either applicationSid or answerUrl must be provided to handle the call."})
		return
	}
    if callReq.ApplicationSid != "" && callReq.AnswerUrl != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide either applicationSid or answerUrl, not both."})
		return
	}


	// Populate the base Call model for durable storage
	// Transient fields from callReq (like SendDigits, StatusCallbackUrl) will be used by service directly
	call := &models.Call{
		AccountSid:     accountSid,
		CallerId:       callReq.From, 
		CallTo:         callReq.To,
		ApplicationSid: callReq.ApplicationSid, 
		AnswerUrl:      callReq.AnswerUrl,      
		Direction:      "outbound-api",
		Status:         models.CallStatusQueued, 
		Timeout:        callReq.TimeLimit, // Service will parse this for 'call_timeout' var
		// Price and Duration are defaults.
		// StartTime, DateCreated, DateUpdated handled by service.
	}
    if call.StartTime.IsZero() { 
        call.StartTime = time.Now().UTC()
    }
    
    // Pass the full callReq DTO to the service method
	createdCall, err := api.callService.CreateCall(c.Request.Context(), call, &callReq) // <-- MODIFIED HERE
	if err != nil {
        if createdCall != nil && createdCall.Status == models.CallStatusFailed {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Failed to originate call via FreeSWITCH", 
                "details": err.Error(),
                "call_record": createdCall, 
            })
        } else {
		    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create call", "details": err.Error()})
        }
		return
	}
	
	if createdCall.Status == models.CallStatusFailed {
        c.JSON(http.StatusInternalServerError, gin.H{ 
            "message": "Call record created but FreeSWITCH origination failed.",
            "call": createdCall,
            "error_details": err.Error(), 
        })
    } else {
	    c.JSON(http.StatusCreated, createdCall)
    }
}

// ModifyCallHandler and containsSubstring
// (Ensure these are the correct, complete versions from Turn 74/75)
func (api *CallAPI) ModifyCallHandler(c *gin.Context) { 
	accountSid, ok := api.authAndAccountCheck(c); if !ok { return }
	callSid := c.Param("callSid"); if callSid == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Call SID missing"}); return }
	var req models.CallRequest; if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload", "details": err.Error()}); return }
	existingCall, err := api.callService.GetCall(c.Request.Context(), callSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") { c.JSON(http.StatusNotFound, gin.H{"error": "Call not found"}) } else { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve call", "details": err.Error()}) }
		return
	}
	if existingCall.AccountSid != accountSid { c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"}); return }
	if req.AnswerUrl != "" { existingCall.AnswerUrl = req.AnswerUrl }
	// Update other fields if necessary from req
    if req.StatusCallbackUrl != "" {
        // If Call model had StatusCallbackUrl, it would be updated here.
        // existingCall.StatusCallbackUrl = req.StatusCallbackUrl 
    }
	updatedCall, err := api.callService.UpdateCall(c.Request.Context(), existingCall)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to modify call", "details": err.Error()}); return }
	c.JSON(http.StatusOK, updatedCall)
}
func containsSubstring(s, substr string) bool { 
    var errStr string; if err, ok := s.(error); ok { errStr = err.Error() } else if str, ok := s.(string); ok { errStr = str } else { return false }
    for i := 0; i <= len(errStr)-len(substr); i++ { if errStr[i:i+len(substr)] == substr { return true } }; return false
}
