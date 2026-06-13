package health

import (
	"testing"
	"time"

	"oba-twilio/client"
	"oba-twilio/handlers/common"
	"oba-twilio/localization"
)

func TestSystemHealthChecker(t *testing.T) {
	checker := &SystemHealthChecker{}

	if checker.Name() != "system" {
		t.Errorf("Expected name to be 'system', got %s", checker.Name())
	}

	result := checker.Check()

	if result.Name != "system" {
		t.Errorf("Expected result name to be 'system', got %s", result.Name)
	}

	if result.Status == "" {
		t.Error("Expected status to be set")
	}

	if result.Duration < 0 {
		t.Error("Expected duration to be non-negative")
	}

	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if result.Metadata == nil {
		t.Error("Expected metadata to be present")
	}

	// Check for expected metadata keys
	expectedKeys := []string{"goroutines", "memory_alloc", "memory_sys", "gc_cycles"}
	for _, key := range expectedKeys {
		if _, exists := result.Metadata[key]; !exists {
			t.Errorf("Expected metadata key %s to be present", key)
		}
	}
}

func TestOneBusAwayHealthChecker_NilClient(t *testing.T) {
	checker := NewOneBusAwayHealthChecker(nil)

	if checker.Name() != "onebusaway_api" {
		t.Errorf("Expected name to be 'onebusaway_api', got %s", checker.Name())
	}

	result := checker.Check()

	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status to be unhealthy with nil client, got %s", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message for nil client")
	}
}

func TestOneBusAwayHealthChecker_WithClient(t *testing.T) {
	// Create a client for testing
	client := client.NewOneBusAwayClient("https://api.pugetsound.onebusaway.org", "test-key")
	checker := NewOneBusAwayHealthChecker(client)

	result := checker.Check()

	if result.Name != "onebusaway_api" {
		t.Errorf("Expected result name to be 'onebusaway_api', got %s", result.Name)
	}

	// Should have metrics metadata
	if result.Metadata == nil {
		t.Error("Expected metadata to be present")
	}

	expectedKeys := []string{"cache_hits", "cache_misses", "api_calls", "api_errors"}
	for _, key := range expectedKeys {
		if _, exists := result.Metadata[key]; !exists {
			t.Errorf("Expected metadata key %s to be present", key)
		}
	}
}

func TestSessionStoreHealthChecker_NilStore(t *testing.T) {
	checker := NewSessionStoreHealthChecker(nil)

	if checker.Name() != "session_store" {
		t.Errorf("Expected name to be 'session_store', got %s", checker.Name())
	}

	result := checker.Check()

	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status to be unhealthy with nil store, got %s", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message for nil store")
	}
}

func TestSessionStoreHealthChecker_WithStore(t *testing.T) {
	// Create a session store for testing
	store := common.NewImprovedSessionStore()
	defer store.Close()

	checker := NewSessionStoreHealthChecker(store)

	result := checker.Check()

	if result.Name != "session_store" {
		t.Errorf("Expected result name to be 'session_store', got %s", result.Name)
	}

	// Should have metrics metadata
	if result.Metadata == nil {
		t.Error("Expected metadata to be present")
	}

	expectedKeys := []string{"total_sessions", "cache_hits", "cache_misses", "memory_usage_bytes"}
	for _, key := range expectedKeys {
		if _, exists := result.Metadata[key]; !exists {
			t.Errorf("Expected metadata key %s to be present", key)
		}
	}
}

func TestLocalizationHealthChecker_NilManager(t *testing.T) {
	checker := NewLocalizationHealthChecker(nil)

	if checker.Name() != "localization" {
		t.Errorf("Expected name to be 'localization', got %s", checker.Name())
	}

	result := checker.Check()

	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status to be unhealthy with nil manager, got %s", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message for nil manager")
	}
}

func TestLocalizationHealthChecker_WithManager(t *testing.T) {
	// Create a localization manager for testing without relying on disk files
	manager := localization.NewTestManager()

	checker := NewLocalizationHealthChecker(manager)

	result := checker.Check()

	if result.Name != "localization" {
		t.Errorf("Expected result name to be 'localization', got %s", result.Name)
	}

	// Should have metadata
	if result.Metadata == nil {
		t.Error("Expected metadata to be present")
	}

	expectedKeys := []string{"supported_languages", "primary_language", "language_count"}
	for _, key := range expectedKeys {
		if _, exists := result.Metadata[key]; !exists {
			t.Errorf("Expected metadata key %s to be present", key)
		}
	}
}

func TestHTTPServerHealthChecker(t *testing.T) {
	checker := NewHTTPServerHealthChecker("8080")

	if checker.Name() != "http_server" {
		t.Errorf("Expected name to be 'http_server', got %s", checker.Name())
	}

	result := checker.Check()

	if result.Name != "http_server" {
		t.Errorf("Expected result name to be 'http_server', got %s", result.Name)
	}

	// Should have metadata with port
	if result.Metadata == nil {
		t.Error("Expected metadata to be present")
	}

	if port, exists := result.Metadata["port"]; !exists {
		t.Error("Expected port in metadata")
	} else if port != "8080" {
		t.Errorf("Expected port to be '8080', got %s", port)
	}

	// Note: This test will likely fail because the server isn't running
	// In a real scenario, you might start a test server or mock the HTTP client
}

func TestHealthCheckerPerformance(t *testing.T) {
	checker := &SystemHealthChecker{}

	// Test that health checks are fast
	start := time.Now()
	for i := 0; i < 100; i++ {
		checker.Check()
	}
	duration := time.Since(start)

	// 100 system health checks should complete in under 1 second
	if duration > time.Second {
		t.Errorf("Expected 100 health checks to complete in under 1s, took %v", duration)
	}

	// Average check should be under 10ms
	avgDuration := duration / 100
	if avgDuration > 10*time.Millisecond {
		t.Errorf("Expected average check duration to be under 10ms, got %v", avgDuration)
	}
}

func TestHealthCheckerStability(t *testing.T) {
	checker := &SystemHealthChecker{}

	// Test that health checks are consistent
	results := make([]CheckResult, 10)
	for i := 0; i < 10; i++ {
		results[i] = checker.Check()
		time.Sleep(10 * time.Millisecond) // Small delay between checks
	}

	// All results should have the same name
	for i, result := range results {
		if result.Name != "system" {
			t.Errorf("Check %d: expected name 'system', got %s", i, result.Name)
		}

		if result.Status == "" {
			t.Errorf("Check %d: expected status to be set", i)
		}

		if result.Duration < 0 {
			t.Errorf("Check %d: expected non-negative duration", i)
		}

		if result.Timestamp.IsZero() {
			t.Errorf("Check %d: expected timestamp to be set", i)
		}
	}
}

func BenchmarkSystemHealthChecker(b *testing.B) {
	checker := &SystemHealthChecker{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check()
	}
}

func BenchmarkOneBusAwayHealthChecker(b *testing.B) {
	client := client.NewOneBusAwayClient("https://api.pugetsound.onebusaway.org", "test-key")
	checker := NewOneBusAwayHealthChecker(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check()
	}
}

func BenchmarkSessionStoreHealthChecker(b *testing.B) {
	store := common.NewImprovedSessionStore()
	defer store.Close()

	checker := NewSessionStoreHealthChecker(store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check()
	}
}

// Test helper functions

func createTestOneBusAwayClient() *client.OneBusAwayClient {
	return client.NewOneBusAwayClient("https://api.pugetsound.onebusaway.org", "test-key")
}

func createTestSessionStore() *common.ImprovedSessionStore {
	return common.NewImprovedSessionStore()
}

func createTestLocalizationManager() *localization.LocalizationManager {
	return localization.NewTestManager()
}

// Test that checkers implement the HealthChecker interface
func TestHealthCheckerInterface(t *testing.T) {
	var checkers []HealthChecker

	checkers = append(checkers, &SystemHealthChecker{})
	checkers = append(checkers, NewOneBusAwayHealthChecker(createTestOneBusAwayClient()))
	checkers = append(checkers, NewSessionStoreHealthChecker(createTestSessionStore()))
	checkers = append(checkers, NewHTTPServerHealthChecker("8080"))

	if manager := createTestLocalizationManager(); manager != nil {
		checkers = append(checkers, NewLocalizationHealthChecker(manager))
	}

	for i, checker := range checkers {
		if checker.Name() == "" {
			t.Errorf("Checker %d: expected non-empty name", i)
		}

		result := checker.Check()
		if result.Name == "" {
			t.Errorf("Checker %d: expected non-empty result name", i)
		}

		if result.Status == "" {
			t.Errorf("Checker %d: expected non-empty status", i)
		}
	}
}
