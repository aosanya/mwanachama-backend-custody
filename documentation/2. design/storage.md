# Storage

Why the store layer is shaped as it is. The code carries no comments, so the
reasoning that would have sat above each function lives here.

The objects themselves are declared in
[custody.blueprint.json](../../custody.blueprint.json), one field at a time
with its own `description`. This page is about the Go that a declaration
cannot replace.

## The two logs' ids are minted, not auto-incremented

`spec` declares an `int` primary key and offers no auto-increment, so
`store.go`'s `nextSequenceID` supplies the value: a Postgres `SEQUENCE` where
the dialect has one, and the table's own high-water mark on sqlite, which is
tests only and single-writer there. `provision.go`'s `createSequences`
creates the two sequences, and `cmd/ddl` prints them alongside the tables.

**The id has to increase with time**, which is why the high-water-mark
fallback is acceptable and a random id would not be. Both logs order by
`occurred_at DESC, id DESC`, and instants are stored to the second, so within
any one second the id *is* the ordering. A uuid would make the order of two
acts in the same second arbitrary.

This is the one place the conversion could not be a pure declaration. If
`spec` ever grows a sequential-key type, `nextSequenceID` and
`createSequences` both go.

## Timestamps are text, and the layout is load-bearing

`spec.TypeTimestamp` is stored as text so every dialect compares it the same
way, and `specstore`'s codec has no arm for `time.Time` — it would render a
struct through `fv.String()`. So `models` carries instants as `string`, in
`models.TimeLayout` (`time.RFC3339`), always UTC.

**Comparing them as text compares them chronologically**, which is what lets
`occurred_at >= ?` and `file_removed_due_at <= ?` be ordinary SQL. That only
holds because the format is fixed-width and single-zone; a local offset, or a
variable number of fractional digits, would order two instants wrongly.
`textVersionLess` relies on the same property when it sorts in Go.

`models.DueAt` and `Retention` stay `time.Time`/`time.Duration` arithmetic.
The string boundary is at the store, not through the whole module.

Shared is adding a `time.Time` arm to `specstore` with a nanosecond-precision
layout of its own; board row CU6 covers deciding whether to adopt it, and why
it is a stored-format change rather than only a type change.

## What each non-obvious piece of the store is for

- **`declaredValues`** (`validate.go`) reads an enum's values in the order the
  blueprint lists them, because that order is the order the log draws its
  filter chips. It used to be two hand-kept Go slices of the same thing.
- **`actScope`** (`custody_impl.go`) adds the structure filter only when the
  slice is non-empty. **This reproduces DEV-1697 exactly**, including the
  leak: an empty resolved subtree matches every structure rather than none.
  It is written this way on purpose, so the conversion changed no behaviour;
  the fix and its pinning test are on that board row.
- **`ListVersions`** (`consent_impl.go`) branches on dialect. Postgres puts
  NULLs first under `DESC` and needs `NULLS LAST` spelled out; sqlite has no
  such syntax and puts them last already, so that dialect sorts in Go. Only
  `postgres_integration_test.go` sees the branch that needed the clause.
- **`CreateVersion`** clears `published_at`, `published_by`, `superseded_at`
  and `copied_mechanics_id` whatever the caller supplied. A publish is the
  only writer of those, and the in-force rule derives from two of them with
  no guard of its own, so a caller-supplied value would put text into force
  without the act that authorises it. It does **not** clear `id`, which is
  DEV-1695.
- **`Progress`** puts the write-once guard on `resumed_from_pct` in the
  UPDATE's `WHERE` clause rather than in a read-then-write, so two workers
  racing cannot both believe they recorded the first figure. `explainNoRows`
  then tells the two reasons an UPDATE matched nothing apart: an unknown id,
  or a write-once violation on an id that does exist.
- **`MarkFileRemoved`** stamps the column and keeps the row. **There is no
  DELETE statement anywhere in `custody_impl.go` or `export_impl.go`**, and
  that absence is the append-only invariant rather than a rule inside one — a
  method that existed and refused would be one `if` away from not refusing.
- **`mustExist`** (`consent_impl.go`) checks both version references before
  writing a record, and returns `ErrInvalidReference` naming the column at
  fault. The records table has no real foreign key, because the two
  references point at a table whose physical name is a hash the domain spec
  decides.

## Why the wire shapes are written out

`wire.go` builds the act-log and custody-log responses as maps rather than
tagging the domain structs. **`ActorID` empty means the timer acted** — a
real state a client draws differently from a missing field — and the same for
the system on the custody log. A struct tag would make the stored value and
the rendered field move together by accident, and `omitempty` would erase
exactly the distinction that matters.

The field names also differ from the column names on purpose: `kind` is
rendered `act_kind` and `event_kind`, because the gateway's own clients read
them that way and the conversion held the wire shape.

## Error classification

`errors.go`'s `classify` maps a driver error onto `ErrInvalidReference` or
`ErrConflict` on either dialect: Postgres reports a structured
`pgconn.PgError` with a SQLSTATE in class 23, and sqlite reports plain text,
so that branch matches on the constraint-violation phrase. It is the reason
`consent_version`'s natural key can be a database constraint rather than a
pre-check — the race is lost in the database and surfaces as a typed error.

`ErrConflict` reaching the HTTP layer as a 500 rather than a 409 is
DEV-1696, and [routes.md](routes.md) says why it was left alone.
