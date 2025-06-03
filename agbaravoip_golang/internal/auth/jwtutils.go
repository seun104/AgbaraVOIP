package auth

import (
	"fmt"
	"time"
	// "errors" // Not strictly needed for this basic version, but often useful in auth packages

	"github.com/golang-jwt/jwt/v4" // Using v4 as specified
)

// JWTClaims defines the custom claims for AgbaraVOIP JWT tokens.
type JWTClaims struct {
	AccountSID       string   `json:"acc_sid"`
	ParentAccountSID string   `json:"parent_acc_sid,omitempty"` // If subaccounts exist
	Role             string   `json:"role,omitempty"`           // e.g., "user", "admin"
	// Add other claims as needed, e.g., username, permissions
	jwt.RegisteredClaims // Embed standard claims (iss, sub, aud, exp, nbf, iat, jti)
}

var jwtSecretKey []byte // This will be initialized from config

// InitJWTSecret initializes the JWT secret key.
// This should be called once at application startup from main.go or config loading.
func InitJWTSecret(secret string) {
	if secret == "" {
		// In a real app, panic or ensure a default secure key is loaded,
		// or prevent startup if no key. For this subtask, we'll assume a valid secret is always passed during init.
		// If logging was available here (it's not, as this is a utility pkg), one might log a warning.
	}
	jwtSecretKey = []byte(secret)
}

// GenerateToken creates a new JWT token with the provided claims.
func GenerateToken(claims *JWTClaims, expirationTime time.Time) (string, error) {
	if len(jwtSecretKey) == 0 {
		return "", fmt.Errorf("JWT secret key not initialized")
	}

	// Set standard claims if not already set by caller
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	}
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(time.Now())
	}
	if claims.NotBefore == nil {
		// Allow for slight clock skew, token valid slightly in the past
		claims.NotBefore = jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))
	}
	// Example: Set Issuer and Subject if desired
	// claims.Issuer = "AgbaraVOIP"
	// if claims.Subject == "" && claims.AccountSID != "" { // Set subject if not already set
	//     claims.Subject = claims.AccountSID
	// }


	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken parses and validates a JWT token string.
// It returns the custom claims if the token is valid, or an error otherwise.
func ValidateToken(tokenString string) (*JWTClaims, error) {
	if !IsJWTSecretInitialized() { // Use the helper
		return nil, fmt.Errorf("JWT secret key not initialized")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		// Example of more detailed error checking:
		// if ve, ok := err.(*jwt.ValidationError); ok {
		//     if ve.Errors&jwt.ValidationErrorMalformed != 0 {
		//         return nil, errors.New("malformed token")
		//     } else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
		//         return nil, errors.New("token is either expired or not active yet")
		//     } else {
		//         return nil, fmt.Errorf("token validation error: %w", err)
		//     }
		// }
		return nil, fmt.Errorf("token parsing/validation failed: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token or claims type assertion failed")
}

// IsJWTSecretInitialized checks if the JWT secret key has been set.
func IsJWTSecretInitialized() bool {
	return len(jwtSecretKey) > 0
}
