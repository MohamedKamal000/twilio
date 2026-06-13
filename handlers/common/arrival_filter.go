package common

import "oba-twilio/models"

// ArrivalFilterConfig controls filtering of suspicious realtime predictions.
type ArrivalFilterConfig struct {
	Enabled               bool
	MaxPredictedEarlyMins int
	FallbackToUnfiltered  bool
}

// FilterArrivals removes arrivals where predicted time is unrealistically earlier than scheduled.
// Returns filtered arrivals, number of excluded arrivals, and whether fallback was used.
func FilterArrivals(arrivals []models.Arrival, cfg ArrivalFilterConfig) ([]models.Arrival, int, bool) {
	if !cfg.Enabled || len(arrivals) == 0 {
		return arrivals, 0, false
	}

	threshold := cfg.MaxPredictedEarlyMins
	if threshold <= 0 {
		threshold = 15
	}
	thresholdMs := int64(threshold) * 60 * 1000

	filtered := make([]models.Arrival, 0, len(arrivals))
	excluded := 0

	for _, a := range arrivals {
		// Filter only when both timestamps are available and predicted is much earlier.
		if a.PredictedArrivalTime > 0 && a.ScheduledArrivalTime > 0 {
			if a.ScheduledArrivalTime-a.PredictedArrivalTime >= thresholdMs {
				excluded++
				continue
			}
		}
		filtered = append(filtered, a)
	}

	if cfg.FallbackToUnfiltered && len(filtered) == 0 && len(arrivals) > 0 {
		return arrivals, excluded, true
	}

	return filtered, excluded, false
}
