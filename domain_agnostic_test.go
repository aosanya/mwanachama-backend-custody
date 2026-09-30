package mwanachamacustody_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

// domainWords are words that mean something in one domain and nothing in
// another. This module records who did what to whose data; it must not learn
// what the data is about. A structure is a chapter to one organization and a
// ward to another, and neither word belongs here.
//
// "member" is listed and "membership" is not: every organization has
// membership, but only some have members. Matching is per word, so
// ActMembershipEnded and ClassMembership pass and a MemberID would not.
var domainWords = map[string]bool{
	"chapter":   true,
	"member":    true,
	"agency":    true,
	"sacco":     true,
	"sector":    true,
	"archetype": true,
	"patient":   true,
	"ward":      true,
	"clinic":    true,
	"library":   true,
	"borrower":  true,
	"tenant":    true,
}

var agnosticDirs = []string{".", "models", "routes"}

func TestNoDomainWordInAnyIdentifier(t *testing.T) {
	for _, dir := range agnosticDirs {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, notTest, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					if word := offendingWord(id.Name); word != "" {
						t.Errorf("%s: the identifier %s carries the domain word %q",
							path, id.Name, word)
					}
					return true
				})
			}
		}
	}
}

// TestNoDomainWordInAnyStoredValue is the one that matters most: a stored
// enum value outlives a rename. An identifier can be renamed in an
// afternoon; a value already written to a column cannot.
func TestNoDomainWordInAnyStoredValue(t *testing.T) {
	b, err := mwanachamacustody.Blueprint()
	if err != nil {
		t.Fatalf("Blueprint: %v", err)
	}
	seen := 0
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			for _, v := range f.Values {
				seen++
				for _, segment := range strings.Split(v, "_") {
					if domainWords[segment] {
						t.Errorf("%s.%s declares the stored value %q, which carries the domain word %q",
							o.Role, f.Name, v, segment)
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no declared enum values were checked, so this test proves nothing")
	}
	t.Logf("checked %d declared enum values", seen)
}

func notTest(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }

// offendingWord splits a Go identifier into its words and returns the first
// that is a domain word, or "".
func offendingWord(name string) string {
	for _, word := range camelWords(name) {
		if domainWords[word] {
			return word
		}
	}
	return ""
}

func camelWords(name string) []string {
	var out []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			out = append(out, strings.ToLower(current.String()))
			current.Reset()
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '_':
			flush()
		case r >= 'A' && r <= 'Z':
			// A run of capitals is one word until a lowercase letter starts
			// the next, so ActorID is [actor, id] and IDKey is [id, key].
			if i > 0 && (runes[i-1] < 'A' || runes[i-1] > 'Z') {
				flush()
			} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				flush()
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return out
}
