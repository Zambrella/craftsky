// Package firehose defines the contract for consuming the atproto Relay
// firehose. Day one contains only the interface and a NotImplemented stub
// so the CLI's firehose-replay subcommand compiles and returns a clean
// error. Real subscription logic lands in a later commit.
package firehose

import (
	"context"
	"errors"
	"time"
)

// Subscriber replays firehose events into the indexer.
type Subscriber interface {
	// Replay re-indexes firehose events since the given timestamp.
	Replay(ctx context.Context, since time.Time) error
}

// NotImplemented is the day-one stub. Every method returns a descriptive
// error; the CLI surfaces this to stdout with exit code 1.
type NotImplemented struct{}

var _ Subscriber = NotImplemented{}

func (NotImplemented) Replay(ctx context.Context, since time.Time) error {
	return errors.New("firehose: not yet implemented")
}
