# Hub copy diff

`mwanachama-backend-custody` is the custodian of the custody module. The hub (`projects/hub-steward/code/wakala-api/modules/custody`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue CU8, the custody half of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [hub-folder [module [shared [hub-module [hub-shared]]]]]
```

The defaults are `../projects/hub-steward/code/wakala-api/modules/custody` relative to this repository's parent folder, `github.com/aosanya/mwanachama-backend-custody`, `github.com/aosanya/mwanachama-backend-shared`, `wakalaserver/modules/custody` and `wakalaserver/modules/shared`. The exit code is 1 when the trees differ, 0 when they do not. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-git` (G78), generalised to take the module paths.

## What it does

1. Copies the `*.go` and `*.json` files of this repository (without `cmd/`, `documentation/`, `scripts/`) and of the hub folder into two temporary folders.
2. Rewrites the hub's imports: the hub shared path becomes `github.com/aosanya/mwanachama-backend-shared` and the hub custody path becomes `github.com/aosanya/mwanachama-backend-custody`.
3. Rewrites the hub's names for the module: its `package custody` becomes `package mwanachamacustody`, its import alias `custody` becomes `mwanachamacustody`, its selectors `custody.Name` become `mwanachamacustody.Name`, and its error prefix `"custody: ` becomes `"mwanachamacustody: `.
4. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
5. Runs `diff -r backend hub`.

Step 3 exists because the hub's port of 2026-11-07 (hub a1f9d1aa) renamed the package and the error prefix, while this repository keeps `mwanachamacustody` as `mwanachama-backend-git` keeps `mwanachamagit`. The hub imports the module under the alias `mwanachamacustody` in its own callers. Whether the module should instead take the hub's name `custody` is an owner decision (todo TD-1608 on CU8); until it is made the rewrite hides the difference.

## Result of 2026-11-16

Every non-test Go file and every JSON file is identical after the rewrite, and every test file of the hub exists here and is identical. What remains is `Only in backend` lines, each a test file with its reason:

| Difference | Reason |
|---|---|
| Only here: `postgres_integration_test.go` | Build tag `integration`: it needs a Postgres database, which the hub does not run. |
| Only here: `spec_for_test.go` | The red test written for `SpecFor` and `SpecForMount` (hub a1f9d1aa had none). |
| Only here: `consent_in_force_test.go` | The red test written for the consent sheet in force dropping `published_by` (hub 5762d51f proved it from wakala-api). |
| Only here: `routes/give_description_test.go` | The red test written for the declared default language of `custody.consent.give` (hub f6eb3c28 changed a description only). |
| Only here: `consent_text_test.go`, `routes/consent_tools_test.go` | Tests for `PublishConsentText`, `ListConsentVersions`, `GetInForceConsent` and the six consent tools (hub 36920a5b had none in this module). |

## The hub commits carried over

One commit here per hub commit, each naming the hub hash: a1f9d1aa, 0ba49da6, 4695aead, ec77c144, 3b1729a7, 0ccb75a9, 7325c5ce, e5efebc7, 682dd4c4, 23f26b85, bff780fa, 9b603ee4, 5762d51f, f6eb3c28, fbc16a41, 36920a5b, 0a132371.

Files a commit touched lost their comments (the repository rule is no comments in code). The parts of 9b603ee4, 5762d51f, f6eb3c28 and 36920a5b that sit outside `modules/custody` (the wakala-api start-up stamp and its callers) are not part of this module.

## Behaviour that changed in the module

- `Operations()` returns the embedded bytes, and `routes` builds through `dispatch.NewTable`; `BuildWith`, `RoutesWith`, `SplitWith`, `PublicRoutes`, `OperatorRoutes` and `Shape` are gone, and `Table`, `Sentinels`, `BuildTools` and the `Mount` and `Tool` aliases replace them.
- `models.TimeLayout` is nanosecond-resolution (`2006-01-02T15:04:05.000000000Z07:00`) instead of `time.RFC3339`, so an agreement placed after a withdrawal in the same second sorts after it as text (hub bff780fa). Stored custody timestamps stay fixed-width UTC strings, so comparing them as text still compares them chronologically.
- Validation failures of a consent text wrap `ErrInvalidValue`, which the routes answer with 400.
- Two consent text publishes at the same moment are numbered one after the other, and `ErrConflict` answers 409 (hub issue MUN687, see [consent-publish-concurrency.md](consent-publish-concurrency.md)). The hub copy does not have it yet, so the diff is not empty until the hub takes the change or ISS-105 deletes the copy.
