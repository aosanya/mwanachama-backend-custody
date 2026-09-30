//go:build integration

package mwanachamacustody_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newPostgres(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is unset")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamacustody.LoadSpec(testSpec)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	for _, o := range s.Objects {
		if err := db.Exec("drop table if exists " + s.TableFor(o)).Error; err != nil {
			t.Fatalf("drop %s: %v", s.TableFor(o), err)
		}
	}
	if err := db.Exec("drop table if exists " + s.NameRegistryTable()).Error; err != nil {
		t.Fatalf("drop registry: %v", err)
	}
	if err := mwanachamacustody.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	return db, s
}

// TestDetailAbsenceIsNullNotEmptyObject is the behaviour this repo's
// CLAUDE.md calls load-bearing and that SQLite cannot stand in for: a
// structure act with no detail must read back as a true nil, because two
// years after occurred_at the free-text keys are stripped and an empty
// object would make an aged-out row look like it still carried them.
//
// Catalog's CAT2 was the same shape — true on SQLite, false on Postgres.
func TestDetailAbsenceIsNullNotEmptyObject(t *testing.T) {
	db, s := newPostgres(t)
	store, err := mwanachamacustody.NewCustodyStore(db, s, mwanachamacustody.SystemClock)
	if err != nil {
		t.Fatalf("NewCustodyStore: %v", err)
	}
	ctx := context.Background()

	if _, err := store.AppendAct(ctx, models.StructureActLogEntry{
		StructureID: "structure-1",
		Kind:        models.ActRoleGranted,
		ActorLabel:  "operator one",
	}); err != nil {
		t.Fatalf("AppendAct: %v", err)
	}

	acts, err := store.ListActs(ctx, []string{"structure-1"}, "", 0)
	if err != nil {
		t.Fatalf("ListActs: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("ListActs returned %d entries, want 1", len(acts))
	}
	if acts[0].Detail != nil {
		t.Fatalf("Detail round-tripped as %#v, want nil — an absent document must not read back as an empty one", acts[0].Detail)
	}

	object, _ := s.ByRole(mwanachamacustody.RoleStructureAct)
	var isNull bool
	if err := db.Raw("select detail is null from " + s.TableFor(object)).Scan(&isNull).Error; err != nil {
		t.Fatalf("read detail: %v", err)
	}
	if !isNull {
		t.Fatal("the stored detail column is not NULL, so the absence was written as a value")
	}
}

// TestEvidenceRoundTripsThroughJSONB pins that a document written on
// Postgres comes back with its keys intact. jsonb sorts keys and strips
// whitespace, so the bytes are rarely the ones sent and only a semantic
// comparison holds.
func TestEvidenceRoundTripsThroughJSONB(t *testing.T) {
	db, s := newPostgres(t)
	store, err := mwanachamacustody.NewCustodyStore(db, s, mwanachamacustody.SystemClock)
	if err != nil {
		t.Fatalf("NewCustodyStore: %v", err)
	}
	ctx := context.Background()

	if _, err := store.AppendEvent(ctx, models.Entry{
		Kind:       models.EventPhoneSaltRotated,
		ActorLabel: "the system",
		Evidence:   map[string]any{"key_id": "k-2", "duration_ms": float64(1200)},
	}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	events, err := store.ListEvents(ctx, "", 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("ListEvents returned %d entries, want 1", len(events))
	}
	if got := events[0].Evidence["key_id"]; got != "k-2" {
		t.Fatalf("evidence key_id = %#v, want \"k-2\"", got)
	}

	object, _ := s.ByRole(mwanachamacustody.RoleEvent)
	var columnType string
	if err := db.Raw(`select data_type from information_schema.columns
		where table_name = ? and column_name = 'evidence'`, s.TableFor(object)).Scan(&columnType).Error; err != nil {
		t.Fatalf("read column type: %v", err)
	}
	if columnType != "jsonb" {
		t.Fatalf("evidence is %q, want jsonb — filtering inside the document happens in the database", columnType)
	}
}

// TestSequenceMintsMonotonicIDs covers the Postgres branch of the log's id
// minting, which sqlite never reaches: the value comes from a real SEQUENCE
// rather than the table's high-water mark.
func TestSequenceMintsMonotonicIDs(t *testing.T) {
	db, s := newPostgres(t)
	store, err := mwanachamacustody.NewCustodyStore(db, s, mwanachamacustody.SystemClock)
	if err != nil {
		t.Fatalf("NewCustodyStore: %v", err)
	}
	ctx := context.Background()

	var ids []int64
	for range 3 {
		e, err := store.AppendEvent(ctx, models.Entry{
			Kind:       models.EventKeysRotated,
			ActorLabel: "the system",
		})
		if err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
		ids = append(ids, e.ID)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Fatalf("ids are %v, which does not increase — the log is read newest-first and the id is the tiebreaker within one instant", ids)
		}
	}
}

// TestListVersionsOrdersNullsLast covers the Postgres-only branch of
// ListVersions. Postgres puts NULLs first under DESC and needs NULLS LAST
// spelled out; sqlite puts them last already, so the two dialects disagree
// and only this test sees the one that needed the clause.
func TestListVersionsOrdersNullsLast(t *testing.T) {
	db, s := newPostgres(t)
	store, err := mwanachamacustody.NewConsentStore(db, s)
	if err != nil {
		t.Fatalf("NewConsentStore: %v", err)
	}
	ctx := context.Background()

	published, err := store.CreateVersion(ctx, models.TextVersion{
		Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish,
	})
	if err != nil {
		t.Fatalf("CreateVersion v1: %v", err)
	}
	if _, err := store.PublishVersion(ctx, published.ID, "admin-1", models.Now()); err != nil {
		t.Fatalf("PublishVersion: %v", err)
	}
	if _, err := store.CreateVersion(ctx, models.TextVersion{
		Scope: models.ScopeMechanics, Version: "v2", Language: models.LanguageEnglish,
	}); err != nil {
		t.Fatalf("CreateVersion v2: %v", err)
	}

	versions, err := store.ListVersions(ctx, models.ScopeMechanics, models.LanguageEnglish)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("ListVersions returned %d, want 2", len(versions))
	}
	if versions[0].PublishedAt == nil {
		t.Fatal("the unpublished version sorted first; published versions come first and unpublished ones last")
	}
	if versions[1].PublishedAt != nil {
		t.Fatal("the published version sorted last")
	}
}

// TestDuplicateNaturalKeyIsAConflictOnPostgres proves the unique index is a
// real database constraint on the dialect that ships, so a concurrent caller
// loses the race in the database rather than slipping past a read-then-write.
//
// It asserts ErrConflict from the store. The route over it still answers 500
// rather than 409 — that is DEV-1696, pinned separately in routes/, and this
// test is why the store half is known to be right.
func TestDuplicateNaturalKeyIsAConflictOnPostgres(t *testing.T) {
	db, s := newPostgres(t)
	store, err := mwanachamacustody.NewConsentStore(db, s)
	if err != nil {
		t.Fatalf("NewConsentStore: %v", err)
	}
	ctx := context.Background()

	version := models.TextVersion{
		Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish,
	}
	if _, err := store.CreateVersion(ctx, version); err != nil {
		t.Fatalf("first CreateVersion: %v", err)
	}
	_, err = store.CreateVersion(ctx, version)
	if err == nil {
		t.Fatal("a second version on the same (scope, version, language) was accepted")
	}
	if !errors.Is(err, mwanachamacustody.ErrConflict) {
		t.Fatalf("second CreateVersion returned %v, want ErrConflict", err)
	}
}
