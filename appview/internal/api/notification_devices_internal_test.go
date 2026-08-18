package api

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

type notificationSubscriptionRowsStub struct {
	ids     []uuid.UUID
	next    int
	current int
	err     error
	closed  bool
}

func (rows *notificationSubscriptionRowsStub) Next() bool {
	if rows.next >= len(rows.ids) {
		return false
	}
	rows.current = rows.next
	rows.next++
	return true
}

func (rows *notificationSubscriptionRowsStub) Scan(destinations ...any) error {
	*destinations[0].(*uuid.UUID) = rows.ids[rows.current]
	return nil
}

func (rows *notificationSubscriptionRowsStub) Err() error {
	return rows.err
}

func (rows *notificationSubscriptionRowsStub) Close() {
	rows.closed = true
}

func TestScanNotificationSubscriptionIDsRejectsPrefixOnTerminalError(t *testing.T) {
	sentinel := errors.New("iterator failed after prefix")
	rows := &notificationSubscriptionRowsStub{
		ids: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		err: sentinel,
	}

	ids, err := scanNotificationSubscriptionIDs(rows)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want terminal iterator error", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v, want no usable partial prefix", ids)
	}
	if !rows.closed {
		t.Fatal("rows were not closed before returning")
	}
}
