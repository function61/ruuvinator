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

const (
	// one message size in JSON is about 250 B, and SQS max message size is 256 KB so we
	// could theoretically send 500 measurements per one message and still be super safe
	maxObservationsPerOneSqsMessage = 100
)

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
			observations := readMoreUnblocking(maxObservationsPerOneSqsMessage, firstItem, o.observations)

			if len(observations) >= maxObservationsPerOneSqsMessage {
				log.Info(fmt.Sprintf(
					"packed maximum observations (%d) into one SQS message",
					len(observations)))
			}

			observationsAsJson, err := json.Marshal(observations)
			if err != nil {
				panic(err)
			}

			// send just one message in a batch. previously we sent each observation as own
			// msg, but that proved out to be (relatively) expensive
			messages := []*sqs.SendMessageBatchRequestEntry{
				sqsfacade.ToSimpleQueueEntry(string(observationsAsJson), 0),
			}

			// try to limit sending to one msg/second to cheap out on AWS bills
			nextPossibleQueueSubmit := time.Now().Add(1 * time.Second)

			failed := func(err error) {
				log.Error(fmt.Sprintf("Send: %s", err.Error()))
			}

			if err := sqsClient.Send(ctx, messages, 10*time.Second, failed); err != nil {
				failed(err)
			}

			time.Sleep(time.Until(nextPossibleQueueSubmit))
		}
	}
}

func New(ctx context.Context, config ruuvinatortypes.SqsOutputConfig) *output {
	out := &output{
		config: config,
		observations: make(
			chan ruuvinatortypes.ResolvedSensorObservation,
			maxObservationsPerOneSqsMessage*2),
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
