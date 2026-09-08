package models

import (
	"context"
	"time"
)

// DefaultPage is the page size every list method in this package falls back
// to when a caller passes limit <= 0. custody.DefaultPage and
// export.DefaultPage were two identically-valued constants in the source
// packages, each justified the same way — "here rather than in either store,
// so a memory backend cannot page differently from Postgres and make the
// parity tests pass while the product behaves differently in production."
// Now that both live in this one module there is exactly one store per
// domain (see the root package's *_impl.go), so the two constants collapsed
// into this one without losing the reasoning that named them.
const DefaultPage = 200

// CustodyRepository is the persistence boundary for both append-only logs —
// mirrors mwanachama-backend-api-gateway's internal/domain/custody.Repository
// method-for-method.
//
// **There is no update method and no delete method, on either log, and that
// absence is the append-only invariant** — not a comment about one. A method
// that existed and refused would be one `if` away from not refusing.
type CustodyRepository interface {
	AppendAct(ctx context.Context, e StructureActLogEntry) (StructureActLogEntry, error)

	ListActs(ctx context.Context, structures []string, class ActClass, limit int) ([]StructureActLogEntry, error)

	CountActsByClass(ctx context.Context, structures []string, since time.Time) (map[ActClass]int, error)

	// AppendEvent writes one custody log row, deriving Chip from Kind. A Kind
	// reaching no chip returns ErrUnknownEventKind and writes nothing.
	AppendEvent(ctx context.Context, e Entry) (Entry, error)

	ListEvents(ctx context.Context, chip EventChip, limit int) ([]Entry, error)

	// CountEventsByChip returns the per-chip tally, for CountActsByClass's
	// reason. Zero `since` means all time.
	CountEventsByChip(ctx context.Context, since time.Time) (map[EventChip]int, error)

	// GetEvent returns one custody row by id. ErrNotFound if unknown.
	GetEvent(ctx context.Context, id int64) (Entry, error)
}

type ExportRepository interface {
	// Create inserts one job and returns it with ID and RequestedAt filled
	// in. It calls Job.Validate first and returns ErrInvalid without
	// writing. It never fills SnapshotAt — a caller that has not taken a
	// snapshot has not started an export.
	Create(ctx context.Context, j Job) (Job, error)

	// Get returns one job by id. ErrNotFound if unknown. Deliberately
	// unscoped — never reachable from a handler without a scope check on the
	// row it returns.
	Get(ctx context.Context, id string) (Job, error)

	// ListOrganization returns organization-scoped jobs, newest first. Limit
	// <= 0 means DefaultPage.
	ListOrganization(ctx context.Context, limit int) ([]Job, error)

	ListForActor(ctx context.Context, actorID string, limit int) ([]Job, error)

	// Progress records a build's advance: status, progress, and — on a
	// resume — the percentage it continued from. The ONLY mutation this
	// object exposes that belongs to a build worker rather than a client.
	//
	// **ResumedFromPct is write-once.** A call that would overwrite a
	// non-null value returns ErrResumeAlreadyRecorded and writes nothing.
	Progress(ctx context.Context, id string, p ProgressUpdate) (Job, error)

	// Complete records the finish: bytes, checksum, storage path.
	Complete(ctx context.Context, id string, c Completion) (Job, error)

	// MarkFileRemoved stamps FileRemovedAt. It never deletes the row.
	MarkFileRemoved(ctx context.Context, id string, at time.Time) (Job, error)

	// DueForRemoval returns jobs whose file is past FileRemovedDueAt and not
	// yet removed, oldest first — the sweep a retention worker walks.
	DueForRemoval(ctx context.Context, asOf time.Time, limit int) ([]Job, error)
}

type ContactRepository interface {
	// Create inserts one contact_read row and returns it with ID and ReadAt
	// filled in. Event, not counter: it never looks for an existing row, so
	// two reads by one operator are two rows.
	Create(ctx context.Context, r Read) (Read, error)

	// Get returns one read by id. ErrNotFound if unknown.
	Get(ctx context.Context, id string) (Read, error)

	ListForSubject(ctx context.Context, actorID string) ([]Read, error)

	// ListByOperator returns every read operatorID performed, newest first —
	// the operator's own side, scoped to themselves.
	ListByOperator(ctx context.Context, operatorID string) ([]Read, error)
}

// ConsentRepository is the persistence boundary for the consent domain —
// mirrors mwanachama-backend-api-gateway's internal/domain/consent.Repository
// method-for-method.
type ConsentRepository interface {
	// CreateVersion inserts a new, unpublished TextVersion row. Scope,
	// Version and Language together are the natural key. Returns
	// ErrConflict when that triple already exists.
	CreateVersion(ctx context.Context, v TextVersion) (TextVersion, error)

	// PublishVersion stamps PublishedAt/PublishedBy on the named version and
	// SupersededAt=now on whichever version was previously in force for the
	// same (Scope, Language), in one atomic step. Returns ErrAlreadyPublished
	// if the version already carries a PublishedAt, ErrNotFound if the id is
	// unknown.
	PublishVersion(ctx context.Context, id, publishedBy string, now time.Time) (TextVersion, error)

	// GetVersion returns one version by id. ErrNotFound if unknown.
	GetVersion(ctx context.Context, id string) (TextVersion, error)

	// GetInForce returns the version currently in force for a (scope,
	// language): the greatest PublishedAt with a nil SupersededAt, within
	// that scope. Returns ErrNotFound when nothing has ever published there
	// — a real state, not an exceptional one.
	GetInForce(ctx context.Context, scope ConsentScope, language Language) (TextVersion, error)

	// ListVersions returns every version for a (scope, language), newest
	// PublishedAt first (unpublished rows last).
	ListVersions(ctx context.Context, scope ConsentScope, language Language) ([]TextVersion, error)

	// CreateRecord inserts an immutable consent record. Never updated or
	// deleted. Returns ErrInvalidReference if MechanicsVersionID (or a
	// non-empty AppendixVersionID) names no row.
	CreateRecord(ctx context.Context, r Record) (Record, error)

	// GetRecord returns one record by id. ErrNotFound if unknown.
	GetRecord(ctx context.Context, id string) (Record, error)

	ListForActor(ctx context.Context, actorID string) ([]Record, error)

	CurrentForActor(ctx context.Context, actorID string) (Record, error)
}
