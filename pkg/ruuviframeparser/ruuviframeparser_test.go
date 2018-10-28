package ruuviframeparser

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/ruuvinator/pkg/hciframereceiver"
	"github.com/function61/ruuvinator/pkg/ruuvinatortestdata"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
	"strings"
	"testing"
)

func TestParseAnyRuuviFormat(t *testing.T) {
	observations := []*ruuvinatortypes.SensorObservation{}

	err := hciframereceiver.ParseStream(strings.NewReader(ruuvinatortestdata.DemoStream), func(frame hciframereceiver.Frame) {
		observation, _ := Parse(frame)
		if observation != nil {
			observations = append(observations, observation)
		}
	})

	assert.True(t, err == nil)
	assert.True(t, len(observations) == 2)

	obs := observations[0]

	assert.EqualString(t, obs.SensorAddr, "fb:72:36:09:90:15")
	assert.True(t, obs.Measurements.Temperature == 19.68)
	assert.True(t, obs.Measurements.Humidity == 35.5)
	assert.True(t, obs.Measurements.Pressure == 98875)
	assert.True(t, obs.Measurements.Battery == 3.157)
	assert.True(t, obs.Measurements.Acceleration.X == 49)
	assert.True(t, obs.Measurements.Acceleration.Y == -41)
	assert.True(t, obs.Measurements.Acceleration.Z == 1034)

	obs = observations[1]

	assert.EqualString(t, obs.SensorAddr, "e5:fa:12:7e:ef:65")
	assert.True(t, obs.Measurements.Temperature == 1.13)
	assert.True(t, obs.Measurements.Humidity == 87)
	assert.True(t, obs.Measurements.Pressure == 99754)
	assert.True(t, obs.Measurements.Battery == 2.845)
	assert.True(t, obs.Measurements.Acceleration.X == 542)
	assert.True(t, obs.Measurements.Acceleration.Y == 421)
	assert.True(t, obs.Measurements.Acceleration.Z == -726)
}
