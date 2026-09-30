package routes

import (
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	custody "github.com/aosanya/mwanachama-backend-custody"
)

type Route = httpwire.Route

var sentinels = map[string]error{
	"ErrNotFound":            custody.ErrNotFound,
	"ErrAlreadyPublished":    custody.ErrAlreadyPublished,
	"ErrUnknownScope":        custody.ErrUnknownScope,
	"ErrUnknownActClass":     custody.ErrUnknownActClass,
	"ErrUnknownEventChip":    custody.ErrUnknownEventChip,
	"ErrInvalidLimit":        custody.ErrInvalidLimit,
	"ErrInvalidSince":        custody.ErrInvalidSince,
	"ErrStructureIDRequired": custody.ErrStructureIDRequired,
	"ErrNotSelf":             custody.ErrNotSelf,
}

// AnonymousActions is every operation reachable without presenting a caller.
// It is empty: nothing in an audit trail is public, and this module has
// never carried a gate of its own — whatever mounts it supplies the
// Authorizer. It is the allowlist Split is taken over, never the list of
// what is protected, so an operation added to the spec and not named here
// arrives gated.
var AnonymousActions = []string{}

type Mount struct {
	Authorize dispatch.Authorizer
	Caller    dispatch.Caller
}

func Build(cm *custody.CustodyManager) ([]Route, error) { return BuildWith(cm, nil) }

func BuildWith(cm *custody.CustodyManager, authorize dispatch.Authorizer) ([]Route, error) {
	return BuildFor(cm, Mount{Authorize: authorize})
}

func BuildFor(cm *custody.CustodyManager, m Mount) ([]Route, error) {
	s, err := custody.Operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{
		Manager: cm, Errors: sentinels, Authorize: m.Authorize, Caller: m.Caller,
	})
}

func Routes(cm *custody.CustodyManager) []Route { return RoutesWith(cm, nil) }

func RoutesWith(cm *custody.CustodyManager, authorize dispatch.Authorizer) []Route {
	return RoutesFor(cm, Mount{Authorize: authorize})
}

func RoutesFor(cm *custody.CustodyManager, m Mount) []Route {
	out, err := BuildFor(cm, m)
	if err != nil {
		panic(fmt.Sprintf("custody routes: %v", err))
	}
	return out
}

func Split(cm *custody.CustodyManager) dispatch.Split { return SplitWith(cm, nil) }

func SplitWith(cm *custody.CustodyManager, authorize dispatch.Authorizer) dispatch.Split {
	return SplitFor(cm, Mount{Authorize: authorize})
}

func SplitFor(cm *custody.CustodyManager, m Mount) dispatch.Split {
	public := dispatch.Anonymous(RoutesFor(cm, Mount{Caller: m.Caller}), AnonymousActions...)
	gated := dispatch.Anonymous(RoutesFor(cm, m), AnonymousActions...)
	return dispatch.Split{Anonymous: public.Anonymous, Gated: gated.Gated}
}

func PublicRoutes(cm *custody.CustodyManager) []Route { return Split(cm).Anonymous }

func OperatorRoutes(cm *custody.CustodyManager, authorize dispatch.Authorizer) []Route {
	return SplitWith(cm, authorize).Gated
}

func Shape() []Route {
	s, err := custody.Operations()
	if err != nil {
		panic(fmt.Sprintf("custody routes: %v", err))
	}
	return dispatch.Shape(s)
}
