package gormstore

import (
	"encoding/json"
	"time"

	"github.com/aosanya/mwanachama-backend-custody/models"
	"gorm.io/datatypes"
)

type StructureActLogEntryRow struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	ChapterID       string    `gorm:"index;not null"`
	OccurredAt      time.Time `gorm:"index"`
	Kind            string
	Class           string `gorm:"index"`
	ActorID         string
	ActorLabel      string
	ActorChapterID  string
	SubjectRef      string
	SubjectID       string
	Detail          datatypes.JSON
	EscalationLevel *int
	ToChapterID     string
}

func StructureActLogEntryToRow(e models.StructureActLogEntry) (StructureActLogEntryRow, error) {
	var detail datatypes.JSON
	if e.Detail != nil {
		b, err := json.Marshal(e.Detail)
		if err != nil {
			return StructureActLogEntryRow{}, err
		}
		detail = datatypes.JSON(b)
	}
	return StructureActLogEntryRow{
		ID:              e.ID,
		ChapterID:       e.StructureID,
		OccurredAt:      e.OccurredAt,
		Kind:            string(e.Kind),
		Class:           string(e.Class),
		ActorID:         e.ActorID,
		ActorLabel:      e.ActorLabel,
		ActorChapterID:  e.ActorStructureID,
		SubjectRef:      e.SubjectRef,
		SubjectID:       e.SubjectID,
		Detail:          detail,
		EscalationLevel: e.EscalationLevel,
		ToChapterID:     e.ToStructureID,
	}, nil
}

func StructureActLogEntryFromRow(r StructureActLogEntryRow) (models.StructureActLogEntry, error) {
	e := models.StructureActLogEntry{
		ID:               r.ID,
		StructureID:      r.ChapterID,
		OccurredAt:       r.OccurredAt,
		Kind:             models.ActKind(r.Kind),
		Class:            models.ActClass(r.Class),
		ActorID:          r.ActorID,
		ActorLabel:       r.ActorLabel,
		ActorStructureID: r.ActorChapterID,
		SubjectRef:       r.SubjectRef,
		SubjectID:        r.SubjectID,
		EscalationLevel:  r.EscalationLevel,
		ToStructureID:    r.ToChapterID,
	}
	if len(r.Detail) > 0 {
		var detail map[string]any
		if err := json.Unmarshal(r.Detail, &detail); err != nil {
			return models.StructureActLogEntry{}, err
		}
		e.Detail = detail
	}
	return e, nil
}
