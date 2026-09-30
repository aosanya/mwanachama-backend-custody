# Declared-domain audit

This repo against the standard `mwanachama-backend-catalog` sets, per the
strategy in `developer/documentation/2. design/architecture-spec-driven-modules.md`
(decided 2026-09-23). Audited 2026-09-30 against commit `637ed23`, working
tree clean.

Baseline: `go test ./...` passes, no Go file exceeds 300 lines.

## Verdict

**Not converted — one gate of eighteen passes.** Conversion is warranted: this repo is
small, GORM-backed (not entitygraph), and its objects are append-only or
write-once, so there is little update path to get wrong.

**Two decisions gate the work and are not this audit's to make.** Both are in
"Preconditions" below. Steps A–J should not start until they are answered,
because both change what the blueprint declares and would be redone.

## Gates

| # | Gate | Verdict | Evidence |
|---|------|---------|----------|
| 1 | Blueprint at root | **fail** | no `*.blueprint.json` |
| 2 | Blueprint reached from Go | **fail** | no `blueprint.go` |
| 3 | Domain specs (≥2) | **fail** | no spec files at all |
| 4 | No row structs | **fail** | `gormstore/` holds 6 row structs; root `tables.go` re-exports `TableNames`/`DefaultTableNames`/`Migrate`; `gormstore/tables.go:70` calls `AutoMigrate` |
| 5 | Store is thin over `specstore` | **fail** | no `store.go`; six `*_impl.go` files call `gormstore` directly |
| 6 | Constructor cross-check | **fail** | constructors take `(db, TableNames)`, not `(db, *spec.Spec)` |
| 7 | Declared rules | **fail** | no `validate.go`, no `patterns.go`; rules are `Job.Validate`, `ProgressUpdate.Validate`, `Completion.Validate` |
| 8 | Thin models | **fail** | 3 `Validate` methods, 2 membership maps (`actClasses`, `eventChips`) with `IsActClass`/`IsEventChip`/`ActClassOf`/`EventChipOf`, and per-field doc comments throughout `models/` |
| 9 | Operations declared | **fail** | no `*.operations.json` |
| 10 | Routes are an adapter | **fail** | 5 hand-written handler files (`consent.go`, `contact.go`, `custody.go`, `export.go`, `identity.go`) |
| 11 | Gate is data | **fail** | no `AnonymousActions`; `routes/doc.go` says a mounting process wraps each Handler |
| 12 | MCP is the same table | **n/a** | no `mcp/` package |
| 13 | Provisioning | **fail** | no `provision.go`, no `cmd/ddl` |
| 14 | Guard tests | **fail** | none of the six exist |
| 15 | Depends on shared | **fail** | `go.mod` has no `mwanachama-backend-shared` require or replace — this repo is not on the engine at all |
| 16 | Makefile | **fail** | no Makefile |
| 17 | Documentation | **partial** | four-phase layout exists, but `1. requirements/`, `2. design/` and `4. qa/` are **empty**; board rows exist in `3. implementation/todo.md` |
| 18 | House rules | **pass** | no file over 300 lines; comments are being stripped per `CLAUDE.md` |

Only gate 18 passes outright. Gate 12 does not apply (no `mcp/`), gate 17
passes in part. The other fifteen fail.

### What the repo already got right

Two things reduce the conversion's size and are worth naming so they are not
re-litigated:

- **`routes/wire.go` is a local copy of `httpwire`.** Deleting it is a pure
  substitution, the same one `assetmanager` and `insights` already made.
- **Six objects, and only one of them has a real update path.** Five are
  append-only or write-once, so most of Step C carries no risk of getting an
  update wrong.

### The `chapter` → `structure` rename stopped at the storage boundary

Worth flagging separately, because it is easy to read as already done. The
*domain* types are renamed — `models.StructureActLogEntry.StructureID`,
`ErrStructureRequired` — but storage is not:

| Layer | Still says `chapter` |
| --- | --- |
| row struct | `ChapterID`, `ActorChapterID`, `ToChapterID` (`gormstore/chapter_act.go`) |
| physical column | `chapter_id`, queried literally in `custody_impl.go:69` and `:94` |
| physical table | `custody_chapter_act_log_entry` |
| file name | `gormstore/chapter_act.go` |

The `*ToRow`/`*FromRow` converters exist partly to bridge that gap, so the
conversion deletes the gap along with them: the blueprint declares
`structure_id`, `specstore` finds the column from the Go field name, and the
table is a hash regardless. **This is a column rename**, not just a table
one — which matters for precondition 3, because adopting legacy rows would
need the column mapped, not just the table. A fresh table needs nothing.

`gormstore/chapter_act.go` is also the one place a stored name still carries
a word the module has otherwise dropped, which is the failure the standard
calls the worst kind.

## Preconditions

### 1. The stored enum values cannot go where the standard wants them

This repo stores **45 enum values naming Mwanachama platform events** —
14 `ActKind` (`role_granted`, `left_structure`, `answer_read_named`, …) and
31 `EventKind` (`phone_salt_rotated`, `break_glass_write`,
`themes_recomputed`, `survey_question_versioned`, …).

Under the standard these become declared `values` on a blueprint field. But
CAT5 is explicit that a **domain may not supply a value set** — setting
`values` on a field the module declares is refused by name at load. So the
module owns them, which means the module carries 45 values that a library or
a clinic running this code would have to inherit and ignore.

The skill's own rule — "a word that means something in one domain and nothing
in another does not belong in the module" — says these fail. The engine
currently offers nowhere else to put them. `spec.Field` has `Values []string`
and no mechanism for a domain to extend or replace a declared set.

There is a second, related problem the engine also cannot express: the
**kind → class** and **kind → chip** mappings (`models/act.go`'s `actClasses`,
`models/event.go`'s `eventChips`). These are not flat enums, they are
functions, and `spec.Field` has no shape for a mapping. They would stay in Go
as "a rule a spec cannot state" — but then the kinds stay in Go with them,
and `domain_agnostic_test.go` fails on the current names.

**The three ways out, none of which this audit should pick:**

| Option | Cost |
| --- | --- |
| Declare all 45 in the module's blueprint and accept that this module names the platform | Cheapest. Concedes the neutrality rule for this repo, which should then be written into `1. requirements` as a deliberate exception so nobody "fixes" it later. |
| Extend the engine so a domain may supply a declared field's `values` | Changes CAT5, which exists for a good reason. Affects every converted repo. Belongs on shared's board, not in this conversion. |
| Move kind and class inside a `json` document field, the way catalog moved `Sector` | Loses the index on class, which is the one thing `models/act.go` says is stored rather than computed *so the filter is an index scan*. |

This repo's `CLAUDE.md` never claims domain-neutrality — it describes itself
as an extraction of the gateway's audit/compliance cluster. That is an
argument for the first option, but "the requirements record says so" is
exactly the evidence that does not exist here: `documentation/1. requirements/`
is empty.

### 2. The only consumer is already broken

`api-gateway` is the sole importer of this repo, and it imports it deeply —
`cmd/server/stores.go`, `cmd/server/comm_act_writer.go`,
`internal/api/http/deps.go`, `internal/api/http/break_glass.go`. This repo's
`CLAUDE.md` pins the four `*Repository` interfaces as must-not-change for
that reason.

**The gateway does not compile today:**

```text
cmd/server/stores.go:324:38: undefined: mwanachamataskmanager.DefaultTableNames
cmd/server/stores.go:325:34: undefined: mwanachamataskmanager.Migrate
```

plus two `cmd/backfill-*` tools importing `actor/gormstore` and
`taskmanager/gormstore`, both deleted by those conversions. Three completed
conversions have already broken it and nobody repaired it. `wakala-api` is
the live consumer and does not import this repo
([[wakala-api-is-the-live-consumer]]).

So the interface pin protects a consumer that is already down, and
"converting custody breaks the gateway" is no longer true in any way that
matters — it is already broken. But this is the owner's call, not an
auditor's: either the gateway is retired and its requires dropped, or it is
repaired once and kept building. Changing a sibling repo needs approval
either way.

Note that the four Repository interfaces themselves **survive the
conversion** — a manager built by `New…Manager(db, *spec.Spec)` can still
implement them. What changes is the constructor signature, `routes`' exported
builders, and `tables.go`, all three of which the gateway calls.

### 3. No live table data was found, which removes the usual blocker

The readable table names (`custody_chapter_act_log_entry`,
`custody_log_entry`, `custody_export_job`, `custody_contact_read`,
`custody_consent_text_version`, `custody_consent_record`) are created by this
repo's own `AutoMigrate` from the gateway's startup, and the gateway has not
started since the taskmanager conversion broke it. Confirm against
`api.demo`'s database before relying on this — [[demo-db-is-the-working-database]]
is pumped continuously and must never be cleared — but if those six tables
are empty or absent, the legacy-adoption step (`spec.AdoptLegacy`) is not
needed and Step H is just `Provision` plus `cmd/ddl`.

## Conversion shape, once unblocked

Six objects, six roles, all append-only or write-once:

| Role | Today | Key shape |
| --- | --- | --- |
| `structure_act` | `StructureActLogEntryRow` | autoincrement bigint, append-only |
| `event` | `EntryRow` | autoincrement bigint, append-only |
| `export_job` | `JobRow` | minted string; the one object with real updates (`Progress`, `Complete`, `MarkFileRemoved`) |
| `contact_read` | `ReadRow` | minted string, append-only |
| `consent_version` | `TextVersionRow` | minted string; unique `(scope, version, language)`; write-once publish |
| `consent_record` | `RecordRow` | minted string, immutable |

Carry-across items that the conversion loses if not transcribed:

- **`ResumedFromPct` is write-once** and returns `ErrResumeAlreadyRecorded`.
  `spec.Field.Immutable` rejects a change after creation, which is not the
  same thing — this field starts null and may be set exactly once. This stays
  a Go rule.
- **`FileRemovedDueAt` must equal `SnapshotAt + 30 days`**, checked in
  `Job.Validate`. A cross-field equality no spec can state; stays Go.
- **Actor scope refuses `RequestedBy`, `IncludeIDs` and `pg_dump`.** Three
  conditional-presence rules; stay Go, with the type they are about.
- **`(scope, version, language)` is a database unique index**, deliberately,
  so `classify()` turns the race into `ErrConflict` rather than a
  read-then-write. This becomes a declared `Index` with `unique: true`.
- **`Detail`/`Evidence` must round-trip a true `nil`**, not `{}` — the
  two-year free-text ageing depends on it. `spec.TypeJSON` plus
  `nullable: true`; assert it in `postgres_integration_test.go`, since this is
  precisely the class of thing SQLite stands in for badly (catalog's CAT2).
- **The four `*Repository` interfaces stay byte-for-byte**, per `CLAUDE.md`.
- **`ErrUnknownActKind`/`ErrUnknownEventKind` refuse at the write.** A kind
  with no class arm must not land with a null class.

Every sentinel this repo exports must appear in `operations.json`'s `errors`
map or it is redacted to a 500 — catalog's CAT7. The sentinels are
`ErrNotFound`, `ErrUnknownActKind`, `ErrStructureRequired`,
`ErrUnknownEventKind`, `ErrAlreadyPublished`, `ErrInvalid`,
`ErrResumeAlreadyRecorded`, plus `errors.go`'s `ErrInvalidReference` and
`ErrConflict`.

## Also missing, independent of the two decisions

These are pure additions and can land whatever the decisions are:

- ~~A `Makefile` with `build`, `test`, `test-pg`, `vet`, `clean`~~ — **added
  2026-09-30**, gate 16 now passes. `test-pg` runs `-tags=integration` and
  currently matches no files, which is the next item.
- **There is no `postgres_integration_test.go` at all.** Four unit test files
  plus two route test files, all SQLite. That is the gap that matters most
  independent of the conversion, because `Detail`/`Evidence` are the exact
  nullable-`jsonb` round-trip that `CLAUDE.md` says must survive as a true
  `nil` — and SQLite is the dialect `CLAUDE.md` itself says zero-values it to
  `{}` on some dialects. The behaviour the repo documents as load-bearing is
  the one nothing tests. Catalog's CAT2 was exactly this: true on SQLite,
  false on Postgres.
- A `documentation/1. requirements/` record — this repo has none, which is
  why precondition 1 has no decision table to consult.
- `documentation/4. qa/` is empty.
