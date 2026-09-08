package gormstore

import (
	"encoding/json"
	"time"

	"github.com/aosanya/mwanachama-backend-custody/models"
	"gorm.io/datatypes"
)

type EntryRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	OccurredAt time.Time `gorm:"index"`
	Kind       string
	Chip       string `gorm:"index"`
	ActorID    string
	ActorLabel string
	Detail     string
	Evidence   datatypes.JSON
	SubjectID  string
}

func EntryToRow(e models.Entry) (EntryRow, error) {
	var evidence datatypes.JSON
	if e.Evidence != nil {
		b, err := json.Marshal(e.Evidence)
		if err != nil {
			return EntryRow{}, err
		}
		evidence = datatypes.JSON(b)
	}
	return EntryRow{
		ID:         e.ID,
		OccurredAt: e.OccurredAt,
		Kind:       string(e.Kind),
		Chip:       string(e.Chip),
		ActorID:    e.ActorID,
		ActorLabel: e.ActorLabel,
		Detail:     e.Detail,
		Evidence:   evidence,
		SubjectID:  e.SubjectID,
	}, nil
}

// EntryFromRow converts a row back to the domain Entry.
func EntryFromRow(r EntryRow) (models.Entry, error) {
	e := models.Entry{
		ID:         r.ID,
		OccurredAt: r.OccurredAt,
		Kind:       models.EventKind(r.Kind),
		Chip:       models.EventChip(r.Chip),
		ActorID:    r.ActorID,
		ActorLabel: r.ActorLabel,
		Detail:     r.Detail,
		SubjectID:  r.SubjectID,
	}
	if len(r.Evidence) > 0 {
		var evidence map[string]any
		if err := json.Unmarshal(r.Evidence, &evidence); err != nil {
			return models.Entry{}, err
		}
		e.Evidence = evidence
	}
	return e, nil
}
