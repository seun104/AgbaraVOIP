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
)

type conferenceService struct {
	db  *gorm.DB // Changed to gorm.DB
	log *logrus.Entry
}

func NewConferenceService(db *gorm.DB, logger *logrus.Logger) ConferenceService { // Changed to gorm.DB
	return &conferenceService{
		db:  db,
		log: logger.WithField("service", "conference"),
	}
}

func (s *conferenceService) GetConferenceBySID(ctx context.Context, sid string) (*domain.Conference, error) {
	var conf domain.Conference
	if err := s.db.WithContext(ctx).Where("sid = ?", sid).First(&conf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying conference by SID %s: %w", sid, err)
	}
	return &conf, nil
}

func (s *conferenceService) GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error) {
	var conf domain.Conference
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

func (s *conferenceService) UpdateConferenceStatus(ctx context.Context, sid string, status domain.ConferenceStatus) error {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&domain.Conference{}).Where("sid = ?", sid).Updates(map[string]interface{}{"status": status, "updated_at": now})
	if res.Error != nil { return fmt.Errorf("updating status for conference %s: %w", sid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Updated status for conference %s to %s", sid, status)
	return nil
}

func (s *conferenceService) EndConference(ctx context.Context, sid string, endTime time.Time) error {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&domain.Conference{}).Where("sid = ?", sid).Updates(map[string]interface{}{
		"status":     domain.ConferenceStatusCompleted,
		"end_time":   endTime.UTC(),
		"updated_at": now,
	})
	if res.Error != nil { return fmt.Errorf("ending conference %s: %w", sid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Ended conference %s at %v", sid, endTime)
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

func (s *conferenceService) GetParticipant(ctx context.Context, pSid string) (*domain.ConferenceParticipant, error) {
	var part domain.ConferenceParticipant
	err := s.db.WithContext(ctx).Where("sid = ?", pSid).First(&part).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, domain.ErrNotFound }
		return nil, fmt.Errorf("querying participant by SID %s: %w", pSid, err)
	}
	return &part, nil
}

func (s *conferenceService) GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error) {
	var part domain.ConferenceParticipant
	err := s.db.WithContext(ctx).Where("call_sid = ? AND leave_time IS NULL", callSid).First(&part).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, domain.ErrNotFound }
		return nil, fmt.Errorf("querying participant by CallSID %s: %w", callSid, err)
	}
	return &part, nil
}

func (s *conferenceService) UpdateParticipantMuteStatus(ctx context.Context, pSid string, isMuted bool) error {
	res := s.db.WithContext(ctx).Model(&domain.ConferenceParticipant{}).Where("sid = ? AND leave_time IS NULL", pSid).Update("is_muted", isMuted)
	if res.Error != nil { return fmt.Errorf("updating mute status for participant %s: %w", pSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Updated mute status for participant %s to %t", pSid, isMuted)
	return nil
}

func (s *conferenceService) UpdateParticipantModeratorStatus(ctx context.Context, pSid string, isModerator bool) error {
	res := s.db.WithContext(ctx).Model(&domain.ConferenceParticipant{}).Where("sid = ? AND leave_time IS NULL", pSid).Update("is_moderator", isModerator)
	if res.Error != nil { return fmt.Errorf("updating moderator status for participant %s: %w", pSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Updated moderator status for participant %s to %t", pSid, isModerator)
	return nil
}

func (s *conferenceService) RemoveParticipant(ctx context.Context, pSid string, leaveTime time.Time) error {
	res := s.db.WithContext(ctx).Model(&domain.ConferenceParticipant{}).Where("sid = ? AND leave_time IS NULL", pSid).Update("leave_time", leaveTime.UTC())
	if res.Error != nil { return fmt.Errorf("setting leave_time for participant %s: %w", pSid, res.Error) }
	if res.RowsAffected == 0 { return domain.ErrNotFound }
	s.log.Infof("Participant %s left conference at %v", pSid, leaveTime)
	return nil
}

func (s *conferenceService) ListParticipants(ctx context.Context, confSid string) ([]*domain.ConferenceParticipant, error) {
	var participants []*domain.ConferenceParticipant
	err := s.db.WithContext(ctx).Where("conference_sid = ? AND leave_time IS NULL", confSid).Find(&participants).Error
	if err != nil { return nil, fmt.Errorf("listing participants for conference %s: %w", confSid, err) }
	return participants, nil
}
