package health

import (
	"strings"
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	if collector == nil {
		t.Fatal("Expected metrics collector to be created")
	}

	if collector.systemStartTime.IsZero() {
		t.Error("Expected system start time to be set")
	}
}

func TestUpdateMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	results := map[string]CheckResult{
		"test1": {
			Status:  StatusHealthy,
			Message: "OK",
		},
		"test2": {
			Status:  StatusDegraded,
			Message: "Slow",
		},
		"test3": {
			Status: StatusUnhealthy,
			Error:  "Failed",
		},
	}

	duration := 100 * time.Millisecond
	collector.UpdateMetrics(results, duration)

	metrics := collector.GetMetrics()

	if metrics.HealthChecksTotal != 1 {
		t.Errorf("Expected health checks total to be 1, got %d", metrics.HealthChecksTotal)
	}

	if metrics.HealthChecksFailuresTotal != 2 { // degraded + unhealthy
		t.Errorf("Expected health check failures to be 2, got %d", metrics.HealthChecksFailuresTotal)
	}

	if metrics.HealthChecksDurationSeconds <= 0 {
		t.Error("Expected health check duration to be positive")
	}
}

func TestGetMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	metrics := collector.GetMetrics()

	if metrics.SystemUptimeSeconds < 0 {
		t.Error("Expected system uptime to be non-negative")
	}

	if metrics.SystemGoroutinesTotal <= 0 {
		t.Error("Expected goroutines count to be positive")
	}

	if metrics.SystemMemoryAllocBytes == 0 {
		t.Error("Expected memory allocation to be non-zero")
	}

	if metrics.SystemMemoryUsagePercent < 0 || metrics.SystemMemoryUsagePercent > 100 {
		t.Errorf("Expected memory usage percent to be between 0-100, got %.2f", metrics.SystemMemoryUsagePercent)
	}
}

func TestPrometheusMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Update some metrics
	results := map[string]CheckResult{
		"test": {Status: StatusHealthy},
	}
	collector.UpdateMetrics(results, 50*time.Millisecond)

	prometheusText := collector.GetPrometheusMetrics()

	if prometheusText == "" {
		t.Error("Expected non-empty Prometheus metrics")
	}

	// Check for expected metric names
	expectedMetrics := []string{
		"health_checks_total",
		"health_checks_duration_seconds",
		"health_checks_failures_total",
		"system_goroutines_total",
		"system_memory_alloc_bytes",
		"system_memory_usage_percent",
		"system_uptime_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(prometheusText, metric) {
			t.Errorf("Expected Prometheus metrics to contain %s", metric)
		}
	}

	// Check for HELP comments
	if !strings.Contains(prometheusText, "# HELP") {
		t.Error("Expected Prometheus HELP comments")
	}

	// Check for TYPE comments
	if !strings.Contains(prometheusText, "# TYPE") {
		t.Error("Expected Prometheus TYPE comments")
	}
}

func TestMetricsCollectorConcurrency(t *testing.T) {
	collector := NewMetricsCollector()

	// Test concurrent updates
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			results := map[string]CheckResult{
				"test": {Status: StatusHealthy},
			}
			collector.UpdateMetrics(results, 10*time.Millisecond)
			done <- true
		}()
	}

	// Wait for all updates to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := collector.GetMetrics()
	if metrics.HealthChecksTotal != 10 {
		t.Errorf("Expected 10 health checks, got %d", metrics.HealthChecksTotal)
	}
}

func TestIncrementDependencyRequests(t *testing.T) {
	collector := NewMetricsCollector()

	collector.IncrementDependencyRequests()
	collector.IncrementDependencyRequests()

	metrics := collector.GetMetrics()
	if metrics.DependencyRequestsTotal != 2 {
		t.Errorf("Expected 2 dependency requests, got %d", metrics.DependencyRequestsTotal)
	}
}

func TestIncrementDependencyErrors(t *testing.T) {
	collector := NewMetricsCollector()

	collector.IncrementDependencyErrors()
	collector.IncrementDependencyErrors()
	collector.IncrementDependencyErrors()

	metrics := collector.GetMetrics()
	if metrics.DependencyErrorsTotal != 3 {
		t.Errorf("Expected 3 dependency errors, got %d", metrics.DependencyErrorsTotal)
	}
}

func TestUpdateDependencyResponseTime(t *testing.T) {
	collector := NewMetricsCollector()

	duration := 150 * time.Millisecond
	collector.UpdateDependencyResponseTime(duration)

	metrics := collector.GetMetrics()
	expectedSeconds := duration.Seconds()
	if metrics.DependencyResponseTimeSeconds != expectedSeconds {
		t.Errorf("Expected dependency response time %.6f, got %.6f",
			expectedSeconds, metrics.DependencyResponseTimeSeconds)
	}
}

func TestUpdateSessionStoreMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	collector.UpdateSessionStoreMetrics(100, 80, 20)

	metrics := collector.GetMetrics()
	if metrics.SessionStoreSessionsTotal != 100 {
		t.Errorf("Expected 100 sessions, got %d", metrics.SessionStoreSessionsTotal)
	}

	if metrics.SessionStoreCacheHitsTotal != 80 {
		t.Errorf("Expected 80 cache hits, got %d", metrics.SessionStoreCacheHitsTotal)
	}

	if metrics.SessionStoreCacheMissesTotal != 20 {
		t.Errorf("Expected 20 cache misses, got %d", metrics.SessionStoreCacheMissesTotal)
	}
}

func TestUpdateCircuitBreakerState(t *testing.T) {
	collector := NewMetricsCollector()

	// Test closed state
	collector.UpdateCircuitBreakerState(0)
	metrics := collector.GetMetrics()
	if metrics.CircuitBreakerState != 0 {
		t.Errorf("Expected circuit breaker state 0, got %d", metrics.CircuitBreakerState)
	}

	// Test open state
	collector.UpdateCircuitBreakerState(1)
	metrics = collector.GetMetrics()
	if metrics.CircuitBreakerState != 1 {
		t.Errorf("Expected circuit breaker state 1, got %d", metrics.CircuitBreakerState)
	}

	// Test half-open state
	collector.UpdateCircuitBreakerState(2)
	metrics = collector.GetMetrics()
	if metrics.CircuitBreakerState != 2 {
		t.Errorf("Expected circuit breaker state 2, got %d", metrics.CircuitBreakerState)
	}
}

func TestFormatPrometheusMetrics(t *testing.T) {
	metrics := MetricsInfo{
		HealthChecksTotal:           100,
		HealthChecksDurationSeconds: 0.150,
		HealthChecksFailuresTotal:   5,
		SystemGoroutinesTotal:       50,
		SystemMemoryAllocBytes:      1024000,
		SystemMemoryUsagePercent:    75.5,
		SystemUptimeSeconds:         3600,
	}

	prometheusText := formatPrometheusMetrics(metrics)

	// Check metric values are formatted correctly
	if !strings.Contains(prometheusText, "health_checks_total 100.000000") {
		t.Error("Expected health_checks_total to be formatted correctly")
	}

	if !strings.Contains(prometheusText, "system_memory_usage_percent 75.500000") {
		t.Error("Expected memory usage percent to be formatted correctly")
	}

	if !strings.Contains(prometheusText, "system_uptime_seconds 3600.000000") {
		t.Error("Expected uptime to be formatted correctly")
	}
}

func TestPrometheusMetricsWithZeroValues(t *testing.T) {
	metrics := MetricsInfo{
		HealthChecksTotal:           0,
		HealthChecksDurationSeconds: 0,
		HealthChecksFailuresTotal:   0,
	}

	prometheusText := formatPrometheusMetrics(metrics)

	// Should still include metrics even with zero values
	if !strings.Contains(prometheusText, "health_checks_total 0.000000") {
		t.Error("Expected zero value metrics to be included")
	}
}

func TestPrometheusMetricsConditionalInclusion(t *testing.T) {
	// Test that dependency metrics are only included when they have values
	metrics := MetricsInfo{
		HealthChecksTotal:         1,
		DependencyRequestsTotal:   0,  // Should not be included
		DependencyErrorsTotal:     0,  // Should not be included
		SessionStoreSessionsTotal: 10, // Should be included
	}

	prometheusText := formatPrometheusMetrics(metrics)

	// Should not include dependency metrics with zero values
	if strings.Contains(prometheusText, "dependency_requests_total") {
		t.Error("Expected dependency_requests_total to be excluded when zero")
	}

	if strings.Contains(prometheusText, "dependency_errors_total") {
		t.Error("Expected dependency_errors_total to be excluded when zero")
	}

	// Should include session store metrics
	if !strings.Contains(prometheusText, "session_store_sessions_total") {
		t.Error("Expected session_store_sessions_total to be included")
	}
}

func BenchmarkUpdateMetrics(b *testing.B) {
	collector := NewMetricsCollector()
	results := map[string]CheckResult{
		"test1": {Status: StatusHealthy},
		"test2": {Status: StatusDegraded},
	}
	duration := 50 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.UpdateMetrics(results, duration)
	}
}

func BenchmarkGetMetrics(b *testing.B) {
	collector := NewMetricsCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics()
	}
}

func BenchmarkGetPrometheusMetrics(b *testing.B) {
	collector := NewMetricsCollector()
	results := map[string]CheckResult{
		"test": {Status: StatusHealthy},
	}
	collector.UpdateMetrics(results, 50*time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetPrometheusMetrics()
	}
}

func TestMetricsThreadSafety(t *testing.T) {
	collector := NewMetricsCollector()

	// Test that concurrent operations don't cause race conditions
	done := make(chan bool, 20)

	// Concurrent updates
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				collector.IncrementDependencyRequests()
				collector.IncrementDependencyErrors()
				collector.UpdateDependencyResponseTime(time.Duration(j) * time.Millisecond)
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				collector.GetMetrics()
				collector.GetPrometheusMetrics()
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	metrics := collector.GetMetrics()
	if metrics.DependencyRequestsTotal != 1000 { // 10 goroutines * 100 increments
		t.Errorf("Expected 1000 dependency requests, got %d", metrics.DependencyRequestsTotal)
	}
}

func TestMetricsAccuracy(t *testing.T) {
	collector := NewMetricsCollector()

	// Test that metrics calculations are accurate
	results1 := map[string]CheckResult{
		"test1": {Status: StatusHealthy},
		"test2": {Status: StatusDegraded},
	}
	duration1 := 100 * time.Millisecond

	results2 := map[string]CheckResult{
		"test1": {Status: StatusHealthy},
		"test2": {Status: StatusUnhealthy},
	}
	duration2 := 200 * time.Millisecond

	collector.UpdateMetrics(results1, duration1)
	collector.UpdateMetrics(results2, duration2)

	metrics := collector.GetMetrics()

	// Should have 2 total checks
	if metrics.HealthChecksTotal != 2 {
		t.Errorf("Expected 2 total checks, got %d", metrics.HealthChecksTotal)
	}

	// Should have 2 failures (1 degraded + 1 unhealthy)
	if metrics.HealthChecksFailuresTotal != 2 {
		t.Errorf("Expected 2 failures, got %d", metrics.HealthChecksFailuresTotal)
	}

	// Average duration should be 150ms
	expectedAvg := 0.150 // (100ms + 200ms) / 2 = 150ms = 0.150s
	if metrics.HealthChecksDurationSeconds != expectedAvg {
		t.Errorf("Expected average duration %.3f, got %.3f",
			expectedAvg, metrics.HealthChecksDurationSeconds)
	}
}
