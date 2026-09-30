package mwanachamacustody

import (
	"encoding/json"
	"fmt"
	"reflect"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

const (
	RoleStructureAct   = "structure_act"
	RoleEvent          = "event"
	RoleExportJob      = "export_job"
	RoleContactRead    = "contact_read"
	RoleConsentVersion = "consent_version"
	RoleConsentRecord  = "consent_record"
)

func Roles() []string {
	return []string{
		RoleStructureAct,
		RoleEvent,
		RoleExportJob,
		RoleContactRead,
		RoleConsentVersion,
		RoleConsentRecord,
	}
}

type store = specstore.Store

func carriers() map[string]any {
	return map[string]any{
		RoleStructureAct:   models.StructureActLogEntry{},
		RoleEvent:          models.Entry{},
		RoleExportJob:      models.Job{},
		RoleContactRead:    models.Read{},
		RoleConsentVersion: models.TextVersion{},
		RoleConsentRecord:  models.Record{},
	}
}

func newStore(db *gorm.DB, s *spec.Spec) (*store, error) {
	return specstore.New(db, s, carriers())
}

func columnName(field string) string { return specstore.ColumnName(field) }

func encode(o spec.Object, v any) (map[string]any, error) { return specstore.Encode(o, v) }

func decode(o spec.Object, row map[string]any, out any) error {
	return specstore.Decode(o, row, out)
}

// nextSequenceID mints the monotonic key the two append-only logs order by
// after occurred_at. spec declares the column and nothing more, so the value
// comes from a Postgres SEQUENCE where there is one, and from the table's own
// high-water mark on sqlite, which is tests only and single-writer there.
func nextSequenceID(tx *gorm.DB, table, sequence string) (int64, error) {
	if tx.Dialector.Name() == "postgres" {
		var n int64
		if err := tx.Raw("SELECT nextval(?::regclass)", sequence).Scan(&n).Error; err != nil {
			return 0, fmt.Errorf("nextSequenceID: %s: %w", sequence, err)
		}
		return n, nil
	}
	var high *int64
	if err := tx.Table(table).Select("max(id)").Scan(&high).Error; err != nil {
		return 0, fmt.Errorf("nextSequenceID: %s: %w", table, err)
	}
	if high == nil {
		return 1, nil
	}
	return *high + 1, nil
}

func sequenceFor(role string) string {
	return "custody_" + role + "_seq"
}

func marshalDocument(v any) (any, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		if rv.Len() == 0 {
			return nil, nil
		}
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}
