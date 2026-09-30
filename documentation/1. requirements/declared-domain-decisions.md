# Declared-domain conversion — decisions

The scope decisions the conversion to the declared-domain standard turned on.
Recorded because two of them are deliberate departures from what the standard
asks for, and without a record here the next reader would read them as drift
and "fix" them.

Decided by the owner 2026-09-30. The audit that raised them is
[documentation/2. design/declared-domain-audit.md](../2.%20design/declared-domain-audit.md).

## D1 — This module names the platform, on purpose

**Decision: the 45 stored kind values are declared in this module's own
blueprint, and this repo is a deliberate exception to the domain-neutrality
rule.**

`catalog`'s rule is that a word meaning something in one domain and nothing
in another does not belong in the module. By that test, 14 `ActKind` and 31
`EventKind` values fail: `phone_salt_rotated`, `break_glass_write`,
`themes_recomputed`, `survey_question_versioned` are Mwanachama platform
words, and a library running this code would inherit and ignore them.

They are declared here anyway, because the alternatives cost more:

- A domain cannot supply them. CAT5 refuses a domain spec that sets `values`
  on a field the module declares, by name, at load. That rule exists so a
  domain cannot quietly widen a module's vocabulary, and relaxing it would
  change the engine's contract for all nine converted repos.
- Moving kind and class into a `json` document field — catalog's own answer
  for `Sector` — would lose the index on class. `models/act.go` is explicit
  that class is *stored rather than computed at read time so the filter is an
  index scan*, and an expression index over a document path is not the same
  index.

There is a second reason the exception is defensible rather than merely
cheap: **this module's subject is the platform's own operations.** It was
extracted from `api-gateway`'s `internal/domain/{custody,export,contact,consent}`
to be an immutable record of who did what to whose data *in this system*.
Unlike `catalog`, it was never written to run a library. Its `CLAUDE.md` has
never claimed neutrality.

**What follows from this:**

- `domain_agnostic_test.go` is written with an **allowlist of the declared
  kind values**, not with the kinds removed. The test still guards the
  identifiers in `models/`, `routes/` and the root package, which is where a
  domain word would do real damage.
- A future domain that needs its own kinds declares **its own object with no
  role**, the way `library.catalog.json` declares `copy`. It does not extend
  `event.kind`.
- If the engine ever grows a way for a domain to supply a declared field's
  values, this decision should be revisited first — it is the clearest case
  in the fleet.

## D2 — Timestamps become RFC 3339 strings

**Decision: every `time.Time` field on every type in `models/` becomes a
`string`.**

Not a preference. `spec.TypeTimestamp` is stored as text so that every
dialect compares it identically, and `specstore`'s codec has no arm for
`time.Time`: `cell` falls through to `fv.String()`, which on a struct yields
`"<time.Time Value>"` rather than an instant. Every converted repo carries
timestamps as strings for this reason — `catalog`'s `models` has
`CreatedAt string`, `ApprovedAt string`, `RetiredAt string` throughout.

This is wider than the audit anticipated. It changes the signatures of the
four `*Repository` interfaces, which `CLAUDE.md` pinned as must-not-change:

| Interface | Signature that changes |
|---|---|
| `CustodyRepository` | `CountActsByClass(ctx, structures, since time.Time)` |
| `CustodyRepository` | `CountEventsByChip(ctx, since time.Time)` |
| `ExportRepository` | `MarkFileRemoved(ctx, id, at time.Time)` |
| `ExportRepository` | `DueForRemoval(ctx, asOf time.Time, limit)` |
| `ConsentRepository` | `PublishVersion(ctx, id, publishedBy, now time.Time)` |

The pin is released by D3.

`models.DueAt` and `Retention` stay `time.Time`/`time.Duration` arithmetic —
they are a calculation, not storage, and the string boundary is at the store.

## D3 — The `CLAUDE.md` pin on the four Repository interfaces is released

**Decision: `api-gateway` is treated as retired. This conversion does not
preserve its call sites and does not repair it.**

`CLAUDE.md` pinned `models.CustodyRepository`/`.ExportRepository`/
`.ContactRepository`/`.ConsentRepository` byte-for-byte because
`api-gateway`'s route paths, request shapes and status codes depended on
them. That justification no longer holds:

- **The gateway has not compiled since taskmanager's conversion.**
  `cmd/server/stores.go:324` wants `mwanachamataskmanager.DefaultTableNames`,
  deleted there; two `cmd/backfill-*` tools import `actor/gormstore` and
  `taskmanager/gormstore`, deleted by those conversions. Three conversions
  broke it and none repaired it.
- **`wakala-api` is the live consumer** and does not import this repo at all.

So the interfaces are free to change shape. They keep their names and their
method sets — only the timestamp parameters move to strings, per D2.

Dropping the gateway's `require` lines on the converted repos is tracked
org-wide in
[developer/documentation/3. implementation/spec-driven-module-conversion-status.md](../../../developer/documentation/3.%20implementation/spec-driven-module-conversion-status.md),
not here.

## D4 — Storage-layer legacy names are dropped, not carried

The `chapter` → `structure` and `member` → `actor` renames had been applied to
the domain types only. Storage still spoke the old words, and the
`*ToRow`/`*FromRow` converters existed partly to bridge the two vocabularies:

| Domain field | Row field | Column |
|---|---|---|
| `StructureID` | `ChapterID` | `chapter_id` |
| `ActorStructureID` | `ActorChapterID` | `actor_chapter_id` |
| `ToStructureID` | `ToChapterID` | `to_chapter_id` |
| `ActorID` | `MemberID` | `member_id` |
| `SubjectActorID` | `SubjectMemberID` | `subject_member_id` |

**Decision: the blueprint declares the domain vocabulary** — `structure_id`,
`actor_id`, `subject_actor_id` — and the converters are deleted with
`gormstore/`. `specstore` finds a column from the Go field name, so the
agreement is direct and nothing bridges.

This is a column rename as well as a table one. It is safe because no live
rows exist under the old names: the readable tables were only ever created by
this repo's `AutoMigrate` at the gateway's startup, and the gateway has not
started since it stopped compiling. **Confirm against `api.demo` before
provisioning anywhere real** — that database is pumped continuously and must
never be cleared — but a fresh instance needs no adoption step and none is
written.
