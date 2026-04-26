# Lexicon → Go types codegen — design

Date: 2026-04-26
Status: implemented (CI wiring deferred — see §3.9)

## 1. Problem

The AppView hand-rolls a narrow Go struct for every record shape it needs to read off the firehose. Today there is one such struct ([`craftskyProfileRecord` in `appview/internal/index/craftsky_profile.go:49`](../../../appview/internal/index/craftsky_profile.go#L49)), but `feed.post`, `feed.like`, `feed.repost`, and per-craft `project.*` indexers are imminent. Each one will repeat the pattern: read the JSON in `lexicon/social/craftsky/**/*.json`, transcribe a subset of the fields into a Go struct in the indexer file, keep the two in sync by code review.

This is fine for one tiny record. It is bad for the post lexicon: `feed.post` has nested `project.{common, details}` with an open union on `details` keyed by craft ([ADR 001](../../../adr/001-post-lexicon-project-extensibility.md)). Transcribing the `$type` discriminator dispatch by hand for every new craft sub-lexicon is the kind of work that drifts from the schema silently and is only caught by the AppView dropping records that should have indexed.

The lexicon JSON should be the single source of truth for record shape. The Go types should be derived from it.

## 2. Goal

Stand up a `lexgen` step that reads `lexicon/social/craftsky/**/*.json`, generates a Go package of typed structs (one per record / def), checks the output into the repo, and runs in CI as a drift guard. Migrate `craftskyProfileRecord` to the generated type as the worked example.

Non-goals:

- **Generating the indexers themselves.** Codegen produces the wire-shape types (`json.Unmarshal` targets). The indexer logic — what to upsert, what to ignore, idempotency keys — stays hand-written, because it encodes AppView policy not lexicon shape.
- **Codegen for `app.bsky.*` or `com.atproto.*` types.** Those already exist in indigo (`github.com/bluesky-social/indigo/api/{bsky,atproto}`); we re-export nothing and import the indigo packages directly when the generator emits a cross-package ref.
- **A second wire format.** No protobuf, no flatbuffers. JSON only at the indexer boundary; CBOR support comes from the same generator chain but is unused today.
- **A custom fork of `lexgen`.** Output is taken as-is, including the `init()` registration calls and the union dispatch shape. We adapt to the generator, not the other way around.
- **Sharing generated types with the Flutter client.** The Dart side has its own codegen path via `atproto.dart`. This spec is Go-only.
- **Generating XRPC client/server scaffolding.** The AppView reads firehose records; it does not call XRPC. We pass `lexgen` without `--gen-server`.

## 3. Approach

### 3.1 Tooling: indigo's `cmd/lexgen` + `cbor-gen`

We already depend on `github.com/bluesky-social/indigo`. Its `cmd/lexgen` is the de facto Go lexicon code generator and is what Bluesky uses to maintain its own `api/bsky` package. No third-party generator is competitive; adopting `lexgen` keeps us on the same toolchain as the rest of the atproto Go ecosystem.

`lexgen` always emits an `init()` block that registers each generated type into a global `lexutil` registry. The registry function signature requires `cbg.CBORMarshaler`, so the generated package will not compile from JSON-shape files alone — `MarshalCBOR` / `UnmarshalCBOR` methods must exist. Two ways to satisfy this:

- **Run `cbor-gen` as a paired step** (indigo's pattern). A small Go main lists the types and `cbor-gen` writes a single `cbor_gen.go` next to them with the methods.
- **Strip the `init()` block and union CBOR methods in a post-process** (`sed` / `awk`).

We pick the first. It matches indigo's setup, has no fragile string-matching, and gives us CBOR support as a free side-benefit if we ever need to validate against PDS-canonical bytes directly. Cost is one ~30 LOC file (`cmd/lexgen/cborgen/main.go`) listing the types — updated when a record is added, which is the same cadence as a lexicon edit anyway.

### 3.2 File layout

```
appview/
  cmd/
    lexgen/
      build.json              # package config (prefix → outdir + import path)
      cborgen/
        main.go               # cbor-gen runner — enumerates types
  internal/
    lexicon/
      craftsky/               # generated package — checked in
        actorprofile.go       # ← lexgen output
        feedpost.go           # ← lexgen output
        feedlike.go           # ← lexgen output
        feedrepost.go         # ← lexgen output
        projectsewing.go      # ← lexgen output
        feeddefs.go           # ← lexgen output (empty for token-only schemas)
        sewingdefs.go         # ← lexgen output (empty)
        cbor_gen.go           # ← cbor-gen output
        doc.go                # package comment, import-side ergonomics notes
```

`internal/lexicon/craftsky/` is the only generated location. `cmd/lexgen/` holds the inputs to the toolchain — the build config and the `cborgen` runner. There is no `cmd/lexgen/main.go`; we invoke indigo's binary via `go run github.com/bluesky-social/indigo/cmd/lexgen` from the just recipe (no need for a wrapper).

### 3.3 Build config

`appview/cmd/lexgen/build.json`:

```json
[
  {
    "package": "craftsky",
    "prefix": "social.craftsky",
    "outdir": "internal/lexicon/craftsky",
    "import": "social.craftsky/appview/internal/lexicon/craftsky"
  },
  {
    "package": "bsky",
    "prefix": "app.bsky",
    "outdir": "_external_bsky_unused",
    "import": "github.com/bluesky-social/indigo/api/bsky"
  },
  {
    "package": "atproto",
    "prefix": "com.atproto",
    "outdir": "_external_atproto_unused",
    "import": "github.com/bluesky-social/indigo/api/atproto"
  }
]
```

Only the first entry has files emitted — the other two exist purely to tell `lexgen` which import path to write for refs into `app.bsky.*` and `com.atproto.*`. The `outdir` values for those entries are placeholder paths that no schemas resolve to (the prototype confirmed that `lexgen` only writes to outdirs that match an input schema's prefix).

### 3.4 External lexicons

The generator needs to *resolve* refs to `app.bsky.richtext.facet` and `com.atproto.repo.strongRef` (used by `feed.post`, `feed.like`, `feed.repost`) so it can emit correct cross-package types — but it does not need to emit code for them. We pass them via `--external-lexicons`, sourced from the indigo module cache pinned to the version in `go.mod`:

```
INDIGO_DIR=$(go list -m -f '{{.Dir}}' github.com/bluesky-social/indigo)
--external-lexicons "$INDIGO_DIR/lexicons/app/bsky/richtext/facet.json"
--external-lexicons "$INDIGO_DIR/lexicons/com/atproto/repo/strongRef.json"
```

This keeps the external-lexicon version automatically in lockstep with the indigo Go API we generate against. No vendoring; no separate version pin.

If a future lexicon adds a ref to a new `app.bsky.*` or `com.atproto.*` type, the recipe gains another `--external-lexicons` line. We do *not* pass the entire `lexicons/app/bsky/` tree — over-supplying external lexicons can change `lexgen`'s behaviour around which packages are considered known, and we want the set of references to be explicit and reviewable.

### 3.5 The just recipe

`justfile`:

```just
# Regenerate Go types from lexicon/ JSON schemas.
# Runs in two phases: lexgen (struct shapes) → cbor-gen (CBOR methods).
# Both phases overwrite checked-in files; commit the result.
lexgen:
    cd appview && \
      INDIGO_DIR=$(go list -m -f '{{{{.Dir}}}}' github.com/bluesky-social/indigo) && \
      go run github.com/bluesky-social/indigo/cmd/lexgen \
        --build-file cmd/lexgen/build.json \
        --external-lexicons "$INDIGO_DIR/lexicons/app/bsky/richtext/facet.json" \
        --external-lexicons "$INDIGO_DIR/lexicons/com/atproto/repo/strongRef.json" \
        ../lexicon/social/craftsky && \
      go run ./cmd/lexgen/cborgen && \
      gofmt -w internal/lexicon/craftsky

# Drift guard: regenerate and fail if the working tree changes.
# Run in CI on every push.
lexgen-check: lexgen
    cd appview && git diff --exit-code internal/lexicon/craftsky cmd/lexgen
```

Two recipes: the human-facing `lexgen` (regenerate + commit) and the CI-facing `lexgen-check` (regenerate + fail on diff). The two-phase invocation is sequential because `cborgen` imports the lexgen output package.

### 3.6 Bootstrap

`cbor-gen`'s runner imports the generated package, but the generated package's `init()` calls require `MarshalCBOR` / `UnmarshalCBOR` to compile, which `cbor-gen` is what produces. The first `just lexgen` would deadlock.

The unblocker is a one-time stub `internal/lexicon/craftsky/cbor_gen.go` checked in alongside the first batch of generated files. The stub provides empty `MarshalCBOR` / `UnmarshalCBOR` methods on every type referenced by `init()` and union dispatch. With the stub in place, the package compiles, `cborgen` runs, and `cborgen` overwrites the stub with the real methods. From the second `just lexgen` onwards, the previous run's `cbor_gen.go` is the bootstrap input for the next, and the cycle is self-sustaining.

The stub does not need to be a separate file — we just commit the bootstrap output as the initial `cbor_gen.go`. The phrase "stub" only matters during the first commit; after that it is the real generated file.

When a new record type is added: add it to the type list in `cmd/lexgen/cborgen/main.go` *before* running the recipe. The first run will fail to compile the cborgen runner because the type does not yet have CBOR methods; this is solved the same way as the original bootstrap — manually add a temporary stub method for the new type, run `just lexgen`, the real method overwrites the stub. This is mildly annoying.

The "delete `cbor_gen.go` before each run" workaround that was floated in the spec draft does *not* work — confirmed during implementation. The cborgen runner imports the lexicon package, and the package's own `init()` registrations and union dispatch require the CBOR methods at compile time. With `cbor_gen.go` absent, neither phase can run. The previous `cbor_gen.go` is the only thing that breaks the cycle, so the manual stub-on-new-types workflow stays.

### 3.7 Generated-file conventions

- The generated package lives under `internal/lexicon/craftsky/` so it is not part of the AppView's public Go API. Indexers and storage layers import it; nothing outside `appview/` does.
- `.gitattributes` marks `internal/lexicon/craftsky/*.go` and `appview/cmd/lexgen/cborgen/main.go` as `linguist-generated=true` so GitHub diffs collapse them by default. They are *not* listed in `.gitignore` — these files are checked in.
- A hand-written `internal/lexicon/craftsky/doc.go` documents the package, points at this spec, and notes that any change to a `.go` file under this directory will be overwritten on the next `just lexgen`.
- The `// Code generated by cmd/lexgen … DO NOT EDIT.` header that `lexgen` already emits is the authoritative warning for the per-file headers.

### 3.8 Migrating `craftskyProfileRecord`

[`appview/internal/index/craftsky_profile.go:49-51`](../../../appview/internal/index/craftsky_profile.go#L49) declares:

```go
type craftskyProfileRecord struct {
    Crafts []string `json:"crafts"`
}
```

After this spec lands, the indexer imports the generated package and consumes the wider type:

```go
import craftskylex "social.craftsky/appview/internal/lexicon/craftsky"

// inside Handle:
var rec craftskylex.ActorProfile
if err := json.Unmarshal(ev.Record, &rec); err != nil {
    return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
}
if rec.Crafts == nil {
    rec.Crafts = []string{}
}
// ... unchanged from here
```

The hand-rolled `craftskyProfileRecord` type is deleted. The "wider type that parses fields the indexer doesn't use" trade-off (§4.2) is conscious: drift between the lexicon and the type vector becomes structurally impossible, and the cost is a few extra `json.Unmarshal` cycles on a payload that is already cheap to parse.

The existing test ([`craftsky_profile_test.go`](../../../appview/internal/index/craftsky_profile_test.go)) needs no changes — it constructs `tap.Event{Record: json.RawMessage(...)}` payloads and asserts on what the indexer writes to Postgres. The wire format is unchanged; only the unmarshal target swaps.

### 3.9 CI integration

There is no CI workflow in this repo yet. The `just lexgen-check` recipe is committed alongside this spec so that whenever CI is stood up, wiring the drift guard is a one-liner: add a step that runs `just lexgen-check` after `go mod download` (so the indigo module cache is populated). A failure means a contributor edited `lexicon/` without running `just lexgen`, or edited a generated file by hand. The fix message in the failure output should say: "run `just lexgen` and commit the result."

### 3.10 What the generated package looks like

For reference (taken from the prototype, output verified to compile and JSON-round-trip on every interesting case):

- **`ActorProfile`** — single field (`Crafts []string`), `$type` const tag, `key: literal:self` produces no special handling (just a normal record struct).
- **`FeedPost`** — top-level struct with nested `FeedPost_Project`, `FeedPost_ProjectCommon`, `FeedPost_Pattern`, `FeedPost_Image`, `FeedPost_QuoteEmbed`, `FeedPost_ReplyRef` types in the same file. Cross-package refs (`*appbsky.RichtextFacet`, `*comatproto.RepoStrongRef`) emit correctly.
- **Open union dispatch** — for `FeedPost.Embed` and `FeedPost.Project.Details`, the generator emits a wrapper struct with one `*Variant` field per known union member plus `MarshalJSON` / `UnmarshalJSON` methods that switch on the `$type` discriminator. Unknown variants unmarshal silently to nil — the open-union semantics we want.
- **Empty `defs` files** — `feed.defs` and `project.sewing.defs` contain only `token` types; they generate empty `.go` files (just the package clause). Harmless.

## 4. Trade-offs

### 4.1 Generated naming

Type names are mechanical — `FeedPost_Project_Details`, `FeedPost_QuoteEmbed`, `ProjectSewing_Details`. This is uglier than what we would write by hand. We accept it because: (a) the names appear only in indexer code, not in HTTP responses, (b) IDE auto-import smooths the verbosity at use sites, and (c) the alternative is a hand-maintained translation table between lexicon NSIDs and Go names — exactly the drift we are trying to eliminate.

If a particular name becomes painful, the indexer can introduce a local type alias (`type ActorProfile = craftskylex.ActorProfile`) to shorten it without modifying the generated code.

### 4.2 Wide structs, narrow indexers

A hand-written `craftskyProfileRecord` parses only the fields the indexer cares about — implicit documentation of indexer scope. Generated structs parse everything in the schema. We lose that documentation property. We pay a few microseconds per event to materialise unused fields. We gain: when a lexicon adds a field that the indexer *should* be reading, the change is visible in the generated package and the omission becomes a code-review item rather than an invisible non-action.

### 4.3 Bootstrap awkwardness

§3.6 describes a one-time bootstrap and a recurring annoyance when adding new types. This is the price of running both `lexgen` and `cbor-gen` against types that mutually reference each other. The open question in §3.6 may resolve it; if not, the workaround is documented and tractable.

### 4.4 Test payloads written by hand

Indexer tests construct JSON payloads as `[]byte` literals. This continues — the test is asserting "given this wire payload, the indexer does X", and using the generated type to *construct* the test payload would couple the test to the type rather than the wire. The generated type is the indexer's input contract, not its test fixture format.

## 5. Acceptance bar

This spec is done when:

1. `appview/cmd/lexgen/build.json` and `appview/cmd/lexgen/cborgen/main.go` exist with the contents described in §3.3 and §3.1.
2. `appview/internal/lexicon/craftsky/` exists with the generated `*.go` files and a hand-written `doc.go`.
3. `just lexgen` from a clean checkout runs to completion and produces no diff (idempotent regeneration).
4. `just lexgen-check` is wired into CI and fails on a deliberate test edit (e.g. adding a field to `lexicon/social/craftsky/actor/profile.json` and not running `just lexgen`).
5. [`craftsky_profile.go:49`](../../../appview/internal/index/craftsky_profile.go#L49)'s `craftskyProfileRecord` is deleted; the indexer imports `social.craftsky/appview/internal/lexicon/craftsky` and uses `ActorProfile` as the unmarshal target.
6. `just test` passes — the existing `craftsky_profile_test.go` exercises the migration without modification.
7. `AGENTS.md` grows one line under "Coding Conventions" pointing at this spec, of the shape: "Lexicon-derived Go types are generated by `just lexgen` into `appview/internal/lexicon/craftsky/`. Hand-rolled record structs in indexer files are no longer accepted — see [`docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md`](docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md)."

## 6. Out of scope, explicitly tracked

- **A future lexicon snapshot tool** that vendors `app.bsky.*` and `com.atproto.*` schemas into `lexicon/external/` so the recipe doesn't reach into `GOMODCACHE`. Tracked as a possibility; not done here because the module-cache approach has zero maintenance cost and stays version-locked automatically.
- **Generating Dart types from the same lexicons** for the Flutter client. Separate concern, separate toolchain (`atproto.dart` already does this).
- **Replacing the rest of the would-be hand-rolled record structs** for `feed.post`, `feed.like`, `feed.repost`, etc. They will simply use the generated types from day one — no migration burden because they don't exist yet.
