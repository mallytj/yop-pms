package helpers

import (
	"errors"
	"regexp"
	"unicode/utf8"
)

var (
	ErrStartingTx  = errors.New("error starting transaction")
	ErrCommitingTx = errors.New("error committing transaction")
)

// StringCharCount returns the number of characters in a string, accounting for multi-byte characters.
func StringCharCount(s string) int {
	return utf8.RuneCountInString(s)
}

// IsValidEmail performs a regex check to validate email format.
func IsValidEmail(email string) bool {
	// Simple regex for demonstration purposes; consider using a more robust regex for production use.
	// Checks for general email format: local-part@domain
	const emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}
