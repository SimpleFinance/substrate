package wipe

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type sqsQueue struct {
	region string
	url    string
}

func (r *sqsQueue) String() string {
	// the queue name is the last part of the URL path
	parts := strings.Split(r.url, "/")
	name := parts[len(parts)-1]
	return fmt.Sprintf("SQS queue %s in %s", name, r.region)
}

func (r *sqsQueue) Destroy() error {
	return fmt.Errorf("can't destroy SQS queues yet")
}

func (r *sqsQueue) Priority() int {
	return 100
}

func discoverSQSResources(region string, envName string) []destroyableResource {
	result := []destroyableResource{}
	svc := sqs.New(session.New(), &aws.Config{Region: aws.String(region)})

	fmt.Printf("scanning for SQS queues in %s...\n", region)
	namePrefix := "substrate-" + envName
	queues, err := svc.ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: &namePrefix,
	})
	if err != nil {
		panic(err)
	}
	for _, url := range queues.QueueUrls {
		result = append(result, &sqsQueue{
			region: region,
			url:    *url,
		})
	}
	return result
}
