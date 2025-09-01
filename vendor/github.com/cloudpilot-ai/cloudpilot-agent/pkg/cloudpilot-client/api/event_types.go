package api

import (
	"time"
)

type SpotEvent struct {
	Type               SpotEventType `json:"type"`
	CloudProvider      string        `json:"cloudProvider"`
	Region             string        `json:"region"`
	Zone               string        `json:"zone"`
	InstanceType       string        `json:"instanceType"`
	NodeAvailableHours float64       `json:"NodeAvailableHours"`
	ProviderIDHash     string        `json:"providerIDHash"`
	Time               time.Time     `json:"time"`
}

type SpotEventType string

const (
	SpotEventTypeSpotInterruption SpotEventType = "spot-interruption"
	SpotEventTypeSpotRebalance    SpotEventType = "spot-rebalance"
	SpotEventTypeSpotPrediction   SpotEventType = "spot-prediction"
)
