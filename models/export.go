package models

import (
	"errors"
	"fmt"
	"time"
)

const Retention = 30 * 24 * time.Hour

var ErrInvalid = errors.New("mwanachamacustody: invalid export job")

var ErrResumeAlreadyRecorded = errors.New("mwanachamacustody: resumed_from_pct is already recorded and is write-once")

type Job struct {
	ID string

	Scope          Scope
	SubjectActorID string
	RequestedBy    string

	RequestedAt string
	SnapshotAt  string

	IncludeIDs bool
	Format     Format
	Datasets   []string

	Status           Status
	ProgressPct      *int
	ResumedFromPct   *int
	RowCounts        map[string]int
	Bytes            *int64
	Checksum         string
	StoragePath      string
	FileRemovedAt    *string
	FileRemovedDueAt string
}

func DueAt(snapshotAt time.Time) time.Time { return snapshotAt.Add(Retention) }

func DueAfter(snapshotAt string) (string, error) {
	t, err := ParseTime(snapshotAt)
	if err != nil {
		return "", fmt.Errorf("%w: snapshot_at %q is not an RFC 3339 instant", ErrInvalid, snapshotAt)
	}
	return FormatTime(DueAt(t)), nil
}

func (j Job) FileGone() bool { return j.FileRemovedAt != nil }

func (j Job) Validate() error {
	if err := j.validateInstants(); err != nil {
		return err
	}
	if err := j.validateScope(); err != nil {
		return err
	}
	return validatePct("progress_pct", j.ProgressPct)
}

func (j Job) validateInstants() error {
	if j.SnapshotAt == "" {
		return fmt.Errorf("%w: snapshot_at is unset — the file is one instant, and an empty one here is a build that would read now()", ErrInvalid)
	}
	snapshot, err := ParseTime(j.SnapshotAt)
	if err != nil {
		return fmt.Errorf("%w: snapshot_at %q is not an RFC 3339 instant", ErrInvalid, j.SnapshotAt)
	}
	if j.RequestedAt != "" {
		requested, err := ParseTime(j.RequestedAt)
		if err != nil {
			return fmt.Errorf("%w: requested_at %q is not an RFC 3339 instant", ErrInvalid, j.RequestedAt)
		}
		if snapshot.Before(requested) {
			return fmt.Errorf("%w: snapshot_at is before requested_at", ErrInvalid)
		}
	}
	want := FormatTime(DueAt(snapshot))
	if j.FileRemovedDueAt != want {
		return fmt.Errorf("%w: file_removed_due_at is %s, not snapshot_at + 30 days (%s)", ErrInvalid, j.FileRemovedDueAt, want)
	}
	return nil
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
