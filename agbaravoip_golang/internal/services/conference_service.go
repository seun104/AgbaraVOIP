package services

import (
	"context"
	"errors" // Required for errors.Is
	"fmt"
	"strings"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/utils" // For GenerateSID
	"gorm.io/gorm"
	"github.com/sirupsen/logrus"
	"github.com/user/agbaravoip_golang/internal/esl" // Added ESL import
)

// ErrConferenceInvalidState indicates a conference is not in a valid state for the operation.
var ErrConferenceInvalidState = errors.New("conference is not in a state that allows this operation")


type conferenceService struct {
	db        *gorm.DB
	log       *logrus.Entry
	eslClient *esl.FSInboundClient // Added ESL client
}

// NewConferenceService creates a new IConferenceService.
func NewConferenceService(db *gorm.DB, logger *logrus.Logger, eslClient *esl.FSInboundClient) IConferenceService {
	return &conferenceService{
		db:        db,
		log:       logger.WithField("service", "conference"),
		eslClient: eslClient,
	}
}

// getVerifiedConference is an internal helper to fetch a conference and verify ownership.
func (s *conferenceService) getVerifiedConference(ctx context.Context, accountSid string, confSid string) (*domain.Conference, error) {
	var conf domain.Conference
	if err := s.db.WithContext(ctx).Where("sid = ? AND account_sid = ?", confSid, accountSid).First(&conf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Warnf("getVerifiedConference: Conference %s not found for account %s", confSid, accountSid)
			return nil, fmt.Errorf("%w: conference %s not found for account %s", domain.ErrNotFound, confSid, accountSid)
		}
		s.log.Errorf("getVerifiedConference: Error querying conference %s for account %s: %v", confSid, accountSid, err)
		return nil, fmt.Errorf("querying conference %s for account %s: %w", confSid, accountSid, err)
	}
	return &conf, nil
}


// GetConferenceBySID retrieves a conference by its SID, ensuring it belongs to the account.
func (s *conferenceService) GetConferenceBySID(ctx context.Context, accountSid string, confSid string) (*domain.Conference, error) {
	s.log.Infof("GetConferenceBySID: Attempting to retrieve conference %s for account %s", confSid, accountSid)
	return s.getVerifiedConference(ctx, accountSid, confSid)
}

// ListConferences lists conferences for a specific account.
func (s *conferenceService) ListConferences(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.Conference, error) {
	s.log.Infof("ListConferences: Listing conferences for account SID: %s with filters: %v", accountSid, filters)

	var conferences []*domain.Conference
	query := s.db.WithContext(ctx).Model(&domain.Conference{}).Where("account_sid = ?", accountSid)

	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if friendlyName, ok := filters["friendly_name"].(string); ok && friendlyName != "" {
		query = query.Where("friendly_name LIKE ?", "%"+friendlyName+"%")
	}
	// Add date range filters if needed

	if err := query.Order("created_at DESC").Find(&conferences).Error; err != nil {
		s.log.Errorf("ListConferences: Error querying conferences for account %s: %v", accountSid, err)
		return nil, fmt.Errorf("querying conferences for account %s: %w", accountSid, err)
	}
	return conferences, nil
}


func (s *conferenceService) GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error) {
	var conf domain.Conference
	// This method is primarily for internal use (e.g. AgbaraXML <Conference>)
	// It might not need strict accountSid scoping if friendly_name is unique enough per account context
	// or if the caller (like XML processor) already has established account context.
	// For now, keeping accountSid as it was.
	err := s.db.WithContext(ctx).Where("account_sid = ? AND friendly_name = ? AND status != ?", accountSid, name, domain.ConferenceStatusCompleted).
		Order("created_at DESC").First(&conf).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying conference by name %s for account %s: %w", name, accountSid, err)
	}
	return &conf, nil
}

// CreateConference is used internally, GetOrCreateConference is preferred for most uses.
func (s *conferenceService) CreateConference(ctx context.Context, accountSid, name, sid string) (*domain.Conference, error) {
	now := time.Now().UTC()
	conf := &domain.Conference{
		SID:          sid,
		AccountSID:   accountSid,
		FriendlyName: name,
		Status:       domain.ConferenceStatusInit,
		CreatedAt:    now,
		UpdatedAt:    now,
		// StartTime will be nil as GORM handles default NULL for pointers
	}

	if err := s.db.WithContext(ctx).Create(&conf).Error; err != nil {
		// Check for unique constraint violation on SID
		if strings.Contains(err.Error(), "unique constraint") &&
		   (strings.Contains(err.Error(), "conferences_sid_key") || strings.Contains(err.Error(), "idx_conferences_sid")) { // PostgreSQL specific index name
			 return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("creating conference %s: %w", sid, err)
	}
	// GORM populates ID, CreatedAt, UpdatedAt by default if tags are correct.
	// StartTime remains nil as expected.
	s.log.Infof("Created conference %s (%s)", name, sid)
	return conf, nil
}


func (s *conferenceService) GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error) {
	conf, err := s.GetConferenceByName(ctx, accountSid, name)
	if err == nil && conf != nil && conf.Status != domain.ConferenceStatusCompleted {
		s.log.Infof("Found existing active conference %s (%s)", name, conf.SID)
		if conf.Status == domain.ConferenceStatusInit {
			now := time.Now().UTC()
			updateData := map[string]interface{}{
				"status":     domain.ConferenceStatusInProgress,
				"start_time": now,
				"updated_at": now,
			}
			// Use .Model with a pointer to the struct for updates to work correctly with hooks/timestamps
			if tx := s.db.WithContext(ctx).Model(&domain.Conference{}).Where("sid = ? AND status = ?", conf.SID, domain.ConferenceStatusInit).Updates(updateData); tx.Error != nil {
				s.log.Warnf("Failed to update status to in-progress for conf %s: %v", conf.SID, tx.Error)
				// Continue with existing conf anyway
			} else if tx.RowsAffected > 0 {
				conf.Status = domain.ConferenceStatusInProgress
				conf.StartTime = &now // Update in-memory struct as well
				conf.UpdatedAt = now
				s.log.Infof("Updated conference %s status to in-progress and set start time.", conf.SID)
			}
		}
		return conf, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("error fetching conference by name '%s': %w", name, err)
	}

	newSID := utils.GenerateSID("CF")
	s.log.Infof("Creating new conference %s for account %s with SID %s", name, accountSid, newSID)
	return s.CreateConference(ctx, accountSid, name, newSID)
}

func (s *conferenceService) UpdateConferenceStatus(ctx context.Context, accountSid string, confSid string, status domain.ConferenceStatus) error {
	s.log.Infof("UpdateConferenceStatus: Attempting for conf %s, account %s to status %s", confSid, accountSid, status)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return err
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&domain.Conference{}).Where("sid = ?", confSid).Updates(map[string]interface{}{"status": status, "updated_at": now})
	if res.Error != nil { return fmt.Errorf("updating status for conference %s: %w", confSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound } // Should be caught by getVerifiedConference, but good practice
	s.log.Infof("Updated status for conference %s to %s", confSid, status)
	return nil
}

func (s *conferenceService) EndConference(ctx context.Context, accountSid string, confSid string, endTime time.Time) error {
	s.log.Infof("EndConference: Attempting for conf %s, account %s", confSid, accountSid)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return err
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&domain.Conference{}).Where("sid = ?", confSid).Updates(map[string]interface{}{
		"status":     domain.ConferenceStatusCompleted,
		"end_time":   endTime.UTC(),
		"updated_at": now,
	})
	if res.Error != nil { return fmt.Errorf("ending conference %s: %w", confSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Ended conference %s at %v", confSid, endTime)
	return nil
}

// --- Participant Methods ---
func (s *conferenceService) AddParticipant(ctx context.Context, confSid, callSid, pSid, accountSid string, isMuted, isModerator bool) (*domain.ConferenceParticipant, error) {
	now := time.Now().UTC()
	part := &domain.ConferenceParticipant{
		SID:           pSid,
		ConferenceSID: confSid,
		CallSID:       callSid,
		AccountSID:    accountSid,
		IsMuted:       isMuted,
		IsModerator:   isModerator,
		JoinTime:      now, // GORM will set CreatedAt/UpdatedAt if struct has them and they are zero
	}

	if err := s.db.WithContext(ctx).Create(&part).Error; err != nil {
		if strings.Contains(err.Error(), "unique constraint") &&
		   (strings.Contains(err.Error(), "conference_participants_call_sid_key") || strings.Contains(err.Error(), "idx_conference_participants_call_sid")) { // GORM might use different index name string
			s.log.Warnf("Call %s already a participant (or unique constraint violation on SID %s)", callSid, pSid)
			existingPart, getErr := s.GetParticipantByCallSID(ctx, callSid)
			// Ensure existingPart is not nil before checking LeaveTime
			if getErr == nil && existingPart != nil && existingPart.LeaveTime == nil { return existingPart, domain.ErrConflict }
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("executing insert for participant CallSID %s in conf %s: %w", callSid, confSid, err)
	}
	s.log.Infof("Added participant CallSID %s (PSID %s) to conference %s", callSid, pSid, confSid)
	return part, nil
}

func (s *conferenceService) GetParticipant(ctx context.Context, accountSid string, confSid string, participantSid string) (*domain.ConferenceParticipant, error) {
	s.log.Infof("GetParticipant: Attempting for pSID %s in conf %s, account %s", participantSid, confSid, accountSid)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return nil, err
	}
	var part domain.ConferenceParticipant
	err := s.db.WithContext(ctx).Where("sid = ? AND conference_sid = ?", participantSid, confSid).First(&part).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, domain.ErrNotFound }
		return nil, fmt.Errorf("querying participant by SID %s in conf %s: %w", participantSid, confSid, err)
	}
	return &part, nil
}

func (s *conferenceService) GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error) {
	// This is often used internally by ESL events, so might not need explicit accountSid scoping here
	// if the callSid is globally unique and implies the account.
	// However, if called from an API context that has accountSid, it could be added for consistency.
	// For now, keeping it as is, assuming callSid is sufficient for unique lookup in its primary use case.
	var part domain.ConferenceParticipant
	err := s.db.WithContext(ctx).Where("call_sid = ? AND leave_time IS NULL", callSid).First(&part).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, domain.ErrNotFound }
		return nil, fmt.Errorf("querying participant by CallSID %s: %w", callSid, err)
	}
	return &part, nil
}

func (s *conferenceService) UpdateParticipantMuteStatus(ctx context.Context, accountSid string, confSid string, pSid string, isMuted bool) error {
	s.log.Infof("UpdateParticipantMuteStatus: Attempting for pSID %s in conf %s, account %s to mute=%t", pSid, confSid, accountSid, isMuted)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return err
	}
	// Also verify participant pSid belongs to confSid before updating
	part, err := s.GetParticipant(ctx, accountSid, confSid, pSid)
	if err != nil {
		return err // Not found or other error
	}

	res := s.db.WithContext(ctx).Model(&part).Where("sid = ? AND conference_sid = ? AND leave_time IS NULL", pSid, confSid).Update("is_muted", isMuted)
	if res.Error != nil { return fmt.Errorf("updating mute status for participant %s: %w", pSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound } // Should be caught by GetParticipant, but defense in depth
	s.log.Infof("Updated mute status for participant %s to %t", pSid, isMuted)
	return nil
}

func (s *conferenceService) UpdateParticipantModeratorStatus(ctx context.Context, accountSid string, confSid string, pSid string, isModerator bool) error {
	s.log.Infof("UpdateParticipantModeratorStatus: Attempting for pSID %s in conf %s, account %s to moderator=%t", pSid, confSid, accountSid, isModerator)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return err
	}
	part, err := s.GetParticipant(ctx, accountSid, confSid, pSid)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&part).Where("sid = ? AND conference_sid = ? AND leave_time IS NULL", pSid, confSid).Update("is_moderator", isModerator)
	if res.Error != nil { return fmt.Errorf("updating moderator status for participant %s: %w", pSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Updated moderator status for participant %s to %t", pSid, isModerator)
	return nil
}

func (s *conferenceService) RemoveParticipant(ctx context.Context, accountSid string, confSid string, pSid string, leaveTime time.Time) error {
	s.log.Infof("RemoveParticipant: Attempting for pSID %s in conf %s, account %s", pSid, confSid, accountSid)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return err
	}
	part, err := s.GetParticipant(ctx, accountSid, confSid, pSid)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&part).Where("sid = ? AND conference_sid = ? AND leave_time IS NULL", pSid, confSid).Update("leave_time", leaveTime.UTC())
	if res.Error != nil { return fmt.Errorf("setting leave_time for participant %s: %w", pSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Participant %s left conference %s at %v", pSid, confSid, leaveTime)
	return nil
}

func (s *conferenceService) ListParticipants(ctx context.Context, accountSid string, confSid string) ([]*domain.ConferenceParticipant, error) {
	s.log.Infof("ListParticipants: Attempting for conf %s, account %s", confSid, accountSid)
	if _, err := s.getVerifiedConference(ctx, accountSid, confSid); err != nil {
		return nil, err
	}
	var participants []*domain.ConferenceParticipant
	err := s.db.WithContext(ctx).Where("conference_sid = ? AND leave_time IS NULL", confSid).Find(&participants).Error
	if err != nil { return nil, fmt.Errorf("listing participants for conference %s: %w", confSid, err) }
	return participants, nil
}


// --- Live Conference Control Methods ---

func (s *conferenceService) PlayAudioInConference(ctx context.Context, accountSid string, confSid string, playURL string, loop int) (string, error) {
	s.log.Infof("PlayAudioInConference: Attempting for conf %s, account %s, URL: %s", confSid, accountSid, playURL)
	verifiedConf, err := s.getVerifiedConference(ctx, accountSid, confSid)
	if err != nil {
		return "", err
	}
	if verifiedConf.Status != domain.ConferenceStatusInProgress {
		return "", fmt.Errorf("%w: conference %s is not in-progress (status: %s)", ErrConferenceInvalidState, confSid, verifiedConf.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS // Re-use from call_service or define locally
	}
	// conference <confname> play <file> [async_job_uuid]
	// Note: loop might not be directly supported by this simple command.
	if loop > 1 {
		s.log.Warnf("Looping > 1 for PlayAudioInConference on conf %s may not be supported. Playing once.", confSid)
	}
	command := fmt.Sprintf("bgapi conference %s play %s", verifiedConf.FriendlyName, playURL)
	s.log.Debugf("Sending ESL command for PlayAudioInConference: %s", command)
	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.log.Errorf("ESL PlayAudioInConference command failed for conf %s: %v", confSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err) // Re-use
	}
	return jobID, nil
}

func (s *conferenceService) SayTextInConference(ctx context.Context, accountSid string, confSid string, text string, language *string, voice *string) (string, error) {
	s.log.Infof("SayTextInConference: Attempting for conf %s, account %s", confSid, accountSid)
	verifiedConf, err := s.getVerifiedConference(ctx, accountSid, confSid)
	if err != nil {
		return "", err
	}
	if verifiedConf.Status != domain.ConferenceStatusInProgress {
		return "", fmt.Errorf("%w: conference %s is not in-progress (status: %s)", ErrConferenceInvalidState, confSid, verifiedConf.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}

	ttsEngine := "flite"; ttsVoice := "slt" // Defaults
	if language != nil { /* logic to map language to engine/voice */ }
	if voice != nil { ttsVoice = *voice }

	// conference <confname> say <engine> <voice> <text>
	command := fmt.Sprintf("bgapi conference %s say %s %s '%s'", verifiedConf.FriendlyName, ttsEngine, ttsVoice, text)
	s.log.Debugf("Sending ESL command for SayTextInConference: %s", command)
	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.log.Errorf("ESL SayTextInConference command failed for conf %s: %v", confSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	return jobID, nil
}

func (s *conferenceService) StartRecordingConference(ctx context.Context, accountSid string, confSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (string, string, error) {
	s.log.Infof("StartRecordingConference: Attempting for conf %s, account %s", confSid, accountSid)
	verifiedConf, err := s.getVerifiedConference(ctx, accountSid, confSid)
	if err != nil {
		return "", "", err
	}
	if verifiedConf.Status != domain.ConferenceStatusInProgress {
		return "", "", fmt.Errorf("%w: conference %s is not in-progress (status: %s)", ErrConferenceInvalidState, confSid, verifiedConf.Status)
	}
	if s.eslClient == nil {
		return "", "", ErrESLClientNotAvailable_CS
	}

	actualFormat := "wav"; if format != nil && (*format == "wav" || *format == "mp3") { actualFormat = *format }
	var recordingFileName string
	if fileName != nil && *fileName != "" {
		recordingFileName = fmt.Sprintf("%s.%s", *fileName, actualFormat)
	} else {
		recordingFileName = fmt.Sprintf("%s_%d.%s", confSid, time.Now().UnixNano(), actualFormat)
	}
	fullRecordingPath := fmt.Sprintf("/var/lib/freeswitch/recordings/%s/%s", accountSid, recordingFileName) // Configurable base path needed

	// conference <confname> record <path> [limit]
	command := fmt.Sprintf("bgapi conference %s record %s", verifiedConf.FriendlyName, fullRecordingPath)
	if maxDurationSec != nil && *maxDurationSec > 0 {
		command += fmt.Sprintf(" %d", *maxDurationSec)
	}
	s.log.Debugf("Sending ESL command for StartRecordingConference: %s", command)
	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.log.Errorf("ESL StartRecordingConference command failed for conf %s: %v", confSid, err)
		return "", "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	// TODO: Consider creating a preliminary domain.Recording metadata entry here or rely on events.
	return recordingFileName, jobID, nil
}

func (s *conferenceService) StopRecordingConference(ctx context.Context, accountSid string, confSid string, recordingNameOrUUID string) (string, error) {
	s.log.Infof("StopRecordingConference: Attempting for conf %s, account %s, recording: %s", confSid, accountSid, recordingNameOrUUID)
	verifiedConf, err := s.getVerifiedConference(ctx, accountSid, confSid)
	if err != nil {
		return "", err
	}
	// No strict InProgress check, might want to stop recording even if conf is ending.
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}
	// conference <confname> norecord <path_or_all>
	command := fmt.Sprintf("bgapi conference %s norecord %s", verifiedConf.FriendlyName, recordingNameOrUUID)
	s.log.Debugf("Sending ESL command for StopRecordingConference: %s", command)
	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.log.Errorf("ESL StopRecordingConference command failed for conf %s: %v", confSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	return jobID, nil
}

func (s *conferenceService) MuteParticipantInConference(ctx context.Context, accountSid string, confSid string, participantCallSidOrMemberID string, mute bool) (string, error) {
	s.log.Infof("MuteParticipantInConference: Attempting for conf %s, account %s, participant %s, mute: %t", confSid, accountSid, participantCallSidOrMemberID, mute)
	verifiedConf, err := s.getVerifiedConference(ctx, accountSid, confSid)
	if err != nil {
		return "", err
	}
	if verifiedConf.Status != domain.ConferenceStatusInProgress {
		return "", fmt.Errorf("%w: conference %s is not in-progress (status: %s)", ErrConferenceInvalidState, confSid, verifiedConf.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}
	action := "unmute"; if mute { action = "mute" }
	// conference <confname> <mute|unmute> <member_id>
	command := fmt.Sprintf("bgapi conference %s %s %s", verifiedConf.FriendlyName, action, participantCallSidOrMemberID)
	s.log.Debugf("Sending ESL command for MuteParticipant: %s", command)
	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.log.Errorf("ESL MuteParticipant command failed for conf %s, participant %s: %v", confSid, participantCallSidOrMemberID, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	// DB update for participant's IsMuted status should happen via ESL events ideally.
	return jobID, nil
}

func (s *conferenceService) KickParticipantFromConference(ctx context.Context, accountSid string, confSid string, participantCallSidOrMemberID string) (string, error) {
	s.log.Infof("KickParticipantFromConference: Attempting for conf %s, account %s, participant %s", confSid, accountSid, participantCallSidOrMemberID)
	verifiedConf, err := s.getVerifiedConference(ctx, accountSid, confSid)
	if err != nil {
		return "", err
	}
	if verifiedConf.Status != domain.ConferenceStatusInProgress {
		return "", fmt.Errorf("%w: conference %s is not in-progress (status: %s)", ErrConferenceInvalidState, confSid, verifiedConf.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}
	// conference <confname> kick <member_id>
	command := fmt.Sprintf("bgapi conference %s kick %s", verifiedConf.FriendlyName, participantCallSidOrMemberID)
	s.log.Debugf("Sending ESL command for KickParticipant: %s", command)
	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.log.Errorf("ESL KickParticipant command failed for conf %s, participant %s: %v", confSid, participantCallSidOrMemberID, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	// DB update for participant's LeaveTime should happen via ESL events.
	return jobID, nil
}
