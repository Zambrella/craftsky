# CraftSky Lexicon publication

The official publishing account is `craftsky.social`, DID
`did:plc:lmmx63zcns6gewgxqfdt4kof`. Schemas live in its
`com.atproto.lexicon.schema` collection, keyed by their NSID.
The checked-in `lexicon/social/craftsky/` files are the publication source.

From the repository root:

```bash
scripts/lexicon-publication manifest
scripts/lexicon-publication publish
scripts/lexicon-publication verify
```

Publication requires `goat` authenticated as that exact DID. The script checks
account identity and parses every schema before publishing. It does not use
`--update`: changing an existing published schema requires the project's lexicon
review and ADR process. `publish --bootstrap` allows initial publication before
DNS exists, without changing the schema-update policy.

DNS authority is assigned separately for each NSID group. The manifest lists
all eight TXT records with value `did=did:plc:lmmx63zcns6gewgxqfdt4kof`:

| Authority name |
| --- |
| `_lexicon.actor.craftsky.social` |
| `_lexicon.business.craftsky.social` |
| `_lexicon.feed.craftsky.social` |
| `_lexicon.project.craftsky.social` |
| `_lexicon.crochet.project.craftsky.social` |
| `_lexicon.knitting.project.craftsky.social` |
| `_lexicon.quilting.project.craftsky.social` |
| `_lexicon.sewing.project.craftsky.social` |

Cloudflare's `cf` CLI must have zone-read and DNS-read/write access to
`craftsky.social`. Use a local OAuth login, or a zone-restricted API token;
credentials must stay outside repository files and command output.

```bash
cf auth login --scopes zone.read dns.read dns.write offline_access
scripts/lexicon-publication configure-dns --dry-run
scripts/lexicon-publication configure-dns
scripts/lexicon-publication verify
```

DNS setup verifies every published schema, checks existing authorities for
conflicts before creating anything, and creates only missing TXT records.
It leaves matching records unchanged. Verification compares public schema
contents with the checkout and resolves each exact authority name through
public DNS. Cached negative DNS answers can briefly delay verification after
initial publication; rerun verification after propagation.

The runtime validator uses a separate embedded catalog: all CraftSky schemas,
the pinned external address schema, and the transitive dependencies of the
supported Bluesky records. `just lexgen` synchronizes it from checked-in schemas
and the Indigo version in `go.mod`; `just lexgen-check` checks for drift.

References: [Lexicon publication and resolution](https://atproto.com/specs/lexicon#lexicon-publication-and-resolution),
[Cloudflare CLI](https://github.com/cloudflare/cf/blob/main/packages/cli/README.md).
