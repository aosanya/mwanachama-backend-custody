package gormstore

import (
	"encoding/json"
	"time"

	"github.com/aosanya/mwanachama-backend-custody/models"
	"gorm.io/gorm"
)

// TextVersionRow is the GORM row for a [models.TextVersion]. Unique over
// (scope, version, language) — the natural key ConsentRepository.
// CreateVersion's doc names.
type TextVersionRow struct {
	ID string `gorm:"primaryKey"`

	Scope    string `gorm:"uniqueIndex:custody_consent_text_version_natural_key"`
	Version  string `gorm:"uniqueIndex:custody_consent_text_version_natural_key"`
	Language string `gorm:"uniqueIndex:custody_consent_text_version_natural_key"`

	// Clauses and Effect are plain JSON text for export.go's portability
	// reason: this repo's Migrate runs against both Postgres and sqlite.
	Clauses           string
	Effect            string
	Translated        bool
	ReleaseRef        string
	CopiedMechanicsID string

	PublishedAt  *time.Time
	PublishedBy  string
	SupersededAt *time.Time
}

// BeforeCreate mints an id via mintID when the caller left one unset.
func (r *TextVersionRow) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		id, err := mintID(tx, "consentversion", "custody_consent_text_version_seq")
		if err != nil {
			return err
		}
		r.ID = id
	}
	return nil
}

// TextVersionToRow converts a domain TextVersion to its row shape.
func TextVersionToRow(v models.TextVersion) (TextVersionRow, error) {
	clauses, err := json.Marshal(v.Clauses)
	if err != nil {
		return TextVersionRow{}, err
	}
	var effect []byte
	if len(v.Effect) > 0 {
		if effect, err = json.Marshal(v.Effect); err != nil {
			return TextVersionRow{}, err
		}
	}
	return TextVersionRow{
		ID:                v.ID,
		Scope:             string(v.Scope),
		Version:           v.Version,
		Language:          string(v.Language),
		Clauses:           string(clauses),
		Effect:            string(effect),
		Translated:        v.Translated,
		ReleaseRef:        v.ReleaseRef,
		CopiedMechanicsID: v.CopiedMechanicsID,
		PublishedAt:       v.PublishedAt,
		PublishedBy:       v.PublishedBy,
		SupersededAt:      v.SupersededAt,
	}, nil
}

// TextVersionFromRow converts a row back to the domain TextVersion.
func TextVersionFromRow(r TextVersionRow) (models.TextVersion, error) {
	v := models.TextVersion{
		ID:                r.ID,
		Scope:             models.ConsentScope(r.Scope),
		Version:           r.Version,
		Language:          models.Language(r.Language),
		Translated:        r.Translated,
		ReleaseRef:        r.ReleaseRef,
		CopiedMechanicsID: r.CopiedMechanicsID,
		PublishedAt:       r.PublishedAt,
		PublishedBy:       r.PublishedBy,
		SupersededAt:      r.SupersededAt,
	}
	if r.Clauses != "" {
		if err := json.Unmarshal([]byte(r.Clauses), &v.Clauses); err != nil {
			return models.TextVersion{}, err
		}
	}
	if r.Effect != "" {
		if err := json.Unmarshal([]byte(r.Effect), &v.Effect); err != nil {
			return models.TextVersion{}, err
		}
	}
	return v, nil
}

// RecordRow is the GORM row for a [models.Record].
type RecordRow struct {
	ID                 string `gorm:"primaryKey"`
	MemberID           string `gorm:"index"`
	MechanicsVersionID string
	AppendixVersionID  string
	AgreedAt           time.Time
	Language           string
	ChapterID          string
}

// BeforeCreate mints an id via mintID when the caller left one unset.
func (r *RecordRow) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		id, err := mintID(tx, "consentrecord", "custody_consent_record_seq")
		if err != nil {
			return err
		}
		r.ID = id
	}
	return nil
}

// RecordToRow converts a domain Record to its row shape.
func RecordToRow(r models.Record) RecordRow {
	return RecordRow{
		ID:                 r.ID,
		MemberID:           r.ActorID,
		MechanicsVersionID: r.MechanicsVersionID,
		AppendixVersionID:  r.AppendixVersionID,
		AgreedAt:           r.AgreedAt,
		Language:           string(r.Language),
		ChapterID:          r.StructureID,
	}
}

// RecordFromRow converts a row back to the domain Record.
func RecordFromRow(r RecordRow) models.Record {
	return models.Record{
		ID:                 r.ID,
		ActorID:            r.MemberID,
		MechanicsVersionID: r.MechanicsVersionID,
		AppendixVersionID:  r.AppendixVersionID,
		AgreedAt:           r.AgreedAt,
		Language:           models.Language(r.Language),
		StructureID:        r.ChapterID,
	}
}
