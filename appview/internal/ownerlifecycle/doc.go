// Package ownerlifecycle owns the durable authorization state and the
// cross-process fence for effects performed on behalf of an AT Protocol DID.
//
// Lock order is part of the package contract. A path acquires only the classes
// it needs, always in this order:
//
//  1. owner-effect advisory fences, with DIDs sorted by their canonical string;
//  2. an optional object or work advisory fence;
//  3. parent-session advisory fences sorted by (DID, session ID);
//  4. a short database transaction.
//
// Row locks inside that transaction are acquired in this order:
//
//  1. owner lifecycle/auth-epoch row;
//  2. authorization requests by primary key;
//  3. account-deletion operation;
//  4. OAuth parents by session ID;
//  5. CraftSky children by primary key;
//  6. handoff exchanges and receipts by primary key;
//  7. cleanup and outbox rows by primary key.
//
// The owner fence is session-scoped and uses a dedicated pgx connection. The
// package never holds a database transaction over an external network call.
// Callbacks must not recursively acquire another owner fence.
package ownerlifecycle
