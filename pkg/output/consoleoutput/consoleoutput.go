package consoleoutput

import (
	"encoding/json"
	"fmt"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
)

type output struct {
	ch chan ruuvinatortypes.ResolvedSensorObservation
}

func (o *output) GetObservationsChan() chan<- ruuvinatortypes.ResolvedSensorObservation {
	return o.ch
}

func New() *output {
	ch := make(chan ruuvinatortypes.ResolvedSensorObservation, 1)

	go func() {
		for observation := range ch {
			observationAsJson, _ := json.Marshal(observation)

			fmt.Printf("%s\n", observationAsJson)
		}
	}()

	return &output{
		ch: ch,
	}
}
