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
