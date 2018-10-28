package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// AWS SQS requires too much boilerplate that you can get wrong. therefore I had to
// build an abstraction
type sqsFacade struct {
	queueUrl string
	client   *sqs.SQS
}

func (a *sqsFacade) Receive() (*sqs.ReceiveMessageOutput, error) {
	output, err := a.client.ReceiveMessage(&sqs.ReceiveMessageInput{
		MaxNumberOfMessages: aws.Int64(10),
		QueueUrl:            &a.queueUrl,
		WaitTimeSeconds:     aws.Int64(10),
	})

	return output, err
}

func (a *sqsFacade) AckReceived(receiveOutput *sqs.ReceiveMessageOutput) error {
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

	_, err := a.client.DeleteMessageBatch(&sqs.DeleteMessageBatchInput{
		Entries:  ackList,
		QueueUrl: &a.queueUrl,
	})

	return err
}

func NewSQS(queueUrl string, accessKeyId string, accessKeySecret string) *sqsFacade {
	sess := session.Must(session.NewSession())

	return &sqsFacade{
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
