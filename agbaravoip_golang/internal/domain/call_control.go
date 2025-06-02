package domain

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings" 
	// "time" // No direct usage of time in this file anymore

	"github.com/fiorix/go-eventsocket/eventsocket" // Still needed for the interface definition if not fully abstracted
	"github.com/sirupsen/logrus" 
)

type CallControlAction int
const (
	ActionContinue CallControlAction = iota 
	ActionRedirect                          
	ActionHangup                            
	ActionError                             
)
type CallControlResult struct {
	Action   CallControlAction
	NextURL  string 
	Err      error  
}

type MinimalCallContext interface {
	GetFreeswitchUUID() string
	GetAgbaraCallSID() string
	GetLoggerEntry() *logrus.Entry 
}

// EslConnectionExecutor defines the ESL methods needed by Execute functions.
// Both *eventsocket.Connection and test mocks will implement this.
type EslConnectionExecutor interface {
	Execute(app string, arg string, lock bool) (*eventsocket.Event, error)
	Send(cmd string, args ...string) (*eventsocket.Event, error)
	// Add other methods like ExecuteUUID, API, BgAPI if they become needed by verbs
}

type CallControlElement interface {
	// Changed eslConn to EslConnectionExecutor interface
	Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc interface{}) CallControlResult
}

type SayElement struct {
	Text     string `xml:",chardata"` 
	Voice    string `xml:"voice,attr,omitempty"` 
	Language string `xml:"language,attr,omitempty"` 
	Loop     int    `xml:"loop,attr,omitempty"`
	Engine   string `xml:"engine,attr,omitempty"` 
}
type PlayElement struct {
	URL  string `xml:",chardata"` 
	Loop int    `xml:"loop,attr,omitempty"`
}
type HangupElement struct {
	Reason   string `xml:"reason,attr,omitempty"` 
	Schedule *int   `xml:"schedule,attr,omitempty"` 
}
type PauseElement struct {
	Length int `xml:"length,attr"` 
}
type RedirectElement struct {
	URL    string `xml:",chardata"` 
	Method string `xml:"method,attr,omitempty"` 
}
type ResponseElement struct {
	XMLName  xml.Name      `xml:"Response"` 
	Verbs    []interface{} `xml:",any"` 
	Elements []CallControlElement `xml:"-"` 
}

// Updated Execute methods to use EslConnectionExecutor
func (s *SayElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc interface{}) CallControlResult {
	logger := ctx.GetLoggerEntry(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil")} }
	if strings.TrimSpace(s.Text) == "" { logger.Debugf("Say empty text for %s", ctx.GetAgbaraCallSID()); return CallControlResult{Action: ActionContinue} }
	ttsEngine := "flite"; if s.Engine != "" { ttsEngine = s.Engine }
	ttsVoice := "slt"; if strings.ToUpper(s.Voice) == "MAN" { ttsVoice = "kal" } else if s.Voice != "" { ttsVoice = s.Voice }
	eslText := strings.ReplaceAll(s.Text, "'", "\\''")
	speakCmdArg := fmt.Sprintf("%s|%s|%s", ttsEngine, ttsVoice, eslText)
	loops := 1; if s.Loop > 0 { loops = s.Loop }; if s.Loop == 0 { loops = 1000 }
	logger.Infof("EXEC Say for CallSID %s, Loops %d", ctx.GetAgbaraCallSID(), loops)
	for i := 0; i < loops; i++ {
		ev, err := eslConn.Execute("speak", speakCmdArg, true) 
		if err != nil { logger.Errorf("ESL Speak failed for %s: %v", ctx.GetAgbaraCallSID(), err); return CallControlResult{Action: ActionError, Err: fmt.Errorf("speak failed: %w", err)} }
		if !strings.Contains(ev.Get("Reply-Text"), "+OK") { logger.Errorf("ESL Speak non-OK for %s: %s", ctx.GetAgbaraCallSID(), ev.Get("Reply-Text")); return CallControlResult{Action: ActionError, Err: fmt.Errorf("speak reply not OK: %s", ev.Get("Reply-Text"))} }
	}
	return CallControlResult{Action: ActionContinue}
}

func (p *PlayElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc interface{}) CallControlResult {
	logger := ctx.GetLoggerEntry(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil")} }
	if strings.TrimSpace(p.URL) == "" { logger.Debugf("Play empty URL for %s", ctx.GetAgbaraCallSID()); return CallControlResult{Action: ActionContinue} }
	loops := 1; if p.Loop > 0 { loops = p.Loop }; if p.Loop == 0 { loops = 1000 }
	logger.Infof("EXEC Play for CallSID %s, URL %s, Loops %d", ctx.GetAgbaraCallSID(), p.URL, loops)
	for i := 0; i < loops; i++ {
		ev, err := eslConn.Execute("playback", p.URL, true)
		if err != nil { logger.Errorf("ESL Playback failed for %s: %v", ctx.GetAgbaraCallSID(), err); return CallControlResult{Action: ActionError, Err: fmt.Errorf("playback failed: %w", err)} }
		if !strings.Contains(ev.Get("Reply-Text"), "+OK") { logger.Errorf("ESL Playback non-OK for %s: %s", ctx.GetAgbaraCallSID(), ev.Get("Reply-Text")); return CallControlResult{Action: ActionError, Err: fmt.Errorf("playback reply not OK: %s", ev.Get("Reply-Text"))} }
	}
	return CallControlResult{Action: ActionContinue}
}

func (h *HangupElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc interface{}) CallControlResult {
	logger := ctx.GetLoggerEntry(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil")} }
	hangupReason := "NORMAL_CLEARING"; if h.Reason != "" { hangupReason = h.Reason }
	logger.Infof("EXEC Hangup for CallSID %s, Reason %s", ctx.GetAgbaraCallSID(), hangupReason)
	if h.Schedule != nil && *h.Schedule > 0 {
		_, _ = eslConn.Execute("sched_hangup", fmt.Sprintf("+%d %s", *h.Schedule, hangupReason), true)
		return CallControlResult{Action: ActionHangup} 
	}
	_, _ = eslConn.Execute("hangup", hangupReason, true) 
	return CallControlResult{Action: ActionHangup}
}

func (p *PauseElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc interface{}) CallControlResult {
	logger := ctx.GetLoggerEntry(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil")} }
	if p.Length <= 0 { logger.Infof("Pause invalid length %d for %s", p.Length, ctx.GetAgbaraCallSID()); return CallControlResult{Action: ActionContinue} }
	logger.Infof("EXEC Pause for CallSID %s, Length %d sec", ctx.GetAgbaraCallSID(), p.Length)
	playbackArg := fmt.Sprintf("silence_stream://%d", p.Length*1000) 
	ev, err := eslConn.Execute("playback", playbackArg, true)
	if err != nil { logger.Errorf("ESL Pause failed for %s: %v", ctx.GetAgbaraCallSID(), err); return CallControlResult{Action: ActionError, Err: fmt.Errorf("pause failed: %w", err)} }
	if !strings.Contains(ev.Get("Reply-Text"), "+OK") { logger.Errorf("ESL Pause non-OK for %s: %s", ctx.GetAgbaraCallSID(), ev.Get("Reply-Text")); return CallControlResult{Action: ActionError, Err: fmt.Errorf("pause reply not OK: %s", ev.Get("Reply-Text"))} }
	return CallControlResult{Action: ActionContinue}
}

func (r *RedirectElement) Execute(ctx MinimalCallContext, eslConn EslConnectionExecutor, callSvc interface{}) CallControlResult {
	logger := ctx.GetLoggerEntry(); if logger == nil { return CallControlResult{Action: ActionError, Err: errors.New("logger nil")} }
	logger.Infof("EXEC Redirect for CallSID %s, URL %s", ctx.GetAgbaraCallSID(), r.URL) 
	if strings.TrimSpace(r.URL) == "" { return CallControlResult{Action: ActionError, Err: errors.New("redirect URL empty")} }
	return CallControlResult{Action: ActionRedirect, NextURL: r.URL}
}
