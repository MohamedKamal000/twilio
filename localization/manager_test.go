package localization

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// Create a temporary directory for test locales
	tempDir := t.TempDir()

	// Create locales directory
	localesDir := filepath.Join(tempDir, "locales")
	err := os.MkdirAll(localesDir, 0755)
	require.NoError(t, err)

	// Create test locale file
	enUSContent := `{
		"voice.welcome": "Welcome to {brand}",
		"sms.no_stops_found": "No stops found with ID %s"
	}`

	err = os.WriteFile(filepath.Join(localesDir, "en-US.json"), []byte(enUSContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Test creating manager
	manager, err := NewManager("en-US")
	require.NoError(t, err)
	assert.NotNil(t, manager)

	// Test string retrieval
	welcome := manager.GetString("voice.welcome", "en-US")
	assert.Equal(t, "Welcome to OneBusAway", welcome)

	// Test string with parameters
	noStops := manager.GetString("sms.no_stops_found", "en-US", "12345")
	assert.Equal(t, "No stops found with ID 12345", noStops)

	// Test fallback for missing key
	missing := manager.GetString("missing.key", "en-US")
	assert.Equal(t, "missing.key", missing)

	manager.SetBrandDisplayName("Acme Transit")
	assert.Equal(t, "Welcome to Acme Transit", manager.GetString("voice.welcome", "en-US"))
}

func TestManagerMultipleLanguages(t *testing.T) {
	// Create a temporary directory for test locales
	tempDir := t.TempDir()

	// Create locales directory
	localesDir := filepath.Join(tempDir, "locales")
	err := os.MkdirAll(localesDir, 0755)
	require.NoError(t, err)

	// Create test locale files
	enUSContent := `{"voice.welcome": "Welcome to {brand}"}`
	esUSContent := `{"voice.welcome": "Bienvenido a {brand}"}`

	err = os.WriteFile(filepath.Join(localesDir, "en-US.json"), []byte(enUSContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(localesDir, "es-US.json"), []byte(esUSContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Test creating manager with multiple languages
	manager, err := NewManager("en-US,es-US")
	require.NoError(t, err)

	// Test English
	welcome := manager.GetString("voice.welcome", "en-US")
	assert.Equal(t, "Welcome to OneBusAway", welcome)

	// Test Spanish
	bienvenido := manager.GetString("voice.welcome", "es-US")
	assert.Equal(t, "Bienvenido a OneBusAway", bienvenido)

	// Test language count
	assert.Equal(t, 2, manager.GetLanguageCount())

	// Test supported languages
	assert.True(t, manager.IsSupported("en-US"))
	assert.True(t, manager.IsSupported("es-US"))
	assert.False(t, manager.IsSupported("fr-US"))
}
