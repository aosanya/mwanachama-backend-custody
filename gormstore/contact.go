package gormstore

import (
	"time"

	"github.com/aosanya/mwanachama-backend-custody/models"
	"gorm.io/gorm"
)

// ReadRow is the GORM row for a [models.Read].
type ReadRow struct {
	ID string `gorm:"primaryKey"`

	MemberID   string `gorm:"index"`
	ReadBy     string `gorm:"index"`
	ChapterID  string
	ReadAt     time.Time
	NotifiedAt *time.Time
}

// BeforeCreate mints an id via mintID when the caller left one unset.
func (r *ReadRow) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		id, err := mintID(tx, "contactread", "custody_contact_read_seq")
		if err != nil {
			return err
		}
		r.ID = id
	}
	return nil
}

// ReadToRow converts a domain Read to its row shape.
func ReadToRow(r models.Read) ReadRow {
	return ReadRow{
		ID:         r.ID,
		MemberID:   r.ActorID,
		ReadBy:     r.ReadBy,
		ChapterID:  r.StructureID,
		ReadAt:     r.ReadAt,
		NotifiedAt: r.NotifiedAt,
	}
}

// ReadFromRow converts a row back to the domain Read.
func ReadFromRow(r ReadRow) models.Read {
	return models.Read{
		ID:          r.ID,
		ActorID:     r.MemberID,
		ReadBy:      r.ReadBy,
		StructureID: r.ChapterID,
		ReadAt:      r.ReadAt,
		NotifiedAt:  r.NotifiedAt,
	}
}
