// Package gormstore holds every GORM-specific piece of this repo: row
// structs, their conversion to/from the domain types in
// mwanachama-backend-custody/models, and table migration. Nothing outside
// this package (and the root mwanachama-backend-custody package's *_impl.go
// files, which call it) needs to know GORM exists — mirrors
// mwanachama-backend-actor/gormstore's and mwanachama-backend-comm/gormstore's
// split.
package gormstore

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TableNames configures which physical tables a store reads and writes.
// Struct fields rather than hard-coded names so a test can migrate a
// differently-named scratch set without colliding with a concurrent test
// run, the same reason comm's TableNames exists.
type TableNames struct {
	StructureActLog string
	CustodyEvent    string

	ExportJob string

	ContactRead string

	ConsentTextVersion string
	ConsentRecord      string
}

func DefaultTableNames() TableNames {
	return TableNames{
		StructureActLog: "custody_chapter_act_log_entry",
		CustodyEvent:    "custody_log_entry",

		ExportJob: "custody_export_job",

		ContactRead: "custody_contact_read",

		ConsentTextVersion: "custody_consent_text_version",
		ConsentRecord:      "custody_consent_record",
	}
}

// Migrate creates or updates the six tables t names, via GORM's AutoMigrate
// scoped to each table name in turn, plus the Postgres SEQUENCEs the
// id-minting BeforeCreate hooks below read from (see mintID). Callers run
// this once at startup (or in test setup) before constructing a store with
// the same db and t.
func Migrate(db *gorm.DB, t TableNames) error {
	if db.Dialector.Name() == "postgres" {
		if err := createSequences(db); err != nil {
			return err
		}
	}
	migrations := []struct {
		table string
		row   any
	}{
		{t.StructureActLog, &StructureActLogEntryRow{}},
		{t.CustodyEvent, &EntryRow{}},
		{t.ExportJob, &JobRow{}},
		{t.ContactRead, &ReadRow{}},
		{t.ConsentTextVersion, &TextVersionRow{}},
		{t.ConsentRecord, &RecordRow{}},
	}
	for _, m := range migrations {
		if err := db.Table(m.table).AutoMigrate(m.row); err != nil {
			return fmt.Errorf("gormstore.Migrate: %s: %w", m.table, err)
		}
	}
	return nil
}

var seqNames = []string{
	"custody_export_job_seq",
	"custody_contact_read_seq",
	"custody_consent_text_version_seq",
	"custody_consent_record_seq",
}

func createSequences(db *gorm.DB) error {
	for _, seq := range seqNames {
		if err := db.Exec("CREATE SEQUENCE IF NOT EXISTS " + seq).Error; err != nil {
			return fmt.Errorf("gormstore.Migrate: %s: %w", seq, err)
		}
	}
	return nil
}

// mintID produces a human-readable, prefix-and-number id by reading the next
// value of a server-side SEQUENCE on Postgres — mirrors comm's mintID.
//
// sqlite (this repo's fast-test dialect only) has no SEQUENCE, so it falls
// back to a uuid-suffixed id with the same prefix. Every behavioural
// assertion this repo's tests make holds under either dialect; only the
// exact digits after the prefix differ.
func mintID(tx *gorm.DB, prefix, seq string) (string, error) {
	if tx.Dialector.Name() == "postgres" {
		var n int64
		if err := tx.Raw("SELECT nextval(?::regclass)", seq).Scan(&n).Error; err != nil {
			return "", fmt.Errorf("mintID: %s: %w", seq, err)
		}
		return fmt.Sprintf("%s-%d", prefix, n), nil
	}
	return prefix + "-" + uuid.NewString(), nil
}
