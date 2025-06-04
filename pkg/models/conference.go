package models

import (
	"time"
)

// ConferenceStatus represents the status of a conference.
type ConferenceStatus string

const (
	ConferenceStatusInit        ConferenceStatus = "initializing" // Or "init" from C#
	ConferenceStatusInProgress  ConferenceStatus = "in-progress"
	ConferenceStatusCompleted   ConferenceStatus = "completed"
	ConferenceStatusFailed      ConferenceStatus = "failed"
)

// Conference model
type Conference struct {
	Sid          string           `json:"sid"`
	AccountSid   string           `json:"accountSid"`
	FriendlyName string           `json:"friendlyName,omitempty"`
	DateCreated  time.Time        `json:"dateCreated"`
	DateUpdated  time.Time        `json:"dateUpdated"`
	Status       ConferenceStatus `json:"status"`
}

// Participant model for a conference
type Participant struct {
	CallSid                string    `json:"callSid"` // Call SID of the participant
	AccountSid             string    `json:"accountSid"`
	ConferenceSid          string    `json:"conferenceSid"`
	FriendlyName           string    `json:"friendlyName,omitempty"` // Optional participant-specific name
	DateCreated            time.Time `json:"dateCreated"`
	DateUpdated            time.Time `json:"dateUpdated"`
	Muted                  bool      `json:"muted"`
	StartConferenceOnEnter bool      `json:"startConferenceOnEnter"`
	EndConferenceOnExit    bool      `json:"endConferenceOnExit"`
}

// --- Request/Response DTOs ---

// CreateConferenceRequest model for creating a new conference.
type CreateConferenceRequest struct {
	FriendlyName string `json:"friendlyName" binding:"required"`
	// AccountSid is typically derived from authenticated user context.
}

// MuteParticipantRequest model for muting/unmuting a participant.
type MuteParticipantRequest struct {
	Muted bool `json:"muted"` // Target state: true to mute, false to unmute
}

// ConferenceRecordRequest model for starting a conference recording.
// Fields are examples; actual fields depend on what `apiserver` expects.
type ConferenceRecordRequest struct {
	FileName string `json:"fileName,omitempty"` // Optional name for the recording file
	Format   string `json:"format,omitempty"`   // Optional format (e.g., "wav", "mp3")
	// Other options like Transcription, etc. could be added.
}

// ConferencePlayRequest model for playing audio into a conference.
// Fields are examples.
type ConferencePlayRequest struct {
	Url  string `json:"url" binding:"required"` // URL of the audio file to play
	Loop int    `json:"loop,omitempty"`         // Number of times to loop; 0 or 1 means play once
}

// ConferenceActionResponse is a generic response for conference actions.
type ConferenceActionResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	ConferenceSid      string `json:"conferenceSid,omitempty"`
	ParticipantCallSid string `json:"participantCallSid,omitempty"` // For participant-specific actions
	Operation          string `json:"operation,omitempty"`          // e.g., "mute", "kick", "record_start"
}
