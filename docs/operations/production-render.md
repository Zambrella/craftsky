# Production on Render

This runbook implements [ADR 016](../../adr/016-render-managed-production-infrastructure.md).
The source of truth for resource configuration is [`render.yaml`](../../render.yaml).

## Production topology

| Resource | Configuration |
|---|---|
| AppView | Render Docker web service, Frankfurt, 0.5 CPU / 512 MB, one instance |
| Tap | Render private image service, Frankfurt, 0.5 CPU / 512 MB, one instance |
| PostgreSQL | Render Postgres 16, Frankfurt, Basic 1 GB RAM, 5 GB autoscaling storage |
| Scheduled media | Private AWS S3 bucket in `eu-central-1` |

Both compute services have 1 GB persistent disks. Tap stores `/data/tap.db` on
its disk. AppView's disk at `/var/lib/craftsky-deploy-serialization` must remain
empty: it exists only because Render disables overlapping deploys for services
with disks. Application state belongs in PostgreSQL or S3.

The expected baseline Render resource cost is approximately USD 34-36 per month,
plus the USD 25/month Pro workspace fee, before excess bandwidth. AWS S3 is
usage-priced separately.

## Initial setup

### 1. Add Render billing details

The Blueprint cannot validate or create its paid resources until the CraftSky
workspace has a payment method. Add one in the Render Dashboard, then run:

```sh
render blueprints validate
```

The result must contain `"valid": true` before applying the Blueprint.

### 2. Create the S3 bucket

Create a globally unique bucket in `eu-central-1` with:

- Block Public Access enabled for every setting.
- Bucket owner enforced; ACLs disabled.
- Versioning disabled because AppView does not track or delete S3 version IDs.
- Default SSE-S3 encryption, or SSE-KMS if its key operations are also granted.
- No lifecycle expiration rule. AppView owns deletion while database state says
  an object is live.

Create a dedicated IAM principal. Replace `BUCKET_NAME` in this policy before
attaching it:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "InspectCraftSkyScheduledMediaBucket",
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": "arn:aws:s3:::BUCKET_NAME"
    },
    {
      "Sid": "ManageCraftSkyScheduledMediaObjects",
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::BUCKET_NAME/scheduled-media/v2/*"
    }
  ]
}
```

AppView performs `HeadBucket` at startup and then uses `HeadObject`, `GetObject`,
`PutObject`, and `DeleteObject`. Do not grant access to other buckets.

### 3. Generate application secrets

Generate the confidential OAuth key locally:

```sh
just oauth-keygen
```

Generate the 32-byte unpadded base64url handoff key without writing it to disk:

```sh
openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'
```

Enter both outputs directly into Render when prompted. Do not place them in a
shell history, issue, chat message, CI log, or repository file.

### 4. Configure Sentry and Firebase push

AppView sends production errors, safe logs, sampled traces, and metrics to the
configured Sentry project. The DSN and conservative sample rates are declared in
`render.yaml`. `SENTRY_RELEASE` may override the release identifier; when it is
omitted on Render, AppView uses the platform-provided `RENDER_GIT_COMMIT` SHA.
An invalid configured Sentry client fails dependency construction instead of
silently disabling production reporting.

Firebase Admin authenticates with Application Default Credentials. In the
`craftsky-appview` Render service, create a secret file named
`firebase-admin.json` containing the dedicated `craftsky-app` service-account
JSON. Render mounts it at `/etc/secrets/firebase-admin.json`; never store the
JSON in an environment variable, Blueprint, repository, issue, or CI output.
The service account must be limited to sending Firebase Cloud Messaging messages,
and the Firebase Cloud Messaging API must be enabled for `craftsky-app`.

For iOS delivery, upload the production APNs authentication key, key ID, and
Apple team ID to the `social.craftsky.app` iOS application in Firebase Console.
Confirm the signed TestFlight application has the production `aps-environment`
entitlement before relying on production delivery.

### 5. Apply the Blueprint

The Blueprint must be present on `main`. Open:

<https://dashboard.render.com/blueprint/new?repo=https://github.com/Zambrella/craftsky>

Disable Blueprint Auto Sync during creation. Service-level
`autoDeployTrigger: off` does not disable Blueprint Auto Sync. Apply later
Blueprint changes manually only after the same reviewed CI gate used for code;
otherwise a merge can change production outside the tagged promotion workflow.

Before applying, verify all three resources are in Frankfurt and enter:

| Variable | Value source |
|---|---|
| `OAUTH_CLIENT_SECRET_KEY` | `just oauth-keygen` output |
| `OAUTH_HANDOFF_RECEIPT_KEY` | 32-byte base64url output |
| `SCHEDULED_POSTS_S3_BUCKET` | AWS bucket name |
| `SCHEDULED_POSTS_S3_ACCESS_KEY_ID` | Dedicated IAM access key |
| `SCHEDULED_POSTS_S3_SECRET_ACCESS_KEY` | Dedicated IAM secret key |

The AppView service uses Render Postgres's direct internal URL. Do not replace
it with `connectionPoolString` or enable transaction-mode PgBouncer because
owner lifecycle fences use session advisory locks.

The pre-deploy command runs `/app/cli --env prod ping` before migrations. This
constructs the production dependency graph and checks PostgreSQL and S3, so an
invalid OAuth key, handoff key, database connection, or object-store credential
fails before `migrate up` can change the production schema.

The Pro workspace and Blueprint isolate the production environment's private
network boundary and protect it from non-admin destructive changes. Keep these
controls enabled while Tap's private admin endpoints are unauthenticated.

### 6. Verify DNS and TLS

The DNS zone is on Cloudflare. Add the CNAME target Render supplies for
`appview.craftsky.social` with Cloudflare proxying disabled. Verify the domain in
Render and wait for its certificate. Keep the Render subdomain disabled.

Render health probes use a verified custom domain as their `Host`. Before DNS is
verified, AppView permits only `/health` to bypass canonical Host enforcement;
OAuth and `/v1/*` remain unavailable through alternate authorities.

## CI/CD setup

Create a GitHub environment named `production`. Add these environment secrets:

- `RENDER_API_KEY`: a non-expiring Render API key scoped to the operator.
- `RENDER_SERVICE_ID`: the `srv-...` ID of `craftsky-appview`.

Require an environment reviewer for initial releases. Add repository rulesets:

- `main`: require pull requests and the `AppView release gate` check; block
  force-pushes and deletion.
- `prod-v*`: restrict creation to release maintainers and block update/deletion.

`Backend CI` runs the release-equivalent AppView gate for relevant pull requests
and pushes to `main`, and lints the GitHub Actions workflows. Flutter and the
standalone Instagram importer are separate artifacts and retain their own manual
release checks.

## Release

Production has two authorized change paths. Routine application releases use
immutable `prod-v*` tags. Infrastructure changes use a reviewed manual Blueprint
sync after the required `AppView release gate` passes on the current `main` tip.
Before a sync, run `render blueprints validate`, review the planned actions, and
afterward wait for every affected resource to become healthy. A manual sync can
redeploy affected services from `main`; treat and record it as a production
deployment. Auto Sync remains disabled because Render CLI 2.28.0 can validate
Blueprints but cannot apply or synchronize an existing one.

Do not combine routine application changes with Blueprint changes after initial
provisioning: synchronize infrastructure in a dedicated reviewed change before
creating the application release tag. Record the Blueprint sync and resulting
resource configuration in the release notes.

Outside reviewed Blueprint synchronization, only a strict semantic production
tag at the current remote `main` tip deploys the AppView application commit:

```sh
git switch main
git pull --ff-only origin main
git tag -a prod-v0.1.0 -m "Production v0.1.0"
git push origin prod-v0.1.0
```

The workflow retests that exact commit, deploys that SHA through Render's public
HTTP API, waits for a terminal result, and polls bounded public health checks
until PostgreSQL and Tap are healthy. Blueprint validation remains part of the
separate reviewed infrastructure-sync path because an application release does
not synchronize `render.yaml`. Render auto-deploy is off.

Record the tag, commit SHA, Render deploy ID, migration version, and health-check
results in the release notes.

## Post-deploy checks

```sh
curl -fsS https://appview.craftsky.social/health
curl -fsS https://appview.craftsky.social/healthz | jq .
```

Expected:

- `/health` returns `{"status":"ok"}`.
- `/healthz` has `db: "ok"` and `tap.connected: true`.
- `status` can remain `degraded` until Tap receives its first tracked event.

Use Render logs to confirm migrations, S3 bucket connectivity, one Tap consumer,
one copy of each worker, successful Firebase initialization, and no Sentry
initialization failure. Query PostgreSQL after initial migration:

```sql
SELECT current_setting('server_version');
SELECT extname FROM pg_extension WHERE extname = 'pg_trgm';
SELECT version, dirty FROM schema_migrations;
```

Test requests from two independent external networks. Keep
`HTTP_TRUSTED_PROXY_CIDRS` empty unless Render publishes stable immediate-proxy
CIDRs and the observed forwarding chain matches AppView's trust model. The
initial per-client limit equals the global limit so proxy aggregation cannot
create a lower accidental shared bucket.

Install the store-signed Android and iOS builds on physical devices, grant
notification permission, and confirm each device registers through
`POST /v1/notifications/devices`. Generate a real eligible notification from a
second account and verify foreground, background, and terminated delivery. Check
Sentry for the AppView release SHA, production environment, sampled request and
Tap traces, push metrics, and safe push completion logs without device tokens or
other private identifiers.

Before launch, configure an external monitor to poll `/healthz` and alert when
`db != "ok"`, `tap.connected != true`, or a non-empty `tap.last_event_at` is more
than 15 minutes old. Render's `/health` restart probe intentionally covers
database readiness only; it cannot detect ingestion becoming stale after startup.

## Rollback

1. Identify the last known-good commit from Render deploy history and release
   evidence.
2. If the migration is backward compatible, deploy that exact commit manually:

   ```sh
   curl --fail-with-body --request POST \
     --header "Authorization: Bearer $RENDER_API_KEY" \
     --header 'Content-Type: application/json' \
     --data '{"commitId":"COMMIT_SHA","clearCache":"do_not_clear"}' \
     "https://api.render.com/v1/services/SERVICE_ID/deploys"
   ```

3. Do not run a down migration during an incident unless its data-loss behavior
   has been reviewed and a current backup exists.
4. Verify `/health`, `/healthz`, OAuth metadata, and recent error logs.
5. Follow with a new reviewed commit on `main` and a new immutable production tag.

The persistent AppView disk makes rollback stop-before-start, so expect a short
availability interruption.

## PostgreSQL recovery

The selected paid Render Postgres plan provides a three-day PITR window.
Before launch, create an on-demand logical export and perform a restore drill into
a separate database. Confirm the restored migration version and representative
row counts before deleting the drill resource.

Monitor storage, CPU, memory, and active connections. The initial direct pool and
all operational sessions must remain below the Basic plan's 100-connection limit.
Never delete the production database before exporting it; Render deletes its
backups with the instance.

## Tap disk recovery

Render snapshots the disk every 24 hours and retains snapshots for at least seven
days. A snapshot restore replaces the entire disk and loses changes after the
snapshot.

After restoring a snapshot or attaching a fresh disk:

1. Start Tap and confirm its private TCP port is reachable.
2. Start AppView and confirm `tap.connected: true`.
3. Enumerate active member DIDs from `craftsky_profiles`.
4. For each DID, run `/app/cli --env prod tap reconcile DID` in an AppView shell.
5. Run `/app/cli --env prod tap backlog --limit 100` until durable projection and
   repository work converges.
6. Inspect quarantine state and compare representative profile/post counts.

Tap's disk snapshot is an acceleration mechanism, not the sole recovery path.
Canonical records remain on users' PDSes and reconciliation is the recovery
boundary.
