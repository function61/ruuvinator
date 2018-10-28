package ruuviframeparser

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/function61/ruuvinator/pkg/hciframereceiver"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
	"github.com/function61/ruuvinator/pkg/utils"
	"time"
)

const (
	btAddrLen            = 6
	format3PayloadOffset = 19
)

var (
	errUnknownFormat = errors.New("unknown format")
)

type SensorFormat3 struct {
	ManufacturerID      uint16
	DataFormat          uint8
	Humidity            uint8
	Temperature         uint8
	TemperatureFraction uint8
	Pressure            uint16
	AccelerationX       int16
	AccelerationY       int16
	AccelerationZ       int16
	BatteryVoltageMv    uint16
}

// https://github.com/ruuvi/ruuvi-sensor-protocols

var ruuviFormat3Signature = []byte{0x99, 0x04, 0x03}

func parseTemperature(t uint8, f uint8) float64 {
	mask := uint8(1 << 7)
	isNegative := (t & mask) > 0
	temp := float64(t&^mask) + float64(f)/100.0
	if isNegative {
		temp *= -1
	}
	return temp
}

// thanks https://github.com/Turee/goruuvitag
func parseSensorFormat3(data []byte, addr string) (*ruuvinatortypes.SensorObservation, error) {
	reader := bytes.NewReader(data)
	result := SensorFormat3{}
	err := binary.Read(reader, binary.BigEndian, &result)
	if err != nil {
		return nil, err
	}

	return &ruuvinatortypes.SensorObservation{
		SensorAddr: addr,
		Time:       time.Now(),
		Measurements: ruuvinatortypes.SensorMeasurements{
			Temperature: parseTemperature(result.Temperature, result.TemperatureFraction),
			Humidity:    float64(result.Humidity) / 2.0,
			Pressure:    uint32(result.Pressure) + 50000,
			Battery:     float64(result.BatteryVoltageMv) / 1000.0,
			Acceleration: ruuvinatortypes.AccelerationData{
				X: result.AccelerationX,
				Y: result.AccelerationY,
				Z: result.AccelerationZ,
			},
		},
	}, nil
}

// good reference implementation:
// https://github.com/ttu/ruuvitag-sensor/blob/master/ruuvitag_sensor/ruuvi.py
func Parse(frame hciframereceiver.Frame) (*ruuvinatortypes.SensorObservation, error) {
	if frame.Direction != hciframereceiver.HciDumpDirectionInbound {
		return nil, nil // not an error per se
	}

	if len(frame.Data) < 36 {
		return nil, errUnknownFormat
	}

	// manufacturer specific data
	if frame.Data[18] != 0xff {
		return nil, errUnknownFormat
	}

	if !bytes.Equal(ruuviFormat3Signature, frame.Data[format3PayloadOffset:format3PayloadOffset+len(ruuviFormat3Signature)]) {
		return nil, errUnknownFormat
	}

	btAddrBytes := unfuckBluetoothAddress(frame.Data[7 : 7+btAddrLen])
	btAddrString := utils.SplitStringIntoGroupsOfTwo(hex.EncodeToString(btAddrBytes), ":")

	sensorDataParsed, err := parseSensorFormat3(
		frame.Data[format3PayloadOffset:],
		btAddrString)
	if err != nil {
		return nil, err
	}

	return sensorDataParsed, nil
}

// for some braindead reason (security by obscurity?) the Bluetooth address is in reverse order
func unfuckBluetoothAddress(addr []byte) []byte {
	return []byte{addr[5], addr[4], addr[3], addr[2], addr[1], addr[0]}
}
