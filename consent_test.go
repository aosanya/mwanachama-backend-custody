package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newConsentStore(t *testing.T) *mwanachamacustody.ConsentStore {
	t.Helper()
	db, tables := newTestDB(t)
	s, err := mwanachamacustody.NewConsentStore(db, tables)
	if err != nil {
		t.Fatalf("NewConsentStore: %v", err)
	}
	return s
}

func TestCreateVersionRefusesDuplicateNaturalKey(t *testing.T) {
	s := newConsentStore(t)
	ctx := context.Background()
	v := models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish}
	if _, err := s.CreateVersion(ctx, v); err != nil {
		t.Fatalf("CreateVersion #1: %v", err)
	}
	_, err := s.CreateVersion(ctx, v)
	if !errors.Is(err, mwanachamacustody.ErrConflict) {
		t.Fatalf("CreateVersion duplicate triple: got %v, want ErrConflict", err)
	}
}

func TestCreateVersionClearsPublicationFields(t *testing.T) {
	s := newConsentStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	created, err := s.CreateVersion(ctx, models.TextVersion{
		Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish,
		PublishedAt: &now, PublishedBy: "sneaky", CopiedMechanicsID: "sneaky-id",
	})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if created.PublishedAt != nil || created.PublishedBy != "" || created.CopiedMechanicsID != "" {
		t.Fatalf("CreateVersion did not clear publication fields: %+v", created)
	}
}

func TestPublishVersionSupersedesPriorInForce(t *testing.T) {
	s := newConsentStore(t)
	ctx := context.Background()
	v1, err := s.CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion v1: %v", err)
	}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.PublishVersion(ctx, v1.ID, "admin-1", t0); err != nil {
		t.Fatalf("PublishVersion v1: %v", err)
	}
	v2, err := s.CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v2", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion v2: %v", err)
	}
	t1 := t0.Add(24 * time.Hour)
	if _, err := s.PublishVersion(ctx, v2.ID, "admin-1", t1); err != nil {
		t.Fatalf("PublishVersion v2: %v", err)
	}

	inForce, err := s.GetInForce(ctx, models.ScopeMechanics, models.LanguageEnglish)
	if err != nil {
		t.Fatalf("GetInForce: %v", err)
	}
	if inForce.ID != v2.ID {
		t.Fatalf("GetInForce = %q, want v2 (%q)", inForce.ID, v2.ID)
	}
	old, err := s.GetVersion(ctx, v1.ID)
	if err != nil {
		t.Fatalf("GetVersion v1: %v", err)
	}
	if old.SupersededAt == nil {
		t.Fatalf("v1.SupersededAt is nil, want set once v2 published")
	}
}

func TestPublishVersionRefusesAlreadyPublished(t *testing.T) {
	s := newConsentStore(t)
	ctx := context.Background()
	v, err := s.CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	now := time.Now().UTC()
	if _, err := s.PublishVersion(ctx, v.ID, "admin-1", now); err != nil {
		t.Fatalf("PublishVersion #1: %v", err)
	}
	_, err = s.PublishVersion(ctx, v.ID, "admin-1", now.Add(time.Hour))
	if !errors.Is(err, models.ErrAlreadyPublished) {
		t.Fatalf("PublishVersion #2: got %v, want ErrAlreadyPublished", err)
	}
}

func TestGetInForceUnpublishedReturnsErrNotFound(t *testing.T) {
	s := newConsentStore(t)
	_, err := s.GetInForce(context.Background(), models.ScopeAppendix, models.LanguageEnglish)
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("GetInForce with nothing published: got %v, want ErrNotFound", err)
	}
}

func TestCreateRecordRefusesUnknownMechanicsVersion(t *testing.T) {
	s := newConsentStore(t)
	_, err := s.CreateRecord(context.Background(), models.Record{
		ActorID: "mem-1", MechanicsVersionID: "no-such-version", Language: models.LanguageEnglish, StructureID: "ward-1",
	})
	if !errors.Is(err, mwanachamacustody.ErrInvalidReference) {
		t.Fatalf("CreateRecord unknown mechanics version: got %v, want ErrInvalidReference", err)
	}
}

func TestEnrollWritesRecordWithNilableAppendix(t *testing.T) {
	s := newConsentStore(t)
	ctx := context.Background()
	mechanics, err := s.CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion mechanics: %v", err)
	}
	now := time.Now().UTC()
	if _, err := s.PublishVersion(ctx, mechanics.ID, "admin-1", now); err != nil {
		t.Fatalf("PublishVersion mechanics: %v", err)
	}

	rec, err := mwanachamacustody.Enroll(ctx, s, "mem-1", "ward-1", models.LanguageEnglish, now)
	if err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	if rec.MechanicsVersionID != mechanics.ID {
		t.Fatalf("Enroll MechanicsVersionID = %q, want %q", rec.MechanicsVersionID, mechanics.ID)
	}
	if rec.AppendixVersionID != "" {
		t.Fatalf("Enroll AppendixVersionID = %q, want empty (no appendix published)", rec.AppendixVersionID)
	}

	current, err := s.CurrentForActor(ctx, "mem-1")
	if err != nil {
		t.Fatalf("CurrentForActor: %v", err)
	}
	if current.ID != rec.ID {
		t.Fatalf("CurrentForActor = %q, want %q", current.ID, rec.ID)
	}
}

func TestEnrollRefusesWithNoMechanicsInForce(t *testing.T) {
	s := newConsentStore(t)
	_, err := mwanachamacustody.Enroll(context.Background(), s, "mem-1", "ward-1", models.LanguageEnglish, time.Now().UTC())
	if !errors.Is(err, mwanachamacustody.ErrNoMechanicsInForce) {
		t.Fatalf("Enroll with no mechanics: got %v, want ErrNoMechanicsInForce", err)
	}
}
