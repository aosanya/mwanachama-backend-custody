# Routes

Every address this module answers is an entry in
[custody.operations.json](../../custody.operations.json). There are no
hand-written handlers: `routes/routes.go` is the sentinel map, the builder
ladder and nothing else, and adding an address is an edit to the operations
file rather than a new Go function.

The engine is `mwanachama-backend-shared/dispatch`, documented in that
repo's `documentation/2. design/dispatcher.md`.

## The table

Twelve addresses, all of them reads except the two consent writes. Every one
is byte-identical to the hand-written table it replaced.

| Method | Path | Action | Status |
| --- | --- | --- | --- |
| GET | `/structures/{structureID}/act-log` | `custody.structure_act.list` | 200 |
| GET | `/structures/{structureID}/act-log/counts` | `custody.structure_act.count` | 200 |
| GET | `/custody-log` | `custody.event.list` | 200 |
| GET | `/custody-log/counts` | `custody.event.count` | 200 |
| GET | `/export-jobs` | `custody.export_job.list` | 200 |
| GET | `/export-jobs/{jobID}` | `custody.export_job.get` | 200 |
| GET | `/actors/{actorID}/export-jobs` | `custody.export_job.list_own` | 200 |
| GET | `/actors/{actorID}/contact-reads` | `custody.contact_read.list_own` | 200 |
| GET | `/consent/versions/in-force/{scope}/{language}` | `custody.consent_version.in_force` | 200 |
| GET | `/consent/versions/{versionID}` | `custody.consent_version.get` | 200 |
| POST | `/consent/versions` | `custody.consent_version.create` | 201 |
| POST | `/consent/versions/{versionID}/publish` | `custody.consent_version.publish` | 200 |

## What is deliberately not here

The writes to the two append-only logs, and the export build's own
mutations. `AppendAct`, `AppendEvent`, `Progress`, `Complete` and
`MarkFileRemoved` belong to a writer inside the process or to a build
worker, never to a client, so they have no address at all. A route table
that cannot express them is a stronger guarantee than one that refuses them.

`CreateRecord` is likewise absent: a consent record is stamped at the
structure the actor is registered in, and resolving that is the mounting
host's job. It calls `Enroll` directly once the structure is known.

## The gate is data

`routes.AnonymousActions` is **empty**, and that is the whole gate
declaration. Nothing in an audit trail is public.

It is an allowlist of action ids, never a list of what is protected, so the
failure direction is right: an operation added to the operations file and not
named there arrives gated. A mount supplies `dispatch.Authorizer` and gets
`Split`/`PublicRoutes`/`OperatorRoutes` over the same table.

This module still carries no auth model of its own. The role-to-action grant
table is `mwanachama-backend-auth`'s.

## The address and the caller outrank the body

Two bindings carry rules that a handler used to enforce by hand:

- **`{from: path}`** binds every id, so a caller cannot `POST` to one
  address with a body naming another.
- **`{from: caller}`** supplies `published_by` on a publish and the
  comparison value on the two `list_own` addresses. An actor field is never
  a body field: an act that creates authority names who did it, and the
  binding also keeps the caller out of any tool schema, so an agent is
  refused by name rather than quietly ignored.

The self-only check itself lives in `CustodyManager`, because it is a
comparison between two bound values rather than a binding.

## Errors

`custody.operations.json`'s `errors` map is transcribed from the deleted
handlers' status tables, unchanged:

| Sentinel | Status |
| --- | --- |
| `ErrNotFound` | 404 |
| `ErrAlreadyPublished` | 409 |
| `ErrNotSelf` | 403 |
| `ErrUnknownScope` | 400 |
| `ErrUnknownActClass` | 400 |
| `ErrUnknownEventChip` | 400 |
| `ErrInvalidLimit` | 400 |
| `ErrInvalidSince` | 400 |
| `ErrStructureIDRequired` | 400 |

**An unmapped sentinel is redacted to 500 `"internal error"`.** Four exported
sentinels are deliberately unmapped, because the handlers they replaced did
not map them either:

- `ErrConflict` — a duplicate consent version. This is **DEV-1696**, an open
  bug: an ordinary double-submit arrives as an unexplained 500. It is left
  as it was found rather than fixed inside the conversion, and
  `routes/consent_routes_test.go` pins the broken status so the fix cannot
  land unnoticed.
- `ErrInvalidReference`, `ErrInvalid`, `ErrResumeAlreadyRecorded`,
  `ErrUnknownActKind`, `ErrUnknownEventKind`, `ErrStructureRequired` —
  unreachable from any declared address, since the writes they guard have no
  route. They would need mapping the moment one does.

`dispatch.statusTable` catches the opposite mistake — a spec naming a
sentinel nobody supplied — but not this one. Shared is growing
`dispatch.UnmappedSentinels` to close it in both directions; wiring it here
is a follow-up, and it will report the six above, which is correct and wants
a documented allowlist rather than a mapping.

## Reviewing the schema as SQL

`cmd/ddl <spec.json> [postgres|sqlite]` prints exactly what `Provision`
executes, in order, including the two sequences the logs' monotonic keys read
from. A spec can be reviewed as SQL before any database is given it.
