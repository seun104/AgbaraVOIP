package domain

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath" // For generating file paths
	"strings"
	"time" // For unique filenames

	"github.com/user/agbaravoip_golang/internal/callcontrol" // For Pending structs
	// "github.com/fiorix/go-eventsocket/eventsocket" // No longer needed here, EslConnectionExecutor is in this package (interfaces file)
	"github.com/sirupsen/logrus"
)

// CallControlAction defines the type for call control actions.
type CallControlAction string

const (
	ActionContinue CallControlAction = "Continue"
	ActionRedirect CallControlAction = "Redirect"
	ActionHangup   CallControlAction = "Hangup"
	ActionError    CallControlAction = "Error"
	ActionGather   CallControlAction = "Gather"
	ActionRecord   CallControlAction = "Record"
	ActionSay      CallControlAction = "Say"    // Added for SayElement
	ActionPlay     CallControlAction = "Play"   // Added for PlayElement
	ActionPause    CallControlAction = "Pause"  // Added for PauseElement
	ActionDial     CallControlAction = "Dial"   // Added for DialElement
)

// CallControlResult defines the outcome of executing a call control element.
type CallControlResult struct {
	Action         CallControlAction // Next action to take (e.g., ActionRedirect, ActionHangup, ActionContinue)
	Err            error
	RedirectURL    string // Populated if Action is ActionRedirect
	RedirectMethod string // Populated if Action is ActionRedirect
	Digits         string // Populated by Gather with collected digits
	RecordingPath  string // Populated by Record with path to recording
}

// MinimalCallContext, EslConnectionExecutor, CallServicerForESL are now defined in call_control_interfaces.go
// This file will use those definitions.

type CallControlElement interface {
	GetType() CallControlAction
	GetActionURL() string
	GetMethod() string
	// Execute method signature now uses interfaces from call_control_interfaces.go
	Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult
}

// GatherElement Structure
type GatherElement struct {
	XMLName        xml.Name `xml:"Gather"`
	ActionURL      string   `xml:"action,attr,omitempty"`
	Method         string   `xml:"method,attr,omitempty"` // "GET" or "POST"
	TimeoutSeconds int      `xml:"timeout,attr,omitempty"`
	FinishOnKey    string   `xml:"finishOnKey,attr,omitempty"`
	NumDigits      int      `xml:"numDigits,attr,omitempty"`
	CallbackURL    string   `xml:"callbackUrl,attr,omitempty"`
	CallbackMethod string   `xml:"callbackMethod,attr,omitempty"`
	Play           *PlayElement `xml:",omitempty"`
	Say            *SayElement  `xml:",omitempty"`
	// Digits         string   // To store collected digits - this should be part of CallControlResult
}

func (g *GatherElement) GetType() CallControlAction {
	return ActionGather
}

func (g *GatherElement) GetActionURL() string {
	if g.ActionURL != "" {
		return g.ActionURL
	}
	return g.CallbackURL
}

func (g *GatherElement) GetMethod() string {
	if g.Method != "" {
		return g.Method
	}
	if g.CallbackMethod != "" {
		return g.CallbackMethod
	}
	return "POST" // Default method
}

func (g *GatherElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log()
	if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil in GatherElement.Execute")} }
	logger.Info("GatherElement Execute called (not yet implemented)")
	// Implementation for Gather will be added in a later step
	return CallControlResult{Action: ActionContinue, Err: errors.New("GatherElement.Execute not implemented")}
}

// RecordElement Structure
type RecordElement struct {
	XMLName          xml.Name `xml:"Record"`
	ActionURL        string   `xml:"action,attr,omitempty"`
	Method           string   `xml:"method,attr,omitempty"` // "GET" or "POST"
	MaxLengthSeconds int      `xml:"maxLength,attr,omitempty"`
	FinishOnKey      string   `xml:"finishOnKey,attr,omitempty"`
	PlayBeep         bool     `xml:"playBeep,attr,omitempty"`
	FileFormat       string   `xml:"format,attr,omitempty"` // e.g., "wav", "mp3"
	CallbackURL      string   `xml:"callbackUrl,attr,omitempty"`
	CallbackMethod   string   `xml:"callbackMethod,attr,omitempty"`
	TrimSilence      bool     `xml:"trimSilence,attr,omitempty"`
	EscapeDigits     string   `xml:"escapeDigits,attr,omitempty"`
}

func (r *RecordElement) GetType() CallControlAction {
	return ActionRecord
}

func (r *RecordElement) GetActionURL() string {
	if r.ActionURL != "" {
		return r.ActionURL
	}
	return r.CallbackURL
}

func (r *RecordElement) GetMethod() string {
	if r.Method != "" {
		return r.Method
	}
	if r.CallbackMethod != "" {
		return r.CallbackMethod
	}
	return "POST" // Default method
}

func (r *RecordElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log()
	if logger == nil {
		return CallControlResult{Action: ActionError, Err: errors.New("logger nil in RecordElement.Execute")}
	}

	// 1. Determine recording parameters
	fileName := fmt.Sprintf("%s_%d", ctx.GetUuid(), time.Now().UnixNano())
	fileFormat := "wav" // Default format
	if r.FileFormat != "" {
		// Basic validation for common formats, can be expanded
		if r.FileFormat == "wav" || r.FileFormat == "mp3" {
			fileFormat = r.FileFormat
		} else {
			logger.Warnf("Record: Unsupported file format '%s', defaulting to 'wav'.", r.FileFormat)
		}
	}
	fileNameWithFormat := fmt.Sprintf("%s.%s", fileName, fileFormat)

	baseRecPath := "/var/lib/freeswitch/recordings/" // Placeholder - THIS MUST BE CONFIGURABLE
	fullFilePath := filepath.Join(baseRecPath, ctx.GetAccountSid(), fileNameWithFormat) // Store in account-specific subfolder

	logger.Infof("Executing Record: FilePath='%s', MaxLength=%ds, FinishOnKey='%s', PlayBeep=%t, Format='%s'",
		fullFilePath, r.MaxLengthSeconds, r.FinishOnKey, r.PlayBeep, fileFormat)

	// 2. Play beep if requested
	if r.PlayBeep {
		logger.Info("Record: Playing beep.")
		_, err := eslConn.Execute("playback", "tone_stream://%(1000,0,640)") // Standard FS beep
		if err != nil {
			logger.Warnf("Record: Failed to play beep: %v", err)
		}
	}

	// 3. Start recording (non-blocking)
	maxLength := uint32(0)
	if r.MaxLengthSeconds > 0 {
		maxLength = uint32(r.MaxLengthSeconds)
	}

	silenceThreshold := uint(0)
	silenceHits := uint(0)

	logger.Infof("Record: Calling eslConn.RecordSession: Path='%s', MaxLength=%d", fullFilePath, maxLength)
	_, err := eslConn.RecordSession(fullFilePath, maxLength, silenceThreshold, silenceHits)

	if err != nil {
		logger.Errorf("Record: Failed to start recording session for file '%s': %v", fullFilePath, err)
		return CallControlResult{Action: ActionError, Err: fmt.Errorf("failed to start recording: %w", err)}
	}

	logger.Infof("Record: Recording started to %s. MaxLength: %d. FinishOnKey: '%s'. PlayBeep: %t. Format: %s",
		fullFilePath, r.MaxLengthSeconds, r.FinishOnKey, r.PlayBeep, fileFormat)

	// Use the new context methods for pending operations
	pendingRecInfo := callcontrol.PendingRecordInfo{
		OriginalElement:  r,
		ExpectedFilePath: fullFilePath,
	}
	ctx.SetPendingRecording(pendingRecInfo) // Pass the concrete type, context method takes interface{}
	logger.Info("Record: Pending recording info set in call context.")

	return CallControlResult{Action: ActionContinue}
}

// DialElement Structure
type DialElement struct {
	XMLName        xml.Name `xml:"Dial"`
	ActionURL      string   `xml:"action,attr,omitempty"`      // URL to POST to after dial attempt completion
	Method         string   `xml:"method,attr,omitempty"`      // Method for ActionURL (GET or POST)
	CallerID       string   `xml:"callerId,attr,omitempty"`    // Caller ID for the outbound leg
	CalleeIDToDial string   `xml:",chardata"`                  // The number, SIP URI, or user to dial
	TimeoutSeconds int      `xml:"timeout,attr,omitempty"`     // Ringing timeout for the new leg in seconds
	HangupOnStar   bool     `xml:"hangupOnStar,attr,omitempty"`// If true, receiving '*' on this leg hangs up the other leg(s)
}

func (d *DialElement) GetType() CallControlAction { return ActionDial }

func (d *DialElement) GetActionURL() string {
	return d.ActionURL
}

func (d *DialElement) GetMethod() string {
	if d.Method == "" {
		return "POST" // Default to POST
	}
	return d.Method
}

func (d *DialElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log()
	if logger == nil {
		return CallControlResult{Action: ActionError, Err: errors.New("logger nil in DialElement.Execute")}
	}

	if strings.TrimSpace(d.CalleeIDToDial) == "" {
		logger.Error("Dial: CalleeIDToDial is empty.")
		return CallControlResult{Action: ActionError, Err: errors.New("Dial CalleeIDToDial is empty")}
	}

	logger.Infof("Executing Dial: Callee='%s', CallerID='%s', Timeout=%ds, ActionURL='%s'",
		d.CalleeIDToDial, d.CallerID, d.TimeoutSeconds, d.ActionURL)

	var dialString string
	if strings.Contains(d.CalleeIDToDial, "@") {
		dialString = fmt.Sprintf("sofia/internal/%s", d.CalleeIDToDial)
	} else {
		dialString = fmt.Sprintf("user/%s", d.CalleeIDToDial)
	}

	originateVars := make(map[string]string)

	effectiveCallerID := ctx.GetVariable("caller_id_number")
	if d.CallerID != "" {
		effectiveCallerID = d.CallerID
	}
	if effectiveCallerID != "" {
		originateVars["origination_caller_id_number"] = effectiveCallerID
	}

	if d.TimeoutSeconds > 0 {
		originateVars["originate_timeout"] = fmt.Sprintf("%d", d.TimeoutSeconds)
	} else {
		originateVars["originate_timeout"] = "60"
	}

	originateVars["agbara_parent_call_sid"] = ctx.GetUuid()
	originateVars["agbara_dial_action_url"] = d.GetActionURL()
	originateVars["agbara_dial_action_method"] = d.GetMethod()
	originateVars["agbara_dial_hangup_on_star"] = fmt.Sprintf("%t", d.HangupOnStar)
	// originateVars["accountcode"] = ctx.GetAccountSid() // Example for billing

	logger.Infof("Dial: Calling eslConn.Originate: DialString='%s', VarsMapLength=%d", dialString, len(originateVars))

	newChannelUUID, err := eslConn.Originate(dialString, originateVars)

	if err != nil {
		logger.Errorf("Dial: Failed to send originate command for '%s': %v", dialString, err)
		return CallControlResult{Action: ActionError, Err: fmt.Errorf("failed to send originate command: %w", err)}
	}

	logger.Infof("Dial: Originate command sent for '%s'. New potential Channel UUID: %s. Waiting for async outcome.", dialString, newChannelUUID)

	// Use the new context methods for pending operations
	pendingDialInfo := callcontrol.PendingDialInfo{
		OriginalElement:     d,
		ParentAgbaraCallSID: ctx.GetUuid(),
	}
	ctx.AddPendingDial(newChannelUUID, pendingDialInfo) // Pass the concrete type
	logger.Info("Dial: Pending dial operation registered in call context for new UUID: %s", newChannelUUID)

	return CallControlResult{Action: ActionContinue}
}

type SayElement struct {
	XMLName  xml.Name `xml:"Say"` // Added XMLName
	Text     string `xml:",chardata"`
	Voice    string `xml:"voice,attr,omitempty"`
	Language string `xml:"language,attr,omitempty"`
	Loop     int    `xml:"loop,attr,omitempty"`
	Engine   string `xml:"engine,attr,omitempty"`
}

func (s *SayElement) GetType() CallControlAction { return ActionSay }
func (s *SayElement) GetActionURL() string       { return "" }
func (s *SayElement) GetMethod() string          { return "POST" } // Say doesn't have a method, default POST

type PlayElement struct {
	XMLName xml.Name `xml:"Play"` // Added XMLName
	URL     string   `xml:",chardata"`
	Loop    int      `xml:"loop,attr,omitempty"`
}

func (p *PlayElement) GetType() CallControlAction { return ActionPlay }
func (p *PlayElement) GetActionURL() string       { return "" }
func (p *PlayElement) GetMethod() string          { return "POST" } // Play doesn't have a method, default POST

type HangupElement struct {
	XMLName  xml.Name `xml:"Hangup"` // Added XMLName
	Reason   string   `xml:"reason,attr,omitempty"`
	Schedule *int     `xml:"schedule,attr,omitempty"`
}

func (h *HangupElement) GetType() CallControlAction { return ActionHangup }
func (h *HangupElement) GetActionURL() string       { return "" }
func (h *HangupElement) GetMethod() string          { return "POST" } // Hangup doesn't have a method, default POST

type PauseElement struct {
	XMLName xml.Name `xml:"Pause"` // Added XMLName
	Length  int      `xml:"length,attr"`
}

func (p *PauseElement) GetType() CallControlAction { return ActionPause }
func (p *PauseElement) GetActionURL() string       { return "" }
func (p *PauseElement) GetMethod() string          { return "POST" } // Pause doesn't have a method, default POST

type RedirectElement struct {
	XMLName xml.Name `xml:"Redirect"` // Added XMLName
	URL     string   `xml:",chardata"`
	Method  string   `xml:"method,attr,omitempty"`
}

func (r *RedirectElement) GetType() CallControlAction { return ActionRedirect }
func (r *RedirectElement) GetActionURL() string       { return r.URL }
func (r *RedirectElement) GetMethod() string {
	if r.Method == "" {
		return "POST" // Default to POST
	}
	return r.Method
}

type ResponseElement struct {
	XMLName  xml.Name             `xml:"Response"`
	Verbs    []CallControlElement `xml:",any"` // This field holds all the parsed verb elements
}

// Updated Execute methods to use EslConnectionExecutor from domain interfaces
// and new MinimalCallContext methods (ctx.Log(), ctx.GetUuid())
// and new callSvc type CallServicerForESL.

func (s *SayElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil in Say")} }
	if strings.TrimSpace(s.Text) == "" { logger.Debugf("Say empty text for %s", ctx.GetUuid()); return CallControlResult{Action: ActionContinue} }

	ttsEngine := "flite"; if s.Engine != "" { ttsEngine = s.Engine }
	ttsVoice := "slt"; if strings.ToUpper(s.Voice) == "MAN" { ttsVoice = "kal" } else if s.Voice != "" { ttsVoice = s.Voice }
	// Freeswitch speak app format: speak <tts_engine>|<tts_voice>|<text_to_speak>
	speakArgString := fmt.Sprintf("%s|%s|%s", ttsEngine, ttsVoice, s.Text)

	loops := 1; if s.Loop > 0 { loops = s.Loop }; if s.Loop == 0 { loops = 1000 } // 0 means loop indefinitely practically
	logger.Infof("EXEC Say for CallSID %s, Text: '%s', Loops %d", ctx.GetUuid(), s.Text, loops)

	for i := 0; i < loops; i++ {
		// EslConnectionExecutor interface: Execute(command string, args ...string) (event string, err error)
		_, err := eslConn.Execute("speak", speakArgString)
		if err != nil {
			logger.Errorf("ESL Speak failed for %s: %v", ctx.GetUuid(), err)
			// Return ActionError to allow XML interpreter to decide if it should hangup or continue
			return CallControlResult{Action: ActionError, Err: fmt.Errorf("speak failed: %w", err)}
		}
		// Success is implied by err == nil with the new interface.
	}
	return CallControlResult{Action: ActionContinue}
}

func (p *PlayElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil in Play")} }
	if strings.TrimSpace(p.URL) == "" { logger.Debugf("Play empty URL for %s", ctx.GetUuid()); return CallControlResult{Action: ActionContinue} }

	loops := 1; if p.Loop > 0 { loops = p.Loop }; if p.Loop == 0 { loops = 1000 }
	logger.Infof("EXEC Play for CallSID %s, URL %s, Loops %d", ctx.GetUuid(), p.URL, loops)

	playbackArg := p.URL

	for i := 0; i < loops; i++ {
		// EslConnectionExecutor interface: Execute(command string, args ...string) (event string, err error)
		_, err := eslConn.Execute("playback", playbackArg)
		if err != nil {
			logger.Errorf("ESL Playback failed for %s: %v", ctx.GetUuid(), err)
			return CallControlResult{Action: ActionError, Err: fmt.Errorf("playback failed: %w", err)}
		}
	}
	return CallControlResult{Action: ActionContinue}
}

func (h *HangupElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil in Hangup")} }
	hangupReason := "NORMAL_CLEARING"; if h.Reason != "" { h.Reason = h.Reason } // Corrected assignment
	logger.Infof("EXEC Hangup for CallSID %s, Reason %s", ctx.GetUuid(), hangupReason)

	var err error
	if h.Schedule != nil && *h.Schedule > 0 {
		// EslConnectionExecutor interface: Execute(command string, args ...string) (event string, err error)
		// sched_hangup <+seconds> <reason>
		scheduleArg := fmt.Sprintf("+%d", *h.Schedule)
		_, err = eslConn.Execute("sched_hangup", scheduleArg, hangupReason)
		if err != nil {
			logger.Errorf("ESL SchedHangup failed for %s: %v", ctx.GetUuid(), err)
			// Even if sched_hangup fails, the intention is hangup.
			// The interpreter will likely treat ActionHangup + error as fatal.
			return CallControlResult{Action: ActionHangup, Err: fmt.Errorf("sched_hangup failed: %w", err)}
		}
		logger.Infof("Call for %s scheduled to hangup in %d seconds.", ctx.GetUuid(), *h.Schedule)
	} else {
		// EslConnectionExecutor interface has Hangup(reason string) (string, error)
		_, err = eslConn.Hangup(hangupReason)
		if err != nil {
			logger.Errorf("ESL Hangup failed for %s: %v", ctx.GetUuid(), err)
			return CallControlResult{Action: ActionHangup, Err: fmt.Errorf("hangup failed: %w", err)}
		}
	}
	// Notify CallServicer if needed, e.g. callSvc.UpdateCallStatus(ctx, domain.CallStatusHangingUp, hangupReason)
	// For now, the XML interpreter handles state based on ActionHangup.
	return CallControlResult{Action: ActionHangup}
}

func (p *PauseElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil in Pause")} }
	if p.Length <= 0 {
		logger.Infof("Pause invalid or zero length %d for %s, skipping pause.", p.Length, ctx.GetUuid())
		return CallControlResult{Action: ActionContinue}
	}
	logger.Infof("EXEC Pause for CallSID %s, Length %d sec", ctx.GetUuid(), p.Length)

	// Freeswitch uses 'sleep' app for milliseconds.
	sleepDurationMs := p.Length * 1000
	// EslConnectionExecutor interface: Execute(command string, args ...string) (event string, err error)
	_, err := eslConn.Execute("sleep", fmt.Sprintf("%d", sleepDurationMs))
	if err != nil {
		logger.Errorf("ESL Pause (sleep) failed for %s: %v", ctx.GetUuid(), err)
		return CallControlResult{Action: ActionError, Err: fmt.Errorf("pause (sleep) failed: %w", err)}
	}
	return CallControlResult{Action: ActionContinue}
}

func (r *RedirectElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc CallServicerForESL) CallControlResult {
	logger := ctx.Log(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil in Redirect")} }
	logger.Infof("EXEC Redirect for CallSID %s, URL %s, Method %s", ctx.GetUuid(), r.URL, r.GetMethod())
	if strings.TrimSpace(r.URL) == "" {
		logger.Error("Redirect URL is empty")
		// Return ActionError so interpreter can decide to hangup.
		return CallControlResult{Action: ActionError, Err: errors.New("redirect URL empty")}
	}
	// Redirect is handled by the interpreter loop, not an ESL command itself.
	return CallControlResult{Action: ActionRedirect, RedirectURL: r.URL, RedirectMethod: r.GetMethod()}
}
