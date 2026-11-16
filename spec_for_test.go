package mwanachamacustody_test

import (
	"testing"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func TestSpecForNamesTheInstanceAndTheSixObjects(t *testing.T) {
	s, err := mwanachamacustody.SpecFor("app")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if s.Instance != "app" {
		t.Errorf("Instance = %q, want app", s.Instance)
	}
	if len(s.Objects) != 6 {
		t.Errorf("objects = %d, want 6", len(s.Objects))
	}
	if s.MountName() != "main" {
		t.Errorf("mount = %q, want the default", s.MountName())
	}
}

func TestSpecForMountNamesTheMount(t *testing.T) {
	s, err := mwanachamacustody.SpecForMount("app", "clinic")
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	if s.MountName() != "clinic" {
		t.Errorf("mount = %q, want clinic", s.MountName())
	}
}
