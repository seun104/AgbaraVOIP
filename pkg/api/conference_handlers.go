package api

import (
	"agbara-go/pkg/models"
	"agbara-go/pkg/services"
	"database/sql"
	"errors"
	"fmt" // Ensure fmt is imported for the containsSubstring version from Turn 42
	"net/http"
	// "strings" // Remove if using the fmt-based containsSubstring from Turn 42

	"github.com/gin-gonic/gin"
)

// ConferenceAPI holds handlers for conference-related API endpoints.
type ConferenceAPI struct {
	conferenceService services.ConferenceService
	accountService    services.AccountService // To verify account existence
}

// NewConferenceAPI creates a new ConferenceAPI instance.
func NewConferenceAPI(conferenceService services.ConferenceService, accountService services.AccountService) *ConferenceAPI {
	return &ConferenceAPI{
		conferenceService: conferenceService,
		accountService:    accountService,
	}
}

// authAndAccountCheckForConference is similar to authAndAccountCheck in call_handlers,
// ensuring the :accountSidInPath exists and matches X-Auth-User-Sid.
func (api *ConferenceAPI) authAndAccountCheck(c *gin.Context) (string, bool) {
	authenticatedAccountSid := c.GetHeader("X-Auth-User-Sid") // Using placeholder auth
	if authenticatedAccountSid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required: X-Auth-User-Sid header missing"})
		return "", false
	}

	pathAccountSid := c.Param("accountSidInPath")
	if pathAccountSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account SID missing in path"})
		return "", false
	}

	if pathAccountSid != authenticatedAccountSid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access to this account's conferences is forbidden."})
		return "", false
	}

	// Verify account exists
	_, err := api.accountService.GetAccount(c.Request.Context(), pathAccountSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") { // Using containsSubstring from this version
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account", "details": err.Error()})
		}
		return "", false
	}
	return pathAccountSid, true // Return the validated pathAccountSid
}


// RegisterConferenceRoutes sets up the routes for the Conference API.
// Expects to be registered on a group like router.Group("/Accounts/:accountSidInPath")
func (api *ConferenceAPI) RegisterConferenceRoutes(router *gin.RouterGroup) {
	confRoutes := router.Group("/Conferences")
	{
		confRoutes.POST("", api.CreateConferenceHandler)
		confRoutes.GET("", api.ListConferencesHandler)
		confRoutes.GET("/:conferenceSid", api.GetConferenceHandler)

		participantsRoutes := confRoutes.Group("/:conferenceSid/Participants")
		{
			participantsRoutes.GET("", api.ListParticipantsHandler)
			participantsRoutes.GET("/:callSid", api.GetParticipantHandler)
			participantsRoutes.POST("/:callSid", api.MuteParticipantHandler) 
			participantsRoutes.DELETE("/:callSid", api.KickParticipantHandler)
		}

		confRoutes.POST("/:conferenceSid/Record", api.StartRecordingHandler)
		confRoutes.DELETE("/:conferenceSid/Record", api.StopRecordingHandler)
		confRoutes.POST("/:conferenceSid/Play", api.PlayAudioHandler)
		confRoutes.DELETE("/:conferenceSid/Play", api.StopAudioHandler)
	}
}

func (api *ConferenceAPI) CreateConferenceHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	var req models.CreateConferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}
	conference, err := api.conferenceService.CreateConference(c.Request.Context(), accountSid, req.FriendlyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conference", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, conference)
}

func (api *ConferenceAPI) ListConferencesHandler(c *gin.Context) {
	accountSid, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferences, err := api.conferenceService.ListConferences(c.Request.Context(), accountSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conferences", "details": err.Error()})
		return
	}
	if conferences == nil {
		conferences = []*models.Conference{}
	}
	c.JSON(http.StatusOK, conferences)
}

func (api *ConferenceAPI) GetConferenceHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c) 
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	if conferenceSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID missing in path"})
		return
	}
	conference, err := api.conferenceService.GetConference(c.Request.Context(), conferenceSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conference not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conference", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, conference)
}

func (api *ConferenceAPI) ListParticipantsHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	if conferenceSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID missing in path"})
		return
	}
	_, err := api.conferenceService.GetConference(c.Request.Context(), conferenceSid)
	if err != nil {
         if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conference not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify conference existence", "details": err.Error()})
		}
		return
	}
	participants, err := api.conferenceService.ListParticipants(c.Request.Context(), conferenceSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list participants", "details": err.Error()})
		return
	}
	if participants == nil {
		participants = []*models.Participant{}
	}
	c.JSON(http.StatusOK, participants)
}

func (api *ConferenceAPI) GetParticipantHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	callSid := c.Param("callSid") 
	if conferenceSid == "" || callSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID or Participant Call SID missing in path"})
		return
	}
	participant, err := api.conferenceService.GetParticipant(c.Request.Context(), conferenceSid, callSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve participant", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, participant)
}

func (api *ConferenceAPI) MuteParticipantHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	callSid := c.Param("callSid")
	if conferenceSid == "" || callSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID or Participant Call SID missing in path"})
		return
	}
	var req models.MuteParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload for mute/unmute", "details": err.Error()})
		return
	}
	updatedParticipant, err := api.conferenceService.MuteParticipant(c.Request.Context(), conferenceSid, callSid, req.Muted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found for mute/unmute"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mute/unmute participant", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, updatedParticipant) 
}

func (api *ConferenceAPI) KickParticipantHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	callSid := c.Param("callSid")
	if conferenceSid == "" || callSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID or Participant Call SID missing in path"})
		return
	}
	err := api.conferenceService.KickParticipant(c.Request.Context(), conferenceSid, callSid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsSubstring(err.Error(), "not found") { 
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found to kick"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to kick participant", "details": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent) 
}

func (api *ConferenceAPI) StartRecordingHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	if conferenceSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID missing in path"})
		return
	}
	var req models.ConferenceRecordRequest
	_ = c.ShouldBindJSON(&req) // Error ignored as per original logic for this specific handler
	
	response, err := api.conferenceService.StartRecording(c.Request.Context(), conferenceSid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start conference recording (stub)", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (api *ConferenceAPI) StopRecordingHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	if conferenceSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID missing in path"})
		return
	}
	response, err := api.conferenceService.StopRecording(c.Request.Context(), conferenceSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop conference recording (stub)", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response) 
}

func (api *ConferenceAPI) PlayAudioHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	if conferenceSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID missing in path"})
		return
	}
	var req models.ConferencePlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload for play audio", "details": err.Error()})
		return
	}
	response, err := api.conferenceService.PlayAudio(c.Request.Context(), conferenceSid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to play audio to conference (stub)", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (api *ConferenceAPI) StopAudioHandler(c *gin.Context) {
	_, ok := api.authAndAccountCheck(c)
	if !ok {
		return
	}
	conferenceSid := c.Param("conferenceSid")
	if conferenceSid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conference SID missing in path"})
		return
	}
	response, err := api.conferenceService.StopAudio(c.Request.Context(), conferenceSid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop audio in conference (stub)", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response) 
}

// Using the version of containsSubstring from Turn 42 prompt for conference_handlers.go
func containsSubstring(s, substr string) bool {
    // Check if s is an error and try to get its message
    var errStr string
    if err, ok := s.(error); ok {
        errStr = err.Error()
    } else if str, ok := s.(string); ok {
        errStr = str
    } else {
        return false // Not a type we can check
    }

    // Now check errStr for substr. This is a basic substring check.
    // A more robust solution might involve specific error types or codes.
    for i := 0; i <= len(errStr)-len(substr); i++ {
        if errStr[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
