# 022. Shared Bluesky profile links

Date: 2026-10-27

Status: Accepted for implementation

CraftSky will use the published `app.bsky.actor.link` record and the ordered
`app.bsky.actor.profile.links` strong references, rather than define a separate
CraftSky link collection. This keeps profile links portable between clients.

Schemas and behavior are pinned to Bluesky social-app commit
`0d30bbead1df8409fac03370e4097bb8f33000fa` (October 2026). The record permits ten
references, titles of 40 graphemes / 320 bytes, optional 100,000-byte PNG/JPEG/WebP
icons, self-labels, an HTTPS destination and a creation timestamp. The editor
uses Bluesky's narrower 30-grapheme title limit. Support styling is derived
from destination hosts and account paths, never stored as record metadata.

All Flutter reads and record writes remain AppView-mediated. OAuth credentials
remain server-side. Existing command idempotency, owner fencing, repository
compare-and-swap, URL-fetch protections and image clearance apply to links.
The profile's reference order is authoritative; unreferenced, missing, foreign,
unsafe or moderated links are not public profile content.

Adding the link collection to OAuth authority requires a fresh consent for
existing sessions. Production migrations, deployment and live-account checks
remain separate maintainer operations.

Removal from the profile is distinct from deleting its public PDS record.
PDS cleanup must follow the explicitly authorized owner-removal policy and
must never become a moderation, retention or generic deletion mechanism.
