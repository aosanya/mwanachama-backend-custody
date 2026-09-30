package mwanachamacustody_test

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

const testSpec = "spec/examples/mwanachama.custody.json"

// newTestDB opens a fresh in-memory sqlite database, provisioned the same
// way a real deployment would: the declared objects through
// [mwanachamacustody.Provision], with no row structs anywhere.
func newTestDB(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamacustody.LoadSpec(testSpec)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if err := mwanachamacustody.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	return db, s
}

// monotonicClock advances a whole second per call. A second, not a
// millisecond: instants are stored as RFC 3339 to the second, so a
// millisecond step would render every row at the same instant and the tests
// that distinguish "recorded before" from "recorded in the same instant"
// would be relying on insertion order instead.
func monotonicClock(start time.Time) mwanachamacustody.Clock {
	t := start
	return func() time.Time {
		t = t.Add(time.Second)
		return t
	}
}
