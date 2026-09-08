package routes

import (
	"net/http"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

// Route is one address this package answers, relative to wherever the
// mounting process prefixes it — mirrors mwanachama-backend-actor/routes.
// Route and mwanachama-backend-comm/routes.Route exactly.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Pattern returns the http.ServeMux registration pattern for this route
// once mounted under prefix.
func (r Route) Pattern(prefix string) string {
	return r.Method + " " + prefix + r.Path
}

func CustodyRoutes(custody mwanachamacustody.CustodyRepository, resolver ScopeResolver) []Route {
	return []Route{
		{Method: http.MethodGet, Path: "/structures/{structureID}/act-log", Handler: listStructureActs(custody, resolver)},
		{Method: http.MethodGet, Path: "/structures/{structureID}/act-log/counts", Handler: countStructureActs(custody, resolver)},
		{Method: http.MethodGet, Path: "/custody-log", Handler: listCustodyEvents(custody)},
		{Method: http.MethodGet, Path: "/custody-log/counts", Handler: countCustodyEvents(custody)},
	}
}

// ExportRoutes is the three safe reads over export_job — see doc.go for why
// there is no write route.
func ExportRoutes(export mwanachamacustody.ExportRepository, identity Identity) []Route {
	return []Route{
		{Method: http.MethodGet, Path: "/export-jobs/{jobID}", Handler: getExportJob(export)},
		{Method: http.MethodGet, Path: "/export-jobs", Handler: listOrganizationExports(export)},
		{Method: http.MethodGet, Path: "/actors/{actorID}/export-jobs", Handler: listActorExports(export, identity)},
	}
}

// ContactRoutes is the one route that qualifies over contact_read — see
// doc.go for why the decorated gateway response cannot move here.
func ContactRoutes(contact mwanachamacustody.ContactRepository, identity Identity) []Route {
	return []Route{
		{Method: http.MethodGet, Path: "/actors/{actorID}/contact-reads", Handler: listContactReads(contact, identity)},
	}
}

func ConsentRoutes(consent mwanachamacustody.ConsentRepository, identity Identity) []Route {
	return []Route{
		{Method: http.MethodGet, Path: "/consent/versions/in-force/{scope}/{language}", Handler: getInForceConsentVersion(consent)},
		{Method: http.MethodGet, Path: "/consent/versions/{versionID}", Handler: getConsentVersion(consent)},
		{Method: http.MethodPost, Path: "/consent/versions", Handler: createConsentVersion(consent)},
		{Method: http.MethodPost, Path: "/consent/versions/{versionID}/publish", Handler: publishConsentVersion(consent, identity)},
	}
}

// Routes is every address this package answers today: CustodyRoutes,
// ExportRoutes, ContactRoutes and ConsentRoutes concatenated. A mounting
// process that wants to wrap each domain's gate differently (the gateway
// does, today) calls the four functions separately instead.
func Routes(custody mwanachamacustody.CustodyRepository, export mwanachamacustody.ExportRepository, contact mwanachamacustody.ContactRepository, consent mwanachamacustody.ConsentRepository, resolver ScopeResolver, identity Identity) []Route {
	out := CustodyRoutes(custody, resolver)
	out = append(out, ExportRoutes(export, identity)...)
	out = append(out, ContactRoutes(contact, identity)...)
	out = append(out, ConsentRoutes(consent, identity)...)
	return out
}
