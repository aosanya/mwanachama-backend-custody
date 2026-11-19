package routes_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-custody/routes"
)

const declaredRouteCount = 19

func getStatus(t *testing.T, base, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(base + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func muxOf(rs []routes.Route) *http.ServeMux {
	mux := http.NewServeMux()
	for _, rt := range rs {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	return mux
}

func TestEveryBuilderYieldsTheNineteenDeclaredAddresses(t *testing.T) {
	cm, _ := newManager(t, nil)
	built, err := routes.Build(cm)
	if err != nil || len(built) != declaredRouteCount {
		t.Fatalf("Build = %d routes, %v; want %d", len(built), err, declaredRouteCount)
	}
	if n := len(routes.RoutesFor(cm, routes.Mount{})); n != declaredRouteCount {
		t.Fatalf("RoutesFor = %d", n)
	}
	shape, err := routes.Table.Shape()
	if err != nil || len(shape) != declaredRouteCount {
		t.Fatalf("Shape = %d routes, %v", len(shape), err)
	}
}

func TestShapeCarriesEveryActionWithoutAHandler(t *testing.T) {
	shape, err := routes.Table.Shape()
	if err != nil {
		t.Fatalf("Shape: %v", err)
	}
	var actions []string
	for _, rt := range shape {
		if rt.Action == "" {
			t.Errorf("shape route %s has no action", rt.Pattern(""))
		}
		actions = append(actions, rt.Action)
	}
	sort.Strings(actions)
	for i := 1; i < len(actions); i++ {
		if actions[i] == actions[i-1] {
			t.Errorf("action %s declared twice", actions[i])
		}
	}
	if !strings.HasPrefix(actions[0], "custody.") {
		t.Errorf("actions are not under the custody namespace: %v", actions)
	}
}

func TestSplitGatesEveryRoute(t *testing.T) {
	cm, _ := newManager(t, nil)
	split := routes.Split(cm, routes.Mount{})
	if len(split.Anonymous) != 0 || len(split.Gated) != declaredRouteCount {
		t.Fatalf("Split = %d anonymous, %d gated", len(split.Anonymous), len(split.Gated))
	}
}

func TestARefusedActionNamesTheEventListAndAnswers403(t *testing.T) {
	cm, _ := newManager(t, nil)
	var asked []string
	deny := func(_ context.Context, action string) error {
		asked = append(asked, action)
		return errors.New("no")
	}
	srv := httptest.NewServer(muxOf(routes.RoutesFor(cm, routes.Mount{Caller: callerIs("operator-1"), Authorize: deny})))
	t.Cleanup(srv.Close)

	status, body := getStatus(t, srv.URL, "/custody-log")
	if status != http.StatusForbidden {
		t.Fatalf("status = %d %s, want 403", status, body)
	}
	if len(asked) != 1 || asked[0] != "custody.event.list" {
		t.Fatalf("authorizer asked %v, want [custody.event.list]", asked)
	}
}

func TestAnAllowingAuthorizerLetsTheCallThrough(t *testing.T) {
	cm, _ := newManager(t, nil)
	allow := func(context.Context, string) error { return nil }
	srv := httptest.NewServer(muxOf(routes.RoutesFor(cm, routes.Mount{Caller: callerIs("operator-1"), Authorize: allow})))
	t.Cleanup(srv.Close)

	if status, body := getStatus(t, srv.URL, "/custody-log"); status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", status, body)
	}
}

func TestRefusalsMapToTheirDeclaredStatuses(t *testing.T) {
	cm, _ := newManager(t, nil)
	srv := serve(t, cm, "operator-1")
	cases := []struct {
		name string
		path string
		want int
	}{
		{"a missing export job is 404", "/export-jobs/nope", http.StatusNotFound},
		{"a missing consent version is 404", "/consent/versions/nope", http.StatusNotFound},
		{"no version in force is 404", "/consent/versions/in-force/mechanics/en", http.StatusNotFound},
		{"an unknown scope is 400", "/structures/s1/act-log?scope=galaxy", http.StatusBadRequest},
		{"an unknown act class is 400", "/structures/s1/act-log?class=bogus", http.StatusBadRequest},
		{"a zero limit is 400", "/structures/s1/act-log?limit=0", http.StatusBadRequest},
		{"a non-numeric limit is 400", "/structures/s1/act-log?limit=abc", http.StatusBadRequest},
		{"a malformed since is 400", "/structures/s1/act-log/counts?since=yesterday", http.StatusBadRequest},
		{"an unknown chip is 400", "/custody-log?chip=bogus", http.StatusBadRequest},
		{"another actor's export jobs are 403", "/actors/someone-else/export-jobs", http.StatusForbidden},
		{"another actor's contact reads are 403", "/actors/someone-else/contact-reads", http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if status, body := getStatus(t, srv.URL, c.path); status != c.want {
				t.Fatalf("GET %s = %d %s, want %d", c.path, status, body, c.want)
			}
		})
	}
}

func TestTheCallerMayReadTheirOwnExportsAndContactReads(t *testing.T) {
	cm, _ := newManager(t, nil)
	srv := serve(t, cm, "actor-7")
	for _, path := range []string{"/actors/actor-7/export-jobs", "/actors/actor-7/contact-reads"} {
		if status, body := getStatus(t, srv.URL, path); status != http.StatusOK {
			t.Fatalf("GET %s = %d %s, want 200", path, status, body)
		}
	}
}

func TestAnEmptyOrganizationListsNoExportJobs(t *testing.T) {
	cm, _ := newManager(t, nil)
	srv := serve(t, cm, "operator-1")
	status, body := getStatus(t, srv.URL, "/export-jobs")
	if status != http.StatusOK || strings.Contains(body, "\"id\"") {
		t.Fatalf("GET /export-jobs = %d %s, want 200 with no jobs", status, body)
	}
}
