# Declared-domain audit

This repo against the standard `mwanachama-backend-catalog` sets, per the
strategy in `developer/documentation/2. design/architecture-spec-driven-modules.md`
(decided 2026-09-23). Audited 2026-09-30 against commit `637ed23`, working
tree clean.

Baseline: `go test ./...` passes, no Go file exceeds 300 lines.

> **Superseded 2026-09-30.** The conversion landed (board CU5) and this repo
> now passes every gate; the re-audit is at the end of this page. What
> follows is the audit as taken, kept because the two preconditions it
> raised are the decisions the conversion turned on, and because the
> carry-across list is the record of what had to survive.

## Verdict (as audited, before the conversion)

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

---

## Re-audit, 2026-09-30, after the conversion

| # | Gate | Verdict |
|---|------|---------|
| 1 | Blueprint at root | pass — `custody.blueprint.json`, six objects |
| 2 | Blueprint reached from Go | pass — `blueprint.go`, `//go:embed` + `sync.OnceValues` |
| 3 | Domain specs (≥2) | pass — `mwanachama.custody.json` and `clinic.custody.json` |
| 4 | No row structs | pass — `gormstore/` and `tables.go` deleted, no `AutoMigrate` anywhere |
| 5 | Store is thin over `specstore` | pass — `store.go` is roles, a `store` alias and wrappers |
| 6 | Constructor cross-check | pass — every constructor takes `(db, *spec.Spec)` and passes all six carriers |
| 7 | Declared rules | pass — `validate.go` reads `required`/`values`/`matches`; `patterns.go` supplies `instant` |
| 8 | Thin models | pass with a stated exception, below |
| 9 | Operations declared | pass — `custody.operations.json` + `operations.go` |
| 10 | Routes are an adapter | pass — `routes/routes.go` is 92 lines of sentinel map and builder ladder |
| 11 | Gate is data | pass — `AnonymousActions` is an empty allowlist of action ids |
| 12 | MCP is the same table | n/a — no `mcp/` package |
| 13 | Provisioning | pass — `Provision` + `cmd/ddl` |
| 14 | Guard tests | pass, with one unrun — below |
| 15 | Depends on shared | pass |
| 16 | Makefile | pass |
| 17 | Documentation | pass — all four phases now carry a page |
| 18 | House rules | pass — no comments in the new code, largest file 246 lines |

### Gate 8's exception

Three `Validate` methods survive in `models/`, and they should:

- `Job.Validate` holds the rules a spec cannot state — `file_removed_due_at`
  equals `snapshot_at` plus the retention window, `snapshot_at` is not before
  `requested_at`, and the three actor-scope refusals. Its **enum** switches
  on scope, format and status are gone; the declared `values` carry them.
- `ProgressUpdate.Validate` and `Completion.Validate` are on types that are
  **not declared objects at all** — they are partial-update commands, so
  there is no blueprint field for a spec to validate them against.

The two membership maps, their `IsX` functions and every per-field doc
comment are gone.

### Gate 14's caveat

Five of the six guards were broken on purpose and watched go red. The sixth,
`postgres_integration_test.go`, **has never been run**: no Postgres was
reachable and standing one up for scratch verification is against this
repo's own convention. It compiles and skips correctly. Board row CU4.

One thing worth recording from that exercise: `TestTwoDomainsCoexist` is
narrower than it looks. Giving the clinic `mwanachama`'s instance did **not**
make it fail, because the table hash covers the domain's own table name too.
It only failed once both the instance *and* the table names matched.

### What the conversion did not do

- **No legacy-adoption step**, because no live rows exist under the old
  readable names — the gateway that created them has not started since it
  stopped compiling. A fresh instance needs none. Confirm against `api.demo`
  before provisioning anywhere real.
- **No SQL mirror in the gateway's active `migrations/`.** The usual rule is
  that every declared domain also needs one so `cmd/migrate up` alone can
  provision a fresh database, but the gateway is retired by D3 and its
  migration path is not being maintained. `cmd/ddl` prints the statements for
  whatever does provision this module.
