## Architecture Decision Record

- Status: Accepted
- Aspect: Production hosting, persistence, deployment operations
- Date: 2026-09-14
- Decision: Run AppView, Tap, and PostgreSQL on Render and use an external managed S3-compatible service for AppView-managed media

### Why I needed to decide this

CraftSky needs a production environment for the AppView HTTP API, atproto OAuth
callbacks, firehose ingestion, private application state, and scheduled-post media.
The original architecture reference recommended a Hetzner VPS running Docker
Compose, Caddy, and PostgreSQL. That is inexpensive and close to the development
topology, but it makes CraftSky responsible for host patching, TLS termination,
database operation, backup automation, deployment orchestration, and recovery
from a single-host failure.

The production workload is not wholly stateless. AppView currently runs the HTTP
server, a long-lived Tap WebSocket consumer, and its background workers in one
process. Tap is a separate long-lived service and stores its relay cursor and
repository tracking in SQLite. PostgreSQL stores the indexed public view and
private-by-intent data including OAuth material, sessions, drafts, moderation
state, notification work, and scheduled-post state. Scheduled media needs private
S3-compatible storage until publication transfers it to the user's PDS.

The hosting decision therefore needs to provide:

- continuously running container services rather than request-only functions;
- a persistent local disk for Tap's SQLite database;
- PostgreSQL with ordinary session semantics and the extensions and locking
  features used by CraftSky;
- private service networking;
- public HTTPS ingress for AppView and its OAuth endpoints;
- pre-deploy database migrations, secrets, health checks, logs, and controlled
  shutdown; and
- durable private object storage without making CraftSky operate another
  stateful storage server.

### Options I considered

**Option 1: Hetzner VPS with Docker Compose, Caddy, and self-hosted PostgreSQL - not chosen**

Pros:

- Lowest expected baseline cost.
- Closely matches the existing development Compose topology.
- Gives full control over networking, disks, deployment, and database settings.
- Naturally supports Tap's persistent SQLite file.

Cons:

- Makes CraftSky responsible for operating and securing the host and database.
- AppView, Tap, PostgreSQL, and potentially object storage share one failure
  domain unless additional infrastructure is built.
- Requires separate work for off-host backups, point-in-time recovery, restore
  drills, host monitoring, certificate management, and deployment automation.
- Increases operational work before the first production release.

**Option 2: Render services, Render Postgres, and external managed object storage - chosen**

Pros:

- Maps the existing services onto long-running containers without redesigning
  the application as functions.
- Provides managed TLS, custom domains, deployments, secrets, health checks,
  logs, private networking, and PostgreSQL backups.
- Supports a private Tap service with an attached persistent disk.
- Keeps AppView and PostgreSQL on a low-latency private network in one region.
- Removes host, reverse-proxy, and primary database administration from the
  initial production workload.
- An external managed object store provides replication and durability without
  another always-running storage process.

Cons:

- Costs more than a small self-managed VPS.
- Introduces dependence on Render's service, database, networking, and deployment
  behavior.
- Tap remains a disk-bound singleton and has brief downtime during replacement.
- Render's normal zero-downtime AppView rollout temporarily overlaps old and new
  instances, which is not yet approved by CraftSky's single-replica admission
  boundary.
- External object storage adds another provider and public TLS network path.

**Option 3: Render with self-hosted MinIO - not chosen**

Pros:

- Keeps compute and storage administration in one Render workspace.
- Uses the S3-compatible API already implemented by AppView.
- Render persistent disks provide encryption at rest and daily snapshots.

Cons:

- MinIO would be another continuously running, disk-bound singleton operated by
  CraftSky rather than a managed object-storage service.
- A single attached disk does not provide the replication and availability
  characteristics of managed object storage.
- Disk-backed deployment and maintenance interrupt the MinIO service.
- Render private service addresses normally use HTTP, while CraftSky deliberately
  requires an HTTPS S3 endpoint in production. Providing internal TLS or exposing
  MinIO through public Render HTTPS adds complexity and attack surface.
- At CraftSky's expected initial storage volume, fixed MinIO compute and disk can
  cost more than usage-priced managed object storage.

**Option 4: Render compute with an external managed PostgreSQL provider - not chosen initially**

PlanetScale Postgres, Neon, Supabase, Aiven, and similar services can provide a
managed database while Render runs AppView and Tap.

Pros:

- May provide database branching, different pricing, or specialized operational
  features.
- Keeps database choice independent from the compute platform.

Cons:

- Adds cross-provider networking, latency, egress, credentials, and incident
  boundaries without a demonstrated initial need.
- Transaction-mode poolers are incompatible with CraftSky's session-level
  advisory locks; provider-specific direct connection behavior must be verified.
- Render Postgres already supports the required starting topology and keeps
  traffic on Render's private network.

An external PostgreSQL provider remains a migration option if Render Postgres no
longer meets cost, compatibility, availability, or scale requirements.

**Option 5: Pure serverless functions - not chosen**

Pros:

- Scales request handling automatically and can reduce idle HTTP compute cost.
- Removes management of an always-running HTTP container.

Cons:

- Does not fit Tap's persistent WebSocket and SQLite disk.
- Does not fit the continuous ingestion, publication, push, cleanup, and lifecycle
  workers currently hosted by AppView.
- Would require splitting execution roles, shared admission control, singleton
  ingestion leadership, and new deployment and observability boundaries before
  production.
- An always-running Tap and worker tier would remain, limiting the benefit.

**Option 6: Fly.io, Railway, or a hyperscaler container platform - not chosen initially**

These platforms can run the workload, but do not provide a stronger initial fit
than Render. Fly.io and Railway still require careful singleton-volume operation;
AWS, Google Cloud, and Azure add substantially more infrastructure design and IAM
work. They remain valid migration targets if Render's constraints become material.

### What I decided

CraftSky's initial production backend will use Render in one region:

- AppView runs as one Render web service from the repository's production Docker
  image. It exposes `appview.craftsky.social` through Render-managed HTTPS.
- Tap runs as one Render private service from its pinned Indigo Tap image. It is
  reachable only over Render's private network and has a persistent disk mounted
  at `/data` for `tap.db`.
- PostgreSQL runs as Render Postgres, initially on PostgreSQL 16 to match the
  development and release-tested database version. External database access is
  disabled unless an explicit operational procedure temporarily requires it.
- AppView uses Render Postgres's direct internal connection, not a
  transaction-mode PgBouncer endpoint. Session-level advisory locks must retain
  one database session until explicitly released.
- Database migrations run as a Render pre-deploy command using
  `/app/cli migrate up` before a new AppView version receives traffic.
- The existing AppView background workers remain in the AppView process for the
  initial production release. Moving periodic work to cron or extracting a
  continuous worker service requires a later decision backed by an operational
  need.
- AppView-managed private media uses an external managed S3-compatible object
  store with HTTPS, provider-managed encryption at rest, private access, and
  least-privilege credentials. AWS S3 is the preferred initial candidate, but the
  exact provider and bucket region are deployment choices. Another managed
  S3-compatible provider may be selected without changing this ADR if it meets the
  same security, durability, API, and recovery requirements.
- CraftSky will not self-host PostgreSQL or MinIO for the initial production
  deployment.

The object store covered by this decision initially holds private scheduled-post
media. Canonical public records and blobs remain on users' PDSes. Video upload and
playback remain governed by ADR 012 and ADR 013; this decision does not silently
move canonical atproto media into CraftSky infrastructure.

### Deployment constraints

- AppView, Tap, and PostgreSQL must use the same Render region. Frankfurt is the
  initial preference for the expected UK and European audience.
- AppView remains configured with `APPVIEW_REPLICA_COUNT=1` until shared edge
  admission and multi-replica worker behavior are separately approved.
- Render's default overlapping AppView rollout must not be assumed safe. Before
  production, the deployment specification must either enforce stop-before-start
  replacement or provide evidence that transient overlap preserves Tap, worker,
  OAuth, and admission invariants.
- Render proxy and forwarded-client-address behavior must be tested before setting
  `HTTP_TRUSTED_PROXY_CIDRS`. CraftSky must not trust arbitrary forwarding headers,
  and the process-local per-client limiter must not collapse all traffic onto one
  proxy address.
- Render uses `/health` for service readiness because it returns a non-success
  status when PostgreSQL is unavailable. External monitoring uses `/healthz` and
  inspects the response body for Tap disconnection or stale ingestion because that
  endpoint intentionally always returns HTTP 200.
- The AppView custom domain is the canonical production host used by OAuth
  metadata and callbacks. Any Render-provided public hostname must not become an
  alternate OAuth origin.
- Production secrets live in Render secret environment variables or secret files,
  never in the Blueprint, image, repository, or build arguments.
- AppView's object-store endpoint remains HTTPS. A private bucket policy and
  application credential must grant only the operations and bucket scope needed
  for scheduled media.

### Compatibility and evolution

- Render is a deployment choice, not an application protocol. The AppView remains
  a portable Go container using PostgreSQL and an S3-compatible API.
- Development continues to use Docker Compose with local PostgreSQL, Tap, and
  MinIO. Production does not run that Compose file as one Render service.
- PostgreSQL migrations, SQL semantics, and direct connection behavior remain the
  compatibility boundary. Provider-specific database APIs must not enter AppView.
- S3 object keys and database state must not encode an AWS- or Render-specific
  hostname, account identifier, or storage class that prevents provider migration.
- A future move to another container platform or managed database does not require
  an ADR unless it changes application trust boundaries, persistence semantics,
  availability guarantees, or the ownership of canonical atproto data.
- A future split into HTTP, ingestion, continuous-worker, and cron roles requires
  explicit requirements for leadership, leases, cache invalidation, deployment
  overlap, and monitoring.

### Consequences and notes

- Production infrastructure needs a Render Blueprint or equivalent reviewed
  configuration for AppView, Tap, PostgreSQL, environment groups, health checks,
  domains, and the migration command.
- The current AppView Dockerfile should expose a lean production target so Render
  does not rebuild release-evidence binaries and tests on every deployment. CI
  continues to build the evidence targets with the pinned Go toolchain.
- The Tap disk needs a documented backup classification and recovery procedure.
  Render snapshots are helpful but do not replace a tested repository
  reconciliation plan.
- Render Postgres needs point-in-time recovery, backup retention, connection-limit
  monitoring, storage alerts, and a restore drill before launch.
- The external object store needs lifecycle and recovery rules coordinated with
  scheduled-post database cleanup. Provider lifecycle rules must not independently
  delete objects that AppView still considers live.
- External uptime monitoring and alerting must cover AppView availability,
  PostgreSQL connectivity, Tap connection state, stale `lastEventAt`, queue age,
  disk capacity, and failed migrations.
- Render and object-store costs should be reviewed after real traffic is known.
  Cost alone is not a reason to replace managed persistence without accounting for
  backup, recovery, patching, and operator time.

### Out of scope for this ADR

- Exact Render instance sizes, database plan, disk size, and spend limits.
- Final selection between AWS S3 and another managed S3-compatible provider.
- Backup retention periods, recovery-point objective, recovery-time objective,
  and restore runbooks.
- CI/CD workflow and production promotion or rollback procedure.
- DNS, Cloudflare proxying, transactional email, and app-store distribution.
- Changing AppView's worker topology or adding cron jobs.
- Hosting a CraftSky PDS, Ozone, video transcoding, or a public media CDN.

### Related references

- `atproto-craft-social-app-reference.md` describes the AppView/PDS boundary and
  the original superseded Hetzner recommendation.
- `appview/environments/prod.env.example` defines the production service, OAuth,
  admission, Tap, object-store, and observability constraints.
- `docker-compose.yml` remains the local development topology.
- `docs/roadmap.md` tracks the production deployment, backup, monitoring, DNS,
  and app-distribution work that follows this decision.
- ADR 012 governs the narrow direct video-service authorization handoff.
- ADR 013 governs the standard atproto video embed stored on the PDS.
- Render private services: `https://render.com/docs/private-services`
- Render persistent disks: `https://render.com/docs/disks`
- Render Postgres: `https://render.com/docs/postgresql`
- Render deployments: `https://render.com/docs/deploys`
