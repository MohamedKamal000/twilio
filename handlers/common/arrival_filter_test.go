package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"oba-twilio/models"
)

func TestFilterArrivals_Disabled_NoChange(t *testing.T) {
	input := []models.Arrival{
		{PredictedArrivalTime: 1000, ScheduledArrivalTime: 2000},
	}

	got, excluded, fallback := FilterArrivals(input, ArrivalFilterConfig{Enabled: false})
	assert.Equal(t, input, got)
	assert.Equal(t, 0, excluded)
	assert.False(t, fallback)
}

func TestFilterArrivals_ExcludesPredictedTooEarly(t *testing.T) {
	// scheduled-predicted = 20 min -> excluded for 15-minute threshold
	bad := models.Arrival{PredictedArrivalTime: 10 * 60 * 1000, ScheduledArrivalTime: 30 * 60 * 1000}
	good := models.Arrival{PredictedArrivalTime: 10 * 60 * 1000, ScheduledArrivalTime: 20 * 60 * 1000}

	got, excluded, fallback := FilterArrivals(
		[]models.Arrival{bad, good},
		ArrivalFilterConfig{Enabled: true, MaxPredictedEarlyMins: 15, FallbackToUnfiltered: true},
	)

	assert.Equal(t, []models.Arrival{good}, got)
	assert.Equal(t, 1, excluded)
	assert.False(t, fallback)
}

func TestFilterArrivals_FallbackToUnfiltered(t *testing.T) {
	bad := models.Arrival{PredictedArrivalTime: 10 * 60 * 1000, ScheduledArrivalTime: 50 * 60 * 1000}

	got, excluded, fallback := FilterArrivals(
		[]models.Arrival{bad},
		ArrivalFilterConfig{Enabled: true, MaxPredictedEarlyMins: 15, FallbackToUnfiltered: true},
	)

	assert.Len(t, got, 1)
	assert.Equal(t, 1, excluded)
	assert.True(t, fallback)
}
