package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/utils" // For SID generation and password hashing
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt" // For password hashing
	"gorm.io/gorm"
)

// ErrFreeswitchServerNotFound indicates that a FreeswitchServer record was not found.
var ErrFreeswitchServerNotFound = errors.New("freeswitch server not found")
// ErrFreeswitchServerValidation indicates a validation error for FreeswitchServer data.
var ErrFreeswitchServerValidation = errors.New("freeswitch server validation failed")
// ErrFreeswitchServerAlreadyExists indicates a FreeswitchServer with the same unique identifier already exists.
var ErrFreeswitchServerAlreadyExists = errors.New("freeswitch server already exists")


type freeswitchServerService struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewFreeswitchServerService creates a new IFreeswitchServerService.
func NewFreeswitchServerService(db *gorm.DB, logger *logrus.Logger) IFreeswitchServerService {
	return &freeswitchServerService{
		db:     db,
		logger: logger.WithField("service", "freeswitch_server"),
	}
}

// hashPassword hashes a given password using bcrypt.
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

func (s *freeswitchServerService) CreateFreeswitchServer(host string, port int, password string, outboundAddress string, isActive *bool) (*domain.FreeswitchServer, error) {
	s.logger.Infof("Attempting to create Freeswitch server for host: %s", host)

	if host == "" || password == "" || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("%w: host, password, and valid port are required", ErrFreeswitchServerValidation)
	}

	hashedPass, err := hashPassword(password)
	if err != nil {
		s.logger.Errorf("Failed to hash password for Freeswitch server %s: %v", host, err)
		return nil, err // internal error
	}

	actualIsActive := true // Default to true
	if isActive != nil {
		actualIsActive = *isActive
	}

	fsServer := &domain.FreeswitchServer{
		SID:             utils.GenerateSID("FS"), // Assuming "FS" prefix for Freeswitch Servers
		Host:            host,
		Port:            port,
		Password:        hashedPass,
		OutboundAddress: outboundAddress,
		IsActive:        actualIsActive,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.db.Create(fsServer).Error; err != nil {
		// TODO: Check for unique constraint errors if any are defined beyond SID (e.g. host:port unique)
		s.logger.Errorf("Failed to create Freeswitch server %s in DB: %v", host, err)
		return nil, fmt.Errorf("could not create Freeswitch server: %w", err)
	}

	s.logger.Infof("Freeswitch server created successfully with SID: %s", fsServer.SID)
	fsServer.Password = "" // Clear password before returning
	return fsServer, nil
}

func (s *freeswitchServerService) GetFreeswitchServerBySID(sid string) (*domain.FreeswitchServer, error) {
	s.logger.Infof("Attempting to retrieve Freeswitch server with SID: %s", sid)
	var fsServer domain.FreeswitchServer
	if err := s.db.Where("sid = ?", sid).First(&fsServer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warnf("Freeswitch server with SID %s not found", sid)
			return nil, ErrFreeswitchServerNotFound
		}
		s.logger.Errorf("Error retrieving Freeswitch server %s from DB: %v", sid, err)
		return nil, err
	}
	fsServer.Password = "" // Clear password before returning
	return &fsServer, nil
}

func (s *freeswitchServerService) ListFreeswitchServers(filters map[string]interface{}) ([]*domain.FreeswitchServer, error) {
	s.logger.Info("Listing Freeswitch servers")
	var servers []*domain.FreeswitchServer
	query := s.db.Model(&domain.FreeswitchServer{})

	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where("is_active = ?", isActive)
	}
	// Add other filters as needed

	if err := query.Find(&servers).Error; err != nil {
		s.logger.Errorf("Error listing Freeswitch servers from DB: %v", err)
		return nil, err
	}
    for _, srv := range servers {
        srv.Password = "" // Clear password
    }
	return servers, nil
}

func (s *freeswitchServerService) UpdateFreeswitchServer(sid string, updates map[string]interface{}) (*domain.FreeswitchServer, error) {
	s.logger.Infof("Attempting to update Freeswitch server with SID: %s", sid)

	fsServer, err := s.GetFreeswitchServerBySID(sid) // Use existing method to find and check existence
	if err != nil {
		return nil, err // Handles ErrFreeswitchServerNotFound
	}
    // Restore original hashed password for GORM update if password is not in updates map
    // This is important because GetFreeswitchServerBySID clears it.
    // A better way would be to fetch without clearing password for internal use.
    var originalServerForPassword domain.FreeswitchServer
    if err := s.db.Select("password").Where("sid = ?", sid).First(&originalServerForPassword).Error; err != nil {
        s.logger.Warnf("Could not fetch original password for server SID %s during update: %v", sid, err)
        // Decide if this is critical. For now, proceed, but password won't be re-saved if not in 'updates'.
    } else {
        fsServer.Password = originalServerForPassword.Password
    }


	// Handle password update separately: if "password" is in updates, hash it.
	if newPass, ok := updates["password"].(string); ok && newPass != "" {
		hashedNewPass, err := hashPassword(newPass)
		if err != nil {
			s.logger.Errorf("Failed to hash new password for Freeswitch server %s: %v", sid, err)
			return nil, err
		}
		updates["password"] = hashedNewPass
	} else if _, ok := updates["password"]; ok && newPass == "" { // Explicitly setting empty password not allowed
        delete(updates, "password") // Or return validation error
    }


	// Ensure `updated_at` is set
	updates["updated_at"] = time.Now().UTC()

	// GORM automatically handles partial updates with map[string]interface{}
	if err := s.db.Model(&fsServer).Where("sid = ?", sid).Updates(updates).Error; err != nil {
		s.logger.Errorf("Failed to update Freeswitch server %s in DB: %v", sid, err)
		// TODO: Check for unique constraint errors if any
		return nil, err
	}

    // Fetch the updated record to return it (Updates does not return full model)
    updatedServer, err := s.GetFreeswitchServerBySID(sid)
    if err != nil {
        s.logger.Errorf("Failed to retrieve updated Freeswitch server %s: %v", sid, err)
        return nil, err
    }

	s.logger.Infof("Freeswitch server %s updated successfully", sid)
	updatedServer.Password = "" // Clear password before returning
	return updatedServer, nil
}

func (s *freeswitchServerService) DeleteFreeswitchServer(sid string) error {
	s.logger.Infof("Attempting to delete Freeswitch server with SID: %s", sid)

	// First, check if server exists
	_, err := s.GetFreeswitchServerBySID(sid)
	if err != nil {
		return err // Handles ErrFreeswitchServerNotFound
	}

	if err := s.db.Where("sid = ?", sid).Delete(&domain.FreeswitchServer{}).Error; err != nil {
		s.logger.Errorf("Failed to delete Freeswitch server %s from DB: %v", sid, err)
		return err
	}

	s.logger.Infof("Freeswitch server %s deleted successfully", sid)
	return nil
}
