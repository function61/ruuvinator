package sqsoutput

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/function61/gokit/logger"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
	"github.com/function61/ruuvinator/pkg/sqsfacade"
	"time"
)

var log = logger.New("sqs-output")

type output struct {
	config       ruuvinatortypes.SqsOutputConfig
	observations chan ruuvinatortypes.ResolvedSensorObservation
}

func (o *output) GetObservationsChan() chan<- ruuvinatortypes.ResolvedSensorObservation {
	return o.observations
}

func (o *output) processor(ctx context.Context) {
	log.Info("starting")
	defer log.Info("stopped")

	sqsClient := sqsfacade.New(
		o.config.QueueUrl,
		o.config.AwsAccessKeyId,
		o.config.AwsAccessKeySecret)

	for {
		select {
		case <-ctx.Done():
			return // stop loop
		case firstItem := <-o.observations:
			// try to grab at most 10 observations from channel
			observations := readMoreUnblocking(sqsfacade.MaxItemsPerBatch, firstItem, o.observations)

			batch := []*sqs.SendMessageBatchRequestEntry{}

			for idx, observation := range observations {
				observationAsJson, _ := json.Marshal(observation)

				batch = append(batch, sqsfacade.ToSimpleQueueEntry(string(observationAsJson), idx))
			}

			// group all following observations that arrive within one
			// second to the next batch submit
			nextPossibleQueueSubmit := time.Now().Add(1 * time.Second)

			failed := func(err error) {
				log.Error(fmt.Sprintf("Send: %s", err.Error()))
			}

			if err := sqsClient.Send(ctx, batch, 10*time.Second, failed); err != nil {
				failed(err)
			}

			// only sleep if we're not submitting at max capacity
			if len(batch) < sqsfacade.MaxItemsPerBatch {
				time.Sleep(time.Until(nextPossibleQueueSubmit))
			} else {
				log.Info(fmt.Sprintf("operating at queue send max capacity: %d", len(batch)))
			}
		}
	}
}

func New(ctx context.Context, config ruuvinatortypes.SqsOutputConfig) *output {
	out := &output{
		config:       config,
		observations: make(chan ruuvinatortypes.ResolvedSensorObservation, 32),
	}

	go out.processor(ctx)

	return out
}

// this could really benefit from generics
func readMoreUnblocking(
	limit int,
	firstItem ruuvinatortypes.ResolvedSensorObservation,
	ch <-chan ruuvinatortypes.ResolvedSensorObservation,
) []ruuvinatortypes.ResolvedSensorObservation {
	items := []ruuvinatortypes.ResolvedSensorObservation{firstItem}

	// -1 becase we already have firstItem
	for i := 0; i < limit-1; i++ {
		// peek into the channel
		select {
		case item := <-ch:
			items = append(items, item)
		default:
			i = limit // exit from peek loop
		}
	}

	return items
}
