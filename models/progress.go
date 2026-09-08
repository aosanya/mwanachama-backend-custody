package models

import "fmt"

// ProgressUpdate is one advance of an export build. Every field is optional,
// and a nil field means *leave it* rather than *clear it*.
type ProgressUpdate struct {
	// Status is the new state, or "" to leave it.
	Status Status
	// Pct is the live progress.
	Pct *int
	// ResumedFrom is the percentage this run continued from. Set exactly
	// once per job — see ErrResumeAlreadyRecorded.
	ResumedFrom *int
	// RowCounts is the per-dataset tally so far. Replaces rather than merges.
	RowCounts map[string]int
}

// Validate refuses a progress update the schema would refuse.
func (p ProgressUpdate) Validate() error {
	if p.Status != "" {
		switch p.Status {
		case StatusBuilding, StatusInterrupted, StatusCompleted, StatusFailed:
		default:
			return fmt.Errorf("%w: status %q reaches no state", ErrInvalid, p.Status)
		}
	}
	if err := validatePct("progress_pct", p.Pct); err != nil {
		return err
	}
	return validatePct("resumed_from_pct", p.ResumedFrom)
}

// Completion is an export build's finish. Bytes and Checksum are what the
// screen renders and the checksum is what the collected file is verified
// against, so a completion without one is not a completion.
type Completion struct {
	Bytes       int64
	Checksum    string
	StoragePath string
	RowCounts   map[string]int
}

// Validate refuses a completion that could not be verified against.
func (c Completion) Validate() error {
	if c.Checksum == "" {
		return fmt.Errorf("%w: completion with no checksum", ErrInvalid)
	}
	if c.Bytes < 0 {
		return fmt.Errorf("%w: completion with %d bytes", ErrInvalid, c.Bytes)
	}
	return nil
}
