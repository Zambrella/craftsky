# CraftSky architecture

This overview describes the implemented system. Read it alongside the
[architecture decisions](adr/) before changing architectural boundaries.
ADRs record decisions and exceptions; component guides, migrations, Lexicons
and code define the implementation details. Flag discrepancies between them.
This document does not establish which version or optional integrations are
currently enabled in production.

## Components

| Component | Responsibility |
| --- | --- |
| [Flutter app](../app/README.md) | iOS and Android client; calls AppView for application reads and record-write requests |
| [SvelteKit web client](../web-client/README.md) | Browser UI and server-side rendering, with a Cloudflare Worker proxy/session boundary in front of AppView |
| [AppView](../appview/README.md) | Go HTTP API, PDS OAuth, record-write mediation, indexed reads and private application workflows |
| Tap | Consumes federated repository changes and delivers events to AppView over a WebSocket with acknowledgements |
| PostgreSQL | Public-record projections and authoritative private application/session state |
| User's PDS | Authoritative public records and referenced blobs; identity and repository authority remain with the user's account |

Flutter supports mobile platforms only. The browser application is separate
from the public website in `web/`, statistics site in `stats/`, and
service-status Worker. Instagram features live in the clients and AppView.

## Data ownership

Published CraftSky records use the `social.craftsky.*` namespace. Shared Bluesky
records are used where appropriate, including profiles and the social graph.
The [Lexicons](../lexicon/README.md) define public record shapes; changes require
an ADR and regenerated types.

| Data | Authority / storage |
| --- | --- |
| Published posts, public profiles and public graph records | User's PDS; AppView maintains searchable, filtered projections |
| Drafts, scheduled publication state, mutes, saved-content state and push registrations | AppView PostgreSQL |
| OAuth credentials, DPoP keys and CraftSky session/lifecycle state | AppView PostgreSQL, accessed through server-owned session coordination |
| Unpublished scheduled media and controlled safety evidence | Private object storage with their respective access and retention rules |
| Local UI state and native session credentials | Client storage under the client/session lifecycle rules |

PDS repository records and referenced blobs are public; hiding something from
CraftSky does not make the underlying record private. Logged-out visibility is
an AppView presentation policy, not PDS access control. Only the explicit shared
`!no-unauthenticated` profile self-label opts an account out of guest viewing.
Missing or unknown preference state permits guests without a refresh or backfill;
account lifecycle, moderation and media-safety restrictions still apply.
Bluesky chat uses its separate authorization and service boundary, not private
CraftSky PDS records.

Public projections can be reconstructed from authoritative records, subject to
source availability. Private database state and private objects require backups
and recovery; they cannot be reconstructed from the firehose.

## Authentication

AppView is the Token Mediating Backend. A client requests sign-in or registration
from AppView, opens the returned provider authorization URL, and completes the
CraftSky handoff. AppView owns provider discovery, PAR, PKCE, DPoP, the provider
callback, token exchange and refresh. Clients receive CraftSky sessions, not PDS
OAuth access/refresh tokens or DPoP keys.

Native sessions use device-bound requests and secure client storage. The web
client keeps its AppView child-session credentials in host-only HttpOnly cookies
and proxies browser operations through its Worker. AppView remains the session
and membership authority. Browser handoff, same-origin/CSRF protection and native
verified links are described in [ADR 025](adr/025-browser-consumer-handoff.md).

Short-lived upload capabilities are narrow exceptions to the credential boundary;
they are memory-only, destination-bound and never a general PDS session handoff.

## Reads, writes and ingestion

- Application reads come from AppView. Member and anonymous endpoints apply
  their own visibility, moderation and relationship policies.
- Clients ask AppView to create or change records. AppView authorizes the action
  and writes to the owner's PDS; it does not treat a database insert as public
  publication. Relevant mutation paths also reconcile local projections for
  read-after-write behavior.
- PDS changes reach the relay and Tap. AppView persists accepted ingestion work
  before acknowledgement and applies idempotent projections. Events also include
  changes made outside CraftSky; local write results do not replace ingestion.

The client API uses JSON/HTTP under `/v1/`, camelCase fields, structured errors
and opaque pagination cursors. OAuth endpoints retain protocol-specific formats.
See the [API architecture](superpowers/specs/2026-04-21-appview-api-architecture-design.md)
and [wire conventions](superpowers/specs/2026-04-22-api-wire-alignment-design.md).
Schema details belong in [migrations](../appview/migrations/) and
[queries](../appview/queries/), not a second illustrative schema here.

## Media

Immediate native images can upload directly to the session-authorized PDS blob
endpoint with the ephemeral service JWT approved by
[ADR 017](adr/017-direct-image-upload-service-token-handoff.md).
Record publication remains AppView-mediated. Scheduled private media retain
their separate private-storage flow.

AppView selects the video provider. The Bluesky path uses a constrained service
JWT under [ADR 012](adr/012-ephemeral-video-service-token-handoff.md).
The Bunny path uses video-scoped upload authorization, processes video through
Bunny Stream, stores the processed MP4 on the owner's PDS, and keeps an
owner/blob-to-playback mapping in AppView. The public record still references a
standard video blob. Provider verification precedes publication; Bunny asset
cleanup and CDN purge are separate from deleting a user's PDS content.
See [video development and operations](development/bunny-video.md).

## Private workflows and integrations

AppView also owns scheduling, moderation, safety intake and alert delivery,
account lifecycle/deletion, billing entitlements, notification dispatch and
authorized integrations. It is more than a public index. Firebase push, Brevo,
RevenueCat, Instagram, Sentry and video services have separate configuration and
enablement requirements; client UI flags do not replace server authorization.
Follow the [configuration guide](development/configuration.md) and
[logging/error-reporting rules](development/logging-error-reporting.md).

Moderation controls CraftSky visibility and private enforcement state. It must
not delete another user's PDS content. The explicit owner-approved permanent
CraftSky account-deletion flow and owner removal of their referenced profile-link
record are narrow exceptions; their scope is specified in [AGENTS.md](../AGENTS.md).

## Deployment and development

Local backend development uses Docker Compose for AppView, Tap, PostgreSQL,
MinIO, migrations and bootstrap services. The wrapper isolates worktree ports
and volumes. Go tests run on the host against the development services.

Production AppView, Tap and PostgreSQL use Render; private objects use AWS S3.
The web application uses a Cloudflare Worker with static assets. The hosted web
preview uses production AppView and real accounts/data. A frontend rollback
does not undo backend or PDS writes.

AppView releases deploy an exact immutable production tag through the maintainer
scripts. Web releases upload a tagged candidate for review, then manually promote
that same Worker version without rebuilding. See the
[Render runbook](operations/production-render.md),
[release runbook](operations/releases.md) and
[web promotion ADR](adr/026-web-client-preview-promotion.md).

Go/PostgreSQL keep the backend aligned with the Indigo ecosystem and familiar
contributor tooling: `pgx`, SQL migrations, `sqlc`, stdlib HTTP and structured
logging. Flutter provides the native clients; SvelteKit provides the browser
client. These choices do not change the PDS ownership or server-held OAuth
boundaries above.
