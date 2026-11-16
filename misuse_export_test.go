package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func TestActorExports_OtherResidentCannotListAnothersExports(t *testing.T) {
	db, sp := newTestDB(t)
	m, err := mwanachamacustody.NewCustodyManager(db, sp, mwanachamacustody.SystemClock, nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	ctx := context.Background()
	owner := "6e594821-8a31-4e9e-afc4-66b9dc750a32"
	other := "82323eea-17d8-4416-a310-886a513f6479"
	if _, err := m.ListActorExports(ctx, owner, other); !errors.Is(err, mwanachamacustody.ErrNotSelf) {
		t.Fatalf("another resident listing exports: got %v, want ErrNotSelf", err)
	}
	if _, err := m.ListActorExports(ctx, owner, owner); err != nil {
		t.Fatalf("resident listing own exports: %v", err)
	}
}
