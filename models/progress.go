package models

import "fmt"

type ProgressUpdate struct {
	Status      Status
	Pct         *int
	ResumedFrom *int
	RowCounts   map[string]int
}

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

type Completion struct {
	Bytes       int64
	Checksum    string
	StoragePath string
	RowCounts   map[string]int
}

func (c Completion) Validate() error {
	if c.Checksum == "" {
		return fmt.Errorf("%w: completion with no checksum", ErrInvalid)
	}
	if c.Bytes < 0 {
		return fmt.Errorf("%w: completion with %d bytes", ErrInvalid, c.Bytes)
	}
	return nil
}
