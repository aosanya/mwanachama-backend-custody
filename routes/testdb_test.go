package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/routes"
)

const testSpec = "../spec/examples/mwanachama.custody.json"

func callerIs(id string) dispatch.Caller {
	return func(context.Context) string { return id }
}

func newManager(t *testing.T, resolver mwanachamacustody.ScopeResolver) (*mwanachamacustody.CustodyManager, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamacustody.LoadSpec(testSpec)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if err := mwanachamacustody.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	cm, err := mwanachamacustody.NewCustodyManager(db, s, mwanachamacustody.SystemClock, resolver)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	return cm, db
}

func serve(t *testing.T, cm *mwanachamacustody.CustodyManager, caller string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(cm, routes.Mount{Caller: callerIs(caller)}) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
