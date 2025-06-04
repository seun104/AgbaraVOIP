package api

import (
	"agbara-go/pkg/models"   // For Application model if needed by service
	"agbara-go/pkg/services" // For CallService, ApplicationService
	"agbara-go/pkg/twiml"    // For TwiML generation
	"fmt"
	"net/http"
	"log" // For logging

	"github.com/gin-gonic/gin"
)

// VoiceControlAPI holds handlers for FreeSWITCH call control requests.
type VoiceControlAPI struct {
	callService        services.CallService
	applicationService services.ApplicationService 
	// accountService services.AccountService // Might be needed for account-specific logic later
}

// NewVoiceControlAPI creates a new VoiceControlAPI instance.
func NewVoiceControlAPI(cs services.CallService, as services.ApplicationService) *VoiceControlAPI {
	return &VoiceControlAPI{
		callService:        cs,
		applicationService: as,
	}
}

// RegisterVoiceControlRoutes sets up the routes for voice control.
// This might be a single route that Application.VoiceUrl points to.
func (api *VoiceControlAPI) RegisterVoiceControlRoutes(router *gin.RouterGroup) {
	// Example: A generic route. Specific apps might have their VoiceUrl point here with query params,
	// or the path itself could be derived from Application.VoiceUrl if it's relative.
	// For now, one main handler.
	voiceRoutes := router.Group("/voice") // Base path e.g., /api/v1/voice
	{
		voiceRoutes.POST("/control", api.HandleVoiceControl) // FreeSWITCH usually POSTs form data
		voiceRoutes.GET("/control", api.HandleVoiceControl) // Allow GET for easier browser testing
	}
}

// FreeSwitchRequest captures common parameters sent by FreeSWITCH.
// FreeSWITCH sends data as application/x-www-form-urlencoded.
type FreeSwitchCallbackParams struct {
	// Standard Channel Variables (names might vary based on FS config/lua script)
	CallSid      string `form:"CallSid"`      // Typically 'Channel-Call-UUID' or 'variable_uuid'
	AccountSid   string `form:"AccountSid"`   // Custom variable we should ensure is set by originate
	ApplicationSid string `form:"ApplicationSid"` // Custom variable for Agbara App SID
	From         string `form:"From"`         // 'Caller-Caller-ID-Number' or 'variable_caller_id_number'
	To           string `form:"To"`           // 'Caller-Destination-Number' or 'variable_destination_number'
	CallStatus   string `form:"CallStatus"`   // 'Channel-State' or 'Answer-State'
	Direction    string `form:"Direction"`    // 'Call-Direction'
	Digits       string `form:"Digits"`       // From <Gather> if available, often 'DTMF-Digit' or 'variable_digits'
	// Add other relevant FreeSWITCH variables as needed
}


// HandleVoiceControl is the main handler for incoming FreeSWITCH HTTP requests.
func (api *VoiceControlAPI) HandleVoiceControl(c *gin.Context) {
	var params FreeSwitchCallbackParams
	// FreeSWITCH usually sends data as x-www-form-urlencoded
	if err := c.ShouldBind(&params); err != nil {
		log.Printf("ERROR: Failed to bind FreeSWITCH params: %v\n", err)
		// Respond with TwiML <Hangup/> or an error message if possible,
		// but FreeSWITCH might not handle non-200 well without specific config.
		// For now, send a simple error response that might not be valid TwiML for FS.
		c.XML(http.StatusBadRequest, gin.H{"error": "Invalid parameters", "details": err.Error()})
		return
	}

	log.Printf("INFO: Received FreeSWITCH voice control request: %+v\n", params)

	// --- Basic Logic based on params (to be expanded) ---
	// This is where you'd look up Application details using params.ApplicationSid,
	// check call status from params.CallStatus or by fetching call from DB using params.CallSid,
	// and apply business logic.

	response := twiml.NewResponse()

	// Example 1: Simple Say & Hangup
	// response.Add(&twiml.Say{Text: "Welcome to Agbara Go powered by FreeSWITCH."})
	// response.Add(&twiml.Hangup{})

	// Example 2: Echo Digits if Gathered
	if params.Digits != "" {
		response.Add(&twiml.Say{Text: fmt.Sprintf("You entered digits: %s. Goodbye.", params.Digits)})
		response.Add(&twiml.Hangup{})
	} else {
		// Example 3: Gather input
		gather := &twiml.Gather{
			Action:      "/api/v1/voice/control/gathered", // A different endpoint to process gathered digits
			Method:      "POST",
			NumDigits:   3,
			Timeout:     10,
			FinishOnKey: "#",
		}
		gather.AddNestedVerb(&twiml.Say{Text: "Please enter three digits, followed by the pound key."})
		response.Add(gather)
		// If no input, Gather's action URL won't be called unless actionOnEmptyResult="true"
		// So, add a fallback if Gather completes without input and no actionOnEmptyResult.
		response.Add(&twiml.Say{Text: "We did not receive your input. Goodbye."})
		response.Add(&twiml.Hangup{})
	}
	
	// Example 4: Based on Application (Illustrative - needs ApplicationService methods)
	/*
	if params.ApplicationSid != "" {
		app, err := api.applicationService.GetApplication(c.Request.Context(), params.ApplicationSid)
		if err == nil && app.FriendlyName == "EchoTestApp" {
			response = twiml.NewResponse() // Clear previous
			response.Add(&twiml.Say{Text: "This is the Echo Test Application."})
			// In a real scenario, FreeSWITCH's "echo" app is different. This would be a TwiML <Echo/> if we made one.
			// For now, just say something and hangup.
			response.Add(&twiml.Hangup{})
		}
	}
	*/


	renderedXML, err := response.Render()
	if err != nil {
		log.Printf("ERROR: Failed to render TwiML: %v\n", err)
		c.XML(http.StatusInternalServerError, gin.H{"error": "Failed to render TwiML"})
		return
	}

	log.Printf("INFO: Responding to FreeSWITCH with TwiML:\n%s\n", renderedXML)
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(renderedXML))
}

// HandleGatheredDigits (Example for Gather action URL)
// func (api *VoiceControlAPI) HandleGatheredDigits(c *gin.Context) {
//     // This handler would be registered to "/api/v1/voice/control/gathered"
//     var params FreeSwitchCallbackParams
//     if err := c.ShouldBind(&params); err != nil {
//         // ... error handling ...
//         return
//     }
//     log.Printf("INFO: Digits gathered: %s for CallSid: %s\n", params.Digits, params.CallSid)
//
//     response := twiml.NewResponse()
//     response.Add(&twiml.Say{Text: fmt.Sprintf("You successfully entered %s. Thank you!", params.Digits)})
//     response.Add(&twiml.Hangup{})
//     // ... render and send XML ...
// }
