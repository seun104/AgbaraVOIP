package services

import (
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"errors" // For pre-defined errors like ErrNotImplemented
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrConferenceActionNotImplemented = errors.New("this conference action is not implemented in the service layer yet")

// ConferenceService defines the interface for conference management operations.
type ConferenceService interface {
	CreateConference(ctx context.Context, accountSid string, friendlyName string) (*models.Conference, error)
	GetConference(ctx context.Context, conferenceSid string) (*models.Conference, error)
	ListConferences(ctx context.Context, accountSid string) ([]*models.Conference, error)
	UpdateConferenceStatus(ctx context.Context, conferenceSid string, status models.ConferenceStatus) (*models.Conference, error)

	AddParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error)
	GetParticipant(ctx context.Context, conferenceSid string, callSid string) (*models.Participant, error)
	ListParticipants(ctx context.Context, conferenceSid string) ([]*models.Participant, error)
	UpdateParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error) // For Mute, etc.
	RemoveParticipant(ctx context.Context, conferenceSid string, callSid string) error

	// Methods that would typically interact with a telephony server (e.g., FreeSWITCH)
	// These will be stubbed for now.
	MuteParticipant(ctx context.Context, conferenceSid string, callSid string, muteState bool) (*models.Participant, error)
	KickParticipant(ctx context.Context, conferenceSid string, callSid string) error
	StartRecording(ctx context.Context, conferenceSid string, params models.ConferenceRecordRequest) (*models.ConferenceActionResponse, error)
	StopRecording(ctx context.Context, conferenceSid string) (*models.ConferenceActionResponse, error)
	PlayAudio(ctx context.Context, conferenceSid string, params models.ConferencePlayRequest) (*models.ConferenceActionResponse, error)
	StopAudio(ctx context.Context, conferenceSid string) (*models.ConferenceActionResponse, error)
}

// PostgresConferenceService implements ConferenceService for PostgreSQL.
type PostgresConferenceService struct {
	db *sql.DB
}

// NewPostgresConferenceService creates a new PostgresConferenceService.
func NewPostgresConferenceService(db *sql.DB) *PostgresConferenceService {
	return &PostgresConferenceService{db: db}
}

// --- Conference CRUD ---

func scanConference(scanner interface{ Scan(...interface{}) error }) (*models.Conference, error) {
	conf := &models.Conference{}
	err := scanner.Scan(
		&conf.Sid, &conf.AccountSid, &conf.FriendlyName,
		&conf.Status, &conf.DateCreated, &conf.DateUpdated,
	)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (s *PostgresConferenceService) CreateConference(ctx context.Context, accountSid string, friendlyName string) (*models.Conference, error) {
	conf := &models.Conference{
		Sid:          "CO" + uuid.NewString(),
		AccountSid:   accountSid,
		FriendlyName: friendlyName,
		Status:       models.ConferenceStatusInit,
		DateCreated:  time.Now().UTC(),
		DateUpdated:  time.Now().UTC(),
	}

	query := `
		INSERT INTO conferences (sid, account_sid, friendly_name, status, date_created, date_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING sid, account_sid, friendly_name, status, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		conf.Sid, conf.AccountSid, conf.FriendlyName, conf.Status, conf.DateCreated, conf.DateUpdated,
	)
	return scanConference(row)
}

func (s *PostgresConferenceService) GetConference(ctx context.Context, conferenceSid string) (*models.Conference, error) {
	query := `SELECT sid, account_sid, friendly_name, status, date_created, date_updated FROM conferences WHERE sid = $1;`
	row := s.db.QueryRowContext(ctx, query, conferenceSid)
	conference, err := scanConference(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conference with SID %s not found: %w", conferenceSid, err)
		}
		return nil, fmt.Errorf("failed to get conference: %w", err)
	}
	return conference, nil
}

func (s *PostgresConferenceService) ListConferences(ctx context.Context, accountSid string) ([]*models.Conference, error) {
	query := `SELECT sid, account_sid, friendly_name, status, date_created, date_updated 
	          FROM conferences WHERE account_sid = $1 ORDER BY date_created DESC;`
	rows, err := s.db.QueryContext(ctx, query, accountSid)
	if err != nil {
		return nil, fmt.Errorf("failed to query conferences: %w", err)
	}
	defer rows.Close()

	var conferences []*models.Conference
	for rows.Next() {
		conference, err := scanConference(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conference row: %w", err)
		}
		conferences = append(conferences, conference)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conference rows: %w", err)
	}
    if conferences == nil {
        conferences = []*models.Conference{}
    }
	return conferences, nil
}

func (s *PostgresConferenceService) UpdateConferenceStatus(ctx context.Context, conferenceSid string, status models.ConferenceStatus) (*models.Conference, error) {
	query := `UPDATE conferences SET status = $1, date_updated = $2 WHERE sid = $3
	          RETURNING sid, account_sid, friendly_name, status, date_created, date_updated;`
	row := s.db.QueryRowContext(ctx, query, status, time.Now().UTC(), conferenceSid)
	conference, err := scanConference(row)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("conference %s not found for status update: %w", conferenceSid, err)
        }
        return nil, fmt.Errorf("failed to update conference status: %w", err)
    }
    return conference, nil
}


// --- Participant CRUD ---

func scanParticipant(scanner interface{ Scan(...interface{}) error }) (*models.Participant, error) {
	part := &models.Participant{}
	var friendlyName sql.NullString
	err := scanner.Scan(
		&part.CallSid, &part.ConferenceSid, &part.AccountSid, &friendlyName,
		&part.Muted, &part.StartConferenceOnEnter, &part.EndConferenceOnExit,
		&part.DateCreated, &part.DateUpdated,
	)
	if err != nil {
		return nil, err
	}
	part.FriendlyName = friendlyName.String
	return part, nil
}

func (s *PostgresConferenceService) AddParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error) {
	participant.DateCreated = time.Now().UTC()
	participant.DateUpdated = time.Now().UTC()
	// Assuming participant.AccountSid is already set correctly (e.g., copied from conference)

	query := `
		INSERT INTO participants (call_sid, conference_sid, account_sid, friendly_name, muted, 
		                          start_conference_on_enter, end_conference_on_exit, date_created, date_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING call_sid, conference_sid, account_sid, friendly_name, muted, 
		          start_conference_on_enter, end_conference_on_exit, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		participant.CallSid, participant.ConferenceSid, participant.AccountSid,
		sql.NullString{String: participant.FriendlyName, Valid: participant.FriendlyName != ""},
		participant.Muted, participant.StartConferenceOnEnter, participant.EndConferenceOnExit,
		participant.DateCreated, participant.DateUpdated,
	)
	return scanParticipant(row)
}

func (s *PostgresConferenceService) GetParticipant(ctx context.Context, conferenceSid string, callSid string) (*models.Participant, error) {
	query := `SELECT call_sid, conference_sid, account_sid, friendly_name, muted, 
	                 start_conference_on_enter, end_conference_on_exit, date_created, date_updated 
	          FROM participants WHERE conference_sid = $1 AND call_sid = $2;`
	row := s.db.QueryRowContext(ctx, query, conferenceSid, callSid)
	p, err := scanParticipant(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("participant with CallSID %s in ConferenceSID %s not found: %w", callSid, conferenceSid, err)
		}
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}
	return p, nil
}

func (s *PostgresConferenceService) ListParticipants(ctx context.Context, conferenceSid string) ([]*models.Participant, error) {
	query := `SELECT call_sid, conference_sid, account_sid, friendly_name, muted, 
	                 start_conference_on_enter, end_conference_on_exit, date_created, date_updated 
	          FROM participants WHERE conference_sid = $1 ORDER BY date_created ASC;`
	rows, err := s.db.QueryContext(ctx, query, conferenceSid)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants for conference %s: %w", conferenceSid, err)
	}
	defer rows.Close()

	var participants []*models.Participant
	for rows.Next() {
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant row: %w", err)
		}
		participants = append(participants, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating participant rows: %w", err)
	}
    if participants == nil {
        participants = []*models.Participant{}
    }
	return participants, nil
}

// UpdateParticipant is a general update, can be used for Mute, etc.
func (s *PostgresConferenceService) UpdateParticipant(ctx context.Context, participant *models.Participant) (*models.Participant, error) {
	participant.DateUpdated = time.Now().UTC()
	query := `
		UPDATE participants SET friendly_name = $1, muted = $2, start_conference_on_enter = $3, 
		                      end_conference_on_exit = $4, date_updated = $5
		WHERE conference_sid = $6 AND call_sid = $7
		RETURNING call_sid, conference_sid, account_sid, friendly_name, muted, 
		          start_conference_on_enter, end_conference_on_exit, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		sql.NullString{String: participant.FriendlyName, Valid: participant.FriendlyName != ""},
		participant.Muted, participant.StartConferenceOnEnter, participant.EndConferenceOnExit,
		participant.DateUpdated, participant.ConferenceSid, participant.CallSid,
	)
	updatedP, err := scanParticipant(row)
    if err != nil {
        if err == sql.ErrNoRows { // Should not happen if GetParticipant was called before, but good check
            return nil, fmt.Errorf("participant %s in conference %s not found for update: %w", participant.CallSid, participant.ConferenceSid, err)
        }
        return nil, fmt.Errorf("failed to update participant: %w", err)
    }
    return updatedP, nil
}

func (s *PostgresConferenceService) RemoveParticipant(ctx context.Context, conferenceSid string, callSid string) error {
	query := `DELETE FROM participants WHERE conference_sid = $1 AND call_sid = $2;`
	result, err := s.db.ExecContext(ctx, query, conferenceSid, callSid)
	if err != nil {
		return fmt.Errorf("failed to remove participant %s from conference %s: %w", callSid, conferenceSid, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for participant removal: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("participant %s in conference %s not found for removal (no rows affected)", callSid, conferenceSid)
	}
	return nil
}


// --- Stubbed Telephony-Interacting Methods ---

func (s *PostgresConferenceService) MuteParticipant(ctx context.Context, conferenceSid string, callSid string, muteState bool) (*models.Participant, error) {
    // 1. Update DB record
    participant, err := s.GetParticipant(ctx, conferenceSid, callSid)
    if err != nil {
        return nil, err
    }
    participant.Muted = muteState
    updatedP, err := s.UpdateParticipant(ctx, participant)
    if err != nil {
        return nil, err
    }
    // 2. Actual interaction with telephony server would happen here.
    // For now, we just log or return the DB update.
    fmt.Printf("INFO: [Stub] MuteParticipant called. Conf: %s, Call: %s, Mute: %v. Telephony interaction skipped.\n", conferenceSid, callSid, muteState)
	return updatedP, nil // Return the updated participant from DB
}

func (s *PostgresConferenceService) KickParticipant(ctx context.Context, conferenceSid string, callSid string) error {
    // 1. Remove from DB
    err := s.RemoveParticipant(ctx, conferenceSid, callSid)
    if err != nil {
        return err
    }
    // 2. Actual interaction with telephony server
    fmt.Printf("INFO: [Stub] KickParticipant called. Conf: %s, Call: %s. Telephony interaction skipped.\n", conferenceSid, callSid)
	return nil // Or ErrConferenceActionNotImplemented if strict
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
