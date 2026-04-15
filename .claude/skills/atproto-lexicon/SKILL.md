---
name: atproto-lexicon
description: Use when designing, writing, reviewing, or changing AT Protocol Lexicon schemas in the lexicon/ directory — covers NSID naming, type system, style conventions, and schema evolution rules. Invoke before adding or editing any file under lexicon/, or when a change would affect record shapes on a PDS.
---

# AT Protocol Lexicon

A distillation of the three canonical references:

- <https://atproto.com/guides/lexicon>
- <https://atproto.com/guides/lexicon-style-guide>
- <https://atproto.com/specs/lexicon>

Use this when touching anything under `lexicon/`. Lexicons are **load-bearing**: once records are written to real PDSes under a given schema, you cannot retroactively change it without a new NSID. Design carefully up front.

## Mental Model

- A Lexicon is a JSON file describing **one NSID**, containing one or more named definitions under `defs`.
- Definitions are either **primary** (`record`, `query`, `procedure`, `subscription`, `permission-set`) — at most one per file, conventionally named `main` — or **reusable sub-defs** (objects, tokens, etc.) with other names.
- The file's NSID (e.g. `social.craftsky.craft.post`) resolves to an authority via DNS TXT (`_lexicon` record on the domain).
- Records in a PDS are just JSON/CBOR objects that conform to a Lexicon and carry a `$type` discriminator.

Required top-level fields in every Lexicon file:

| Field | Type | Notes |
|---|---|---|
| `lexicon` | int | Always `1` for this spec version |
| `id` | string | The NSID of this file |
| `defs` | object | Map of name → definition (must be non-empty) |
| `description` | string | Optional but recommended, especially on `main` |

## Type System

| Category | Types |
|---|---|
| Concrete | `boolean`, `integer`, `string`, `bytes`, `cid-link`, `blob` |
| Container | `array`, `object` |
| Sub-type | `params` (query params only), `permission` (permission-set only) |
| Meta | `token`, `ref`, `union`, `unknown` |
| Primary | `record`, `query`, `procedure`, `subscription`, `permission-set` |

Key constraints:

- **`ref`** points to another def by NSID (optionally `#name`) or local fragment (`#name`). Can't point to another `ref`, `union`, or `token`.
- **`union`** defaults to **open** (future-extensible). Use `"closed": true` only when you genuinely want to freeze the variant list — and you almost never do. Variants must be `object` or `record`.
- **`token`** is an empty named value for use in `knownValues` / `enum` — references must be fully qualified NSIDs (e.g. `com.example.defs#tokenOne`).
- **`unknown`** accepts any object with no schema validation. **Avoid in `record` definitions.**
- **`$type` is required** in: record objects (always), union variants (always, except top-level subscription messages), blob objects. Use bare NSID for `main` (never `#main`).

## NSID & Naming

- **NSIDs** are reverse-DNS (`social.craftsky.craft.post`). Authority is rooted in DNS control.
- **Schemas & field names:** `lowerCamelCase`, ASCII alphanumeric, first char not a digit, no hyphens, case-sensitive. Don't start data fields with `$` (reserved).
- **API error names:** `UpperCamelCase`.
- **Fixed strings in `knownValues`:** `kebab-case`.
- **Records:** singular nouns — `post`, `like`, `profile`.
- **Queries/procedures:** `verbNoun` — `getPost`, `listLikes`, `createReport`. Common verbs: `get`, `list`, `search`, `query` (queries); `create`, `update`, `delete`, `upsert`, `put` (procedures).
- **Subscriptions:** `subscribePluralNoun` — `subscribeLabels`.
- **Experimental/unstable:** put `.temp.` or `.unspecced.` in the hierarchy.
- **Shared defs file:** `{group}.defs` (e.g. `social.craftsky.feed.defs`). `defs` files generally should not have a `main`.
- **Avoid collisions** like `app.bsky.feed.post#main` vs `app.bsky.feed.post.main`, or defining both `social.craftsky.feed` (a record) and `social.craftsky.feed.post` (a group).
- **Avoid programming-language keywords** like `default` or `length` as field names.

## String Constraints

- Always set `maxLength` on free-form strings (unless a `format` implies bounds). For visible text, add `maxGraphemes` too — graphemes match human "characters" across languages, bytes don't.
  - Recommended ratio: **10–20 bytes per grapheme** (so `maxGraphemes: 300`, `maxLength: 3000` or similar).
- Don't redundantly set both `format` and length limits.
- Use `format` when it applies — see the list below.
- For **large text or binary data**, use a `blob` reference, not `string`/`bytes`.
- Prefer `knownValues` (open list of suggestions, optionally pointing to tokens) over `enum` (closed set). `enum` is stuck forever.
- Tokens in `knownValues` let you add new recognized values over time without breaking old clients. Great for categories, reasons, statuses.

### Supported string formats

`at-identifier`, `at-uri`, `cid`, `datetime`, `did`, `handle`, `nsid`, `tid`, `record-key`, `uri`, `language`.

Notes:
- `datetime`: RFC 3339 / ISO 8601 intersection. Upper-case `T` separator, required timezone (prefer `Z`), whole-second precision required. Round-trip as **string**, not a native type, to avoid precision drift breaking hashes.
- For account identifiers in **query params**, prefer `at-identifier` (accepts either handle or DID) — lets clients skip a `resolveHandle` call.
- For account references in **records**, always use `did`, not `handle`. Handles change; DIDs don't.

## Objects & Nullability

- `required: [...]` lists non-optional fields. Once set, a field can't become optional or be removed — only V2 schemas can fix that.
- `nullable: [...]` lists fields that may be explicitly `null`. Three distinct semantic states: absent, `null`, falsey value (`false`, `0`, `""`, `[]`).
- Only mark a field `required` if it's genuinely load-bearing for every record. Err toward optional.

## Primary Type Cheatsheet

### `record`
- `key`: record key type (see record-key spec; common values: `tid`, `literal:self`, `any`)
- `record`: an `object` definition describing the payload
- Records with a single known instance per account (e.g. a profile) should use `literal:self` and a stable record key.

### `query` / `procedure`
- `parameters`: `params` type — HTTP query string. Restricted to `boolean`, `integer`, `string`, or arrays of those.
- `output`: `{ encoding, schema? }`. Always specify `output` with `encoding` even if no body schema — use `application/json` with an empty object schema as the fallback.
- `input` (procedures only): same shape as `output`.
- `errors`: array of `{ name, description? }`. Names are `UpperCamelCase`.
- Auth requirements and personalization behavior should live in the description.

### `subscription`
- `parameters`: same shape as query.
- `message.schema`: **must** be a `union` of refs. Variants don't need `$type` at the top level of subscription messages.
- Sequencing convention: include a monotonic `seq: integer` in each message; support a `cursor` query param for backfill.

### `permission-set`
- Bundle of `permission` entries for OAuth scopes. Fields: `title`, `detail`, `permissions`, plus `title:lang` / `detail:lang` maps for i18n.
- Permission resource types in lexicon context: `repo` (write access to collections), `rpc` (XRPC proxying). `blob`, `account`, `identity` are not supported in lexicon — ignore if present.

## Style & Convention Checklist

- [ ] `main` defs have a `description`.
- [ ] Ambiguously-named fields (`uri`, `cid`, `ref`) have descriptions clarifying what they refer to.
- [ ] String fields have `maxLength` (and `maxGraphemes` for visible text), unless a `format` covers them.
- [ ] No `enum` for extensible value sets — use `knownValues` + tokens.
- [ ] Unions are open (no `closed: true`) unless you're absolutely sure the variant set can never grow.
- [ ] Record fields referring to accounts use `did`, not `handle`.
- [ ] Query params for accounts use `at-identifier`.
- [ ] Boolean flags default to `false` (phrase them so the default is the common case — `excludeFoo`, not `includeFoo`).
- [ ] Arrays of data use arrays of **objects** (with a named element field like `account`), not arrays of scalars — leaves room for future context.
- [ ] Shared definitions live in a `{group}.defs` file, not duplicated.
- [ ] Reusing `com.atproto.repo.strongRef` for versioned record references, and `com.atproto.label.defs#label` where labels apply — don't reinvent these.
- [ ] Hydrated "view" responses include the original record verbatim rather than a superset schema (easier to maintain, doesn't strip extension data).
- [ ] Viewer-specific fields (e.g. "did the current user like this?") are optional and grouped under a sub-object so the same schema works for public and logged-in views.

## Evolution Rules (memorize these)

All old data must still validate under the updated Lexicon, and new data must validate under the old one. Concretely:

- ✅ You **can** add new **optional** fields.
- ✅ You **can** add variants to an **open** union.
- ✅ You **can** add values to `knownValues`.
- ❌ You **cannot** add new `required` fields.
- ❌ You **cannot** remove or rename fields. Mark them deprecated in the description instead.
- ❌ You **cannot** change a field's type.
- ❌ You **cannot** tighten a constraint (narrowing `maxLength`, adding `required`, closing a union, etc.).
- ❌ You **cannot** loosen a constraint in a way that breaks old consumers either — old clients will reject data they don't recognize.

If you need a breaking change: **publish a new NSID** (convention: append `V2`, `V3`, …).

A Lexicon effectively becomes "set in stone" once **anyone else** starts using it. For pre-launch or experimental work, use `.temp.` or `.unspecced.` in the NSID so it's clear things can still break.

## Pagination Convention

For list-style queries:

- Inputs: optional `limit: integer`, optional `cursor: string`.
- Output: required array (pluralized, domain-specific field name), optional `cursor: string`.
- First request omits `cursor`. Responses with a `cursor` mean "more results available; pass this back."
- Absence of `cursor` in the response means pagination is complete, **not** that `limit` was reached — results may be filtered, tombstoned, or just sparse.

## Extension Patterns

Two idiomatic ways to extend records without breaking them:

- **Open unions** for polymorphic fields (e.g. a feed item that might be a post, repost, or future variant).
- **Sidecar records** — a separate record type in the same repo with the **same record key**. Lets a third party (or your own future self) attach data to an existing record without mutating its CID or breaking strong references.

App modality signaling: define a "profile" or "declaration" record with a known fixed record key (e.g. `self`) to signal that an account actively uses your app. Deletion = opted out. Backfill services can enumerate all accounts in the network with that record.

## When Authoring a New Lexicon — Workflow

1. **Decide the NSID.** Group by feature (`social.craftsky.feed.*`, `social.craftsky.graph.*`, `social.craftsky.actor.*`). Use `.temp.` if you're not ready to commit.
2. **Write an ADR.** Lexicon design decisions are load-bearing — use the `writing-architecture-decision-records` skill and capture what you considered, why you chose these fields, and what's explicitly out of scope.
3. **Draft the schema.** Start from a sibling Bluesky Lexicon (`app.bsky.feed.post`, `app.bsky.actor.profile`, `app.bsky.graph.follow`) as a structural reference, then adapt to the craft domain.
4. **Stress-test the evolution path.** For every field: "What if we want to remove this? Add context? Allow multiple values?" If any of those require a `V2`, reconsider now.
5. **Cross-check the style checklist above.**
6. **Review record shape end-to-end.** Trace: `app` → `appview` → write to PDS → firehose → index into Postgres. Do the field types make sense at every hop?
7. **Commit with an ADR reference** in the message.

## Bluesky Lexicons as Reference

When in doubt, read how `app.bsky` solves a similar problem:

- <https://github.com/bluesky-social/atproto/tree/main/lexicons>

Useful specific references:
- `app.bsky.feed.post` — the canonical "text + media + facets" record
- `app.bsky.actor.profile` — single-record-per-user pattern with fixed record key
- `app.bsky.graph.follow` / `app.bsky.graph.block` — minimal social graph records
- `app.bsky.feed.defs` — a good example of a shared `.defs` file
- `com.atproto.repo.strongRef` — reuse for versioned record references
- `app.bsky.richtext.facet` — reusable text-annotation type (mentions, links, tags)

Prefer reusing these where semantics match. Only redefine when the craft domain genuinely differs.
