package routes_test

import (
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func TestGiveConsentDeclaresTheContentDefaultLanguage(t *testing.T) {
	declared, err := dispatch.Parse(mwanachamacustody.Operations())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, op := range declared.Operations {
		if op.Action != "custody.consent.give" {
			continue
		}
		if !strings.Contains(op.Description, "the agency's content default language when left out") {
			t.Errorf("description = %q, want the agency's content default language named", op.Description)
		}
		return
	}
	t.Fatal("custody.consent.give is not declared")
}
