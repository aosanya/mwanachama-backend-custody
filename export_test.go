package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newExportStore(t *testing.T, clock mwanachamacustody.Clock) *mwanachamacustody.ExportStore {
	t.Helper()
	db, tables := newTestDB(t)
	s, err := mwanachamacustody.NewExportStore(db, tables, clock)
	if err != nil {
		t.Fatalf("NewExportStore: %v", err)
	}
	return s
}

// baseOrgJob builds a valid organization-scope job snapshotted at the given
// instant. RequestedAt is stamped explicitly, equal to snapshot, rather than
// left zero for Create's own clock to fill: Create's clock reads real system
// time, and Validate refuses a job whose SnapshotAt precedes RequestedAt —
// a fixture with an old (or even future) snapshot needs a RequestedAt no
// later than it to stay valid regardless of when the test actually runs.
func baseOrgJob(snapshot time.Time) models.Job {
	return models.Job{
		Scope:            models.ScopeOrganization,
		RequestedAt:      snapshot,
		SnapshotAt:       snapshot,
		Format:           models.FormatPgDump,
		Status:           models.StatusBuilding,
		FileRemovedDueAt: models.DueAt(snapshot),
	}
}

func TestCreateValidatesJob(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	_, err := s.Create(context.Background(), models.Job{Scope: models.ScopeOrganization})
	if !errors.Is(err, models.ErrInvalid) {
		t.Fatalf("Create with no snapshot_at: got %v, want ErrInvalid", err)
	}
}

func TestCreateAndGetRoundTrip(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	snap := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	created, err := s.Create(ctx, baseOrgJob(snap))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.RequestedAt.IsZero() {
		t.Fatalf("Create did not fill ID/RequestedAt: %+v", created)
	}
	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Scope != models.ScopeOrganization || !got.SnapshotAt.Equal(snap) {
		t.Fatalf("Get = %+v, want scope/snapshot round-tripped", got)
	}
}

func TestListOrganizationExcludesActorScope(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	snap := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.Create(ctx, baseOrgJob(snap)); err != nil {
		t.Fatalf("Create org: %v", err)
	}
	actorJob := models.Job{
		Scope: models.ScopeActor, SubjectActorID: "mem-1", RequestedAt: snap, SnapshotAt: snap,
		Format: models.FormatCSVZip, Status: models.StatusBuilding, FileRemovedDueAt: models.DueAt(snap),
	}
	if _, err := s.Create(ctx, actorJob); err != nil {
		t.Fatalf("Create actor: %v", err)
	}
	orgJobs, err := s.ListOrganization(ctx, 0)
	if err != nil {
		t.Fatalf("ListOrganization: %v", err)
	}
	if len(orgJobs) != 1 || orgJobs[0].Scope != models.ScopeOrganization {
		t.Fatalf("ListOrganization = %+v, want exactly one organization job", orgJobs)
	}
	actorJobs, err := s.ListForActor(ctx, "mem-1", 0)
	if err != nil {
		t.Fatalf("ListForActor: %v", err)
	}
	if len(actorJobs) != 1 || actorJobs[0].SubjectActorID != "mem-1" {
		t.Fatalf("ListForActor = %+v, want exactly one job for mem-1", actorJobs)
	}
}

func TestProgressResumedFromPctIsWriteOnce(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	snap := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	created, err := s.Create(ctx, baseOrgJob(snap))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	first := 40
	if _, err := s.Progress(ctx, created.ID, models.ProgressUpdate{Status: models.StatusInterrupted}); err != nil {
		t.Fatalf("Progress interrupt: %v", err)
	}
	if _, err := s.Progress(ctx, created.ID, models.ProgressUpdate{Status: models.StatusBuilding, ResumedFrom: &first}); err != nil {
		t.Fatalf("Progress resume #1: %v", err)
	}
	second := 68
	_, err = s.Progress(ctx, created.ID, models.ProgressUpdate{Status: models.StatusInterrupted, ResumedFrom: &second})
	if !errors.Is(err, models.ErrResumeAlreadyRecorded) {
		t.Fatalf("second resume: got %v, want ErrResumeAlreadyRecorded", err)
	}
	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ResumedFromPct == nil || *got.ResumedFromPct != first {
		t.Fatalf("ResumedFromPct = %v, want %d (first recorded value preserved)", got.ResumedFromPct, first)
	}
}

func TestProgressUnknownIDReturnsErrNotFound(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	_, err := s.Progress(context.Background(), "no-such-job", models.ProgressUpdate{Status: models.StatusBuilding})
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("Progress unknown id: got %v, want ErrNotFound", err)
	}
}

func TestCompleteAndMarkFileRemoved(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	snap := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	created, err := s.Create(ctx, baseOrgJob(snap))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	completed, err := s.Complete(ctx, created.ID, models.Completion{Bytes: 1024, Checksum: "sha256:abc", StoragePath: "s3://x"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed.Status != models.StatusCompleted || completed.ProgressPct == nil || *completed.ProgressPct != 100 {
		t.Fatalf("Complete result = %+v, want status completed, progress 100", completed)
	}
	if completed.FileGone() {
		t.Fatalf("FileGone() true right after Complete, want false")
	}
	removed, err := s.MarkFileRemoved(ctx, created.ID, time.Time{})
	if err != nil {
		t.Fatalf("MarkFileRemoved: %v", err)
	}
	if !removed.FileGone() {
		t.Fatalf("FileGone() false after MarkFileRemoved, want true")
	}
}

func TestDueForRemoval(t *testing.T) {
	s := newExportStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	past := time.Now().UTC().Add(-60 * 24 * time.Hour)
	created, err := s.Create(ctx, baseOrgJob(past))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	due, err := s.DueForRemoval(ctx, time.Now().UTC(), 0)
	if err != nil {
		t.Fatalf("DueForRemoval: %v", err)
	}
	if len(due) != 1 || due[0].ID != created.ID {
		t.Fatalf("DueForRemoval = %+v, want exactly the overdue job", due)
	}
}
