package mwanachamacustody_test

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

// newTestDB opens a fresh in-memory sqlite database, migrated the same way a
// real deployment would via [mwanachamacustody.Migrate] — mirrors
// mwanachama-backend-comm's newTestDB: exercising real GORM/SQL behaviour
// catches more than a hand-rolled memory store ever could, while staying
// fully in-process.
func newTestDB(t *testing.T) (*gorm.DB, mwanachamacustody.TableNames) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := mwanachamacustody.DefaultTableNames()
	if err := mwanachamacustody.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db, tables
}

// monotonicClock returns a [mwanachamacustody.Clock] that advances by 1ms on
// every call starting from start — deterministic ordering for tests that
// need to distinguish "created before" from "created in the same instant".
func monotonicClock(start time.Time) mwanachamacustody.Clock {
	t := start
	return func() time.Time {
		t = t.Add(time.Millisecond)
		return t
	}
}
