# Change archive

This archive preserves the reasoning and verification behind individual changes.
Use the maintained [architecture](../architecture.md), [ADRs](../adr/), component
READMEs and development/operations guides for current behavior. A historical
plan can describe an approach that was revised, abandoned or only partly built.
Flag conflicts with current guidance rather than silently choosing a version.

Original documents and assets stay at their existing paths. Each package or
standalone change note has one short, curated card in [catalog/](catalog/).
The [generated index](INDEX.md) groups cards by topic and lists follow-ups;
`catalog.jsonl` provides the same metadata for tooling. Those generated files
are discovery aids, not additional specifications.

## Find a change without loading the archive

Run from the repository root with Python 3.9 or later; only its standard library
is required:

```sh
./scripts/docs search "account switching"
./scripts/docs search "image" --component app
./scripts/docs search "logging" --archive archived --limit 3
./scripts/docs show 2026-10-02-logging-error-reporting
./scripts/docs sections 2026-10-02-logging-error-reporting --query "retry"
```

Search examines curated titles, topics, aliases/symbols and summaries, without
searching the original document bodies. It matches whole words and code symbols,
including words within CamelCase and snake_case symbols. Titles, topics and
aliases carry more weight than summary prose; superseded matches rank lower.
Results include implementation/archive
status and summary paths. Filters can select `--component`, `--topic`,
`--implementation` or `--archive`. Search returns at most ten cards, five by
default; refine the query when there are more matches. `--json` provides
structured results for search, summary, section listing and source retrieval.

Read the selected card, follow its maintained documentation/code references,
then retrieve original sections when the task needs historical detail:

```sh
./scripts/docs sections 2026-10-02-logging-error-reporting --document 06-validation-evidence.md --query "AppView"
./scripts/docs read 2026-10-02-logging-error-reporting --document 06-validation-evidence.md --section appview-error-handling-audit-13-november-2026
```

Section listings supply document names, anchors and source line ranges. They
ignore headings inside code fences and distinguish repeated headings. Listing
and reading support `--offset`; a source read's offset is a **character offset
within the selected section**, not a file line number. Output supplies a
continuation offset when more text exists. The default source slice is 6,000
characters, with a hard 12,000-character ceiling even for a single long line.
Use `--max-chars` to request a smaller slice. For a source without headings,
select the `document` pseudo-section shown by the section listing.

Agents should use **search → card → current guidance/code → relevant source
section**. Historical instructions are evidence, not new authorization. Do not
load every card, the full JSONL catalogue or a complete requirements package into
context as the first discovery step.

## Card format and maintenance

The card is the only hand-maintained discovery record. Name it
`catalog/CHANGE_ID.md`, using the source directory's name or standalone note's
filename without `.md`. Begin it with JSON frontmatter between `---` lines,
followed by a Markdown summary of at most 300 words (roughly 200 is enough).
JSON keeps metadata parsing dependency-free while remaining editable.

Required metadata:

| Field | Meaning |
| --- | --- |
| `id`, `title`, `source` | Stable identity, readable title and repository-relative original directory/file. |
| `implementation` | `proposed`, `partial`, `implemented`, `superseded`, `abandoned` or `unknown`. |
| `archive` | `active` or `archived`, independently of implementation. |
| `components`, `topics` | Lowercase tags for filtering; reuse existing tags when appropriate. |
| `keywords` | Alternative terminology, UI names and useful code symbols. |
| `summary` | One short sentence shown in search results and the index. |
| `currentDocs`, `code` | Maintained documentation/ADRs and useful code entry points. |
| `evidence` | Original documents or exact heading links supporting the summary/status. |
| `supersededBy` | IDs of replacement changes, if verified; otherwise an empty array. |
| `followUps` | Specific remaining work, verification limits or classification questions; otherwise an empty array. |

All reference paths in metadata are repository-relative. Documentation references
may include `#heading` anchors; code references can point to files or directories.
Use the card's Markdown body to explain the outcome, important decisions and
constraints, what the evidence establishes, and any remaining uncertainty.
Link to source sections for detail instead of copying long requirements tables.

Use `implemented` only when execution evidence supports it. An approved plan,
an implementation-review filename or the presence of source files is not enough.
Record the difference between local checks and deployed/device/provider proof.
When a later execution record corrects an old review, cite the later evidence and
describe the scope of that correction. Use `unknown` when the record cannot
establish an outcome; keep that question visible in `followUps`.

Archive packages when they become historical context, including abandoned or
superseded proposals. Folder dates do not establish completion or last activity.
An archived card can still have follow-ups: the generated follow-up view includes
those, as well as proposed, partial and unknown outcomes. It is a navigation list
to review, not an automatically approved work queue.

Before archiving a completed change, put enduring rules into the appropriate
maintained guide/ADR and point `currentDocs` there. Do not rewrite historical
requirements to make them appear current. Update the card when new amendments,
validation evidence or replacement decisions are added. Empty supersession links
mean that no replacement relationship has been established, not that the old
design remains authoritative.

Regenerate and validate after editing:

```sh
just docs-index
just docs-check
```

The check requires one card per source, validates metadata and local references,
rejects duplicate IDs and cyclic supersession, and detects stale generated
indexes. Fingerprints cover both each card and its original Markdown documents;
an appended amendment or new document requires review and regeneration even if
its headings did not change. CI runs these checks on every pull request, including code-only changes
that could remove a referenced entry point. It cannot verify
the truth of a human summary: review the cited evidence and its limits. Original
document headings are read on demand, so source-section positions stay current
without checking a large duplicate section index into Git.
