package services

import (
	"agbara-go/pkg/freeswitch_events" // <-- NEW IMPORT
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"errors" 
	"fmt"
	"log"
	"strconv" 
	"strings" 
	"time"

	"github.com/google/uuid"
)

// ConferenceService interface (ensure it's complete from Turn 39)
type ConferenceService interface {
	CreateConference(ctx context.Context, accountSid string, friendlyName string) (*models.Conference, error)
	GetConference(ctx context.Context, conferenceSid string) (*models.Conference, error)
	ListConferences(ctx context.Context, accountSid string) ([]*models.Conference, error)
	UpdateConferenceStatus(ctx context.Context, conferenceSid string, status models.ConferenceStatus) (*models.Conference, error)
	AddParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error)
	GetParticipant(ctx context.Context, conferenceSid string, callSid string) (*models.Participant, error)
	ListParticipants(ctx context.Context, conferenceSid string) ([]*models.Participant, error)
	UpdateParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error) 
	RemoveParticipant(ctx context.Context, conferenceSid string, callSid string) error
	MuteParticipant(ctx context.Context, conferenceSid string, callSid string, muteState bool) (*models.Participant, error)
	KickParticipant(ctx context.Context, conferenceSid string, callSid string) error
	StartRecording(ctx context.Context, conferenceSid string, params models.ConferenceRecordRequest) (*models.ConferenceActionResponse, error)
	StopRecording(ctx context.Context, conferenceSid string) (*models.ConferenceActionResponse, error)
	PlayAudio(ctx context.Context, conferenceSid string, params models.ConferencePlayRequest) (*models.ConferenceActionResponse, error)
	StopAudio(ctx context.Context, conferenceSid string) (*models.ConferenceActionResponse, error)
}


// Update PostgresConferenceService struct
type PostgresConferenceService struct {
	db             *sql.DB
	eventDispatcher *freeswitch_events.EventDispatcher // <-- ADDED
}

// Update NewPostgresConferenceService constructor
func NewPostgresConferenceService(db *sql.DB, evtDisp *freeswitch_events.EventDispatcher) *PostgresConferenceService {
	s := &PostgresConferenceService{
		db:             db,
		eventDispatcher: evtDisp,
	}
	if evtDisp != nil {
		log.Println("ConferenceService: Subscribing to FreeSWITCH events: CUSTOM conference::maintenance, CHANNEL_HANGUP_COMPLETE")
		evtDisp.Subscribe("CUSTOM", s.handleCustomConferenceEvent) // Listen to all CUSTOM, filter by subclass in handler
        evtDisp.Subscribe("CHANNEL_HANGUP_COMPLETE", s.handleParticipantHangup) 
	}
	return s
}

// --- Event Handler Methods for ConferenceService ---

func (s *PostgresConferenceService) handleCustomConferenceEvent(event *freeswitch_events.ParsedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

    if event.Subclass != "conference::maintenance" {
        return // Not the event subclass we are interested in for conference direct maintenance
    }

	conferenceName := event.Headers["Conference-Name"] // This is usually the Agbara Conference SID
	memberIDStr := event.Headers["Member-ID"]      
	action := event.Headers["Action"]
    callerIDNum := event.Headers["Caller-Caller-ID-Number"] 
    participantChannelUUID := event.Headers["Unique-ID"] 

	if conferenceName == "" {
		log.Printf("ConferenceService: CUSTOM Event: Conference-Name is empty. Cannot process. Headers: %+v\n", event.Headers)
		return
	}
    
	conf, err := s.GetConference(ctx, conferenceName) 
	if err != nil {
		log.Printf("ConferenceService: CUSTOM Event: Conference %s not found: %v\n", conferenceName, err)
		return
	}

	log.Printf("ConferenceService: Handling CUSTOM conference::maintenance event: ConfSID: %s, MemberID: %s, Action: %s, ParticipantChannelUUID: %s\n",
		conf.Sid, memberIDStr, action, participantChannelUUID)

    var participantCallSid string
    // Prefer variable_agbara_call_sid if it exists (should be set on the channel when it joins conference).
    if val, ok := event.Headers["variable_agbara_call_sid"]; ok && val != "" {
        participantCallSid = val
    } else if participantChannelUUID != "" && (action == "add-member" || action == "del-member" || action == "mute-member" || action == "unmute-member" || action == "start-talking" || action == "stop-talking") {
        participantCallSid = participantChannelUUID
    } else if memberIDStr != "" {
        if _, errAtoi := strconv.Atoi(memberIDStr); errAtoi != nil {
            participantCallSid = memberIDStr
        }
    }

    if participantCallSid == "" && action != "conference-destroy" { 
        log.Printf("ConferenceService: CUSTOM Event: Could not determine participant CallSid for Conf %s, MemberID %s, Action %s. Headers: %+v\n", conf.Sid, memberIDStr, action, event.Headers)
        return
    }

	var participant *models.Participant
    if participantCallSid != "" { 
        participant, err = s.GetParticipant(ctx, conf.Sid, participantCallSid)
        if err != nil && action != "add-member" { 
            if errors.Is(err, sql.ErrNoRows) {
                log.Printf("ConferenceService: CUSTOM Event: Participant %s in conf %s not found for action '%s'.\n", participantCallSid, conf.Sid, action)
            } else {
                log.Printf("ConferenceService: CUSTOM Event: Error fetching participant %s in conf %s: %v\n", participantCallSid, conf.Sid, err)
            }
            return 
        }
    }

	switch action {
	case "add-member":
		if participant == nil && participantCallSid != "" { // Ensure participantCallSid is valid before adding
            newP := &models.Participant{
                CallSid:       participantCallSid, 
                ConferenceSid: conf.Sid,
                AccountSid:    conf.AccountSid, 
                Muted:         event.Headers["Speak"] == "false" || event.Headers["Listen"] == "false", 
                FriendlyName:  callerIDNum, 
            }
            _, addErr := s.AddParticipant(ctx, newP)
            if addErr != nil {
                log.Printf("ERROR: ConferenceService: Failed to add participant %s for conf %s: %v\n", participantCallSid, conf.Sid, addErr)
            } else {
                 log.Printf("INFO: ConferenceService: Participant %s added to conf %s via 'add-member' event.\n", participantCallSid, conf.Sid)
            }
        } else if participant != nil {
            log.Printf("INFO: ConferenceService: 'add-member' event for existing participant %s in conf %s.\n", participantCallSid, conf.Sid)
        } else {
             log.Printf("WARN: ConferenceService: 'add-member' event received but could not determine participantCallSid. Headers: %+v\n", event.Headers)
        }
        if conf.Status == models.ConferenceStatusInit {
            _, _ = s.UpdateConferenceStatus(ctx, conf.Sid, models.ConferenceStatusInProgress)
        }
	case "del-member":
        if participantCallSid != "" { 
            delErr := s.RemoveParticipant(ctx, conf.Sid, participantCallSid)
            if delErr != nil && !errors.Is(delErr, sql.ErrNoRows) && !strings.Contains(strings.ToLower(delErr.Error()), "not found"){ // Don't log error if already removed
                log.Printf("ConferenceService: CUSTOM Event: Failed to remove participant %s from conf %s: %v\n", participantCallSid, conf.Sid, delErr)
            } else {
                log.Printf("INFO: ConferenceService: Participant %s removed from conf %s due to 'del-member' event.\n", participantCallSid, conf.Sid)
            }
        }
	case "mute-member", "stop-speaking":
		if participant != nil {
			participant.Muted = true
			_, err = s.UpdateParticipant(ctx, participant)
            if err != nil { log.Printf("ERROR: Failed to update participant %s to muted: %v\n", participant.CallSid, err) }
		}
	case "unmute-member", "start-speaking":
		if participant != nil {
			participant.Muted = false
			_, err = s.UpdateParticipant(ctx, participant)
            if err != nil { log.Printf("ERROR: Failed to update participant %s to unmuted: %v\n", participant.CallSid, err) }
		}
    case "conference-destroy": 
        log.Printf("INFO: ConferenceService: 'conference-destroy' event for conf %s. Setting status to completed.\n", conf.Sid)
        _, _ = s.UpdateConferenceStatus(ctx, conf.Sid, models.ConferenceStatusCompleted)
	default:
		log.Printf("ConferenceService: Unhandled conference action '%s' for conf %s, member %s\n", action, conf.Sid, memberIDStr)
	}
}

func (s *PostgresConferenceService) handleParticipantHangup(event *freeswitch_events.ParsedEvent) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

    participantCallSid := event.UniqueID 
    conferenceName := event.Headers["variable_conference_name"] 
    if conferenceName == "" {
        conferenceName = event.Headers["variable_conference_uuid"] 
    }

    if conferenceName != "" && participantCallSid != "" {
        log.Printf("ConferenceService: Participant %s hung up from conference %s (name from event). Removing.\n", participantCallSid, conferenceName)
        err := s.RemoveParticipant(ctx, conferenceName, participantCallSid)
        if err != nil && !errors.Is(err, sql.ErrNoRows) && !strings.Contains(strings.ToLower(err.Error()), "not found") { 
            log.Printf("ConferenceService: Error removing participant %s from conf %s on hangup: %v\n", participantCallSid, conferenceName, err)
        } else if err == nil {
            log.Printf("ConferenceService: Participant %s successfully removed from conference %s after hangup.\n", participantCallSid, conferenceName)
            remainingParticipants, listErr := s.ListParticipants(ctx, conferenceName)
            if listErr == nil && len(remainingParticipants) == 0 {
                log.Printf("ConferenceService: Conference %s is now empty after participant %s hung up. Setting status to completed.\n", conferenceName, participantCallSid)
                _, _ = s.UpdateConferenceStatus(ctx, conferenceName, models.ConferenceStatusCompleted)
            }
        }
    }
}


// --- Existing methods (scanConference, scanParticipant, CreateConference, etc. from Turn 39/40) ---
func scanConference(scanner interface{ Scan(...interface{}) error }) (*models.Conference, error) { 
	conf := &models.Conference{}
	err := scanner.Scan( &conf.Sid, &conf.AccountSid, &conf.FriendlyName, &conf.Status, &conf.DateCreated, &conf.DateUpdated, )
	if err != nil { return nil, err }
	return conf, nil
}
func scanParticipant(scanner interface{ Scan(...interface{}) error }) (*models.Participant, error) { 
	part := &models.Participant{}
	var friendlyName sql.NullString
	err := scanner.Scan( &part.CallSid, &part.ConferenceSid, &part.AccountSid, &friendlyName, &part.Muted, &part.StartConferenceOnEnter, &part.EndConferenceOnExit, &part.DateCreated, &part.DateUpdated, )
	if err != nil { return nil, err }
	part.FriendlyName = friendlyName.String
	return part, nil
}
func (s *PostgresConferenceService) CreateConference(ctx context.Context, accountSid string, friendlyName string) (*models.Conference, error) { 
	conf := &models.Conference{ Sid: "CO" + uuid.NewString(), AccountSid: accountSid, FriendlyName: friendlyName, Status: models.ConferenceStatusInit, DateCreated:  time.Now().UTC(), DateUpdated:  time.Now().UTC(), }
	query := `INSERT INTO conferences (sid, account_sid, friendly_name, status, date_created, date_updated) VALUES ($1, $2, $3, $4, $5, $6) RETURNING sid, account_sid, friendly_name, status, date_created, date_updated;`
	row := s.db.QueryRowContext(ctx, query, conf.Sid, conf.AccountSid, conf.FriendlyName, conf.Status, conf.DateCreated, conf.DateUpdated, )
	return scanConference(row)
}
func (s *PostgresConferenceService) GetConference(ctx context.Context, conferenceSid string) (*models.Conference, error) { 
	query := `SELECT sid, account_sid, friendly_name, status, date_created, date_updated FROM conferences WHERE sid = $1;`
	row := s.db.QueryRowContext(ctx, query, conferenceSid)
	conference, err := scanConference(row)
	if err != nil { if errors.Is(err, sql.ErrNoRows) { return nil, fmt.Errorf("conference with SID %s not found: %w", conferenceSid, err) } ; return nil, fmt.Errorf("failed to get conference: %w", err) }
	return conference, nil
}
func (s *PostgresConferenceService) ListConferences(ctx context.Context, accountSid string) ([]*models.Conference, error) { 
	query := `SELECT sid, account_sid, friendly_name, status, date_created, date_updated FROM conferences WHERE account_sid = $1 ORDER BY date_created DESC;`
	rows, err := s.db.QueryContext(ctx, query, accountSid)
	if err != nil { return nil, fmt.Errorf("failed to query conferences: %w", err) }
	defer rows.Close()
	var conferences []*models.Conference
	for rows.Next() { conference, err := scanConference(rows); if err != nil { return nil, fmt.Errorf("failed to scan conference row: %w", err) }; conferences = append(conferences, conference) }
	if err = rows.Err(); err != nil { return nil, fmt.Errorf("error iterating conference rows: %w", err) }
    if conferences == nil { conferences = []*models.Conference{} }
	return conferences, nil
}
func (s *PostgresConferenceService) UpdateConferenceStatus(ctx context.Context, conferenceSid string, status models.ConferenceStatus) (*models.Conference, error) { 
	query := `UPDATE conferences SET status = $1, date_updated = $2 WHERE sid = $3 RETURNING sid, account_sid, friendly_name, status, date_created, date_updated;`
	row := s.db.QueryRowContext(ctx, query, status, time.Now().UTC(), conferenceSid)
	conference, err := scanConference(row)
    if err != nil { if errors.Is(err, sql.ErrNoRows) { return nil, fmt.Errorf("conference %s not found for status update: %w", conferenceSid, err) }; return nil, fmt.Errorf("failed to update conference status: %w", err) }
    return conference, nil
}
func (s *PostgresConferenceService) AddParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error) { 
	participant.DateCreated = time.Now().UTC(); participant.DateUpdated = time.Now().UTC()
	query := `INSERT INTO participants (call_sid, conference_sid, account_sid, friendly_name, muted, start_conference_on_enter, end_conference_on_exit, date_created, date_updated) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING call_sid, conference_sid, account_sid, friendly_name, muted, start_conference_on_enter, end_conference_on_exit, date_created, date_updated;`
	row := s.db.QueryRowContext(ctx, query, participant.CallSid, participant.ConferenceSid, participant.AccountSid, sql.NullString{String: participant.FriendlyName, Valid: participant.FriendlyName != ""}, participant.Muted, participant.StartConferenceOnEnter, participant.EndConferenceOnExit, participant.DateCreated, participant.DateUpdated, )
	return scanParticipant(row)
}
func (s *PostgresConferenceService) GetParticipant(ctx context.Context, conferenceSid string, callSid string) (*models.Participant, error) { 
	query := `SELECT call_sid, conference_sid, account_sid, friendly_name, muted, start_conference_on_enter, end_conference_on_exit, date_created, date_updated FROM participants WHERE conference_sid = $1 AND call_sid = $2;`
	row := s.db.QueryRowContext(ctx, query, conferenceSid, callSid)
	p, err := scanParticipant(row)
	if err != nil { if errors.Is(err, sql.ErrNoRows) { return nil, fmt.Errorf("participant with CallSID %s in ConferenceSID %s not found: %w", callSid, conferenceSid, err) }; return nil, fmt.Errorf("failed to get participant: %w", err) }
	return p, nil
}
func (s *PostgresConferenceService) ListParticipants(ctx context.Context, conferenceSid string) ([]*models.Participant, error) { 
	query := `SELECT call_sid, conference_sid, account_sid, friendly_name, muted, start_conference_on_enter, end_conference_on_exit, date_created, date_updated FROM participants WHERE conference_sid = $1 ORDER BY date_created ASC;`
	rows, err := s.db.QueryContext(ctx, query, conferenceSid)
	if err != nil { return nil, fmt.Errorf("failed to query participants for conference %s: %w", conferenceSid, err) }
	defer rows.Close()
	var participants []*models.Participant
	for rows.Next() { p, err := scanParticipant(rows); if err != nil { return nil, fmt.Errorf("failed to scan participant row: %w", err) }; participants = append(participants, p) }
	if err = rows.Err(); err != nil { return nil, fmt.Errorf("error iterating participant rows: %w", err) }
    if participants == nil { participants = []*models.Participant{} }
	return participants, nil
}
func (s *PostgresConferenceService) UpdateParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error) { 
	participant.DateUpdated = time.Now().UTC()
	query := `UPDATE participants SET friendly_name = $1, muted = $2, start_conference_on_enter = $3, end_conference_on_exit = $4, date_updated = $5 WHERE conference_sid = $6 AND call_sid = $7 RETURNING call_sid, conference_sid, account_sid, friendly_name, muted, start_conference_on_enter, end_conference_on_exit, date_created, date_updated;`
	row := s.db.QueryRowContext(ctx, query, sql.NullString{String: participant.FriendlyName, Valid: participant.FriendlyName != ""}, participant.Muted, participant.StartConferenceOnEnter, participant.EndConferenceOnExit, participant.DateUpdated, participant.ConferenceSid, participant.CallSid, )
	updatedP, err := scanParticipant(row)
    if err != nil { if errors.Is(err, sql.ErrNoRows) { return nil, fmt.Errorf("participant %s in conference %s not found for update: %w", participant.CallSid, participant.ConferenceSid, err) }; return nil, fmt.Errorf("failed to update participant: %w", err) }
    return updatedP, nil
}
func (s *PostgresConferenceService) RemoveParticipant(ctx context.Context, conferenceSid string, callSid string) error { 
	query := `DELETE FROM participants WHERE conference_sid = $1 AND call_sid = $2;`
	result, err := s.db.ExecContext(ctx, query, conferenceSid, callSid)
	if err != nil { return fmt.Errorf("failed to remove participant %s from conference %s: %w", callSid, conferenceSid, err) }
	rowsAffected, err := result.RowsAffected(); if err != nil { return fmt.Errorf("failed to get rows affected for participant removal: %w", err) }
	if rowsAffected == 0 { return fmt.Errorf("participant %s in conference %s not found for removal (no rows affected)", callSid, conferenceSid) } 
	return nil
}
func (s *PostgresConferenceService) MuteParticipant(ctx context.Context, conferenceSid string, callSid string, muteState bool) (*models.Participant, error) { 
    participant, err := s.GetParticipant(ctx, conferenceSid, callSid); if err != nil { return nil, err }
    participant.Muted = muteState
    updatedP, err := s.UpdateParticipant(ctx, participant); if err != nil { return nil, err }
    fmt.Printf("INFO: [Stub] MuteParticipant called. Conf: %s, Call: %s, Mute: %v. Telephony interaction skipped.\n", conferenceSid, callSid, muteState)
	return updatedP, nil 
}
func (s *PostgresConferenceService) KickParticipant(ctx context.Context, conferenceSid string, callSid string) error { 
    err := s.RemoveParticipant(ctx, conferenceSid, callSid); if err != nil { return err }
    fmt.Printf("INFO: [Stub] KickParticipant called. Conf: %s, Call: %s. Telephony interaction skipped.\n", conferenceSid, callSid)
	return nil 
}
func (s *PostgresConferenceService) StartRecording(ctx context.Context, conferenceSid string, params models.ConferenceRecordRequest) (*models.ConferenceActionResponse, error) { 
	fmt.Printf("INFO: [Stub] StartRecording called for Conf: %s with params: %+v. Telephony interaction skipped.\n", conferenceSid, params)
	return &models.ConferenceActionResponse{Success: true, Message: "Recording started (stubbed)", ConferenceSid: conferenceSid, Operation: "record_start"}, nil
}
func (s *PostgresConferenceService) StopRecording(ctx context.Context, conferenceSid string) (*models.ConferenceActionResponse, error) { 
	fmt.Printf("INFO: [Stub] StopRecording called for Conf: %s. Telephony interaction skipped.\n", conferenceSid)
	return &models.ConferenceActionResponse{Success: true, Message: "Recording stopped (stubbed)", ConferenceSid: conferenceSid, Operation: "record_stop"}, nil
}
func (s *PostgresConferenceService) PlayAudio(ctx context.Context, conferenceSid string, params models.ConferencePlayRequest) (*models.ConferenceActionResponse, error) { 
	fmt.Printf("INFO: [Stub] PlayAudio called for Conf: %s with params: %+v. Telephony interaction skipped.\n", conferenceSid, params)
	return &models.ConferenceActionResponse{Success: true, Message: "Audio play started (stubbed)", ConferenceSid: conferenceSid, Operation: "play_start"}, nil
}
func (s *PostgresConferenceService) StopAudio(ctx context.Context, conferenceSid string) (*models.ConferenceActionResponse, error) { 
	fmt.Printf("INFO: [Stub] StopAudio called for Conf: %s. Telephony interaction skipped.\n", conferenceSid)
	return &models.ConferenceActionResponse{Success: true, Message: "Audio play stopped (stubbed)", ConferenceSid: conferenceSid, Operation: "play_stop"}, nil
}
