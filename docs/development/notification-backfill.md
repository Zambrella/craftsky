# Post notifications during backfill

Durable ingestion projects a CraftSky post and its notifications separately.
The serving projection commits the post and a `post_notifications` job in the
same transaction. A missing notification dependency therefore does not hide
the post or mark its notification work complete.

The notification job waits for the referenced reply parent or quoted post
(`subject_uri`), and for recipient membership (`member_did`). Post and profile
projection wake these dependencies. Blocked notification jobs also retry once
a minute to recover a dependency wake that races with registering the blocked
job. Available recipients can receive notifications while another mentioned
recipient is still waiting for membership.

Notification work passes through the existing source-version, owner lifecycle,
record validation and image-safety gates. Updates and deletes invalidate old
notification leases and dependencies. Retries reuse the notification service's
duplicate protection, category precedence, preferences and relationship rules.
The activity date remains the post's original `createdAt`, including during
backfill; the existing push delivery deadline remains based on that date.

Direct indexer calls retain immediate notification generation. Durable worker
claims supply a projection kind so the post projector can distinguish serving
work from notification work. Operator source/repository repair queries continue
to select the serving job, while the ordinary worker handles auxiliary jobs.

This change uses the existing ingestion job schema. It does not add an API,
lexicon or database migration, and does not requeue previously completed posts
to repair historical omissions. Testing an initial backfill requires a fresh
local environment.
