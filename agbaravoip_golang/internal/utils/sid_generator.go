package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	// Base62 characters (0-9, a-z, A-Z)
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sidLength   = 30 // Desired length of the random part of the SID
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// RandStringBytesBase62 generates a random string of length n using base62 characters.
func RandStringBytesBase62(n int) string {
	if n <= 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(base62Chars[seededRand.Intn(len(base62Chars))])
	}
	return sb.String()
}

// GenerateSID creates a new unique SID with a given prefix.
// Example: GenerateSID("CA") -> "CA_randomstring..."
// The total length will be len(prefix) + 1 (for underscore) + sidLength.
// For consistency with previous SIDs, this might be too long.
// Let's make it prefix + random part.
func GenerateSID(prefix string) string {
	// Ensure prefix is not too long if there's a total length constraint on SIDs (e.g., VARCHAR(64))
	// Max prefix length could be e.g. 4 to keep total SID around 34-35 chars.
	// This version just prepends.
	return fmt.Sprintf("%s%s", prefix, RandStringBytesBase62(sidLength))
}

// GenerateSIDWithTimestamp creates a time-sortable SID (like KSUID) but simpler.
// Format: <Prefix>_<Base36Timestamp>_<RandomChars>
// This helps in sorting by creation time if needed and adds more uniqueness.
// For now, sticking to the simpler GenerateSID.
/*
func GenerateSIDWithTimestamp(prefix string) string {
	timestampPart := strings.ToUpper(strconv.FormatInt(time.Now().UnixNano(), 36))
	randomPartLength := sidLength - len(timestampPart) - 1 // -1 for an potential underscore
	if randomPartLength < 4 { // Ensure at least some random characters
		randomPartLength = 4
	}
	randomPart := RandStringBytesBase62(randomPartLength)
	return fmt.Sprintf("%s_%s_%s", prefix, timestampPart, randomPart)
}
*/
