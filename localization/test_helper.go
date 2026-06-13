package localization

import (
	"sync"
)

// NewTestManagerWithStrings creates a localization manager with custom strings and languages
func NewTestManagerWithStrings(strings map[string]map[string]string, supportedLanguages []string) *LocalizationManager {
	return &LocalizationManager{
		strings:            strings,
		mu:                 sync.RWMutex{},
		defaultLanguage:    supportedLanguages[0],
		supportedLanguages: supportedLanguages,
	}
}

// NewTestManager creates a basic localization manager for testing
func NewTestManager() *LocalizationManager {
	return &LocalizationManager{
		strings: map[string]map[string]string{
			"en-US": {
				// Generic keys used by health checks
				"welcome": "Welcome",
				"error":   "An error occurred",
				"help":    "Help information",
				"goodbye": "Goodbye",

				// Voice prompts
				"voice.welcome":                   "Welcome to {brand} transit information. Please enter your stop ID followed by the pound key.",
				"voice.timeout":                   "Sorry, I didn't hear anything. Please call back and try again.",
				"voice.error.invalid_request":     "Invalid request format.",
				"voice.error.no_stops_found":      "Sorry, I couldn't find any stops with that ID. Please check the stop ID and try again.",
				"voice.error.template_failed":     "Error generating response.",
				"voice.error.service_unavailable": "{brand} service is temporarily unavailable. Please try again in a moment.",
				"voice.menu.more_departures":      "To hear more departures, press 1.",
				"voice.menu.main_menu":            "To go back to the main menu, press 2.",
				"voice.language.spanish_option":   "Para continuar en Español, presione asterisco.",
				"voice.language.choose_language":  "To choose another language, press asterisk.",
				"sms.no_stops_found":              "Sorry, no stops found with ID %s. Please check and try again.",
				"sms.no_arrivals":                 "No upcoming arrivals found for this stop.",
				"sms.service_unavailable":         "{brand} service is temporarily unavailable. Please try again.",
				"sms.error.invalid_choice":        "Please choose a number between 1 and %d.",
				"voice.error.invalid_stop_id":     "Invalid stop ID. Please try calling again with a valid stop ID.",
				"voice.error.invalid_choice":      "Please press a number between 1 and %d.",
				"voice.error.no_digits":           "I didn't receive any digits. Please try calling again.",
				"voice.error.stop_not_found":      "Sorry, I couldn't find any stops with that ID. Please check the stop ID and try again.",
				"voice.error.no_active_session":   "No active selection. Please call again and enter a stop ID to get started.",
				"voice.disambiguation.prompt":     "I found %d stops with ID %s.",
				"error.internal_error":            "An internal error occurred. Please try again.",
				"sms.menu.more_hint":              "Reply 'more' for later buses",
				"sms.menu.help_hint":              "Reply 'help' for usage info",
				"sms.error.invalid_stop":          "Please send a valid stop ID (e.g., 75403).",
				"sms.help":                        "Send a stop ID (e.g., 75403) to get bus arrivals. Reply 'more' for later buses, 'help' for this message.",
				"sms.keyword.more":                "Showing buses in the next %d minutes:",
				"sms.keyword.invalid":             "I don't understand '%s'. Send a stop ID or reply 'help' for usage info.",
				"sms.session.expired":             "Session expired. Please send a stop ID to get started.",
				"sms.language.switched":           "Language switched to English",
				"sms.menu.new_hint":               "Reply 'new' for a different stop",
				"sms.error.search_failed":         "Sorry, I couldn't search for stop %s. Please check the stop ID and try again.",
				"voice.arrival.arriving_now":      "arriving now",
				"voice.arrival.in_one_minute":     "in 1 minute",
				"voice.arrival.in_minutes":        "in %d minutes",
				"voice.arrival.route_to":          "Route %s to %s",
				"voice.arrival.arrivals_for":      "Arrivals for %s.",
				"voice.arrival.no_arrivals":       "No upcoming arrivals found for this stop.",
			},
		},
		mu:                 sync.RWMutex{},
		defaultLanguage:    "en-US",
		supportedLanguages: []string{"en-US"},
	}
}
