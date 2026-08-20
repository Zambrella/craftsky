package api

import (
	"errors"
	"testing"

	"social.craftsky/appview/internal/notifications"
)

type notificationPreferenceRowsStub struct {
	category notifications.Category
	scope    notifications.Scope
	enabled  bool
	advanced bool
	err      error
	closed   bool
}

func (rows *notificationPreferenceRowsStub) Next() bool {
	if rows.advanced {
		return false
	}
	rows.advanced = true
	return true
}

func (rows *notificationPreferenceRowsStub) Scan(destinations ...any) error {
	*destinations[0].(*notifications.Category) = rows.category
	*destinations[1].(*notifications.Scope) = rows.scope
	*destinations[2].(*bool) = rows.enabled
	return nil
}

func (rows *notificationPreferenceRowsStub) Err() error { return rows.err }

func (rows *notificationPreferenceRowsStub) Close() { rows.closed = true }

func TestScanNotificationPreferenceRowsRejectsPrefixOnTerminalError(t *testing.T) {
	sentinel := errors.New("iterator failed after prefix")
	rows := &notificationPreferenceRowsStub{
		category: notifications.Like,
		scope:    notifications.Everyone,
		enabled:  true,
		err:      sentinel,
	}

	persisted, err := scanNotificationPreferenceRows(rows)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want terminal iterator error", err)
	}
	if persisted != nil {
		t.Fatalf("persisted = %+v, want no usable partial prefix", persisted)
	}
	if !rows.closed {
		t.Fatal("rows were not closed before returning")
	}
}
