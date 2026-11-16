package routes_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-custody/routes"
)

func TestBuildAndRoutesForGiveTheSameTable(t *testing.T) {
	cm, _ := newManager(t, nil)
	built, err := routes.Build(cm)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	forMount, err := routes.BuildFor(cm, routes.Mount{Caller: callerIs("person-1")})
	if err != nil {
		t.Fatalf("BuildFor: %v", err)
	}
	listed := routes.RoutesFor(cm, routes.Mount{})
	if len(built) == 0 || len(forMount) != len(built) || len(listed) != len(built) {
		t.Fatalf("build %d, build for %d, routes for %d", len(built), len(forMount), len(listed))
	}
}

func TestNothingIsAnonymousAndEveryRouteIsGated(t *testing.T) {
	cm, _ := newManager(t, nil)
	split := routes.Split(cm, routes.Mount{})
	if len(routes.AnonymousActions) != 0 || len(split.Anonymous) != 0 || len(split.Gated) != len(routes.RoutesFor(cm, routes.Mount{})) {
		t.Fatalf("anonymous %d, gated %d", len(split.Anonymous), len(split.Gated))
	}
}

func TestARefusedActionAnswers403BeforeTheManager(t *testing.T) {
	cm, _ := newManager(t, nil)
	var asked []string
	refuse := func(_ context.Context, action string) error {
		asked = append(asked, action)
		return errors.New("no")
	}
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(cm, routes.Mount{Caller: callerIs("person-1"), Authorize: refuse}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/structures/ward-3/act-log", strings.NewReader("")))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("answered %d %s", rec.Code, rec.Body.String())
	}
	if len(asked) != 1 || !strings.HasPrefix(asked[0], "custody.") {
		t.Fatalf("asked %v", asked)
	}
}
