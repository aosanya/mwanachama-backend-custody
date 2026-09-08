// export.go — HTTP routes over mwanachamacustody.ExportRepository. Net-new surface —
// see doc.go: the gateway never exposed export_job over HTTP, so there is
// no existing handler shape to mirror, only the Repository's own read
// contract to expose safely. Create/Progress/Complete/MarkFileRemoved are
// deliberately absent — the Repository's own doc says they belong to a
// build worker, never a client.
package routes

import (
	"errors"
	"net/http"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

// getExportJob handles GET {jobID} — the writer's own read-back and any
// caller that already knows the id. Deliberately unscoped, matching
// ExportRepository.Get's own doc: it must not be reachable without a scope
// check on the row it returns, which is the mounting process's own job —
// wrap this handler behind whatever gate proves the caller may see this
// specific job before mounting it.
func getExportJob(repo mwanachamacustody.ExportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := repo.Get(r.Context(), r.PathValue("jobID"))
		if err != nil {
			writeExportErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// listOrganizationExports handles GET /export-jobs — export-job.md's two
// policies are never one with an `or` (see the Repository's own doc), so
// this route answers ONLY the organization-scoped list; a mounting process
// wraps it in whatever gate proves the caller holds the HQ capability.
func listOrganizationExports(repo mwanachamacustody.ExportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := repo.ListOrganization(r.Context(), 0)
		if err != nil {
			writeExportErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func listActorExports(repo mwanachamacustody.ExportRepository, identity Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID := r.PathValue("actorID")
		if identity.CallerID(r) != actorID {
			writeErr(w, http.StatusForbidden, "only the actor this record is about may read it")
			return
		}
		out, err := repo.ListForActor(r.Context(), actorID, 0)
		if err != nil {
			writeExportErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// writeExportErr classifies the only two shapes an ExportRepository read can
// fail with: an unknown id, or a store fault this package has no room to
// diagnose further — see doc.go for why a fuller classifier stays gateway-side.
func writeExportErr(w http.ResponseWriter, err error) {
	if errors.Is(err, mwanachamacustody.ErrNotFound) {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeErr(w, http.StatusInternalServerError, "internal error")
}
