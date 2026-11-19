# Hub import drift

This page lists every difference between this repository and the hub's in-tree copy of the module (`projects/hub-steward/code/wakala-api/modules/custody`), written before the hub drops its copy and imports this repository. It reads the result of [hub-copy-diff.md](hub-copy-diff.md) afresh and adds what the script cannot see: the hub code that sits around the module.

Checked on 2026-11-19 against the hub branch `dev` at 43f22b26. The hub folder has no uncommitted change. The newest hub commit that touches it is 1e5d971d (a clerk records consent given on paper), and this repository carries that change in 7bfebdc and f32255d.

## Method

1. `scripts/hub-copy-diff.sh` with its defaults. It rewrites the hub's import paths, package name and error prefix to this repository's, drops comments and compares. It exits 1 on the live trees, for the reasons in the tables below.
2. A file listing of both folders, to catch what the script does not copy (it reads `*.go` and `*.json` only). The hub folder holds nothing else.
3. `git status` and `git log` of the hub folder, to catch uncommitted work and hub commits newer than the last one carried here.
4. A search of the whole hub `code/wakala-api` for the module's import path and for the word custody, to list the wiring that is not part of the module.

## Only in the hub

Nothing. The script prints no `Only in hub` line, no hub file is absent here, and no hub behaviour is missing here.

| Item | Side | Disposition |
|---|---|---|
| Files of the hub folder | none only in the hub | Nothing to move. |
| Hub commits after 0a132371 | 1e5d971d only, already carried | Nothing to move. |

## Only in this repository

| Item | Kind | Disposition |
|---|---|---|
| `consent_text.go`: a consent text publish repeats up to twenty times with a jittered wait, numbers the next label as the highest `v<number>` plus one, and deletes its draft when the publish fails; `custody_manager.go` holds a mutex for it | Behaviour | Kept. The hub takes it on import. The hub copy numbers by count, so a hand-made draft or two publishes at the same moment give it a 500. |
| `consent_impl.go`: `DeleteDraft`, and a Postgres advisory lock around publishing a version | Behaviour | Kept. |
| `errors.go`: a unique conflict names the index fields only, with no table name and no driver code | Behaviour | Kept. |
| `custody.operations.json` and `routes/routes.go`: `ErrConflict` maps to 409 | Behaviour | Kept. The hub copy answers a duplicate consent version with 500. |
| `consent_text_concurrent_test.go`, `consent_text_failed_publish_test.go`, `consent_text_many_managers_test.go`, `consent_text_taken_label_test.go`, `consent_text_test.go` | Test | Kept. |
| `errors_unique_fields_test.go`, `routes/consent_conflict_body_test.go` | Test | Kept. |
| `consent_in_force_test.go`, `routes/give_description_test.go`, `routes/consent_tools_test.go`, `spec_for_test.go` | Test | Kept. They cover behaviour the hub proved only from its own API layer. |
| `postgres_integration_test.go` | Test, build tag `integration` | Kept. It needs a Postgres database. |
| `routes/routes_cover_test.go` | Test | Kept, and repaired in the same change: it still called the route builders that were removed when the routes moved onto the dispatch table, so the `routes` package did not compile its tests. |
| `cmd/ddl`, `scripts/hub-copy-diff.sh`, `scripts/hubdiff`, `documentation/`, `README.md`, `CLAUDE.md`, `Makefile`, `LICENSE`, `go.mod`, `go.sum` | Repository files | Kept. A module folder in the hub never had them. |

## Differing beyond the import path

| File | How it differs | Cause | Disposition |
|---|---|---|---|
| `consent_impl.go` | Here: `DeleteDraft` and the advisory lock. Hub: neither. | The hub copy lacks the concurrent-publish fix. | This repository's version stands. |
| `consent_text.go` | Here: retry, backoff, highest-label numbering, draft cleanup. Hub: numbers by count, no retry, leaves an orphan draft on a failed publish. | Same. | This repository's version stands. |
| `custody_manager.go` | Here: one extra mutex field, `publishText`. | Same. | This repository's version stands. |
| `errors.go` | Here: the conflict message lists the unique fields. Hub: it carries the driver's constraint text. | The hub copy lacks the fix. | This repository's version stands. |
| `custody.operations.json` | Here: `"ErrConflict": 409`. Hub: no such entry. | The hub copy lacks the fix. | This repository's version stands. |
| `routes/routes.go` | Here: `ErrConflict` is in `Sentinels`. Hub: it is not. | Same. | This repository's version stands. |
| `routes/consent_routes_test.go` | Here: a duplicate natural key answers 409 and names the fields. Hub: the same test asserts 500 and "internal error", with a failure message saying the test must change once the bug is fixed. | Same. | This repository's version stands. The hub's assertion records the old behaviour, so it is not a second behaviour to keep. |

The hub's `package custody` and error prefix `custody:` differ from this repository's `package mwanachamacustody` and prefix `mwanachamacustody:`. The script rewrites them before comparing, so they show in no table above. They are owner question 1.

## Hub wiring that is not part of the module

Not moved. Each file stays in the hub and changes its import when the hub takes this repository.

| Hub path (under `code/wakala-api`) | What it does with the module |
|---|---|
| `internal/api/http/custody_routes.go`, `operation_routes.go`, `router.go` | Mount the route table under `/instances/{instanceID}/custody`, with the caller read from the session. |
| `internal/api/http/mcp_tools_custody.go`, `mcp.go`, `mcp_visible.go`, `mcp_audit_paper_consent.go` | Register the custody tools on the MCP surface, filter what each role sees, and audit the paper-consent tool. |
| `internal/api/http/membership.go`, `authorize.go`, `resident_grants.go`, `consent_required.go` | The resident and permissioned action lists, the authorizer, and the refusal of a data call without consent. |
| `internal/api/http/custody_consent_language.go` | Fills in the agency's content language when a resident agrees without naming one. |
| `internal/agencies/registry.go`, `caches.go`, `mounts.go`, `test_copy.go`, `consent_gate.go`, `consent_withdrawal.go`, `earlier_withdrawals.go` | Build one `CustodyManager` per agency from `SpecForMount`, cache it, subscribe to withdrawals, and stamp what a withdrawal hides. |
| `internal/accounts/router.go` | Hands the registry's `CustodyManager` to the API layer. |
| `modules/agency/models/permissions_catalog.json`, `modules_catalog.json` | The custody permissions, among them `consent.record`, and the custody module entry. |
| Roster pins in the hub's own tests | Counts of tools and permissions that the hub pins. They stay in the hub. |

## Changes made here for the import

The hub has nothing to merge in, so the only edits are the repairs the test run needed. `go test ./...` on in-memory SQLite is green after them.

| File | Change | Pinned count |
|---|---|---|
| `routes/routes_cover_test.go` | Rewritten onto the dispatch table: `routes.Build`, `RoutesFor`, `Split`, `Table.Shape` and `BuildTools` replace the removed `BuildWith`, `Routes`, `RoutesWith`, `Shape`, `PublicRoutes`, `OperatorRoutes` and `SplitWith`. The refusal-status, own-export, own-contact-read, empty-organization, authorizer 403 and allow tests are kept, and the duplicate-action check still runs over the shape. | `declaredRouteCount` 12 to 19, the number of routes the operations file declares since the consent routes joined it. |
| `routes/consent_tools_test.go` | The expected tool list gains `custody_record_paper_consent`. | Tools 6 to 7, because the paper-consent operation is a tool here and in the hub. |

## Things the hub meets when it imports

These are not drift in the module. They are listed so the import step starts with them.

| Item | Detail |
|---|---|
| Shared module | The hub imports its own `wakalaserver/modules/shared`. The module imports `github.com/aosanya/mwanachama-backend-shared`, and its `routes.Mount`, `Tool` and `Route` are aliases of that module's types. The hub wiring passes these into its own shared copy's helpers, so the two copies must agree, or the hub must take the shared module first. |
| Package alias | `operation_routes.go` imports the module as `custody`. Every other hub file uses `mwanachamacustody`. |
| Module tag | The hub needs a tagged version of this repository. |

## Owner questions

1. Should the module's package keep the name `mwanachamacustody` and the error prefix `mwanachamacustody:`, or take the hub's `custody` and `custody:`? Callers see the prefix in the text of a 400, 404 or 409 answer. The hub copy writes `custody: `, this repository writes `mwanachamacustody: `, and both behave the same otherwise. No hub test outside the module folder pins the prefix.
