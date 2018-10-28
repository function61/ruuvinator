package ruuvinatortypes

import (
	"time"
)

type SensorObservation struct {
	SensorAddr   string             `json:"sensor_addr"`
	Time         time.Time          `json:"time"`
	Measurements SensorMeasurements `json:"measurements"`
}

type SensorMeasurements struct {
	Temperature  float64          `json:"temperature"`
	Humidity     float64          `json:"humidity"`
	Pressure     uint32           `json:"pressure"`
	Battery      float64          `json:"battery"`
	Acceleration AccelerationData `json:"acceleration"`
}

type AccelerationData struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
	Z int16 `json:"z"`
}

type Output interface {
	GetObservationsChan() chan<- ResolvedSensorObservation
}

// btAddr => friendlyName
type SensorWhitelist map[string]string

type Config struct {
	Output          string           `json:"output"`
	SensorWhitelist SensorWhitelist  `json:"sensor_whitelist"`
	SqsOutputConfig *SqsOutputConfig `json:"sqsoutput_config"` // used if output=sqsoutput
}

type SqsOutputConfig struct {
	QueueUrl           string `json:"queue_url"`
	AwsAccessKeyId     string `json:"aws_access_key_id"`
	AwsAccessKeySecret string `json:"aws_access_key_secret"`
}

// resolved observation means an observation whose presence is detected against a whitelist
// and thus its friendly name is now also known
type ResolvedSensorObservation struct {
	SensorName  string            `json:"sensor_name"`
	Observation SensorObservation `json:"observation"`
}

type SensorResolver interface {
	Resolve(SensorObservation) (*ResolvedSensorObservation, bool)
}
