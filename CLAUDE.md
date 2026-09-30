# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-custody

Four domains that share one theme — **an immutable record of who did what to
whose data**. Domain logic AND storage both live in this package, imported
directly by whatever mounts it — no separate service, no gRPC, no proto.
Module path `github.com/aosanya/mwanachama-backend-custody`.

- **custody** — the two append-only logs: `structure_act` (a structure's own
  record of what its leadership did) and `event` (the organization's audit
  trail of everything done to its data *as data*).
- **export** — `export_job`, both the administrator's register and an
  actor's own self-export.
- **contact** — `contact_read`, the record of an operator viewing an actor's
  contact details.
- **consent** — `consent_version` and `consent_record`: the versioned sheet
  and an actor's acceptance of it.

Originally extracted from `mwanachama-backend-api-gateway`'s
`internal/domain/{custody,export,contact,consent}`, decided 2026-09-06
(DEV-1640…1648). **That consumer is retired**: the gateway has not compiled
since taskmanager's conversion, and `wakala-api` — the live consumer — does
not import this repo. The pin that once held the four `*Repository`
interfaces byte-for-byte is released, which is what let the conversion move
their timestamp parameters to strings (D3 and D2 in
[documentation/1. requirements/declared-domain-decisions.md](documentation/1.%20requirements/declared-domain-decisions.md)).

Converted to the declared-domain standard on 2026-09-30, against the audit in
[documentation/2. design/declared-domain-audit.md](documentation/2.%20design/declared-domain-audit.md).

## Objects are declared, not written

The tables come from JSON, not from Go row structs — see
`developer/documentation/2. design/architecture-spec-driven-modules.md` for
the strategy and `mwanachama-backend-shared`'s
`documentation/2. design/declared-domains.md` for the engine.

- `custody.blueprint.json` — **the module's six objects, declared once**:
  `structure_act`, `event`, `export_job`, `contact_read`,
  `consent_version`, `consent_record`. Embedded into the root package and
  reached through `Blueprint()`, `LoadSpec(path)` and `ParseSpec(raw)`. Load
  a domain spec through those, never through `spec.Load`, or its roled
  objects arrive with no fields.
- `spec/examples/mwanachama.custody.json` and `clinic.custody.json` — the
  same module under two domains. The first calls a structure a **chapter**,
  which is exactly the word the module must not know. The second is a domain
  nobody has provisioned, so the shipped example is not also production
  config, and it declares `retention_hold` as an object of its own with no
  role.
- `Provision(db, *spec.Spec)` creates the two sequences and calls
  `spec.Migrate`. `cmd/ddl <spec.json> [postgres|sqlite]` prints the same
  statements so a spec can be reviewed as SQL first.

**A domain names objects; it does not re-declare them.** Its spec supplies
`instance`, the name and table each role lands in, its own indexes, and a
default on a declared field — nothing else. Module-level invariants that are
not a domain's choice are declared as blueprint indexes and merge ahead of
the domain's: `consent_version`'s `(scope, version, language)` natural key
stays a **unique database constraint**, so a concurrent caller loses the race
in the database rather than slipping past a read-then-write.

**The spec and the types are checked against each other when a store is
built.** A declared column with no field, or a field with no column, fails in
`newStore` rather than dropping a value on every write. Every constructor
passes a carrier for all six roles, so building any one store validates the
whole spec.

**A column is found by field name, never by json tag**, and **every declared
column is written on every write** — a map missing a key means "leave it
alone" to an update, so omitting empty values would make clearing a field
impossible.

**The two logs' ids are a declared `int` primary key minted from a sequence.**
`spec` has no auto-increment, so the value comes from a Postgres `SEQUENCE`
where there is one and from the table's high-water mark on sqlite, which is
tests only and single-writer there. The id must increase with time because
both logs order by `occurred_at DESC, id DESC` and instants are stored to the
second, so the id is the tiebreaker within one second rather than decoration.

**Timestamps are RFC 3339 strings, not `time.Time`.** `spec.TypeTimestamp`
stores as text and `specstore`'s codec has no arm for `time.Time` — it would
render a struct through `fv.String()`. Comparing them as text compares them
chronologically because they are fixed-width UTC, which is what lets
`occurred_at >= ?` work. `models.DueAt` and `Retention` stay time arithmetic:
the string boundary is at the store.

**A required field must not carry a default.** The default is exactly what
would let an omitted value pass unnoticed.

## Rules are declared too, where they can be

`validate.go` reads `required`, an enum's `values` and `matches` off the spec
and applies them on the way in. `patterns.go` names the one pattern this
module supplies, `instant`; a spec naming one nobody supplies is an error,
not a rule that quietly never runs.

`models/vocabulary.go` keeps only the constants Go compares against. The two
membership maps and their `IsX` functions are gone, and so are the
scope/format/status switches inside `Job.Validate` — the declared `values`
carry them.

**The declared order of `values` is load-bearing.** The log draws its filter
chips in the order the blueprint lists `class` and `chip`, and
`CustodyStore.Classes()`/`.Chips()` read that order off the spec.
`ActClasses()`/`EventChips()` used to be two hand-kept Go lists of the same
thing.

`Check(s, role, v)` is the same validation without a database, for a bulk
import validating what it read before it opens a connection.

**What stays in Go is what a spec cannot say**, and each piece lives with the
type it is about:

- `ActClassOf`/`EventChipOf` — a kind-to-class **mapping**, which is a
  function, not a flat enum. A kind with no arm is refused at the write,
  before a row exists to be miscounted.
- `ResumedFromPct` is **write-once**, and the guard is in the UPDATE's WHERE
  clause rather than a read-then-write, so two workers racing cannot both
  believe they recorded the first figure. This is not `immutable`: the field
  starts null and may be set exactly once.
- `file_removed_due_at == snapshot_at + 30 days` — a cross-field equality.
- The three actor-scope refusals (no `requested_by`, no `include_ids`,
  `csv_zip` only) — conditional presence.
- The append-only invariant, which is the **absence** of an UPDATE and a
  DELETE in `custody_impl.go`. A method that existed and refused would be one
  `if` away from not refusing.

## This module names no domain, with one recorded exception

A word that means something in one domain and nothing in another does not
belong here. A structure is a chapter to one organization and a ward to
another; the module says `structure`.

`domain_agnostic_test.go` enforces it by walking the AST of `.`, `models/`
and `routes/`, with a second test over every declared enum value — a stored
value outlives a rename. `member` is forbidden and `membership` is not:
every organization has membership, only some have members.

**The exception is the 45 kind values.** `phone_salt_rotated`,
`break_glass_write` and `themes_recomputed` are platform words, and they are
declared in this module's own blueprint anyway, because a domain may not
supply a declared field's `values` and moving kind and class into a document
would lose the class index the log's filter relies on. This module's subject
*is* the platform's own operations. The reasoning, and what should reopen it,
is D1 in
[documentation/1. requirements/declared-domain-decisions.md](documentation/1.%20requirements/declared-domain-decisions.md).

## What is superseded

- **`gormstore/`** — **deleted 2026-09-30 (CU5)**. Its six row structs and
  their `*ToRow`/`*FromRow` converters are what `spec.Migrate` and
  `specstore`'s codec replace; root `tables.go` and its
  `TableNames`/`DefaultTableNames`/`Migrate` re-export went with it. Its
  `mintID` and `createSequences` are now `store.go`'s `nextSequenceID` and
  `provision.go`.
- **The converters also bridged two vocabularies**, and that is gone with
  them: rows spoke `ChapterID`/`ActorChapterID`/`ToChapterID` and
  `MemberID`/`SubjectMemberID` while the domain types already said
  `Structure`/`Actor`. The blueprint declares the domain's words and nothing
  bridges (D4).
- **`routes/`** — the five hand-written handler files are **deleted
  2026-09-30**. The table is declared in `custody.operations.json` and built
  by `dispatch`; `routes.go` is the sentinel map and the builder ladder. The
  four constructors (`CustodyRoutes`/`ExportRoutes`/`ContactRoutes`/
  `ConsentRoutes`) are gone on purpose: the gate boundary they encoded is now
  `AnonymousActions`, so a mount names actions rather than this module's Go
  identifiers. See [documentation/2. design/routes.md](documentation/2.%20design/routes.md).
- **`routes/wire.go`** — deleted. It was a byte-for-byte copy of
  `httpwire.WriteJSON`/`WriteErr`, and the local `Route`/`Pattern` pair was a
  copy of `httpwire.Route`. `Route` is now an alias for it.
- **`routes/identity.go`'s `Identity`** — deleted. The caller comes from
  `dispatch.Caller` and is bound with `{from: caller}`. `ScopeResolver`
  survives, on `CustodyManager`, because a structure hierarchy really is an
  externally supplied fact.
- **`models/`** — shrunk: two membership maps, two `IsX` functions, the three
  enum switches in `Job.Validate`, and the per-field doc comments, which were
  a hand-rewritten copy of what the blueprint's `description` now owns.

## Known bugs kept exactly as found

The conversion held behaviour, including three filed bugs. Each keeps a
deliberately-red pinning test green, and fixing any of them inside a refactor
is what would hide it:

- **DEV-1695** — `POST /consent/versions` still honours a caller-supplied
  `id`. One half of it did close as a side effect: minted ids are now opaque
  (`specstore.NewID`) rather than `consentversion-<n>` from a sequence, so the
  future-sequence-collision attack is gone. The squatting half stands.
- **DEV-1696** — a duplicate natural key still answers 500, because the
  deleted handler's status table did not map `ErrConflict` and the table was
  transcribed exactly.
- **DEV-1697** — an empty resolved subtree still matches every structure
  rather than none.

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite via `glebarez/sqlite`) is the expected way to verify
  a change here — do not reach for a real Postgres. What SQLite cannot stand
  in for lives in `postgres_integration_test.go` (`//go:build integration`,
  gated on `POSTGRES_URL`, run by `make test-pg`), which is **not** part of
  `go test ./...`.
- No Go file over 300 lines; split by responsibility.
- **Identifiers reach SQL as text.** `spec.NamePattern` and
  `spec.DocPathPattern` refuse anything outside a strict alphabet; never relax
  either to "escape it instead". Assert on `spec.RawNameFor`, never on a
  physical table name literal, and exclude `%_spec_table_names` from anything
  that counts tables.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
