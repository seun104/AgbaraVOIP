package auth_test

import (
	"testing"
	"time"

	"github.com/user/agbaravoip_golang/internal/auth" // Package being tested
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

const testSecret = "testsecretkey12345678901234567890testsecret" // Ensure sufficient length for HS256

// Helper to ensure a clean state for JWT secret for each test, if needed,
// though InitJWTSecret can be called multiple times.
// This specific test suite for jwtutils will manage the secret key state directly.
func resetJWTSecret() {
	auth.InitJWTSecret("")
}
func setTestJWTSecret() {
	auth.InitJWTSecret(testSecret)
}


func TestInitJWTSecret(t *testing.T) {
	resetJWTSecret() // Start clean
	assert.False(t, auth.IsJWTSecretInitialized(), "Should be false before init")

	auth.InitJWTSecret(testSecret)
	assert.True(t, auth.IsJWTSecretInitialized(), "Should be true after init with testSecret")

	auth.InitJWTSecret("")
	assert.False(t, auth.IsJWTSecretInitialized(), "Should be false after re-init with empty string")

	// Restore for any subsequent tests in this package if they don't manage state themselves
	setTestJWTSecret()
}

func TestGenerateToken_Success(t *testing.T) {
	setTestJWTSecret()
	defer resetJWTSecret() // Clean up after test

	accSID := "AC123"
	role := "user"
	expiration := time.Now().Add(1 * time.Hour)

	claims := &auth.JWTClaims{
		AccountSID: accSID,
		Role:       role,
		// RegisteredClaims will be populated by GenerateToken
	}

	tokenString, err := auth.GenerateToken(claims, expiration)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Validate the generated token
	parsedClaims, err := auth.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, parsedClaims)
	assert.Equal(t, accSID, parsedClaims.AccountSID)
	assert.Equal(t, role, parsedClaims.Role)
	assert.WithinDuration(t, expiration, parsedClaims.ExpiresAt.Time, time.Second)
	assert.WithinDuration(t, time.Now(), parsedClaims.IssuedAt.Time, time.Second)
	assert.True(t, parsedClaims.NotBefore.Time.Before(time.Now().Add(time.Second)), "NBF should be in past or now")
}

func TestGenerateToken_NoSecret(t *testing.T) {
	resetJWTSecret() // Ensure no secret
	defer setTestJWTSecret() // Restore for other tests

	claims := &auth.JWTClaims{AccountSID: "AC123"}
	_, err := auth.GenerateToken(claims, time.Now().Add(1*time.Hour))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT secret key not initialized")
}

func TestValidateToken_Success(t *testing.T) {
	setTestJWTSecret()
	defer resetJWTSecret()

	expiration := time.Now().Add(1 * time.Hour)
	claims := &auth.JWTClaims{
		AccountSID: "ACValid",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)), // Ensure it's valid now
			Issuer: "TestIssuer",
			Subject: "TestSubject",
		},
	}
	tokenString, err := auth.GenerateToken(claims, expiration) // GenerateToken will use claims.ExpiresAt
	assert.NoError(t, err)

	parsedClaims, err := auth.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, parsedClaims)
	assert.Equal(t, "ACValid", parsedClaims.AccountSID)
	assert.Equal(t, "TestIssuer", parsedClaims.Issuer)
	assert.Equal(t, "TestSubject", parsedClaims.Subject)
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	setTestJWTSecret() // Generate with correct secret
	expiration := time.Now().Add(1 * time.Hour)
	claims := &auth.JWTClaims{AccountSID: "ACSig", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiration)}}
	tokenString, _ := auth.GenerateToken(claims, expiration)

	auth.InitJWTSecret("anotherwrongsecretkey1234567890123") // Change secret for validation
	defer setTestJWTSecret() // Restore

	_, err := auth.ValidateToken(tokenString)
	assert.Error(t, err)
	if verr, ok := err.(*jwt.ValidationError); ok {
		assert.True(t, verr.Errors&jwt.ValidationErrorSignatureInvalid != 0)
	} else {
		// Fallback check for generic error message if not ValidationError type
		assert.Contains(t, err.Error(), "signature is invalid", "Error message should indicate invalid signature")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	setTestJWTSecret()
	defer resetJWTSecret()

	expiration := time.Now().Add(-1 * time.Hour) // Token expired 1 hour ago
	claims := &auth.JWTClaims{
		AccountSID: "ACExpired",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)), // Issued before expiry
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)), // Valid in the past
		},
	}
	// Manually create token to ensure precise claims without GenerateToken's defaults overriding specific test conditions
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, _ := token.SignedString([]byte(testSecret))

	_, err := auth.ValidateToken(expiredTokenString)
	assert.Error(t, err)
	if verr, ok := err.(*jwt.ValidationError); ok {
		assert.True(t, verr.Errors&jwt.ValidationErrorExpired != 0, "Error should be due to token expiration")
	} else {
		assert.Contains(t, err.Error(), "token is expired", "Error message should indicate token expired")
	}
}

func TestValidateToken_NotYetActive(t *testing.T) {
	setTestJWTSecret()
	defer resetJWTSecret()

	nbfTime := time.Now().Add(1 * time.Hour) // Token not active for 1 hour
	expiration := time.Now().Add(2 * time.Hour)
	claims := &auth.JWTClaims{
		AccountSID: "ACNotYetActive",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(nbfTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	notYetActiveTokenString, _ := token.SignedString([]byte(testSecret))

	_, err := auth.ValidateToken(notYetActiveTokenString)
	assert.Error(t, err)
	if verr, ok := err.(*jwt.ValidationError); ok {
		assert.True(t, verr.Errors&jwt.ValidationErrorNotValidYet != 0, "Error should be due to token not being active yet")
	} else {
		assert.Contains(t, err.Error(), "token is not valid yet", "Error message should indicate token not active")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	setTestJWTSecret()
	defer resetJWTSecret()

	malformedTokenString := "this.is.not.a.valid.jwt.token"
	_, err := auth.ValidateToken(malformedTokenString)
	assert.Error(t, err)
	if verr, ok := err.(*jwt.ValidationError); ok {
		assert.True(t, verr.Errors&jwt.ValidationErrorMalformed != 0, "Error should be due to malformed token")
	} else {
		// The error from ParseWithClaims for malformed token might not always be ValidationErrorMalformed directly,
		// it could be about segment decoding or invalid character.
		// The wrapper in ValidateToken currently says "token parsing/validation failed".
		assert.Contains(t, err.Error(), "token parsing/validation failed")
	}
}

func TestValidateToken_NoSecret(t *testing.T) {
	resetJWTSecret() // Ensure no secret
	defer setTestJWTSecret() // Restore

	// Any token string would do, as it should fail before parsing with a secret
	_, err := auth.ValidateToken("some.dummy.token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT secret key not initialized")
}
