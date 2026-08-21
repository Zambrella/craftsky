# Tap ingestion durability flow

This document shows how CraftSky receives AT Protocol repository events from
Tap, commits a durable acknowledgement boundary, and asynchronously projects
the retained source into the AppView's user-facing tables.

The five principal durability tables are:

- `tap_ingestion_receipts`: the idempotent ledger of durably handled events.
- `tap_source_records`: the latest retained source record or delete tombstone
  for each AT URI.
- `tap_projection_jobs`: the durable queue for materializing retained sources
  into serving tables.
- `tap_quarantined_events`: bounded evidence and replay state for invalid
  events or sources.
- `tap_repository_jobs`: durable repository registration and authoritative PDS
  reconciliation work.

```mermaid
sequenceDiagram
    autonumber

    participant PDS as User PDS
    participant Tap as Tap sidecar
    participant Consumer as Tap WS consumer
    participant Ingest as Ingestion service
    participant Lifecycle as Owner lifecycle

    participant Receipts as tap_ingestion_receipts
    participant Sources as tap_source_records
    participant Projections as tap_projection_jobs
    participant Quarantine as tap_quarantined_events
    participant Repositories as tap_repository_jobs

    participant ProjectionWorker as Projection worker
    participant Projector as Transactional projector
    participant Serving as AppView serving tables

    participant RepositoryWorker as Repository worker
    participant Operator as Operator / CLI
    participant ReplayWorker as Quarantine replay worker

    PDS->>Tap: Repository commit via firehose
    Tap->>Consumer: WebSocket event

    alt Invalid or unsupported event
        Consumer->>Ingest: Quarantine invalid envelope
        Ingest->>Quarantine: Store bounded evidence and exact replay payload
        Ingest->>Receipts: Record permanent_invalid outcome
        Note over Quarantine,Receipts: Both writes commit atomically
        Ingest-->>Consumer: Durable permanent-invalid outcome
        Consumer->>Tap: ACK event

    else Identity event
        Consumer->>Ingest: Ingest identity event

        alt Ordinary identity update
            Ingest->>Receipts: Record applied outcome
            Ingest->>Lifecycle: Schedule identity refresh
        else Account deleted
            Ingest->>Lifecycle: Terminalize owner and commit purge obligations
            Ingest->>Receipts: Record applied outcome
            Note over Lifecycle,Receipts: Terminal state and receipt commit atomically
        end

        Ingest-->>Consumer: Durable applied outcome
        Consumer->>Tap: ACK event

    else Valid record event
        Consumer->>Ingest: Ingest decoded record
        Ingest->>Lifecycle: Lock and inspect owner lifecycle
        Ingest->>Sources: Compare revision, CID, action and content

        alt Duplicate or stale delivery
            Sources-->>Ingest: Existing source remains authoritative
            Ingest->>Receipts: Deduplicate by event fingerprint

        else New authoritative source
            Ingest->>Sources: Upsert current record or delete tombstone
            Ingest->>Projections: Upsert projection job

            alt Dependency currently missing
                Ingest->>Projections: Set blocked and store dependency key
            else Eligible for projection
                Ingest->>Projections: Set pending
            else Terminal owner or permanently denied
                Ingest->>Projections: Set permanent_denied
            end

            Ingest->>Receipts: Record applied, blocked, or permanent-invalid outcome

        else Ordering conflict or uncertain PDS effect
            Ingest->>Sources: Mark ordering_status as uncertain
            Ingest->>Projections: Block on repository_did
            Ingest->>Repositories: Enqueue pds_reconcile job
            Ingest->>Receipts: Record blocked outcome
        end

        Note over Sources,Receipts: Source, jobs, and receipt commit in one transaction
        Ingest-->>Consumer: Durable acknowledgable outcome
        Consumer->>Tap: ACK event
    end

    opt Storage error or retryable ingestion failure
        Ingest-->>Consumer: Retryable outcome or error
        Consumer--xTap: No ACK
        Note over Tap,Consumer: Tap retains and redelivers the event
    end

    loop Projection processing
        ProjectionWorker->>Projections: Claim pending or expired job lease
        Projections-->>ProjectionWorker: Leased projection job
        ProjectionWorker->>Sources: Lock matching current source version
        ProjectionWorker->>Lifecycle: Recheck owner state and generation
        ProjectionWorker->>Projector: Project source transactionally

        alt Projection applied
            Projector->>Serving: Insert, update, or delete serving state
            Projector->>Projections: Mark complete
            Projector->>Projections: Wake jobs waiting on this dependency
            Note over Serving,Projections: Serving changes and completion commit atomically

        else Dependency missing
            Projector->>Projections: Block on member_did or subject_uri

        else Source permanently invalid
            Projector->>Quarantine: Store replayable source quarantine
            Projector->>Receipts: Record permanent-invalid quarantine outcome
            Projector->>Projections: Mark permanent_denied

        else Transient projection failure
            Projector--xServing: Roll back serving transaction
            ProjectionWorker->>Projections: Reschedule pending with backoff
        end
    end

    loop Repository-level recovery
        RepositoryWorker->>Repositories: Claim pending or expired job lease
        Repositories-->>RepositoryWorker: tap_add_repo or pds_reconcile

        alt tap_add_repo
            RepositoryWorker->>Tap: Register repository with AddRepo
            Tap-->>RepositoryWorker: Registration result

        else pds_reconcile
            RepositoryWorker->>Sources: List uncertain sources for DID
            RepositoryWorker->>PDS: Read authoritative records
            PDS-->>RepositoryWorker: Current record, CID, or not-found
            RepositoryWorker->>Ingest: Reconcile authoritative observation
            Ingest->>Sources: Install authoritative source or tombstone
            Ingest->>Projections: Unblock or replace projection job
        end

        alt Repository operation succeeds
            RepositoryWorker->>Repositories: Mark complete and save revision
        else Remote or transient failure
            RepositoryWorker->>Repositories: Reschedule pending with backoff
        end
    end

    opt Operator repairs quarantined input
        Operator->>Quarantine: Request replay by fingerprint
        Quarantine->>Quarantine: Set replay_state to pending
        ReplayWorker->>Quarantine: Claim replay lease

        alt Original Tap envelope
            ReplayWorker->>Ingest: Re-enter normal decode and ingestion flow
        else Projection-generated source quarantine
            ReplayWorker->>Sources: Verify source version is still current
            ReplayWorker->>Projections: Return matching denied job to pending
        end

        alt Replay succeeds
            ReplayWorker->>Quarantine: Mark resolved
        else Replay remains retryable
            ReplayWorker->>Quarantine: Return to pending
        end
    end
```

## Critical acknowledgement boundary

The Tap consumer sends an ACK only after CraftSky has committed an
acknowledgable outcome. For a valid record, that transaction retains the source
state, creates or updates its downstream work, and writes an ingestion receipt.
For invalid input, it commits quarantine evidence and a receipt. A retryable or
storage failure is not acknowledged, allowing Tap to redeliver it.

Projection is deliberately outside this ACK deadline. The projection,
repository, and replay workers use expiring leases so unfinished work survives
process crashes and can be reclaimed safely.
