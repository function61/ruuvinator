package main

import (
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
)

type whitelistResolver struct {
	whitelist ruuvinatortypes.SensorWhitelist
}

func (w *whitelistResolver) Resolve(observation ruuvinatortypes.SensorObservation) (*ruuvinatortypes.ResolvedSensorObservation, bool) {
	friendlyName, whitelisted := w.whitelist[observation.SensorAddr]
	if !whitelisted {
		return nil, false
	}

	return &ruuvinatortypes.ResolvedSensorObservation{
		SensorName:  friendlyName,
		Observation: observation,
	}, true
}

func NewWhitelistResolver(whitelist ruuvinatortypes.SensorWhitelist) ruuvinatortypes.SensorResolver {
	return &whitelistResolver{
		whitelist: whitelist,
	}
}
