# mwanachama-backend-custody — completed work

## Initial build — 2026-09-06

Scaffolded this repo and ported `mwanachama-backend-api-gateway`'s
`internal/domain/{custody,export,contact,consent}` onto GORM-backed
relational storage, following the `mwanachama-backend-actor`/
`mwanachama-backend-comm` template — see [CLAUDE.md](../../CLAUDE.md) for
the full set of porting decisions. Corresponds to DEV-1640…1645 on the
gateway's `documentation/3. implementation/todo_custody_standup.md`.

| Task | Title | Status | Depends on |
|------|-------|--------|------------|
| DEV-1640 | Scaffold `mwanachama-backend-custody`: `go.mod`, `CLAUDE.md`, `models/`, `gormstore/`, `routes/` — file layout copied from `mwanachama-backend-actor` | ✅ Done | — |
| DEV-1641 | Port `internal/domain/custody` into `models/`+`gormstore/`+root-package `CustodyStore` | ✅ Done | DEV-1640 |
| DEV-1642 | Port `internal/domain/export` into `models/`+`gormstore/`+root-package `ExportStore` (domain+storage port only — no gateway HTTP route existed to mirror) | ✅ Done | DEV-1640 |
| DEV-1643 | Port `internal/domain/contact` into `models/`+`gormstore/`+root-package `ContactStore` | ✅ Done | DEV-1640 |
| DEV-1644 | Port `internal/domain/consent` into `models/`+`gormstore/`+root-package `ConsentStore` (+ `Enroll`) | ✅ Done | DEV-1640 |
| DEV-1645 | Build `routes/` route builders (`CustodyRoutes`, `ExportRoutes`, `ContactRoutes`, `ConsentRoutes`) | ✅ Done | DEV-1641, DEV-1642, DEV-1643, DEV-1644 |

`go build ./...`, `go vet ./...` and `go test ./...` (sqlite, in-process —
no Postgres container) all green.

## Declared-domain conversion — 2026-09-30

| Task | Title | Status | Depends on |
|------|-------|--------|------------|
| CU1 | **Declared-domain audit** — this repo against the standard `mwanachama-backend-catalog` sets. One gate of eighteen passes. Written up in [documentation/2. design/declared-domain-audit.md](../2.%20design/declared-domain-audit.md), which is the reference for CU2–CU5 below. | ✅ Done 2026-09-30 | — |
| CU2 | **RESOLVED (owner 2026-09-30): the module owns all 45 kind values, and this repo is a recorded exception to the domain-neutrality rule.** A domain cannot supply them (CAT5 refuses a domain-supplied `values` by name at load) and moving kind/class into a json document would lose the class index that `models/act.go` says is stored precisely so the filter is an index scan. Reasoning and what follows for `domain_agnostic_test.go` are written up as D1 in [documentation/1. requirements/declared-domain-decisions.md](../1.%20requirements/declared-domain-decisions.md). | ✅ Done 2026-09-30 | — |
| CU3 | **RESOLVED (owner 2026-09-30): `api-gateway` is treated as retired; convert freely and do not repair it.** It has not compiled since taskmanager's conversion (`cmd/server/stores.go:324` wants the deleted `DefaultTableNames`) and `wakala-api`, the live consumer, does not import this repo. This releases `CLAUDE.md`'s pin on the four `*Repository` interfaces — recorded as D3. Dropping the gateway's `require` lines is tracked org-wide, not here. | ✅ Done 2026-09-30 | — |
| CU5 | **Conversion Steps A–J**, per the audit's "Conversion shape, once unblocked": six roles (`structure_act`, `event`, `export_job`, `contact_read`, `consent_version`, `consent_record`), five of them append-only or write-once. Carry-across items the audit enumerates and the conversion loses if not transcribed: `ResumedFromPct` write-once (not `Immutable` — it starts null and is set once), `FileRemovedDueAt == SnapshotAt + 30d`, the three actor-scope refusals, the `(scope, version, language)` unique index that `classify()` turns into `ErrConflict`, and all nine exported sentinels in `operations.json`'s `errors` map or they redact to 500 (catalog's CAT7). Note the `chapter` → `structure` rename stopped at the storage boundary: row structs and the `chapter_id` column still say chapter. | ✅ Done 2026-09-30 | — |
