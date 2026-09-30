package mwanachamacustody_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func examplePaths(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob("spec/examples/*.json")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(paths) < 2 {
		t.Fatalf("this module ships %d domain specs; at least two are needed for domain-neutrality to be exercised rather than asserted", len(paths))
	}
	return paths
}

// TestEveryExampleFitsTheTypes builds a manager over every spec this module
// ships, not the one a test happened to load. Drift under an unloaded domain
// is invisible otherwise, and the construction is what cross-checks the
// declaration against the Go types in both directions.
func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, path := range examplePaths(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			s, err := mwanachamacustody.LoadSpec(path)
			if err != nil {
				t.Fatalf("LoadSpec: %v", err)
			}
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("gorm.Open: %v", err)
			}
			if err := mwanachamacustody.Provision(db, s); err != nil {
				t.Fatalf("Provision: %v", err)
			}
			if _, err := mwanachamacustody.NewCustodyManager(db, s, mwanachamacustody.SystemClock, nil); err != nil {
				t.Fatalf("NewCustodyManager: %v", err)
			}
		})
	}
}

// TestTwoDomainsCoexist migrates every shipped domain into one database. The
// table names are hashed per instance, so two domains sharing a database is
// the property that has to hold rather than an accident of naming.
func TestTwoDomainsCoexist(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	want := 0
	for _, path := range examplePaths(t) {
		s, err := mwanachamacustody.LoadSpec(path)
		if err != nil {
			t.Fatalf("LoadSpec %s: %v", path, err)
		}
		if err := mwanachamacustody.Provision(db, s); err != nil {
			t.Fatalf("Provision %s: %v", path, err)
		}
		want += len(s.Objects)
	}
	var got int64
	if err := db.Raw(`select count(*) from sqlite_master
		where type = 'table' and name not like '%_spec_table_names'`).Scan(&got).Error; err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if int(got) != want {
		t.Fatalf("%d tables across the shipped domains, want %d — a collision means two domains are sharing a table", got, want)
	}
}

// TestRequiredFieldsHaveNoDefault asserts no column is ever both not null
// and defaulted. A default is exactly what lets an omitted required value
// pass unnoticed, so the two together turn a refusal into a silent write.
// Checked on the emitted DDL rather than on the declaration, because the
// DDL is what the database is actually given.
func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	checked := 0
	for _, path := range examplePaths(t) {
		s, err := mwanachamacustody.LoadSpec(path)
		if err != nil {
			t.Fatalf("LoadSpec %s: %v", path, err)
		}
		for _, o := range s.Objects {
			for _, f := range o.Fields {
				if f.Required && f.Default != "" {
					t.Errorf("%s: %s.%s is required and carries the default %q",
						path, o.Name, f.Name, f.Default)
				}
			}
		}
		for _, dialect := range []string{"postgres", "sqlite"} {
			for _, stmt := range s.DDL(dialect) {
				for _, line := range strings.Split(stmt, "\n") {
					column := strings.TrimSpace(strings.TrimSuffix(line, ","))
					if column == "" || !strings.Contains(column, "not null") {
						continue
					}
					checked++
					if strings.Contains(column, "default ") {
						t.Errorf("%s (%s): %q is both not null and defaulted",
							path, dialect, column)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no not-null column was examined, so this test proves nothing")
	}
	t.Logf("checked %d not-null columns", checked)
}

// TestVocabularyMatchesTheBlueprint holds models/vocabulary.go and the
// blueprint's declared values to each other in both directions. The
// constants were promised to agree with the spec and nothing checked it.
//
// The declared order matters too, not just the membership: the log draws its
// filter chips in the order the blueprint lists class and chip, and that
// order is read off the spec rather than kept a second time in Go.
func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	b, err := mwanachamacustody.Blueprint()
	if err != nil {
		t.Fatalf("Blueprint: %v", err)
	}

	goValues := constantsByType(t, "models/vocabulary.go")

	// Effect is deliberately absent: strengthens/neutral/weakens live inside
	// the effect document's values, not in a declared column, so the spec has
	// no values list to agree with.
	declaredBy := map[string]string{
		"ActKind":      "structure_act.kind",
		"ActClass":     "structure_act.class",
		"EventKind":    "event.kind",
		"EventChip":    "event.chip",
		"Scope":        "export_job.scope",
		"Format":       "export_job.format",
		"Status":       "export_job.status",
		"ConsentScope": "consent_version.scope",
		"Language":     "consent_version.language",
	}

	for typeName, where := range declaredBy {
		role, field := splitRoleField(t, where)
		declared := blueprintValues(t, b, role, field)
		got := goValues[typeName]
		if got == nil {
			t.Errorf("models/vocabulary.go declares no constants of type %s, which %s's values need", typeName, where)
			continue
		}
		assertSameSet(t, typeName, where, got, declared)
	}

	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if len(f.Values) == 0 {
				continue
			}
			if !mappedSomewhere(declaredBy, o.Role+"."+f.Name) {
				t.Errorf("%s.%s declares values that no Go type in this test is held against", o.Role, f.Name)
			}
		}
	}

	for typeName := range goValues {
		if typeName == "Effect" {
			continue
		}
		if _, ok := declaredBy[typeName]; !ok {
			t.Errorf("models/vocabulary.go declares the type %s, which this test holds against no declared field", typeName)
		}
	}
}

func assertSameSet(t *testing.T, typeName, where string, got, declared []string) {
	t.Helper()
	inGo := map[string]bool{}
	for _, v := range got {
		inGo[v] = true
	}
	inSpec := map[string]bool{}
	for _, v := range declared {
		inSpec[v] = true
	}
	for _, v := range declared {
		if !inGo[v] {
			t.Errorf("%s declares %q and %s has no constant for it", where, v, typeName)
		}
	}
	for _, v := range got {
		if !inSpec[v] {
			t.Errorf("%s has a constant for %q and %s declares no such value", typeName, v, where)
		}
	}
	if len(got) != len(declared) {
		t.Errorf("%s has %d constants and %s declares %d values", typeName, len(got), where, len(declared))
	}
}

func mappedSomewhere(declaredBy map[string]string, where string) bool {
	for _, mapped := range declaredBy {
		if mapped == where {
			return true
		}
	}
	// A role's language column is declared on two objects and held against
	// one Go type; naming both would be two copies of the same agreement.
	return where == "consent_record.language"
}

func splitRoleField(t *testing.T, where string) (string, string) {
	t.Helper()
	parts := strings.Split(where, ".")
	if len(parts) != 2 {
		t.Fatalf("%q is not role.field", where)
	}
	return parts[0], parts[1]
}

func blueprintValues(t *testing.T, b *spec.Blueprint, role, field string) []string {
	t.Helper()
	o, ok := b.Object(role)
	if !ok {
		t.Fatalf("the blueprint declares no role %q", role)
	}
	for _, f := range o.Fields {
		if f.Name == field {
			if len(f.Values) == 0 {
				t.Fatalf("%s.%s declares no values", role, field)
			}
			return f.Values
		}
	}
	t.Fatalf("%s declares no field %q", role, field)
	return nil
}
