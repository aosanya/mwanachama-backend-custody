package routes_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-custody/routes"
)

func TestOnlyTheConsentOperationsAreTools(t *testing.T) {
	cm, _ := newManager(t, nil)
	tools, err := routes.BuildTools(cm, routes.Mount{Caller: callerIs("resident-1")})
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	var got []string
	for _, tool := range tools {
		got = append(got, tool.Name)
	}
	sort.Strings(got)
	want := "custody_agree_consent custody_get_in_force_consent custody_get_my_consent custody_list_consent_versions custody_publish_consent_version custody_record_paper_consent custody_withdraw_consent"
	if strings.Join(got, " ") != want {
		t.Errorf("tools = %v, want %s", got, want)
	}
}
