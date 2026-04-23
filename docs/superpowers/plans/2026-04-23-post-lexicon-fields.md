# Post Lexicon Fields Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full field set of `social.craftsky.feed.post` (and its new `project.sewing` sibling) per the approved spec, so the first real PDS writes can happen against a stable schema.

**Architecture:** Edit one existing lexicon file in place, add three new lexicon files, update two downstream references (the `actor.profile` field description, and the `lexicon/README.md` index). No runtime code changes — the Go indexer is generic (collection-keyed dispatch), the Flutter app currently has no lexicon-bound code, and no PDS records exist yet. Per ADR 001 and the spec, the shape is one post lexicon with `project: {common, details?}`, open-union details, token-backed `knownValues` strings for extensible enums.

**Tech Stack:** atproto Lexicon v1 (JSON schemas), `social.craftsky.*` NSID namespace. No build tooling — lexicons are static JSON that clients and the AppView consume directly.

**Spec:** [`docs/superpowers/specs/2026-04-23-post-lexicon-fields-design.md`](../specs/2026-04-23-post-lexicon-fields-design.md). **ADR:** [`adr/001-post-lexicon-project-extensibility.md`](../../../adr/001-post-lexicon-project-extensibility.md).

**Pre-flight:** before editing any file under `lexicon/`, re-read the project skill [`.claude/skills/atproto-lexicon/SKILL.md`](../../../.claude/skills/atproto-lexicon/SKILL.md) — it has the NSID / type / style / evolution checklist you'll validate each file against.

---

## File Structure

**New files:**
- `lexicon/social/craftsky/feed/defs.json` — shared tokens for cross-craft values (craftType, status, difficulty).
- `lexicon/social/craftsky/project/sewing.json` — sewing `#details` lexicon (one field: `projectType`).
- `lexicon/social/craftsky/project/sewing.defs.json` — sewing sub-domain tokens.

**Modified files:**
- `lexicon/social/craftsky/feed/post.json` — rewrite to the new shape: top-level fields get updated limits; inline `#projectDetails` replaced by `#project` / `#projectCommon` / `#pattern`; `#image` gains larger `maxSize`; new local defs.
- `lexicon/social/craftsky/actor/profile.json` — update the `crafts` field description to reference the new `feed.defs` tokens instead of the old `feed.post#projectDetails.craftType` path.
- `lexicon/README.md` — add rows to the planned-namespaces table for the new files and the `project.*` branch.

**Not touched by this plan:**
- `like.json`, `repost.json` — unchanged.
- Go code (`appview/`) — the indexer dispatcher in `appview/internal/index/dispatcher.go` is keyed by collection NSID and does not know anything about `#projectDetails`. No Go code currently references `projectDetails`, `craftType`, or any post sub-field (verified via grep). Future indexer work will read the new shape directly when built.
- Flutter code (`app/`) — no lexicon-bound code today (verified via grep).

**Why this split:** each file has one responsibility. `post.json` owns the post record. `feed.defs.json` owns cross-craft tokens (reusable from `#projectCommon` and `#pattern`). `project/sewing.json` owns the sewing `#details` object; its sibling `sewing.defs.json` owns the sewing-only sub-domain tokens. Keeping sewing tokens out of `feed.defs.json` prevents cross-craft `defs` file bloat as more crafts are added.

---

## Validation approach

There is no automated lexicon-schema validator wired into `just test` in this repo. Validation per task is:

1. **JSON validity** — each file must parse as valid JSON. Check with `python3 -m json.tool <file>` (or any JSON parser); zero output on success.
2. **atproto-lexicon style-guide checklist** — after writing, compare the file against the checklist in `.claude/skills/atproto-lexicon/SKILL.md` ("Style & Convention Checklist"). Hand-walk each bullet.
3. **Spec conformance** — read the file alongside the spec and check the field list, types, constraints, and descriptions match.
4. **Cross-reference integrity** — any `ref` or `knownValues` entry that points at another NSID + fragment (e.g. `social.craftsky.feed.defs#wip`) must resolve to a token actually defined in the target file. Hand-check, since there's no validator.

If you find issues, fix them in place before committing. Don't defer.

---

## Task decomposition

Eight tasks, ordered so each produces a self-contained commit that leaves the repo in a consistent state:

1. **Create `feed/defs.json`** — shared tokens. Standalone; no dependencies.
2. **Create `project/sewing.defs.json`** — sewing tokens. Standalone.
3. **Create `project/sewing.json`** — sewing `#details`. References tokens from (2).
4. **Rewrite `feed/post.json`** — the main event. References tokens from (1) and the `#details` union member from (3).
5. **Update `actor/profile.json`** — adjust field description to reference `feed.defs` tokens.
6. **Update `lexicon/README.md`** — planned-namespaces table.
7. **Verify the full set** — JSON + style-guide + cross-reference pass over everything.
8. **Mark ADR 001 status** — (optional, small) update ADR status line from implicit draft to "Approved — implemented by YYYY-MM-DD plan" so future readers know it's landed.

Tasks 1–3 can technically land in any order since none depends on another at the file level, but this ordering (tokens first, then the `#details` that references them, then the post that references both) is the cleanest review trail. Task 4 must come after 1 and 3.

---

### Task 1: Create `feed/defs.json`

Shared tokens for `craftType`, `status`, and pattern `difficulty`. These are referenced by `#projectCommon` (task 4) and `#pattern` (task 4). The initial token set was locked in the spec's Open Questions resolution: `knitting, crochet, sewing, embroidery, quilting` for craft types.

**Files:**
- Create: `lexicon/social/craftsky/feed/defs.json`

- [ ] **Step 1: Write the file**

Write the following content exactly:

```json
{
  "lexicon": 1,
  "id": "social.craftsky.feed.defs",
  "description": "Shared tokens for cross-craft values in social.craftsky.feed.post. Tokens are referenced from #projectCommon.craftType, #projectCommon.status, and #pattern.difficulty via knownValues. knownValues is open: new tokens can be added here without breaking existing records.",
  "defs": {
    "knitting": {
      "type": "token",
      "description": "Knitting projects."
    },
    "crochet": {
      "type": "token",
      "description": "Crochet projects."
    },
    "sewing": {
      "type": "token",
      "description": "Sewing projects."
    },
    "embroidery": {
      "type": "token",
      "description": "Embroidery projects."
    },
    "quilting": {
      "type": "token",
      "description": "Quilting projects."
    },
    "wip": {
      "type": "token",
      "description": "Work in progress."
    },
    "finished": {
      "type": "token",
      "description": "Finished project."
    },
    "beginner": {
      "type": "token",
      "description": "Pattern suitable for newcomers to the craft."
    },
    "intermediate": {
      "type": "token",
      "description": "Pattern assumes familiarity with core techniques of the craft."
    },
    "advanced": {
      "type": "token",
      "description": "Pattern assumes confident skill and some complex techniques."
    },
    "expert": {
      "type": "token",
      "description": "Pattern demands mastery of the craft and uncommon techniques."
    }
  }
}
```

- [ ] **Step 2: Validate JSON**

Run: `python3 -m json.tool lexicon/social/craftsky/feed/defs.json > /dev/null`

Expected: no output, exit 0.

- [ ] **Step 3: Style-guide checklist**

Hand-check against `.claude/skills/atproto-lexicon/SKILL.md` "Style & Convention Checklist". For a defs-only file, relevant bullets are:
- `main` defs have a description — **N/A** (no `main`, per the style guide "defs files generally should not have a main").
- Token names are valid lowerCamelCase? **Yes** (simple lowercase words).
- Token descriptions are short, user-facing? **Yes**.

- [ ] **Step 4: Commit**

```bash
git add lexicon/social/craftsky/feed/defs.json
git commit -m "feat(lexicon): add social.craftsky.feed.defs shared tokens

Adds token defs for craftType (knitting, crochet, sewing, embroidery,
quilting), status (wip, finished), and pattern difficulty (beginner,
intermediate, advanced, expert). Referenced from #projectCommon and
#pattern in social.craftsky.feed.post (landing in a follow-up commit).

Refs adr/001-post-lexicon-project-extensibility.md."
```

---

### Task 2: Create `project/sewing.defs.json`

Sewing sub-domain tokens. Referenced by the `projectType` field in `project/sewing.json` (task 3).

**Files:**
- Create: `lexicon/social/craftsky/project/sewing.defs.json`

- [ ] **Step 1: Make the directory if needed**

Run: `mkdir -p lexicon/social/craftsky/project`

Expected: directory exists (idempotent; no output on success).

- [ ] **Step 2: Write the file**

```json
{
  "lexicon": 1,
  "id": "social.craftsky.project.sewing.defs",
  "description": "Sub-domain tokens for sewing projects. Referenced from social.craftsky.project.sewing#details.projectType via knownValues.",
  "defs": {
    "garment": {
      "type": "token",
      "description": "Clothing: dresses, shirts, trousers, skirts, jackets, etc."
    },
    "homeGoods": {
      "type": "token",
      "description": "Curtains, pillows, bedding, tablecloths, and other home items."
    },
    "accessory": {
      "type": "token",
      "description": "Bags, hats, scarves, and similar worn or carried items."
    },
    "softToy": {
      "type": "token",
      "description": "Plushies, stuffed animals, dolls."
    },
    "costume": {
      "type": "token",
      "description": "Cosplay, theatre, fancy dress."
    },
    "alteration": {
      "type": "token",
      "description": "Modifications, repairs, or mending of existing garments."
    }
  }
}
```

**Note on naming:** the spec uses kebab-case token labels (`home-goods`, `soft-toy`) for the `knownValues` entries. Per the atproto style guide, **token *names* inside a defs file are lowerCamelCase** (ASCII alphanumeric, no hyphens — the spec is explicit: "first char not a digit, no hyphens"). So:
- Token def names: `homeGoods`, `softToy` (lowerCamelCase, valid atproto identifiers).
- `knownValues` string constants in the consumer (`sewing.json`): the **fully-qualified reference** `social.craftsky.project.sewing.defs#homeGoods`.

The spec's kebab-case tokens were a brainstorming-level informal notation; the atproto rules don't permit hyphens in def names. No semantic change — the tokens still represent the same sub-domains.

- [ ] **Step 3: Validate JSON**

Run: `python3 -m json.tool lexicon/social/craftsky/project/sewing.defs.json > /dev/null`

Expected: no output, exit 0.

- [ ] **Step 4: Style-guide checklist**

Walk the same bullets as task 1. All token def names are valid lowerCamelCase. Descriptions are short, user-facing.

- [ ] **Step 5: Commit**

```bash
git add lexicon/social/craftsky/project/sewing.defs.json
git commit -m "feat(lexicon): add social.craftsky.project.sewing.defs tokens

Sewing sub-domain tokens (garment, homeGoods, accessory, softToy,
costume, alteration). Referenced from #details.projectType in
social.craftsky.project.sewing (landing next).

Token def names use lowerCamelCase per atproto style guide; the spec's
kebab-case labels were informal. No semantic change.

Refs adr/001-post-lexicon-project-extensibility.md."
```

---

### Task 3: Create `project/sewing.json`

Sewing `#details` lexicon. One object-type def, one field (`projectType`) referencing the tokens from task 2. No `main` record — this file defines a *referenced type only*, per ADR 001.

**Files:**
- Create: `lexicon/social/craftsky/project/sewing.json`

- [ ] **Step 1: Write the file**

```json
{
  "lexicon": 1,
  "id": "social.craftsky.project.sewing",
  "description": "Sewing-specific fields for a craft project post. Referenced from social.craftsky.feed.post#project.details as one variant of the open union. This file defines a referenced type (#details) only; no main record.",
  "defs": {
    "details": {
      "type": "object",
      "description": "Sewing-specific fields attached to a project post when craftType is sewing.",
      "properties": {
        "projectType": {
          "type": "string",
          "maxLength": 100,
          "maxGraphemes": 100,
          "description": "The kind of sewing project. Helps discovery so readers can filter on, e.g., only garment projects.",
          "knownValues": [
            "social.craftsky.project.sewing.defs#garment",
            "social.craftsky.project.sewing.defs#homeGoods",
            "social.craftsky.project.sewing.defs#accessory",
            "social.craftsky.project.sewing.defs#softToy",
            "social.craftsky.project.sewing.defs#costume",
            "social.craftsky.project.sewing.defs#alteration"
          ]
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON**

Run: `python3 -m json.tool lexicon/social/craftsky/project/sewing.json > /dev/null`

Expected: no output, exit 0.

- [ ] **Step 3: Cross-reference integrity**

Every `knownValues` entry in the `projectType` field is of the form `social.craftsky.project.sewing.defs#<tokenName>`. For each entry, confirm the token is defined in `lexicon/social/craftsky/project/sewing.defs.json` from task 2. Should see exact matches for: `garment`, `homeGoods`, `accessory`, `softToy`, `costume`, `alteration`.

- [ ] **Step 4: Style-guide checklist**

- `main` def: **N/A** — this file defines `#details`, not `main`. Valid per ADR 001 (referenced types only).
- String field has both `maxLength` and `maxGraphemes` — **yes**.
- Using `knownValues` (open) not `enum` (closed) — **yes**.
- Field name `projectType` is lowerCamelCase — **yes**.
- Description clarifies purpose — **yes**.

- [ ] **Step 5: Commit**

```bash
git add lexicon/social/craftsky/project/sewing.json
git commit -m "feat(lexicon): add social.craftsky.project.sewing #details

Adds the first per-craft #details lexicon, proving the {common, details}
extensibility pattern from ADR 001. One field for now: projectType, with
token-backed knownValues pointing at sewing.defs. Deliberately minimal;
richer sewing structure (garment sizing, dimensions, etc.) is out of
scope per the spec's alternatives-considered section.

Refs adr/001-post-lexicon-project-extensibility.md and the post lexicon
fields spec (2026-04-23)."
```

---

### Task 4: Rewrite `feed/post.json`

The main event. Replaces the existing inline `#projectDetails` with `#project` / `#projectCommon` / `#pattern`; updates `#image.maxSize` to 15 MB; bumps `text` length limits to 2000 graphemes / 20000 bytes; adds `materials`, `tags`, `duration`, `title` fields on `#projectCommon`; adds `pattern.difficulty`; references tokens from `feed.defs` (task 1) and the `details` union member from `project.sewing` (task 3).

**Files:**
- Modify: `lexicon/social/craftsky/feed/post.json` (complete rewrite — existing file is not in production per the spec).

- [ ] **Step 1: Read the current file once**

Read `lexicon/social/craftsky/feed/post.json` end to end. Confirm the top-level fields you're preserving unchanged: `#quoteEmbed`, `#replyRef`, `facets` (still `app.bsky.richtext.facet`), `createdAt`. You're replacing: `text` limits, `#image.maxSize`, and the whole `#projectDetails` section.

- [ ] **Step 2: Write the new file**

Replace the file's contents with:

```json
{
  "lexicon": 1,
  "id": "social.craftsky.feed.post",
  "defs": {
    "main": {
      "type": "record",
      "description": "A post on Craftsky. May be a general post or a craft project post; when the optional 'project' field is present the post is a project post. A post with an 'embed' containing a quoteEmbed is a quote post.",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["text", "createdAt"],
        "properties": {
          "text": {
            "type": "string",
            "maxLength": 20000,
            "maxGraphemes": 2000,
            "description": "The primary text content of the post. Plain text; rich-text annotations live in 'facets'. Craftsky allows longer posts than Bluesky (2000 graphemes vs 300) because crafters write fuller project write-ups."
          },
          "facets": {
            "type": "array",
            "items": { "type": "ref", "ref": "app.bsky.richtext.facet" },
            "description": "Byte-range annotations over 'text' for mentions, links, and inline hashtags. Reuses app.bsky.richtext.facet so existing renderers Just Work. Inline hashtag facets are composer-merged into project.common.tags on project posts; see the post lexicon fields spec."
          },
          "project": {
            "type": "ref",
            "ref": "#project",
            "description": "If present, this post is a craft project post. Absence means it's a general social post."
          },
          "images": {
            "type": "array",
            "items": { "type": "ref", "ref": "#image" },
            "maxLength": 4,
            "description": "Images attached to the post. Top-level (not inside 'embed') — a post may carry images alongside a quote embed without needing a wrapper variant."
          },
          "embed": {
            "type": "union",
            "refs": ["#quoteEmbed"],
            "description": "Optional embedded content. Open union; today only quote embeds are defined."
          },
          "reply": {
            "type": "ref",
            "ref": "#replyRef",
            "description": "If present, this post is a reply to another post."
          },
          "createdAt": {
            "type": "string",
            "format": "datetime",
            "description": "Client-declared creation timestamp."
          }
        }
      }
    },
    "project": {
      "type": "object",
      "description": "Wraps a craft project's shared fields (#projectCommon) and optional craft-specific fields (#details, open union). The wrapper exists so shared fields live in one place (common) while per-craft specialisations are added additively via the union. See ADR 001.",
      "required": ["common"],
      "properties": {
        "common": {
          "type": "ref",
          "ref": "#projectCommon",
          "description": "Shared project fields that apply across all crafts."
        },
        "details": {
          "type": "union",
          "refs": [
            "social.craftsky.project.sewing#details"
          ],
          "description": "Optional craft-specific fields. Open union — new crafts add new variants here without breaking existing records."
        }
      }
    },
    "projectCommon": {
      "type": "object",
      "description": "Project fields shared across every craft. A post with project.common present but no project.details is a valid craft-tagged project post without specialised fields — useful when craftType references a craft that has no #details lexicon yet.",
      "required": ["craftType"],
      "properties": {
        "craftType": {
          "type": "string",
          "maxLength": 100,
          "maxGraphemes": 100,
          "description": "The craft this project belongs to. Required — a project post without a craft type is meaningless. knownValues are tokens; new crafts can be added to feed.defs without breaking old clients.",
          "knownValues": [
            "social.craftsky.feed.defs#knitting",
            "social.craftsky.feed.defs#crochet",
            "social.craftsky.feed.defs#sewing",
            "social.craftsky.feed.defs#embroidery",
            "social.craftsky.feed.defs#quilting"
          ]
        },
        "status": {
          "type": "string",
          "maxLength": 100,
          "maxGraphemes": 100,
          "description": "Whether the project is in progress or finished at the time this post was created. Snapshot — not a mutable lifecycle flag. knownValues are tokens so new statuses (e.g. planned, frogged) can be added without breaking old clients.",
          "knownValues": [
            "social.craftsky.feed.defs#wip",
            "social.craftsky.feed.defs#finished"
          ]
        },
        "title": {
          "type": "string",
          "maxLength": 500,
          "maxGraphemes": 200,
          "description": "Optional project name, e.g. 'Hitchhiker Shawl' or 'Linen Summer Dress'. Clients showing grid/card views should fall back to truncated text when title is absent."
        },
        "duration": {
          "type": "string",
          "maxLength": 500,
          "maxGraphemes": 100,
          "description": "Free-text description of how long the project took, e.g. '3 weeks', 'a weekend', '6 months of evenings'. Deliberately unstructured — crafters describe duration informally. Not range-queryable."
        },
        "pattern": {
          "type": "ref",
          "ref": "#pattern",
          "description": "Optional pattern reference."
        },
        "materials": {
          "type": "array",
          "maxLength": 20,
          "items": {
            "type": "string",
            "maxLength": 100,
            "maxGraphemes": 100
          },
          "description": "Materials used in the project, as free-form tags. Indexed for search, e.g. 'show me all projects using linen'. Structure per material is intentionally free-form because every craft uses different material descriptors."
        },
        "tags": {
          "type": "array",
          "maxLength": 10,
          "items": {
            "type": "string",
            "maxLength": 64,
            "maxGraphemes": 64
          },
          "description": "Structured search tags. Composer responsibility to normalise to ASCII kebab-case (pattern ^[a-z0-9]+(-[a-z0-9]+)*$) and to merge any inline #hashtag facets from text into this field. AppView indexer materialises this as a multi-value searchable column."
        }
      }
    },
    "pattern": {
      "type": "object",
      "description": "Optional reference to the pattern used. Every field optional — 'Simplicity 8265' (name only), 'https://ravelry.com/patterns/library/hitchhiker' (URL only), or both plus difficulty are all valid.",
      "properties": {
        "url": {
          "type": "string",
          "format": "uri",
          "description": "Link to the pattern."
        },
        "name": {
          "type": "string",
          "maxLength": 500,
          "maxGraphemes": 200,
          "description": "Pattern name, e.g. 'Simplicity 8265' or 'Hitchhiker Shawl'. Useful when there is no URL (e.g. a physical pattern), or alongside a URL as a display label."
        },
        "difficulty": {
          "type": "string",
          "maxLength": 100,
          "maxGraphemes": 100,
          "description": "Pattern difficulty as rated by the designer. A property of the pattern, not the post — self-drafted or free-formed projects should leave this empty. knownValues are tokens.",
          "knownValues": [
            "social.craftsky.feed.defs#beginner",
            "social.craftsky.feed.defs#intermediate",
            "social.craftsky.feed.defs#advanced",
            "social.craftsky.feed.defs#expert"
          ]
        }
      }
    },
    "image": {
      "type": "object",
      "description": "An image blob with alt text.",
      "required": ["image", "alt"],
      "properties": {
        "image": {
          "type": "blob",
          "accept": ["image/jpeg", "image/png", "image/webp"],
          "maxSize": 15728640
        },
        "alt": {
          "type": "string",
          "maxLength": 1000,
          "maxGraphemes": 1000,
          "description": "Alt text describing the image for accessibility."
        }
      }
    },
    "quoteEmbed": {
      "type": "object",
      "description": "Embed wrapping another record (typically a social.craftsky.feed.post) for a quote post.",
      "required": ["record"],
      "properties": {
        "record": {
          "type": "ref",
          "ref": "com.atproto.repo.strongRef",
          "description": "Strong reference to the quoted record."
        }
      }
    },
    "replyRef": {
      "type": "object",
      "description": "Reference to the parent and root posts of a thread.",
      "required": ["root", "parent"],
      "properties": {
        "root": {
          "type": "ref",
          "ref": "com.atproto.repo.strongRef",
          "description": "The root post of the thread."
        },
        "parent": {
          "type": "ref",
          "ref": "com.atproto.repo.strongRef",
          "description": "The immediate parent post being replied to."
        }
      }
    }
  }
}
```

**Note on the `#project` def name:** the atproto style guide warns against NSID/def collisions like `app.bsky.feed.post#main` vs `app.bsky.feed.post.main`. `#project` as a def name inside `social.craftsky.feed.post` is fine — there's no file called `social.craftsky.feed.post.project`, and per the ADR the per-craft `#details` live under `social.craftsky.project.*` which is a sibling group, not a child. No collision.

**Note on `maxSize: 15728640`:** that's 15 MB in bytes (`15 * 1024 * 1024`). atproto's `blob.maxSize` is bytes.

**Note on `materials` / `tags` array item constraints:** the spec said e.g. "each 100g/100b, array maxLength 20". atproto lexicon arrays apply length constraints to the item via the `items` schema (`maxLength`, `maxGraphemes` on the inner string) and to the array via the outer `maxLength`. Both set as per the spec.

- [ ] **Step 3: Validate JSON**

Run: `python3 -m json.tool lexicon/social/craftsky/feed/post.json > /dev/null`

Expected: no output, exit 0.

- [ ] **Step 4: Cross-reference integrity**

For every ref/knownValues pointer in the file, confirm the target exists:
- `app.bsky.richtext.facet` — external, assumed valid (same as current file).
- `com.atproto.repo.strongRef` — external, assumed valid.
- `#project`, `#projectCommon`, `#pattern`, `#image`, `#quoteEmbed`, `#replyRef` — all defined locally in this file; check each is present under `defs`.
- `social.craftsky.feed.defs#knitting`, `#crochet`, `#sewing`, `#embroidery`, `#quilting`, `#wip`, `#finished`, `#beginner`, `#intermediate`, `#advanced`, `#expert` — confirm each is in `lexicon/social/craftsky/feed/defs.json` (task 1).
- `social.craftsky.project.sewing#details` — confirm `#details` is in `lexicon/social/craftsky/project/sewing.json` (task 3).

- [ ] **Step 5: Style-guide checklist**

Walk the checklist bullets:
- `main` def has description — **yes**.
- Ambiguously-named fields have descriptions (`url`, `name` in `#pattern` say what they refer to) — **yes**.
- String fields have `maxLength` (and `maxGraphemes` for visible text) — **yes**; the only exception is `pattern.url` which uses `format: uri` (format implies bounds per the style guide).
- No `enum` — **yes**, all extensible fields use `knownValues`.
- Unions are open — **yes** (`embed`, `project.details`; neither has `closed: true`).
- Record fields referring to accounts use `did`, not `handle` — **N/A**, no account fields.
- Arrays of data use arrays of objects — `materials` and `tags` are arrays of strings, which the style guide normally discourages. **Intentional exception:** they are user-facing free-form labels, and adding an object wrapper would only add ceremony with no forward-compat benefit — materials/tags are the primitive, not something expected to grow context. Documented in the spec's materials rationale. Leave as-is.
- Shared definitions live in a `{group}.defs` file — **yes**; tokens live in `feed.defs` and `project.sewing.defs`, not inlined here.
- Reusing `com.atproto.repo.strongRef` for record references — **yes**, in `#quoteEmbed` and `#replyRef`.

- [ ] **Step 6: Spec conformance spot-check**

Open the spec ([`docs/superpowers/specs/2026-04-23-post-lexicon-fields-design.md`](../specs/2026-04-23-post-lexicon-fields-design.md)) side-by-side. Tick each field in the `#projectCommon` table against the JSON. Tick the three fields in `#pattern`. Tick top-level `post` shape. Confirm `maxSize` on `#image` is 15 MB (15728640 bytes). Confirm text `maxGraphemes` is 2000.

- [ ] **Step 7: Commit**

```bash
git add lexicon/social/craftsky/feed/post.json
git commit -m "feat(lexicon): restructure feed.post to {common, details} shape

Rewrites social.craftsky.feed.post to implement ADR 001 and the post
lexicon fields spec (2026-04-23):

- project.common / project.details replaces the inline projectDetails,
  with an open union on details for per-craft extensions.
- #projectCommon adds title, duration, materials, tags; pattern becomes
  a sub-object carrying url, name, and difficulty.
- craftType, status, and pattern.difficulty are all token-backed
  knownValues pointing at social.craftsky.feed.defs.
- project.details currently admits social.craftsky.project.sewing#details;
  more crafts land additively.
- text limits bumped to 2000 graphemes / 20000 bytes.
- image.maxSize bumped to 15 MB.

No production records use the prior shape (confirmed by product owner),
so this is a rewrite, not an evolution. Future field additions will
follow atproto evolution rules (optional-only, knownValues open).

Refs adr/001-post-lexicon-project-extensibility.md."
```

---

### Task 5: Update `actor/profile.json` description

The existing `crafts` field in `social.craftsky.actor.profile` references `social.craftsky.feed.post#projectDetails.craftType` in its description. That path is gone (`#projectDetails` no longer exists). Point it at the new `feed.defs` tokens instead.

**Files:**
- Modify: `lexicon/social/craftsky/actor/profile.json`

- [ ] **Step 1: Read the current file**

Re-read `lexicon/social/craftsky/actor/profile.json`. The field in question is `crafts.description`, at line 20.

- [ ] **Step 2: Edit the description**

Replace:

```
Free-form but clients may normalize against the knownValues of social.craftsky.feed.post#projectDetails.craftType.
```

With:

```
Free-form but clients may normalize against the craftType tokens defined in social.craftsky.feed.defs (knitting, crochet, sewing, embroidery, quilting, etc.).
```

No other changes to the file. `crafts` stays as an optional `array<string>` with the same length constraints.

- [ ] **Step 3: Validate JSON**

Run: `python3 -m json.tool lexicon/social/craftsky/actor/profile.json > /dev/null`

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add lexicon/social/craftsky/actor/profile.json
git commit -m "docs(lexicon): update actor.profile.crafts cross-ref to feed.defs

The crafts field's description pointed at the old
social.craftsky.feed.post#projectDetails.craftType path. That def is
gone — craft type tokens now live in social.craftsky.feed.defs. Update
the description to match. No schema change."
```

---

### Task 6: Update `lexicon/README.md` planned-namespaces table

The README's "Planned Namespaces" table is the user-facing index of the lexicon tree. It currently lists `feed.post`, `feed.repost`, `feed.like`, `actor.profile`. Add rows for `feed.defs`, `project.sewing`, `project.sewing.defs`, and mention the `#projectDetails → #project/#projectCommon` shape change for `feed.post`.

**Files:**
- Modify: `lexicon/README.md`

- [ ] **Step 1: Read the current file**

Re-read `lexicon/README.md`. The table in question starts at the line beginning `| Lexicon | Purpose |`.

- [ ] **Step 2: Update the `social.craftsky.feed.post` row and add new rows**

Find the current line:

```
| `social.craftsky.feed.post` | A post — general or a craft project (via optional `project` sub-object). Supports replies, up to 4 images, rich-text facets (reusing `app.bsky.richtext.facet`), and quote embeds. |
```

Replace with:

```
| `social.craftsky.feed.post` | A post — general or a craft project (via optional `project` sub-object with `common` + open-union `details`). Supports replies, up to 4 images, rich-text facets (reusing `app.bsky.richtext.facet`), and quote embeds. |
| `social.craftsky.feed.defs` | Shared tokens for cross-craft values in `#projectCommon` (`craftType`, `status`) and `#pattern` (`difficulty`). |
| `social.craftsky.project.sewing` | Sewing-specific `#details` referenced from `feed.post#project.details`. Defines a referenced type only — no `main` record. |
| `social.craftsky.project.sewing.defs` | Sewing sub-domain tokens (`garment`, `homeGoods`, `accessory`, `softToy`, `costume`, `alteration`). |
```

- [ ] **Step 3: Add a short note below the table** (if one paragraph fits naturally)

After the existing note about comments/quote posts, add:

```
Per-craft `#details` lexicons live under the `social.craftsky.project.*` branch (one file per craft). They define a `#details` object type only — they are referenced types, not standalone records. See [ADR 001](../adr/001-post-lexicon-project-extensibility.md) for the extensibility rationale.
```

- [ ] **Step 4: Commit**

```bash
git add lexicon/README.md
git commit -m "docs(lexicon): index new defs and project.sewing files in README

Adds rows for social.craftsky.feed.defs, project.sewing, and
project.sewing.defs to the planned-namespaces table. Updates the
feed.post row to reflect the {common, details} shape. Adds a note
explaining the project.* branch convention with a pointer to ADR 001."
```

---

### Task 7: Verify the full set

After all five previous tasks, run a final pass across every file to catch anything that slipped.

- [ ] **Step 1: Validate every lexicon JSON file**

Run:

```bash
for f in $(find lexicon -name '*.json'); do
  echo "=== $f ==="
  python3 -m json.tool "$f" > /dev/null && echo "OK" || echo "FAIL: $f"
done
```

Expected: every line ends in `OK`.

- [ ] **Step 2: Cross-reference pass**

For each external ref in `feed/post.json` that targets `feed.defs` or `project.sewing`, grep the target file for the token/def name. Expected: every reference resolves.

```bash
grep -oE 'social\.craftsky\.feed\.defs#[a-zA-Z]+' lexicon/social/craftsky/feed/post.json | sort -u
```

Then for each token name that follows the `#`, run:

```bash
python3 -c "import json,sys; d=json.load(open('lexicon/social/craftsky/feed/defs.json'))['defs']; print('OK' if '<tokenName>' in d else 'MISSING')"
```

Iterate through the unique token names: `knitting`, `crochet`, `sewing`, `embroidery`, `quilting`, `wip`, `finished`, `beginner`, `intermediate`, `advanced`, `expert`. All should print `OK`.

Do the same for `project.sewing` tokens referenced by `project/sewing.json`'s `projectType.knownValues`: `garment`, `homeGoods`, `accessory`, `softToy`, `costume`, `alteration`. All should print `OK` when checked against `project/sewing.defs.json`.

- [ ] **Step 3: Ensure no stale references to `#projectDetails`**

Run:

```bash
grep -r "projectDetails" lexicon/
```

Expected: no output (the old def name is fully removed). If anything matches, fix it before moving on.

- [ ] **Step 4: Ensure no stale `patternUrl` string**

Run:

```bash
grep -r "patternUrl" lexicon/
```

Expected: no output.

- [ ] **Step 5: Verify `actor.profile` points at the new defs file**

Run:

```bash
grep -n "projectDetails\|feed.defs" lexicon/social/craftsky/actor/profile.json
```

Expected: one match mentioning `feed.defs`, no mentions of `projectDetails`.

- [ ] **Step 6: Sanity-check the commit history**

Run:

```bash
git log --oneline -10
```

Expected: five new commits from this plan, each scoped to one file (or the two-file README+post exception if that happened). No mixed commits.

- [ ] **Step 7: If any check failed, fix and commit as a follow-up**

If grep finds stale references or cross-refs don't resolve, make a follow-up commit per the failing file. Do not amend existing commits.

---

### Task 8: Mark ADR 001 as approved/implemented (small, optional)

ADR 001 ([`adr/001-post-lexicon-project-extensibility.md`](../../../adr/001-post-lexicon-project-extensibility.md)) doesn't currently have an explicit status line — the generic ADR template it used doesn't require one. Adding a short status note now makes it clear to future readers that this ADR has landed.

**Files:**
- Modify: `adr/001-post-lexicon-project-extensibility.md`

- [ ] **Step 1: Add a Status line**

Insert at the top of the ADR (above the existing `- Aspect:` line):

```
- Status: Approved — implemented by 2026-04-23 post lexicon fields plan
```

No other changes to the ADR body.

- [ ] **Step 2: Commit**

```bash
git add adr/001-post-lexicon-project-extensibility.md
git commit -m "docs(adr): mark 001 as approved and implemented

The post lexicon fields plan (2026-04-23) implements the shape this ADR
locked in. Add a Status line so future readers see it's landed, not
just drafted."
```

---

## Done criteria

All of these are true at the end:

- [ ] `lexicon/social/craftsky/feed/defs.json` exists with 11 tokens (5 crafts + 2 statuses + 4 difficulties).
- [ ] `lexicon/social/craftsky/project/sewing.json` exists with one `#details` def.
- [ ] `lexicon/social/craftsky/project/sewing.defs.json` exists with 6 tokens.
- [ ] `lexicon/social/craftsky/feed/post.json` has: no `#projectDetails` def, a new `#project` def wrapping `#projectCommon` + `#details` union, `#pattern` def, and updated `text`/`#image` limits.
- [ ] `lexicon/social/craftsky/actor/profile.json` has an updated description referencing `feed.defs`.
- [ ] `lexicon/README.md` lists all four new/changed lexicon files in the planned-namespaces table.
- [ ] Every cross-reference in every file resolves (task 7 passes).
- [ ] `grep -r projectDetails lexicon/` and `grep -r patternUrl lexicon/` both return nothing.
- [ ] Six to seven commits in `git log`, each focused on one concern.
- [ ] (Optional, task 8) ADR 001 has a Status line.

## Out of scope for this plan

- Go AppView indexer changes. The dispatcher is generic; once someone writes a `post` indexer, it'll read the new shape directly. Separate spec.
- Flutter composer changes (help text, tags-field normalisation, inline-hashtag merge). Separate spec.
- More per-craft `#details` lexicons (knitting, crochet, quilting, embroidery). Each needs its own brainstorm + spec.
- Indexer schema (the columns that materialise `craftType`, `status`, `materials[]`, `tags[]`, `pattern.difficulty`, sewing `projectType`). Separate spec.
- Publishing the lexicons to the atproto directory / DNS `_lexicon` TXT record. Production-deploy concern.
