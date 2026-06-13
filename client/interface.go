package client

import "oba-twilio/models"

type OneBusAwayClientInterface interface {
	GetArrivalsAndDepartures(stopID string) (*models.OneBusAwayResponse, error)
	GetArrivalsAndDeparturesWithWindow(stopID string, minutesAfter int) (*models.OneBusAwayResponse, error)
	ProcessArrivals(resp *models.OneBusAwayResponse, maxMinutesOut int) []models.Arrival
	SearchStops(query string) ([]models.Stop, error)
	InitializeCoverage() error
	GetCoverageArea() *models.CoverageArea
	FindAllMatchingStops(stopID string) ([]models.StopOption, error)
	GetStopInfo(fullStopID string) (*models.StopOption, error)
}
