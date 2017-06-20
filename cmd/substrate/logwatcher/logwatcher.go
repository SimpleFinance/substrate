package logwatcher

import (
	"encoding/json"
	"math"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// LogWatcher is an interface to stream a Cloudwatch Logs group to stdout
type LogWatcher struct {

	// a group that collects any errors encountered by the background threads
	errgroup *errgroup.Group

	// a cancel function from the error group that let's us cancel all running threads
	cancel context.CancelFunc

	// a context for all the threads associated with the error group
	ctx context.Context

	// the stream of returned log events
	events chan Event

	// a connection to the CloudWatch Logs API
	svc *cloudwatchlogs.CloudWatchLogs
}

// Start a LogWatcher that will stream logs from the specified AWS region and CloudWatch Logs group
func Start(svc *cloudwatchlogs.CloudWatchLogs, groupName string) *LogWatcher {
	result := &LogWatcher{
		events: make(chan Event),
		svc:    svc,
	}

	// create a new cancel-able context for this background computation
	ctx, cancel := context.WithCancel(context.Background())
	result.cancel = cancel

	// create an errgroup.Group to synchronize errors, wrapping the cancel-able context
	result.errgroup, result.ctx = errgroup.WithContext(ctx)

	// start the first background thread in the errorgroup
	result.errgroup.Go(func() error {
		return result.watchForStreams(groupName)
	})

	return result
}

// Events returns an infinite stream of Records from the watched log streams
func (w *LogWatcher) Events() <-chan Event {
	return w.events
}

// Stop the log watching goroutine
func (w *LogWatcher) Stop() error {
	w.cancel()
	err := w.errgroup.Wait()
	close(w.events)
	if err == context.Canceled {
		return nil
	}
	return err
}

// converts from "A point in time expressed as the number of milliseconds since Jan 1, 1970 00:00:00 UTC" to a standard time.Time
func parseTimestamp(awsTimestamp int64) time.Time {
	return time.Unix(awsTimestamp/1000, (awsTimestamp%1000)*1000000).UTC()
}

func (w *LogWatcher) watchForStreams(groupName string) error {
	// this map tracks which streams we've already started listeners for
	streamsWatched := map[string]bool{}

	// loop forever watching for new streams
	for {
		// call the cloudwatch logs API (possibly multiple times) to get streams
		err := w.svc.DescribeLogStreamsPages(
			&cloudwatchlogs.DescribeLogStreamsInput{LogGroupName: &groupName},
			func(page *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
				for _, stream := range page.LogStreams {
					if _, ok := streamsWatched[*stream.Arn]; !ok {
						streamsWatched[*stream.Arn] = true
						w.errgroup.Go(func() error {
							return w.watchStream(groupName, stream)
						})
					}
				}
				return true
			})
		// a 404 is fine, the group just doesn't exist yet so we'll wait and try again
		awsErr, isAWSErr := err.(awserr.Error)
		if err != nil && (!isAWSErr || awsErr.Code() != "ResourceNotFoundException") {
			// otherwise we need to fail
			return err
		}

		var sleepTime time.Duration
		if len(streamsWatched) > 0 {
			// most of the time, wait a while before we look for any new streams
			sleepTime = 30 * time.Second
		} else {
			// but if we haven't found any streams yet, don't sleep so long
			sleepTime = 1 * time.Second
		}

		select {
		case <-time.After(sleepTime):
			// sleep for a bit before trying again
		case <-w.ctx.Done():
			// unless we get canceled, in which case break out of the loop immediately
			return w.ctx.Err()
		}
	}
}

func (w *LogWatcher) watchStream(groupName string, stream *cloudwatchlogs.LogStream) error {
	params := cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: stream.LogStreamName,
		StartFromHead: aws.Bool(true),
		StartTime:     aws.Int64(0),
		EndTime:       aws.Int64(math.MaxInt64),
	}

	// until we hit an error or get canceled, watch for new events
	for {
		result, err := w.svc.GetLogEvents(&params)
		if err != nil {
			return err
		}

		// this should normally never happen
		if result.NextForwardToken == nil {
			break
		}
		params.NextToken = result.NextForwardToken

		for _, event := range result.Events {
			var output Event
			err = json.Unmarshal([]byte(*event.Message), &output.Record)
			if err != nil {
				// TODO: we're just ignoring any log messages that don't match our format
				continue
			}

			output.LogGroupName = groupName
			output.LogStreamName = *stream.LogStreamName
			output.Timestamp = parseTimestamp(*event.Timestamp)
			output.IngestedTimestamp = parseTimestamp(*event.IngestionTime)
			w.events <- output
		}

		var sleepTime time.Duration
		if len(result.Events) > 100 {
			// if we're still finding _lots_ of new events, don't sleep at all
			sleepTime = 0
		} else if len(result.Events) > 0 {
			// if we're still finding some new events, but not many, sleep for just a little bit
			sleepTime = 2 * time.Second
		} else {
			// if we've caught up to the tail of the stream and we don't see any events, sleep longer
			sleepTime = 30 * time.Second
		}

		select {
		case <-time.After(sleepTime):
			// sleep before we try again
		case <-w.ctx.Done():
			// unless we get canceled, in which case break out of the loop immediately
			return w.ctx.Err()
		}
	}
	return nil
}
