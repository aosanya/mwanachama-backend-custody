# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-custody

Extraction of `mwanachama-backend-api-gateway`'s audit/compliance cluster —
`internal/domain/{custody,export,contact,consent}` — onto GORM-backed
relational storage: domain logic AND storage both live in this package,
imported directly by the gateway process — no separate service, no gRPC, no
proto. Module path `github.com/aosanya/mwanachama-backend-custody`. Follows
the template `mwanachama-backend-actor` and `mwanachama-backend-comm` already
set (GORM over hand-rolled SQL/memory stores, `models/`+`gormstore/` split,
`routes/` for the surface that carries no gateway-internal composition).

Decided 2026-09-06 (full scope, full cutover), `mwanachama-backend-api-gateway/
documentation/2. design/design-register.md` and
`architecture-domain-decomposition.md`. Built for
`documentation/3. implementation/todo_custody_standup.md`, DEV-1640…1648.

Houses four domains that share one theme — an immutable record of who did
what to whose data:

- **custody** — the two append-only logs: `chapter_act_log_entry` (a
  chapter's own record of what leadership did) and `custody_log_entry` (the
  organization's audit trail of everything done to its data *as data*).
- **export** — `export_job`, both the admin custody export and the member
  self-export.
- **contact** — `contact_read`, the record of an operator viewing a member's
  phone number.
- **consent** — the versioned consent sheet and a member's acceptance record.

**This board explicitly depends on `todo_actor_absorb.md` and
`todo_comm_absorb.md`, not the other way round** — those two boards' `role`
and `member` domains write act-log rows into this repo's custody log, so
this repo has to exist and compile first.

## Porting notes

- `models/` holds the domain types, one cluster per file (`act.go`+`event.go`
  for custody, `export.go`+`progress.go` for export, `contact.go`,
  `consent.go`), plus `repository.go` for the four `*Repository` interfaces.
  Ported field-for-field from the gateway's four packages, with two
  deliberate divergences the flattening forced:
  - **One shared `ErrNotFound`**, not four per-domain sentinels. The source
    packages each had their own `ErrNotFound` because each package's own
    callers already knew which domain they were calling; here every method
    on every repository still identifies its own failure mode by which
    method returned it, so a single sentinel loses nothing and four
    identically-worded ones would have been pure repetition.
  - **`ConsentScope`, not `Scope`.** The gateway's `consent.Scope` and
    `export.Scope` were two types in two packages with the same bare name;
    flattened into one `models` package, one of them had to be renamed to
    avoid a collision. `consent`'s `Scope` lost the name (`export.Scope`
    kept it) because "export scope" already reads naturally as a phrase
    on its own, where "consent scope" needed the noun spelled out anyway
    for a reader unfamiliar with the domain.
  - **`models.DefaultPage` (200) replaces two identically-valued constants**
    (`custody.DefaultPage`, `export.DefaultPage`) that existed only because
    the source packages were separate and neither could reference the
    other's. Now that both live in one module there is exactly one page-size
    decision to make, not two copies of the same one.
- `gormstore/` holds the GORM row structs and row↔domain conversion, one
  file per cluster, mirroring comm's split. **The two append-only logs'
  ids are plain database autoIncrement bigints**
  (`ChapterActLogEntryRow.ID`, `EntryRow.ID`), not comm's minted,
  prefixed strings — the source objects' own contract is "monotonic, read
  newest-first, never re-ordered" with no human-readable prefix convention
  the way `export_job`/`contact_read`/`consent_*` ids have, so the
  database's native `BIGSERIAL`/`AUTOINCREMENT` already gives everything
  the column needs. The four string-id domains use `gormstore.mintID`
  against a Postgres `SEQUENCE` (sqlite tests fall back to a uuid suffix),
  copied from comm's own `mintID` unchanged.
- **`Detail`/`Evidence` are raw `datatypes.JSON`, not `datatypes.JSONMap`,**
  on both log rows. Unlike `Actor.Attributes` (always a flat, present map),
  a chapter act's `Detail` is nullable and its absence — after G89's
  two-year free-text ageing, a feature this port does not yet build but
  whose column shape must survive it — has to round-trip as a true `nil`,
  not an empty object; `datatypes.JSONMap` zero-values to `{}` on some
  dialects, which would make an aged-out row look like it still carried
  structured keys. `export_job`'s `Datasets`/`RowCounts` and
  `consent_text_version`'s `Clauses`/`Effect` are plain JSON text columns
  for a related but distinct reason: this repo's own `gormstore.Migrate`
  runs `AutoMigrate` against both Postgres and sqlite, and a column shape
  both dialects render identically is worth more here than a Postgres-only
  `text[]` would be.
- **`consent_text_version`'s natural key (scope, version, language) is a
  database unique index**, not a pre-check-then-insert. `ConsentStore.
  CreateVersion` relies on `classify()` turning the resulting constraint
  violation into `ErrConflict` — the same "let the database catch the race"
  shape actor's unique-attribute indexes use, rather than a
  read-then-write that a concurrent caller could still slip past.
- **This repo carries its own `ErrInvalidReference`/`ErrConflict` and its
  own `classify()`,** copied from comm's `errors.go` (SQLSTATE class 23 on
  Postgres, string-matching on sqlite's plain-text driver errors) rather
  than importing the gateway's `internal/domain/domerr` — this module
  cannot import a package that lives inside the gateway's own module, the
  same constraint comm's identically-shaped `errors.go` documents.
- **`routes/` ports fourteen of the gateway's eighteen custody/export/
  contact/consent routes** — the ones that are, underneath, a plain call
  against one of this repo's four Repository interfaces with no
  gateway-internal domain (chapter/member/role) composed into the handler
  body, the same line actor's and comm's own `routes/doc.go` draw:
  - **Custody**: all four (`listChapterActs`/`countChapterActs`/
    `listCustodyEvents`/`countCustodyEvents`). The chapter-log pair takes a
    `ScopeResolver` for `?scope=subtree`'s descendant-closure walk — an
    externally supplied fact, the same shape `HierarchyChecker` (actor) and
    `Identity` (comm) already established, since walking
    `chapter.Repository.ListChildren` is gateway-internal. No capability
    gate is baked into any of the four; a mounting process wraps each
    Handler the way it already wraps actor's `GroupRoutes`.
  - **Export**: three reads (`Get`/`ListOrganization`/`ListForMember`).
    **Net-new surface** — DEV-1642 confirmed by grep that the gateway
    registers no `export` HTTP path at all, so there was no existing
    handler to mirror, only the Repository's own contract to expose
    safely. `Create`/`Progress`/`Complete`/`MarkFileRemoved` are absent on
    purpose: the Repository's own doc says they belong to a build worker,
    never a client, and this package is not the place to invent one.
  - **Contact**: one of one (`ListForSubject`), **undecorated**. The
    gateway's real `GET /v1/members/{memberID}/contact-reads` renders
    operator name, role name and chapter name — none of which live on
    `contact_read` itself — by composing `member.Repository`/
    `chapter.Repository`/`role.Repository` in the handler body
    (`contactReadViews`/`roleNameAt`). That composition cannot move here,
    for chat/dm/moderation's excluded-route reason in comm's own
    `routes/doc.go`; the gateway's handler stays responsible for
    decorating the plain list this package now serves underneath it.
  - **Consent**: four of seven (`GetInForce`/`GetVersion`/`CreateVersion`/
    `PublishVersion`). `enrollConsent` resolves which chapter a record is
    stamped at from the member's own live registrations
    (`member.Repository`, DEV-1269) before ever calling `Enroll` — that
    resolution is gateway-internal, so enrollment itself stays a gateway
    handler that now calls this repo's `mwanachamacustody.Enroll` directly
    once `chapterID` is resolved. `listMemberConsent`/`currentMemberConsent`
    gate on "the member, or an operator holding `CapConsentAdmin`" — an
    `OR` a capability lookup decides — so both stay gateway handlers over
    `ConsentRepository.ListForMember`/`.CurrentForMember` directly.

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- Before wiring into `mwanachama-backend-api-gateway`, the four Go
  interfaces that must NOT change are `models.CustodyRepository`/
  `.ExportRepository`/`.ContactRepository`/`.ConsentRepository` — route
  paths, request/response shapes and status codes in the gateway's own
  `internal/api/http` all depend on those staying byte-for-byte identical
  to the source packages' `Repository` interfaces they mirror.
