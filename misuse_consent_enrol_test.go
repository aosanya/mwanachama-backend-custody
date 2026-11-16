package mwanachamacustody_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-custody/routes"
)

func TestConsent_ResidentCanGiveConsentThenAnswerAForm(t *testing.T) {
	shape, err := routes.Table.Shape()
	if err != nil {
		t.Fatalf("Shape: %v", err)
	}
	var withdraw, give []string
	for _, rt := range shape {
		if !strings.HasPrefix(rt.Action, "custody.consent.") {
			continue
		}
		if rt.Action == "custody.consent.withdraw" {
			withdraw = append(withdraw, rt.Method+" "+rt.Path)
			continue
		}
		if rt.Method == http.MethodPost {
			give = append(give, rt.Method+" "+rt.Path+" ("+rt.Action+")")
		}
	}
	if len(withdraw) == 0 {
		t.Fatalf("custody table declares no consent withdraw route; the table did not load as expected")
	}
	if len(give) == 0 {
		t.Fatalf("no custody route lets a resident record their own consent to the in-force version (only withdraw: %v); forms answers stay refused with consent: no live consent on record", withdraw)
	}
}
