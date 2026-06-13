package localization

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DefaultBrandDisplayName is substituted for {brand} in locale strings when none is configured.
const DefaultBrandDisplayName = "OneBusAway"

// brandPlaceholder is used in locale JSON files for whitelabel product name (see APP_BRAND_NAME).
const brandPlaceholder = "{brand}"

// LocalizationManager handles string localization with thread-safe access
type LocalizationManager struct {
	strings            map[string]map[string]string // [language][key]string
	mu                 sync.RWMutex                 // Thread safety for concurrent access
	defaultLanguage    string
	supportedLanguages []string
	brandDisplayName   string // empty until SetBrandDisplayName; GetString falls back to DefaultBrandDisplayName
}

// NewManager creates a new LocalizationManager
func NewManager(supportedLanguagesStr string) (*LocalizationManager, error) {
	languages := strings.Split(supportedLanguagesStr, ",")
	for i, lang := range languages {
		languages[i] = strings.TrimSpace(lang)
	}

	if len(languages) == 0 {
		return nil, fmt.Errorf("no supported languages provided")
	}

	lm := &LocalizationManager{
		strings:            make(map[string]map[string]string),
		defaultLanguage:    languages[0],
		supportedLanguages: languages,
	}

	// Load localization files
	if err := lm.loadLocalizationFiles(); err != nil {
		return nil, fmt.Errorf("failed to load localization files: %w", err)
	}

	return lm, nil
}

// loadLocalizationFiles loads all localization JSON files
func (lm *LocalizationManager) loadLocalizationFiles() error {
	for _, language := range lm.supportedLanguages {
		if err := lm.loadLanguageFile(language); err != nil {
			log.Printf("Failed to load language file for %s: %v", language, err)
			// Continue loading other languages, but log the error
			continue
		}
	}

	// Ensure default language is loaded
	if _, exists := lm.strings[lm.defaultLanguage]; !exists {
		return fmt.Errorf("default language %s failed to load", lm.defaultLanguage)
	}

	return nil
}

// loadLanguageFile loads a specific language file
func (lm *LocalizationManager) loadLanguageFile(language string) error {
	filename := filepath.Join("locales", language+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	var strings map[string]string
	if err := json.Unmarshal(data, &strings); err != nil {
		return fmt.Errorf("failed to parse JSON in %s: %w", filename, err)
	}

	lm.strings[language] = strings
	log.Printf("Loaded localization file: %s with %d strings", language, len(strings))
	return nil
}

// resolvedBrandLocked returns the display brand; caller must hold lm.mu (RLock or Lock).
func (lm *LocalizationManager) resolvedBrandLocked() string {
	if lm.brandDisplayName != "" {
		return lm.brandDisplayName
	}
	return DefaultBrandDisplayName
}

func (lm *LocalizationManager) applyBrandLocked(s string) string {
	return strings.ReplaceAll(s, brandPlaceholder, lm.resolvedBrandLocked())
}

// SetBrandDisplayName sets the whitelabel name substituted for {brand} in all locales.
// Empty or whitespace-only values leave the default (OneBusAway).
func (lm *LocalizationManager) SetBrandDisplayName(name string) {
	name = strings.TrimSpace(name)
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if name == "" {
		lm.brandDisplayName = ""
		return
	}
	lm.brandDisplayName = name
}

// BrandDisplayName returns the configured product display name (default OneBusAway if unset).
func (lm *LocalizationManager) BrandDisplayName() string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.resolvedBrandLocked()
}

// GetString retrieves a localized string with fallback logic
func (lm *LocalizationManager) GetString(key, language string, params ...interface{}) string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Try requested language first
	if langStrings, exists := lm.strings[language]; exists {
		if str, exists := langStrings[key]; exists {
			str = lm.applyBrandLocked(str)
			if len(params) > 0 {
				return fmt.Sprintf(str, params...)
			}
			return str
		}
	}

	// Fallback to default language
	if defaultStrings, exists := lm.strings[lm.defaultLanguage]; exists {
		if str, exists := defaultStrings[key]; exists {
			if language != lm.defaultLanguage {
				log.Printf("Using fallback language for key: %s, requested: %s", key, language)
			}
			str = lm.applyBrandLocked(str)
			if len(params) > 0 {
				return fmt.Sprintf(str, params...)
			}
			return str
		}
	}

	// Last resort: return the key itself
	log.Printf("Missing translation for key: %s", key)
	return key
}

// IsSupported checks if a language is supported
func (lm *LocalizationManager) IsSupported(language string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	for _, lang := range lm.supportedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// GetSupportedLanguages returns the list of supported languages
func (lm *LocalizationManager) GetSupportedLanguages() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make([]string, len(lm.supportedLanguages))
	copy(result, lm.supportedLanguages)
	return result
}

// GetPrimaryLanguage returns the primary (default) language
func (lm *LocalizationManager) GetPrimaryLanguage() string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.defaultLanguage
}

// GetLanguageCount returns the number of supported languages
func (lm *LocalizationManager) GetLanguageCount() int {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return len(lm.supportedLanguages)
}

// GetLanguageDisplayName returns the native display name for a language
func (lm *LocalizationManager) GetLanguageDisplayName(language string) string {
	displayNames := map[string]string{
		"en-US": "English",
		"es-US": "Español",
		"zh-CN": "中文",
		"fr-US": "Français",
		"de-US": "Deutsch",
		"pl-PL": "Polski",
		"pt-US": "Português",
		"ru-US": "Русский",
		"ar-US": "العربية",
		"ko-US": "한국어",
	}

	if displayName, exists := displayNames[language]; exists {
		return displayName
	}
	return language // Fallback to language code
}
