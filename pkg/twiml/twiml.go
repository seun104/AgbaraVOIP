package twiml

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// --- Base and Container Elements ---

// Verb is an interface that all TwiML verbs should implement (optional, for type safety).
type Verb interface {
	GetVerbName() string // Returns the XML element name of the verb
}

// Response is the root element for TwiML documents.
type Response struct {
	XMLName xml.Name `xml:"Response"`
	Verbs   []interface{} `xml:",any"` // Holds any TwiML verb
}

// NewResponse creates a new TwiML Response.
func NewResponse() *Response {
	return &Response{}
}

// Add appends a TwiML verb to the Response.
func (r *Response) Add(verb Verb) {
	r.Verbs = append(r.Verbs, verb)
}

// Render generates the XML string for the Response.
func (r *Response) Render() (string, error) {
	byteXML, err := xml.MarshalIndent(r, "", "  ") // Use MarshalIndent for pretty printing
	if err != nil {
		return "", fmt.Errorf("failed to marshal TwiML response: %w", err)
	}
	return xml.Header + string(byteXML), nil // Prepend XML header
}

// --- TwiML Verbs ---

// Say verb for text-to-speech.
type Say struct {
	XMLName  xml.Name `xml:"Say"`
	Text     string   `xml:",chardata"` // Text to say
	Voice    string   `xml:"voice,attr,omitempty"`    // e.g., "alice", "man", "woman"
	Language string   `xml:"language,attr,omitempty"` // e.g., "en-US", "es-MX"
	Loop     int      `xml:"loop,attr,omitempty"`     // Number of times to repeat
}
func (s *Say) GetVerbName() string { return "Say" }

// Play verb for playing an audio file.
type Play struct {
	XMLName xml.Name `xml:"Play"`
	URL     string   `xml:",chardata"`           // URL of the audio file
	Loop    int      `xml:"loop,attr,omitempty"` // Number of times to loop
	Digits  string   `xml:"digits,attr,omitempty"` // DTMF tones to play (w=0.5s pause, W=1s pause)
}
func (p *Play) GetVerbName() string { return "Play" }

// Record verb for recording audio.
type Record struct {
	XMLName           xml.Name `xml:"Record"`
	Action            string   `xml:"action,attr,omitempty"`            // URL to submit recording to
	Method            string   `xml:"method,attr,omitempty"`            // "GET" or "POST"
	MaxLength         int      `xml:"maxLength,attr,omitempty"`         // Max recording duration in seconds
	Timeout           int      `xml:"timeout,attr,omitempty"`           // Seconds of silence before considering recording complete
	FinishOnKey       string   `xml:"finishOnKey,attr,omitempty"`       // DTMF key to end recording (e.g., "#", "*")
	PlayBeep          bool     `xml:"playBeep,attr,omitempty"`          // Play a beep before recording
	Trim              string   `xml:"trim,attr,omitempty"`              // "trim-silence" or "do-not-trim"
	RecordingStatusCallback string `xml:"recordingStatusCallback,attr,omitempty"`
	RecordingStatusCallbackMethod string `xml:"recordingStatusCallbackMethod,attr,omitempty"`
	// Transcription options could be added here too
}
func (r *Record) GetVerbName() string { return "Record" }

// Hangup verb to end the call.
type Hangup struct {
	XMLName xml.Name `xml:"Hangup"`
}
func (h *Hangup) GetVerbName() string { return "Hangup" }

// Redirect verb to redirect call control to another URL.
type Redirect struct {
	XMLName xml.Name `xml:"Redirect"`
	URL     string   `xml:",chardata"`           // URL to redirect to
	Method  string   `xml:"method,attr,omitempty"` // "GET" or "POST"
}
func (r *Redirect) GetVerbName() string { return "Redirect" }


// --- Dial Verb and its Children ---

// Dial verb for calling another party.
type Dial struct {
	XMLName           xml.Name      `xml:"Dial"`
	Targets           []interface{} `xml:",any"` // Can contain Number, Sip, Client, Conference
	Action            string        `xml:"action,attr,omitempty"`    // URL to go to after dial attempt
	Method            string        `xml:"method,attr,omitempty"`    // "GET" or "POST" for action
	Timeout           int           `xml:"timeout,attr,omitempty"`   // Dial timeout
	TimeLimit         int           `xml:"timeLimit,attr,omitempty"` // Max duration of the dialed call
	CallerId          string        `xml:"callerId,attr,omitempty"`
	HangupOnStar      bool          `xml:"hangupOnStar,attr,omitempty"` // Allow original caller to hang up bridged call with *
	Record            string        `xml:"record,attr,omitempty"`       // "true", "false", "record-from-answer", etc.
	RecordingStatusCallback string    `xml:"recordingStatusCallback,attr,omitempty"`
}
func (d *Dial) GetVerbName() string { return "Dial" }
func (d *Dial) AddTarget(target Verb) { d.Targets = append(d.Targets, target) }


// Number noun for dialing a PSTN number.
type Number struct {
	XMLName      xml.Name `xml:"Number"`
	PhoneNumber  string   `xml:",chardata"`
	SendDigits   string   `xml:"sendDigits,attr,omitempty"` // DTMF to send to called party after connect
	Url          string   `xml:"url,attr,omitempty"`        // TwiML URL to execute on called party leg before bridge
	Method       string   `xml:"method,attr,omitempty"`
	StatusCallbackEvent string `xml:"statusCallbackEvent,attr,omitempty"` // initiated, ringing, answered, completed
	StatusCallback      string `xml:"statusCallback,attr,omitempty"`
	StatusCallbackMethod string `xml:"statusCallbackMethod,attr,omitempty"`
}
func (n *Number) GetVerbName() string { return "Number" }

// Sip noun for dialing a SIP URI.
type Sip struct {
	XMLName      xml.Name `xml:"Sip"`
	SipURI       string   `xml:",chardata"` // Full SIP URI, e.g., sip:user@example.com
	Username     string   `xml:"username,attr,omitempty"`
	Password     string   `xml:"password,attr,omitempty"` // Be cautious with exposing passwords
	Headers      string   `xml:"headers,attr,omitempty"`  // Custom SIP headers (e.g., X-Foo=bar&X-Baz=boo)
	Url          string   `xml:"url,attr,omitempty"`      // TwiML URL for called party leg
	Method       string   `xml:"method,attr,omitempty"`
	StatusCallbackEvent string `xml:"statusCallbackEvent,attr,omitempty"`
	StatusCallback      string `xml:"statusCallback,attr,omitempty"`
	StatusCallbackMethod string `xml:"statusCallbackMethod,attr,omitempty"`
}
func (s *Sip) GetVerbName() string { return "Sip" }

// Conference noun for dialing into a conference.
type Conference struct {
	XMLName                xml.Name `xml:"Conference"`
	Name                   string   `xml:",chardata"` // Conference Name/Room
	Muted                  bool     `xml:"muted,attr,omitempty"`
	Beep                   string   `xml:"beep,attr,omitempty"` // "true", "false", "onEnter", "onExit"
	StartConferenceOnEnter bool     `xml:"startConferenceOnEnter,attr,omitempty"`
	EndConferenceOnExit    bool     `xml:"endConferenceOnExit,attr,omitempty"`
	WaitUrl                string   `xml:"waitUrl,attr,omitempty"`    // URL for hold music/message
	WaitMethod             string   `xml:"waitMethod,attr,omitempty"` // "GET" or "POST"
	MaxParticipants        int      `xml:"maxParticipants,attr,omitempty"`
	StatusCallbackEvent    string   `xml:"statusCallbackEvent,attr,omitempty"` // start, end, join, leave, mute, hold
	StatusCallback         string   `xml:"statusCallback,attr,omitempty"`
	StatusCallbackMethod   string   `xml:"statusCallbackMethod,attr,omitempty"`
	// JitterBufferSize, ParticipantLabel etc. can be added
}
func (c *Conference) GetVerbName() string { return "Conference" }


// --- Gather Verb and its Children ---

// Gather verb for collecting DTMF digits.
type Gather struct {
	XMLName      xml.Name      `xml:"Gather"`
	NestedVerbs  []interface{} `xml:",any"` // Can contain Say, Play, Pause
	Action       string        `xml:"action,attr,omitempty"` // URL to submit digits to
	Method       string        `xml:"method,attr,omitempty"` // "GET" or "POST"
	Timeout      int           `xml:"timeout,attr,omitempty"`  // Seconds to wait for digits
	NumDigits    int           `xml:"numDigits,attr,omitempty"`// Max number of digits to collect
	FinishOnKey  string        `xml:"finishOnKey,attr,omitempty"` // e.g., "#"
	ActionOnEmptyResult bool   `xml:"actionOnEmptyResult,attr,omitempty"` // If true, call action URL even if no digits
	// Input hints (speech related, not for DTMF only)
	// SpeechTimeout, Language etc.
}
func (g *Gather) GetVerbName() string { return "Gather" }
func (g *Gather) AddNestedVerb(verb Verb) { g.NestedVerbs = append(g.NestedVerbs, verb) }

// Pause verb (can be nested in Gather or used directly in Response)
type Pause struct {
    XMLName  xml.Name `xml:"Pause"`
    Length   int      `xml:"length,attr,omitempty"` // Duration in seconds
}
func (p *Pause) GetVerbName() string { return "Pause" }


// Helper function to render any TwiML construct to string
func ToXML(v Verb) (string, error) {
    byteXML, err := xml.MarshalIndent(v, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to marshal TwiML verb %s: %w", v.GetVerbName(), err)
    }
    return string(byteXML), nil
}

// Helper to render just the TwiML content (no XML header, for embedding)
func RenderVerbs(verbs ...Verb) (string, error) {
    var builder strings.Builder
    for _, verb := range verbs {
        byteXML, err := xml.Marshal(verb) // No indent for fragments
        if err != nil {
            return "", fmt.Errorf("failed to marshal TwiML verb %s: %w", verb.GetVerbName(), err)
        }
        builder.Write(byteXML)
        // builder.WriteString("\n") // Optional: newlines between verbs if desired
    }
    return builder.String(), nil
}
