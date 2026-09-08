package models

import (
	"errors"
	"fmt"
	"time"
)

// Retention is the file's lifetime, from SnapshotAt — mirrors
// mwanachama-backend-api-gateway's internal/domain/export.Retention.
const Retention = 30 * 24 * time.Hour

// ErrInvalid is returned by Job.Validate. Every wrapping error names which
// invariant it is.
var ErrInvalid = errors.New("mwanachamacustody: invalid export job")

// ErrResumeAlreadyRecorded is returned when a Progress call would overwrite a
// ResumedFromPct that is already set — write-once, so a resumed job cannot be
// re-labelled a fresh one on a second interruption.
var ErrResumeAlreadyRecorded = errors.New("mwanachamacustody: resumed_from_pct is already recorded and is write-once")

// Scope is the discriminator every rule on Job branches on. A job is one or
// the other and never both.
type Scope string

const (
	// ScopeOrganization — the register, requested by an HQ admin, readable
	// through the custody log.
	ScopeOrganization Scope = "organization"
	ScopeActor        Scope = "actor"
)

type Format string

const (
	FormatPgDump Format = "pg_dump"
	FormatCSVZip Format = "csv_zip"
)

// Status is where the job is. StatusInterrupted is a first-class state, not
// an absence — it is what lets a resume be a resume.
type Status string

const (
	StatusBuilding    Status = "building"
	StatusInterrupted Status = "interrupted"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

// Job is one export — mirrors mwanachama-backend-api-gateway's
// internal/domain/export.Job field-for-field.
type Job struct {
	ID string

	Scope          Scope
	SubjectActorID string
	// RequestedBy is the administrator, at organization scope only.
	RequestedBy string

	RequestedAt time.Time
	// SnapshotAt is the instant the build reads from. Never RequestedAt, and
	// never defaulted from it.
	SnapshotAt time.Time

	IncludeIDs bool
	Format     Format
	Datasets   []string

	Status Status
	// ProgressPct is live while building. ResumedFromPct is the evidence for
	// "continued from 68% — not restarted".
	ProgressPct      *int
	ResumedFromPct   *int
	RowCounts        map[string]int
	Bytes            *int64
	Checksum         string
	StoragePath      string
	FileRemovedAt    *time.Time
	FileRemovedDueAt time.Time
}

// DueAt is the retention clock, stated once.
func DueAt(snapshotAt time.Time) time.Time { return snapshotAt.Add(Retention) }

// FileGone reports whether the exported file has been removed. The row
// itself is never deleted.
func (j Job) FileGone() bool { return j.FileRemovedAt != nil }

// Validate refuses every job the schema would refuse, and in the same terms.
func (j Job) Validate() error {
	switch j.Scope {
	case ScopeOrganization, ScopeActor:
	default:
		return fmt.Errorf("%w: scope %q is neither organization nor actor", ErrInvalid, j.Scope)
	}
	switch j.Format {
	case FormatPgDump, FormatCSVZip:
	default:
		return fmt.Errorf("%w: format %q is neither pg_dump nor csv_zip", ErrInvalid, j.Format)
	}
	switch j.Status {
	case StatusBuilding, StatusInterrupted, StatusCompleted, StatusFailed:
	default:
		return fmt.Errorf("%w: status %q reaches no state", ErrInvalid, j.Status)
	}
	if j.SnapshotAt.IsZero() {
		return fmt.Errorf("%w: snapshot_at is unset — the file is one instant, and a zero here is a build that would read now()", ErrInvalid)
	}
	if !j.RequestedAt.IsZero() && j.SnapshotAt.Before(j.RequestedAt) {
		return fmt.Errorf("%w: snapshot_at is before requested_at", ErrInvalid)
	}
	if got, want := j.FileRemovedDueAt, DueAt(j.SnapshotAt); !got.Equal(want) {
		return fmt.Errorf("%w: file_removed_due_at is %s, not snapshot_at + 30 days (%s)", ErrInvalid, got, want)
	}
	if err := j.validateScope(); err != nil {
		return err
	}
	return validatePct("progress_pct", j.ProgressPct)
}

func (j Job) validateScope() error {
	if j.Scope == ScopeActor {
		if j.SubjectActorID == "" {
			return fmt.Errorf("%w: actor scope with no subject_actor_id", ErrInvalid)
		}
		if j.RequestedBy != "" {
			return fmt.Errorf("%w: actor scope naming requested_by %q — an administrator is never the requester of one", ErrInvalid, j.RequestedBy)
		}
		if j.IncludeIDs {
			return fmt.Errorf("%w: actor scope with include_ids — the ID number is never in the file in any form", ErrInvalid)
		}
		if j.Format != FormatCSVZip {
			return fmt.Errorf("%w: actor scope with format %q — a pg_dump of one actor's rows is a format only the organization could open", ErrInvalid, j.Format)
		}
		return validatePct("resumed_from_pct", j.ResumedFromPct)
	}
	if j.SubjectActorID != "" {
		return fmt.Errorf("%w: organization scope naming subject_actor_id %q", ErrInvalid, j.SubjectActorID)
	}
	return validatePct("resumed_from_pct", j.ResumedFromPct)
}

func validatePct(name string, p *int) error {
	if p != nil && (*p < 0 || *p > 100) {
		return fmt.Errorf("%w: %s is %d, outside 0..100", ErrInvalid, name, *p)
	}
	return nil
}
