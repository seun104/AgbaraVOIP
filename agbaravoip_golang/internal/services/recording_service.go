package services

import (
	"context"
	"errors"
	"fmt"
	"strings" // For query builder in ListRecordings
	// "time" // Not strictly needed for these methods unless using time for default filter values

	"github.com/user/agbaravoip_golang/internal/domain"
	// "github.com/jmoiron/sqlx" // Not using sqlx for this service
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ErrRecordingNotFound indicates that a Recording record was not found.
var ErrRecordingNotFound = errors.New("recording not found")
// ErrRecordingDeletionFailed indicates an issue deleting recording metadata.
var ErrRecordingDeletionFailed = errors.New("recording metadata deletion failed")


type recordingService struct {
	db     *gorm.DB // Using GORM for consistency with other new services
	logger *logrus.Entry
}

// NewRecordingService creates a new IRecordingService.
func NewRecordingService(db *gorm.DB, logger *logrus.Logger) IRecordingService {
	return &recordingService{
		db:     db,
		logger: logger.WithField("service", "recording"),
	}
}

func (s *recordingService) GetRecordingBySID(ctx context.Context, accountSid string, recordingSid string) (*domain.Recording, error) {
	s.logger.Infof("GetRecordingBySID: Getting recording SID %s for account %s", recordingSid, accountSid)
	var rec domain.Recording

	// Using WithContext for GORM
	if err := s.db.WithContext(ctx).Where("sid = ? AND account_sid = ?", recordingSid, accountSid).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warnf("GetRecordingBySID: Recording %s not found for account %s", recordingSid, accountSid)
			return nil, ErrRecordingNotFound // Use specific error
		}
		s.logger.Errorf("GetRecordingBySID: Error querying recording %s for account %s: %v", recordingSid, accountSid, err)
		return nil, fmt.Errorf("querying recording by SID %s for account %s: %w", recordingSid, accountSid, err)
	}
	return &rec, nil
}

func (s *recordingService) ListRecordings(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.Recording, error) {
	s.logger.Infof("ListRecordings: Listing recordings for account SID: %s with filters: %v", accountSid, filters)

	var recordings []*domain.Recording
	query := s.db.WithContext(ctx).Model(&domain.Recording{}).Where("account_sid = ?", accountSid)

	// Example filters:
	if callSID, ok := filters["call_sid"].(string); ok && callSID != "" {
		query = query.Where("call_sid = ?", callSID)
	}
	if conferenceSID, ok := filters["conference_sid"].(string); ok && conferenceSID != "" {
		query = query.Where("conference_sid = ?", conferenceSID)
	}
	if format, ok := filters["format"].(string); ok && format != "" {
		query = query.Where("format = ?", format)
	}
	// Add date range filters if needed, e.g., using created_at
	// if dateFromStr, ok := filters["date_from"].(string); ok {
	//     if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
	//         query = query.Where("created_at >= ?", dateFrom)
	//     }
	// }
	// if dateToStr, ok := filters["date_to"].(string); ok {
	//     if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
	//         query = query.Where("created_at <= ?", dateTo.Add(23*time.Hour+59*time.Minute+59*time.Second)) // End of day
	//     }
	// }


	if err := query.Order("created_at DESC").Find(&recordings).Error; err != nil {
		s.logger.Errorf("ListRecordings: Error querying recordings for account %s: %v", accountSid, err)
		return nil, fmt.Errorf("querying recordings for account %s: %w", accountSid, err)
	}
	return recordings, nil
}

func (s *recordingService) DeleteRecording(ctx context.Context, accountSid string, recordingSid string) error {
	s.logger.Infof("DeleteRecording: Attempting to delete recording metadata SID %s for account %s", recordingSid, accountSid)

	// Verify ownership before deleting
	// GetRecordingBySID already logs if not found or other error
	_, err := s.GetRecordingBySID(ctx, accountSid, recordingSid)
	if err != nil {
		return err // Handles ErrRecordingNotFound or other DB errors
	}

	// Perform the delete operation
	// GORM's Delete method requires a struct or a primary key.
	// If deleting by SID (which is unique), this should work.
	result := s.db.WithContext(ctx).Where("sid = ? AND account_sid = ?", recordingSid, accountSid).Delete(&domain.Recording{})
	if result.Error != nil {
		s.logger.Errorf("DeleteRecording: Failed to delete recording metadata %s for account %s from DB: %v", recordingSid, accountSid, result.Error)
		return fmt.Errorf("%w: %v", ErrRecordingDeletionFailed, result.Error)
	}

	if result.RowsAffected == 0 {
		// This case should ideally be caught by GetRecordingBySID check above,
		// but as a safeguard for race conditions or other issues:
		s.logger.Warnf("DeleteRecording: No recording metadata found with SID %s for account %s to delete (RowsAffected = 0)", recordingSid, accountSid)
		return ErrRecordingNotFound
	}

	s.logger.Infof("DeleteRecording: Recording metadata %s for account %s deleted successfully", recordingSid, accountSid)
	// Note: This does NOT delete the actual recording file from storage.
	// That would require additional logic (e.g., interacting with a file storage service).
	return nil
}
