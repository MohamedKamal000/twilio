package validation

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"oba-twilio/models"
)

var (
	// stopIDPattern matches basic stop ID format:
	// - 1–20 characters
	// - letters, digits, underscores, and hyphens
	// Security-sensitive checks (injection patterns, control chars) are applied separately.
	stopIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,20}$`)

	// phoneNumberPattern matches valid phone numbers with optional country code
	phoneNumberPattern = regexp.MustCompile(`^\+?[1-9][0-9]{7,14}$`)

	// dangerousPatterns contains patterns that could be used for injection attacks
	dangerousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>`),                                           // Script tags
		regexp.MustCompile(`(?i)</?\w+[^>]*>`),                                            // Any HTML/XML tags
		regexp.MustCompile(`(?i)(union|select|drop|insert|update|delete|create|alter)\s`), // SQL keywords
		regexp.MustCompile(`['";\\]`),                                                     // SQL injection characters
		regexp.MustCompile(`%[0-9a-f]{2}`),                                                // URL encoding
		regexp.MustCompile(`&[#\w]+;`),                                                    // HTML entities
	}
)

// ValidateStopID validates a stop ID for format, length, and security
func ValidateStopID(stopID string) error {
	// Trim whitespace
	stopID = strings.TrimSpace(stopID)

	// Check if empty
	if stopID == "" {
		return models.NewValidationFailedError("stop ID cannot be empty", nil)
	}

	// Check basic length bounds
	if len(stopID) < 1 || len(stopID) > 20 {
		return models.NewValidationFailedError("stop ID must be between 1 and 20 characters", nil)
	}

	// Check basic format: letters, digits, underscores, hyphens
	if !stopIDPattern.MatchString(stopID) {
		return models.NewValidationFailedError("stop ID must contain only letters, numbers, underscores, and hyphens", nil)
	}

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(stopID) {
			return models.NewValidationFailedError("stop ID contains invalid characters", nil)
		}
	}

	// Additional checks for control characters
	for _, char := range stopID {
		if unicode.IsControl(char) {
			return models.NewValidationFailedError("stop ID contains invalid control characters", nil)
		}
	}

	return nil
}

// ValidatePhoneNumber validates a phone number for Twilio compatibility
func ValidatePhoneNumber(phoneNumber string) error {
	// Trim whitespace
	phoneNumber = strings.TrimSpace(phoneNumber)

	// Check if empty
	if phoneNumber == "" {
		return models.NewValidationFailedError("phone number cannot be empty", nil)
	}

	// Check length (8-16 characters including country code)
	if len(phoneNumber) < 8 || len(phoneNumber) > 16 {
		return models.NewValidationFailedError("phone number must be between 8 and 16 characters", nil)
	}

	// Check format
	if !phoneNumberPattern.MatchString(phoneNumber) {
		return models.NewValidationFailedError("phone number format is invalid", nil)
	}

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(phoneNumber) {
			return models.NewValidationFailedError("phone number contains invalid characters", nil)
		}
	}

	return nil
}

// SanitizeUserInput removes potentially dangerous content from user input
func SanitizeUserInput(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove HTML/XML tags (including content for script tags)
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	input = scriptPattern.ReplaceAllString(input, "")

	// Remove all other HTML/XML tags
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	input = tagPattern.ReplaceAllString(input, "")

	// Remove control characters
	var sanitized strings.Builder
	for _, char := range input {
		if !unicode.IsControl(char) || char == '\t' || char == '\n' || char == '\r' {
			sanitized.WriteRune(char)
		}
	}
	input = sanitized.String()

	// Normalize multiple whitespace/newlines to single spaces
	spacePattern := regexp.MustCompile(`\s+`)
	input = spacePattern.ReplaceAllString(input, " ")

	// Final trim
	return strings.TrimSpace(input)
}

// ValidateDisambiguationChoice validates a user's disambiguation choice
func ValidateDisambiguationChoice(choice string, maxChoices int) error {
	// Trim whitespace
	choice = strings.TrimSpace(choice)

	// Check if empty
	if choice == "" {
		return models.NewValidationFailedError("choice cannot be empty", nil)
	}

	// Parse as integer
	choiceNum, err := strconv.Atoi(choice)
	if err != nil {
		return models.NewValidationFailedError("choice must be a number", err)
	}

	// Check range
	if choiceNum < 1 || choiceNum > maxChoices {
		return models.NewValidationFailedError("choice must be between 1 and "+strconv.Itoa(maxChoices), nil)
	}

	return nil
}

// ValidateLanguageCode validates a language code format
func ValidateLanguageCode(langCode string) error {
	// Trim whitespace
	langCode = strings.TrimSpace(langCode)

	// Check if empty
	if langCode == "" {
		return models.NewValidationFailedError("language code cannot be empty", nil)
	}

	// Check length and format (e.g., "en-US", "es-ES")
	langPattern := regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)
	if !langPattern.MatchString(langCode) {
		return models.NewValidationFailedError("language code must be in format 'en' or 'en-US'", nil)
	}

	return nil
}

// ValidateMinutesWindow validates a time window for arrival queries
func ValidateMinutesWindow(minutes int) error {
	// Check reasonable range (0 to 8 hours = 480 minutes)
	if minutes < 0 || minutes > 480 {
		return models.NewValidationFailedError("minutes window must be between 0 and 480", nil)
	}

	return nil
}

// ValidateTwilioCallSid validates a Twilio call SID format
func ValidateTwilioCallSid(callSid string) error {
	// Trim whitespace
	callSid = strings.TrimSpace(callSid)

	// Check if empty
	if callSid == "" {
		return models.NewValidationFailedError("call SID cannot be empty", nil)
	}

	// Twilio SIDs start with CA, are 34 characters long, and contain only alphanumeric characters
	sidPattern := regexp.MustCompile(`^CA[a-f0-9]{32}$`)
	if !sidPattern.MatchString(callSid) {
		return models.NewValidationFailedError("invalid Twilio call SID format", nil)
	}

	return nil
}

// ValidateMessageBody validates SMS message body content
func ValidateMessageBody(body string) error {
	// Check length (SMS messages can be up to 1600 characters)
	if len(body) > 1600 {
		return models.NewValidationFailedError("message body too long", nil)
	}

	// Check for dangerous patterns (but allow more flexibility than stop IDs)
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>`)
	if scriptPattern.MatchString(body) {
		return models.NewValidationFailedError("message body contains potentially dangerous content", nil)
	}

	return nil
}
