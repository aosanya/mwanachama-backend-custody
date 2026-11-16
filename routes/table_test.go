package routes_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-custody/routes"
)

func TestEveryDeclaredErrorHasASentinel(t *testing.T) {
	unmapped, err := routes.Table.UnmappedSentinels()
	if err != nil {
		t.Fatalf("UnmappedSentinels: %v", err)
	}
	if len(unmapped) > 0 {
		t.Fatalf("operations name errors with no sentinel: %v", unmapped)
	}
}

func TestShapeBuilds(t *testing.T) {
	shape, err := routes.Table.Shape()
	if err != nil {
		t.Fatalf("Shape: %v", err)
	}
	if len(shape) == 0 {
		t.Fatal("the operations file declares no routes")
	}
}
