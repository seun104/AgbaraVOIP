package services

import (
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt" // For AuthToken hashing
)

// AccountService defines the interface for account management operations.
type AccountService interface {
	CreateMasterAccount(ctx context.Context, friendlyName string) (*models.Account, error)
	CreateSubAccount(ctx context.Context, friendlyName string, parentSid string) (*models.Account, error)
	GetAccount(ctx context.Context, accountSid string) (*models.Account, error)
	GetAccountBySidAndToken(ctx context.Context, accountSid string, authToken string) (*models.Account, error) // For Validate
	ListSubAccounts(ctx context.Context, parentSid string) ([]*models.Account, error)
	ChangeAccountStatus(ctx context.Context, accountSid string, status models.AccountStatus) (*models.Account, error)
	ChangeAccountType(ctx context.Context, accountSid string, accType models.AccountType) (*models.Account, error)
    GenerateAuthToken(ctx context.Context, accountSid string) (string, error) // Helper to generate/regenerate token
}

// PostgresAccountService implements AccountService for PostgreSQL.
type PostgresAccountService struct {
	db *sql.DB
}

// NewPostgresAccountService creates a new PostgresAccountService.
func NewPostgresAccountService(db *sql.DB) *PostgresAccountService {
	return &PostgresAccountService{db: db}
}

// Helper to hash an auth token
func hashAuthToken(token string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// scanAccount is a helper to scan a sql.Row or sql.Rows into a models.Account struct.
func scanAccount(scanner interface{ Scan(...interface{}) error }) (*models.Account, error) {
	acc := &models.Account{}
	var parentSid sql.NullString // Handle NULL ParentSid for master accounts
	var phoneNumber sql.NullString

	err := scanner.Scan(
		&acc.Sid, &parentSid, &acc.FriendlyName, &phoneNumber,
		&acc.Type, &acc.Status, &acc.AuthToken, // AuthToken is stored hashed
		&acc.DateCreated, &acc.DateUpdated,
	)
	if err != nil {
		return nil, err
	}

	acc.ParentSid = parentSid.String // If NULL, will be empty string
	acc.PhoneNumber = phoneNumber.String
	return acc, nil
}

// CreateMasterAccount creates a new top-level account.
func (s *PostgresAccountService) CreateMasterAccount(ctx context.Context, friendlyName string) (*models.Account, error) {
	acc := &models.Account{
		Sid:          "AC" + uuid.NewString(),
		FriendlyName: friendlyName,
		Type:         models.AccountTypeTrial,  // Default type
		Status:       models.AccountStatusActive, // Default status
		DateCreated:  time.Now().UTC(),
		DateUpdated:  time.Now().UTC(),
	}
	
	// Generate a new raw token, then hash it for storage
	rawToken := uuid.NewString() // Simple new token
	hashedToken, err := hashAuthToken(rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed to hash auth token: %w", err)
	}
	acc.AuthToken = hashedToken // Store the hashed token

	query := `
		INSERT INTO accounts (sid, friendly_name, type, status, auth_token, date_created, date_updated, parent_sid, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL)
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		acc.Sid, acc.FriendlyName, acc.Type, acc.Status, acc.AuthToken, acc.DateCreated, acc.DateUpdated,
	)
    // Scan to get all fields, including those set by DB defaults (though we set most here)
	return scanAccount(row)
}

// CreateSubAccount creates a new account under a parent account.
func (s *PostgresAccountService) CreateSubAccount(ctx context.Context, friendlyName string, parentSid string) (*models.Account, error) {
	// Verify parent account exists
	_, err := s.GetAccount(ctx, parentSid)
	if err != nil {
		return nil, fmt.Errorf("parent account with SID %s not found: %w", parentSid, err)
	}

	acc := &models.Account{
		Sid:          "AC" + uuid.NewString(),
		ParentSid:    parentSid,
		FriendlyName: friendlyName,
		Type:         models.AccountTypeTrial,
		Status:       models.AccountStatusActive,
		DateCreated:  time.Now().UTC(),
		DateUpdated:  time.Now().UTC(),
	}
	rawToken := uuid.NewString()
	hashedToken, err := hashAuthToken(rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed to hash auth token: %w", err)
	}
	acc.AuthToken = hashedToken

	query := `
		INSERT INTO accounts (sid, parent_sid, friendly_name, type, status, auth_token, date_created, date_updated, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL)
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		acc.Sid, acc.ParentSid, acc.FriendlyName, acc.Type, acc.Status, acc.AuthToken, acc.DateCreated, acc.DateUpdated,
	)
	return scanAccount(row)
}

// GetAccount retrieves an account by its SID.
func (s *PostgresAccountService) GetAccount(ctx context.Context, accountSid string) (*models.Account, error) {
	query := `
		SELECT sid, parent_sid, friendly_name, phone_number, type, status, auth_token, date_created, date_updated
		FROM accounts WHERE sid = $1;
	`
	row := s.db.QueryRowContext(ctx, query, accountSid)
	account, err := scanAccount(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account with SID %s not found: %w", accountSid, err)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return account, nil
}

// GetAccountBySidAndToken retrieves an account if SID and plain text token match. Used for validation.
func (s *PostgresAccountService) GetAccountBySidAndToken(ctx context.Context, accountSid string, plainToken string) (*models.Account, error) {
    account, err := s.GetAccount(ctx, accountSid) // Fetches account, including its hashed token
    if err != nil {
        return nil, err // Handles not found or other DB errors
    }

    // Compare the provided plainToken with the stored hashed token
    err = bcrypt.CompareHashAndPassword([]byte(account.AuthToken), []byte(plainToken))
    if err != nil {
        // Passwords don't match or some other error (e.g. hash too short)
        return nil, fmt.Errorf("auth token validation failed: %w", err) // Consider a more generic auth error
    }
    return account, nil // Token matches
}


// ListSubAccounts retrieves all accounts that have the given parentSid.
func (s *PostgresAccountService) ListSubAccounts(ctx context.Context, parentSid string) ([]*models.Account, error) {
	query := `
		SELECT sid, parent_sid, friendly_name, phone_number, type, status, auth_token, date_created, date_updated
		FROM accounts WHERE parent_sid = $1 ORDER BY date_created DESC;
	`
	rows, err := s.db.QueryContext(ctx, query, parentSid)
	if err != nil {
		return nil, fmt.Errorf("failed to query sub-accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sub-account row: %w", err)
		}
		accounts = append(accounts, account)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sub-account rows: %w", err)
	}
    if accounts == nil {
        accounts = []*models.Account{} // Return empty slice, not nil
    }
	return accounts, nil
}

// ChangeAccountStatus updates the status of an account.
func (s *PostgresAccountService) ChangeAccountStatus(ctx context.Context, accountSid string, status models.AccountStatus) (*models.Account, error) {
	query := `
		UPDATE accounts SET status = $1, date_updated = $2
		WHERE sid = $3
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query, status, time.Now().UTC(), accountSid)
	account, err := scanAccount(row)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("account %s not found for status update: %w", accountSid, err)
        }
        return nil, fmt.Errorf("failed to update account status: %w", err)
    }
    return account, nil
}

// ChangeAccountType updates the type of an account.
func (s *PostgresAccountService) ChangeAccountType(ctx context.Context, accountSid string, accType models.AccountType) (*models.Account, error) {
	query := `
		UPDATE accounts SET type = $1, date_updated = $2
		WHERE sid = $3
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query, accType, time.Now().UTC(), accountSid)
	account, err := scanAccount(row)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("account %s not found for type update: %w", accountSid, err)
        }
        return nil, fmt.Errorf("failed to update account type: %w", err)
    }
	return account, nil
}

// GenerateAuthToken generates a new auth token for an account, stores its hash, and returns the plain token.
func (s *PostgresAccountService) GenerateAuthToken(ctx context.Context, accountSid string) (string, error) {
    plainToken := uuid.NewString() + uuid.NewString() // Make it longer
    hashedToken, err := hashAuthToken(plainToken)
    if err != nil {
        return "", fmt.Errorf("failed to hash new token: %w", err)
    }

    query := `UPDATE accounts SET auth_token = $1, date_updated = $2 WHERE sid = $3 RETURNING sid;`
    var updatedSid string
    err = s.db.QueryRowContext(ctx, query, hashedToken, time.Now().UTC(), accountSid).Scan(&updatedSid)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", fmt.Errorf("account %s not found for token regeneration: %w", accountSid, err)
        }
        return "", fmt.Errorf("failed to update auth token in db: %w", err)
    }
    return plainToken, nil
}
