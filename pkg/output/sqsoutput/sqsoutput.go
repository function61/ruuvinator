package sqsoutput

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/function61/gokit/logger"
	"github.com/function61/ruuvinator/pkg/ruuvinatortypes"
	"time"
)

const (
	maxItemsPerQueueSendBatch = 10 // aws limitation
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

	sess := session.Must(session.NewSession())

	sqsClient := sqs.New(sess, &aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
		Credentials: credentials.NewStaticCredentials(
			o.config.AwsAccessKeyId,
			o.config.AwsAccessKeySecret,
			""),
	})

	for {
		select {
		case <-ctx.Done():
			return // stop loop
		case firstItem := <-o.observations:
			// try to grab at most 10 observations from channel
			observations := readMoreUnblocking(maxItemsPerQueueSendBatch, firstItem, o.observations)

			batch := []*sqs.SendMessageBatchRequestEntry{}

			for idx, observation := range observations {
				observationAsJson, _ := json.Marshal(observation)

				batch = append(batch, toQueueEntry(string(observationAsJson), idx))
			}

			// group all following observations that arrive within one
			// second to the next batch submit
			nextPossibleQueueSubmit := time.Now().Add(1 * time.Second)

			out, err := sqsClient.SendMessageBatch(&sqs.SendMessageBatchInput{
				Entries:  batch,
				QueueUrl: &o.config.QueueUrl,
			})
			if err != nil {
				log.Error(fmt.Sprintf("SendMessageBatch(): %s", err.Error()))
				continue
			}

			if len(out.Failed) > 0 {
				// TODO: retry logic?
				log.Error(fmt.Sprintf("%d/%d entries failed", len(out.Failed), len(batch)))
			}

			// only sleep if we're not submitting at max capacity
			if len(batch) < maxItemsPerQueueSendBatch {
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
		observations: make(chan ruuvinatortypes.ResolvedSensorObservation, 16),
	}

	go out.processor(ctx)

	return out

}

func toQueueEntry(body string, idx int) *sqs.SendMessageBatchRequestEntry {
	return &sqs.SendMessageBatchRequestEntry{
		Id:          aws.String(fmt.Sprintf("%d", idx)),
		MessageBody: aws.String(body),
	}
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
