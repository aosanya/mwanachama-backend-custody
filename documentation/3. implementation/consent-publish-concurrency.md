# Consent text publishes at the same moment

Hub issue MUN687. `PublishConsentText` (the tool `custody_publish_consent_version`) drafts the next version and brings it into force in one step.

## Cause

The draft was labelled `"v"` plus the number of existing versions plus one. Two publishes at the same moment read the same count, so both drafted the same label and the unique index on scope, version and language refused the second insert. That refusal is `ErrConflict`, which no route mapped, so the caller saw "internal error" and a 500. A second race sat behind the first: two publishes in different transactions could each supersede the version the other had not yet committed, leaving two versions in force.

## Rule

Two publishes at the same moment behave like two publishes one after the other: the same text twice is allowed and gives v1 and v2, the later one supersedes the earlier, and exactly one version stays in force. Nobody is refused for sending at the same moment as someone else.

## How

- One mutex per manager serialises the whole draft-and-publish, so within one server process the labels are numbered in the order the publishes run and the one published last is the highest number.
- When the draft insert still hits the unique index (a second server process on the same database), the count is read again and the draft is numbered again, up to five attempts. A conflict that survives the attempts is returned as `ErrConflict`.
- `ErrConflict` is mapped to 409 in `Sentinels` and in the `errors` table of `custody.operations.json`, so a conflict on any create is a clean refusal and never "internal error". The test that recorded DEV-1696 as a 500 now asserts the 409.
- On Postgres, `PublishVersion` takes `pg_advisory_xact_lock` on the scope and language before it supersedes, then re-reads the version, so two transactions cannot both leave a version in force. SQLite writes are already one at a time. The Postgres branch is only reached with a Postgres database; the `integration` build tag test file is where it would be exercised.

## A publish that fails after its draft

Hub issue CU9. Drafting and publishing are two steps. When the publish fails (a transient database error, a busy lock, a cancelled request), `PublishConsentText` deletes the draft it made, with a context that outlives the request, and returns the publish error. The delete only touches a row that is still unpublished and only the row this call made, so a draft someone made by hand with `custody_create_consent_version` is never removed. The choice was a delete rather than one transaction around both steps because the publish already runs its own transaction and the store would have to nest them.

`consent_text_failed_publish_test.go` makes the first update fail once, then asserts that no row is left, that the next publish is v1 and that it is the only row and in force. On 0601f7f it fails with one leftover row.

## Proof

`consent_text_concurrent_test.go` sends 8 publishes at once, five rounds on a fresh database each, on a SQLite file with several connections and on an in-memory SQLite database with one connection, and once more across two managers on one database. All must succeed, the labels must be v1 to v8 with no repeat, and exactly one version, the highest, must be in force. Before the fix the first two cases failed on the first round with the unique violation.

## Hub copy

The hub copy (`projects/hub-steward/code/wakala-api/modules/custody`) does not have this change and needs it until ISS-105 deletes the copy. Until then `scripts/hub-copy-diff.sh` shows differences in `consent_text.go`, `consent_impl.go`, `custody_manager.go`, `routes/routes.go`, `custody.operations.json` and the two test files.
