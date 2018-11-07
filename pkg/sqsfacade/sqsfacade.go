package sqsfacade

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/function61/gokit/retry"
	"time"
)

const (
	MaxItemsPerBatch = 10 // AWS limitation
)

// AWS SQS requires too much boilerplate that you can get wrong, for simple things.
// therefore I had to build a facade to hide this misery

func New(queueUrl string, accessKeyId string, accessKeySecret string) *SQS {
	sess := session.Must(session.NewSession())

	return &SQS{
		queueUrl: queueUrl,
		client: sqs.New(sess, &aws.Config{
			Region: aws.String(endpoints.UsEast1RegionID),
			Credentials: credentials.NewStaticCredentials(
				accessKeyId,
				accessKeySecret,
				""),
		}),
	}
}

type SQS struct {
	queueUrl string
	client   *sqs.SQS
}

func (s *SQS) Receive() (*sqs.ReceiveMessageOutput, error) {
	output, err := s.client.ReceiveMessage(&sqs.ReceiveMessageInput{
		MaxNumberOfMessages: aws.Int64(10),
		QueueUrl:            &s.queueUrl,
		WaitTimeSeconds:     aws.Int64(10),
	})

	return output, err
}

func (s *SQS) AckReceived(receiveOutput *sqs.ReceiveMessageOutput) error {
	ackList := []*sqs.DeleteMessageBatchRequestEntry{}

	for _, msg := range receiveOutput.Messages {
		ackList = append(ackList, &sqs.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		})
	}

	if len(ackList) == 0 {
		return nil
	}

	// TODO: retry failed
	response, err := s.client.DeleteMessageBatch(&sqs.DeleteMessageBatchInput{
		Entries:  ackList,
		QueueUrl: &s.queueUrl,
	})
	if err != nil {
		return err
	}

	if len(response.Failed) > 0 {
		return fmt.Errorf("DeleteMessageBatch() failed for %d/%d entries", len(response.Failed), len(ackList))
	}

	return nil
}

func (s *SQS) Send(
	ctx context.Context,
	batch []*sqs.SendMessageBatchRequestEntry,
	timeout time.Duration,
	failed func(err error),
) error {
	// in the start all msgs remain undelivered
	leftToDeliver := batch

	attempt := func(ctx context.Context) error {
		undelivered, err := s.send(ctx, leftToDeliver)

		leftToDeliver = undelivered

		// err == nil if len(undelivered) == 0
		return err
	}

	ctxRetry, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return retry.Retry(
		ctxRetry,
		attempt,
		retry.DefaultBackoff(),
		failed)
}

func (s *SQS) send(
	ctx context.Context,
	batch []*sqs.SendMessageBatchRequestEntry,
) ([]*sqs.SendMessageBatchRequestEntry, error) {
	response, err := s.client.SendMessageBatchWithContext(ctx, &sqs.SendMessageBatchInput{
		Entries:  batch,
		QueueUrl: &s.queueUrl,
	})
	if err != nil {
		// whole batch left undelivered
		return batch, fmt.Errorf("SendMessageBatch(): %s", err.Error())
	}

	// at least partially undelivered

	undelivered := []*sqs.SendMessageBatchRequestEntry{}

	err = nil

	if len(response.Failed) > 0 {
		err = fmt.Errorf("SendMessageBatch(): %d/%d entries failed", len(undelivered), len(batch))
	}

	for _, batchEntry := range batch {
		for _, failed := range response.Failed {
			if batchEntry.Id == failed.Id {
				undelivered = append(undelivered, batchEntry)
				break
			}
		}
	}

	if len(undelivered) != len(response.Failed) {
		err = fmt.Errorf("SendMessageBatch() some entries failed, and response seems corrupted")
	}

	return undelivered, err
}

func ToSimpleQueueEntry(body string, idx int) *sqs.SendMessageBatchRequestEntry {
	return &sqs.SendMessageBatchRequestEntry{
		Id:          aws.String(fmt.Sprintf("%d", idx)),
		MessageBody: aws.String(body),
	}
}
