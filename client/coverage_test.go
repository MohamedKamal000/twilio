package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"oba-twilio/models"
)

func TestInitializeCoverage_Success(t *testing.T) {
	mockResponse := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool                       `json:"limitExceeded"`
			List          []models.AgencyCoverageRow `json:"list"`
		}{
			LimitExceeded: false,
			List: []models.AgencyCoverageRow{
				{
					AgencyID: "metro",
					Lat:      47.6062,
					LatSpan:  0.5,
					Lon:      -122.3321,
					LonSpan:  0.8,
				},
				{
					AgencyID: "soundtransit",
					Lat:      47.5,
					LatSpan:  0.3,
					Lon:      -122.2,
					LonSpan:  0.4,
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "agencies-with-coverage")
		assert.Equal(t, "test-key", r.URL.Query().Get("key"))

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.NoError(t, err)

	coverage := client.GetCoverageArea()
	assert.NotNil(t, coverage)

	// Center should be the midpoint of all agencies
	// Agency 1: lat 47.6062±0.25, lon -122.3321±0.4 -> bounds: lat[47.3562, 47.8562], lon[-122.7321, -121.9321]
	// Agency 2: lat 47.5±0.15, lon -122.2±0.2 -> bounds: lat[47.35, 47.65], lon[-122.4, -122.0]
	// Combined bounds: lat[47.35, 47.8562], lon[-122.7321, -121.9321]
	// Center: lat=(47.35+47.8562)/2=47.6031, lon=(-122.7321+-121.9321)/2=-122.3321
	assert.InDelta(t, 47.6031, coverage.CenterLat, 0.01)
	assert.InDelta(t, -122.3321, coverage.CenterLon, 0.01)
	assert.Greater(t, coverage.Radius, 5000.0)
	assert.Less(t, coverage.Radius, 100000.0)
}

func TestInitializeCoverage_SingleAgency(t *testing.T) {
	mockResponse := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool                       `json:"limitExceeded"`
			List          []models.AgencyCoverageRow `json:"list"`
		}{
			List: []models.AgencyCoverageRow{
				{
					AgencyID: "unitrans",
					Lat:      38.5553,
					LatSpan:  0.0356,
					Lon:      -121.7360,
					LonSpan:  0.1050,
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.NoError(t, err)

	coverage := client.GetCoverageArea()
	assert.NotNil(t, coverage)

	assert.InDelta(t, 38.5553, coverage.CenterLat, 0.01)
	assert.InDelta(t, -121.7360, coverage.CenterLon, 0.01)
	assert.GreaterOrEqual(t, coverage.Radius, 5000.0)
}

func TestInitializeCoverage_NoAgencies(t *testing.T) {
	mockResponse := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool                       `json:"limitExceeded"`
			List          []models.AgencyCoverageRow `json:"list"`
		}{
			List: []models.AgencyCoverageRow{},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no coverage areas found")
}

func TestInitializeCoverage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestCalculateRadius(t *testing.T) {
	client := NewOneBusAwayClient("test", "test")

	tests := []struct {
		name      string
		latSpan   float64
		lonSpan   float64
		centerLat float64
		minRadius float64
		maxRadius float64
	}{
		{
			name:      "Small area",
			latSpan:   0.01,
			lonSpan:   0.01,
			centerLat: 47.6,
			minRadius: 5000,
			maxRadius: 10000,
		},
		{
			name:      "Large area",
			latSpan:   5.0,
			lonSpan:   5.0,
			centerLat: 47.6,
			minRadius: 100000,
			maxRadius: 100000,
		},
		{
			name:      "Equatorial area",
			latSpan:   1.0,
			lonSpan:   1.0,
			centerLat: 0.0,
			minRadius: 50000,
			maxRadius: 120000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			radius := client.calculateRadius(tt.latSpan, tt.lonSpan, tt.centerLat)
			assert.GreaterOrEqual(t, radius, tt.minRadius)
			assert.LessOrEqual(t, radius, tt.maxRadius)
		})
	}
}

func TestSearchStops_WithCoverage(t *testing.T) {
	mockStopData := models.StopData{
		Data: struct {
			List []struct {
				ID   string  `json:"id"`
				Name string  `json:"name"`
				Lat  float64 `json:"lat"`
				Lon  float64 `json:"lon"`
			} `json:"list"`
		}{
			List: []struct {
				ID   string  `json:"id"`
				Name string  `json:"name"`
				Lat  float64 `json:"lat"`
				Lon  float64 `json:"lon"`
			}{
				{
					ID:   "1_12345",
					Name: "Test Stop",
					Lat:  47.6062,
					Lon:  -122.3321,
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/where/agencies-with-coverage.json":
			mockCoverage := models.AgenciesWithCoverageResponse{
				Data: struct {
					LimitExceeded bool                       `json:"limitExceeded"`
					List          []models.AgencyCoverageRow `json:"list"`
				}{
					List: []models.AgencyCoverageRow{
						{
							AgencyID: "test",
							Lat:      47.6062,
							LatSpan:  0.5,
							Lon:      -122.3321,
							LonSpan:  0.8,
						},
					},
				},
				Code: 200,
				Text: "OK",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockCoverage); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		case "/api/where/stops-for-location.json":
			assert.Equal(t, "test search", r.URL.Query().Get("query"))
			assert.NotEmpty(t, r.URL.Query().Get("lat"))
			assert.NotEmpty(t, r.URL.Query().Get("lon"))
			assert.NotEmpty(t, r.URL.Query().Get("radius"))

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockStopData); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.NoError(t, err)

	stops, err := client.SearchStops("test search")
	assert.NoError(t, err)
	assert.Len(t, stops, 1)
	assert.Equal(t, "1_12345", stops[0].ID)
	assert.Equal(t, "Test Stop", stops[0].Name)
}

func TestSearchStops_WithoutCoverage(t *testing.T) {
	client := NewOneBusAwayClient("test", "test-key")

	_, err := client.SearchStops("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "coverage area not initialized")
}

func TestCalculateCoverageArea_EmptyAgencies(t *testing.T) {
	client := NewOneBusAwayClient("test", "test")

	agencies := []models.AgencyCoverageRow{}

	coverage := client.calculateCoverageArea(agencies)

	assert.Equal(t, 47.6062, coverage.CenterLat)
	assert.Equal(t, -122.3321, coverage.CenterLon)
	assert.Equal(t, 25000.0, coverage.Radius)
}

func TestFindAllMatchingStops_SingleMatch(t *testing.T) {
	mockStopResp := struct {
		Data struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		} `json:"data"`
		Code int    `json:"code"`
		Text string `json:"text"`
	}{
		Data: struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		}{
			Entry: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "1_12345",
				Name: "Test Stop",
			},
			References: struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			}{
				Agencies: []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					{ID: "1", Name: "King County Metro"},
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	mockCoverageResp := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool                       `json:"limitExceeded"`
			List          []models.AgencyCoverageRow `json:"list"`
		}{
			LimitExceeded: false,
			List: []models.AgencyCoverageRow{
				{AgencyID: "1", Lat: 47.6062, LatSpan: 0.5, Lon: -122.3321, LonSpan: 0.8},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/where/agencies-with-coverage.json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockCoverageResp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		} else if strings.Contains(r.URL.Path, "/api/where/stop/1_12345.json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockStopResp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.NoError(t, err)

	stops, err := client.FindAllMatchingStops("12345")
	assert.NoError(t, err)
	assert.Len(t, stops, 1)
	assert.Equal(t, "1_12345", stops[0].FullStopID)
	assert.Equal(t, "King County Metro", stops[0].AgencyName)
	assert.Equal(t, "Test Stop", stops[0].StopName)
}

func TestFindAllMatchingStops_MultipleMatches(t *testing.T) {
	mockStopResp1 := struct {
		Data struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		} `json:"data"`
		Code int    `json:"code"`
		Text string `json:"text"`
	}{
		Data: struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		}{
			Entry: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "1_12345",
				Name: "Pine St & 3rd Ave",
			},
			References: struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			}{
				Agencies: []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					{ID: "1", Name: "King County Metro"},
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	mockStopResp2 := struct {
		Data struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		} `json:"data"`
		Code int    `json:"code"`
		Text string `json:"text"`
	}{
		Data: struct {
			Entry struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"entry"`
			References struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			} `json:"references"`
		}{
			Entry: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "40_12345",
				Name: "University Street Station",
			},
			References: struct {
				Agencies []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"agencies"`
			}{
				Agencies: []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					{ID: "40", Name: "Sound Transit"},
				},
			},
		},
		Code: 200,
		Text: "OK",
	}

	mockCoverageResp := models.AgenciesWithCoverageResponse{
		Data: struct {
			LimitExceeded bool                       `json:"limitExceeded"`
			List          []models.AgencyCoverageRow `json:"list"`
		}{
			LimitExceeded: false,
			List: []models.AgencyCoverageRow{
				{AgencyID: "1", Lat: 47.6062, LatSpan: 0.5, Lon: -122.3321, LonSpan: 0.8},
				{AgencyID: "40", Lat: 47.5, LatSpan: 0.3, Lon: -122.2, LonSpan: 0.4},
			},
		},
		Code: 200,
		Text: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/where/agencies-with-coverage.json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockCoverageResp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		} else if strings.Contains(r.URL.Path, "/api/where/stop/1_12345.json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockStopResp1); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		} else if strings.Contains(r.URL.Path, "/api/where/stop/40_12345.json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockStopResp2); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewOneBusAwayClient(server.URL, "test-key")

	err := client.InitializeCoverage()
	assert.NoError(t, err)

	stops, err := client.FindAllMatchingStops("12345")
	assert.NoError(t, err)
	assert.Len(t, stops, 2)

	// Results may come back in different order due to concurrent requests
	foundMetro := false
	foundSoundTransit := false

	for _, stop := range stops {
		switch stop.FullStopID {
		case "1_12345":
			assert.Equal(t, "King County Metro", stop.AgencyName)
			assert.Equal(t, "Pine St & 3rd Ave", stop.StopName)
			foundMetro = true
		case "40_12345":
			assert.Equal(t, "Sound Transit", stop.AgencyName)
			assert.Equal(t, "University Street Station", stop.StopName)
			foundSoundTransit = true
		}
	}

	assert.True(t, foundMetro, "Should find Metro stop")
	assert.True(t, foundSoundTransit, "Should find Sound Transit stop")
}

func TestGetAgencyNameFromID(t *testing.T) {
	client := NewOneBusAwayClient("test", "test")

	tests := []struct {
		stopID   string
		agencies []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		expected string
	}{
		{
			stopID: "1_12345",
			agencies: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				{ID: "1", Name: "King County Metro"},
			},
			expected: "King County Metro",
		},
		{
			stopID: "40_12345",
			agencies: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{},
			expected: "Sound Transit",
		},
		{
			stopID: "invalid",
			agencies: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{},
			expected: "Unknown",
		},
		{
			stopID: "99_12345",
			agencies: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{},
			expected: "Agency 99",
		},
	}

	for _, tt := range tests {
		result := client.getAgencyNameFromID(tt.stopID, tt.agencies)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFindAllMatchingStops_Timeout(t *testing.T) {
	// Create a server that delays responses beyond the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the HTTP client timeout to trigger timeout mechanism
		time.Sleep(200 * time.Millisecond) // Sleep slightly longer than HTTP client timeout

		mockStopResp := struct {
			Data struct {
				Entry struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"entry"`
				References struct {
					Agencies []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"agencies"`
				} `json:"references"`
			} `json:"data"`
			Code int    `json:"code"`
			Text string `json:"text"`
		}{
			Data: struct {
				Entry struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"entry"`
				References struct {
					Agencies []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"agencies"`
				} `json:"references"`
			}{
				Entry: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					ID:   "1_12345",
					Name: "Test Stop",
				},
				References: struct {
					Agencies []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"agencies"`
				}{
					Agencies: []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					}{
						{ID: "1", Name: "Test Agency"},
					},
				},
			},
			Code: 200,
			Text: "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockStopResp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client with short timeout for testing
	config := &ClientConfig{
		AgencyPriority:  []string{"1", "40"},
		DefaultAgencies: []string{"1", "40"},
		APITimeout:      1 * time.Second, // Short timeout for fast test
	}
	client, err := NewOneBusAwayClientWithConfig(server.URL, "test-key", config)
	assert.NoError(t, err)

	// Override HTTP client timeout to be shorter for testing
	client.Client.Timeout = 100 * time.Millisecond

	// This test should complete quickly due to timeout mechanism
	// even though the server would take 15s to respond
	start := time.Now()
	stops, err := client.FindAllMatchingStops("12345")
	duration := time.Since(start)

	// Should complete within reasonable time due to timeout (100ms HTTP timeout + some buffer)
	assert.Less(t, duration, 500*time.Millisecond, "FindAllMatchingStops should timeout after ~100ms")

	// Should return empty results due to timeout
	assert.NoError(t, err)
	assert.Empty(t, stops, "Should return empty results when timing out")
}
