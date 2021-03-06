package logrus_cloudwatchlogs

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type Hook struct {
	svc               *cloudwatchlogs.CloudWatchLogs
	groupName         string
	streamName        string
	nextSequenceToken *string
}

func NewHook(groupName, streamName string, cfg *aws.Config) (*Hook, error) {
	h := &Hook{
		svc:        cloudwatchlogs.New(session.New(cfg)),
		groupName:  groupName,
		streamName: streamName,
	}

	resp, err := h.svc.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(h.groupName), // Required
		LogStreamNamePrefix: aws.String(h.streamName),
	})

	if err != nil {
		return nil, err
	}

	// grab the next sequence token
	if len(resp.LogStreams) > 0 {
		h.nextSequenceToken = resp.LogStreams[0].UploadSequenceToken
		return h, nil
	}

	// create stream if it doesn't exist. the next sequence token will be null
	_, err = h.svc.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(streamName),
	})
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	switch entry.Level {
	case logrus.PanicLevel:
		fallthrough
	case logrus.FatalLevel:
		fallthrough
	case logrus.ErrorLevel:
		fallthrough
	case logrus.WarnLevel:
		fallthrough
	case logrus.InfoLevel:
		fallthrough
	case logrus.DebugLevel:
		_, err := h.Write([]byte(line))
		return err
	default:
		return nil
	}
}

func (h *Hook) Write(p []byte) (n int, err error) {
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents: []*cloudwatchlogs.InputLogEvent{
			{
				Message:   aws.String(string(p)),
				Timestamp: aws.Int64(int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)),
			},
		},
		LogGroupName:  aws.String(h.groupName),
		LogStreamName: aws.String(h.streamName),
		SequenceToken: h.nextSequenceToken,
	}
	resp, err := h.svc.PutLogEvents(params)

	if err != nil {
		//fmt.Println(reflect.TypeOf(err))
		return 0, err
	}

	h.nextSequenceToken = resp.NextSequenceToken

	return len(p), nil
}

func (h *Hook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

// WriterHook is a hook that just outputs to an io.Writer.
// This is useful because our formatter outputs the file
// and line where it was called, and the callstack for a hook
// is different from the callstack for just writing to logrus.Logger.Out.
type WriterHook struct {
	w io.Writer
}

func NewWriterHook(w io.Writer) *WriterHook {
	return &WriterHook{w: w}
}

func (h *WriterHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	_, err = h.w.Write([]byte(line))
	return err
}

func (h *WriterHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
