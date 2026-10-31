package chat

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
	"social.craftsky/appview/internal/ownerlifecycle"
	"sync"
	"time"
)

// Run has two local task slots in addition to the shared SQL capacity/owner
// claims. A completion immediately yields its slot to the next due owner;
// a slow task therefore cannot hold up the other account slot.
func (worker *Synchronizer) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	var tasks sync.WaitGroup
	defer func() { cancel(); tasks.Wait() }()
	pulses := worker.pulses
	if pulses == nil {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		pulses = ticker.C
	}
	type completion struct {
		owner syntax.DID
		err   error
	}
	completed := make(chan completion, 2)
	active := make(map[syntax.DID]bool)
	deferred := make(map[syntax.DID]bool)
	q := chatdb.New(worker.service.store.pool)
	for {
		if err := runCtx.Err(); err != nil {
			return err
		}
		if len(active) < 2 {
			if err := q.EnsureScheduledChatAccounts(runCtx); err != nil {
				return err
			}
			due, err := q.DueChatOwners(runCtx, dbTime(worker.now()))
			if err != nil {
				return err
			}
			for _, raw := range due {
				if len(active) == 2 {
					break
				}
				owner, err := syntax.ParseDID(raw)
				if err != nil {
					return errorsProviderCause(err)
				}
				if active[owner] || deferred[owner] {
					continue
				}
				active[owner] = true
				tasks.Add(1)
				go func() { defer tasks.Done(); completed <- completion{owner: owner, err: worker.Poll(runCtx, owner)} }()
			}
		}
		select {
		case <-runCtx.Done():
			return runCtx.Err()
		case <-pulses:
			clear(deferred)
		case item := <-completed:
			delete(active, item.owner)
			if errors.Is(item.err, ErrPollNotDue) {
				deferred[item.owner] = true
				continue
			}
			if item.err == nil || errors.Is(item.err, context.Canceled) {
				continue
			}
			// Even a local failure to acquire/finalize a claim yields until
			// the next scheduler pulse instead of hot-looping this owner.
			deferred[item.owner] = true
			if errors.Is(item.err, auth.ErrNoUsableBackgroundSession) || errors.Is(item.err, auth.ErrPDSPermissionsRequired) || errors.Is(item.err, auth.ErrPDSSessionExpired) || errors.Is(item.err, auth.ErrOAuthSessionNotFound) {
				// Poll owns the claimed failure schedule, including preflight.

				continue
			}
			if errors.Is(item.err, ErrDisabled) || errors.Is(item.err, ErrRestricted) || errors.Is(item.err, ErrConnectionChanged) || errors.Is(item.err, ownerlifecycle.ErrOwnerNotActive) || errors.Is(item.err, ownerlifecycle.ErrTerminalOwner) {
				continue
			}
			// The scheduler is the one private diagnostic owner. Expected per-owner
			// authorization recovery above never creates a repeated issue.
			if worker.observer != nil {
				worker.observer.ObservePrivateFailure(runCtx, item.err, item.owner, "", "chat.sync", "poll", "error", 1)
			}
		}
	}
}
