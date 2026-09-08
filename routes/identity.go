package routes

import (
	"context"
	"net/http"
)

// Identity resolves the caller's own identity from an authenticated request
// — the one gateway-session concern this package cannot own itself. Mirrors
// mwanachama-backend-comm/routes.Identity.
type Identity interface {
	CallerID(r *http.Request) string
}

type ScopeResolver interface {
	Subtree(ctx context.Context, structureID string) ([]string, error)
}
