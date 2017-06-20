package zone

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/SimpleFinance/substrate/cmd/substrate/logwatcher"
)

// LogsInput contains the input parameters for tailing logs in a zone
type LogsInput struct {
	Version      string
	ManifestPath string
	Verbose      bool
}

// Logs pulls zone metadata from a manifest file and then tails the logs for that zone
func Logs(params *LogsInput) error {
	zoneManifest, err := ReadManifest(params.ManifestPath)
	if err != nil {
		return err
	}

	log := logwatcher.Start(
		cloudwatchlogs.New(
			session.New(),
			&aws.Config{Region: aws.String(zoneManifest.AWSRegion())}),
		zoneManifest.CloudWatchLogsGroupSystemLogs(),
	)

	for event := range log.Events() {
		if event.Record.Priority == "DEBUG" {
			continue
		}
		id := event.Record.Syslog.Identifier
		if id == "" {
			id = event.Record.SystemdUnit
		}
		fmt.Printf(
			"%s %s:%s - [%s] %s\n",
			event.Timestamp.Format(time.RFC3339),
			event.Record.Hostname,
			id,
			event.Record.Priority,
			event.Record.Message,
		)
	}
	return nil
}
