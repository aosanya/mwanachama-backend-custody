package routes

import (
	"errors"
	"net/http"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func getInForceConsentVersion(repo mwanachamacustody.ConsentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := mwanachamacustody.ConsentScope(r.PathValue("scope"))
		language := mwanachamacustody.Language(r.PathValue("language"))
		out, err := repo.GetInForce(r.Context(), scope, language)
		if err != nil {
			writeConsentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// getConsentVersion handles GET /consent/versions/{versionID} — fetches one
// version by id, for reading back the exact wording a consent record pins.
func getConsentVersion(repo mwanachamacustody.ConsentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := repo.GetVersion(r.Context(), r.PathValue("versionID"))
		if err != nil {
			writeConsentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// createVersionInput is the wire shape for POST /consent/versions —
// PublishedAt/PublishedBy/SupersededAt/CopiedMechanicsID are deliberately
// absent, mirroring the gateway's own DEV-1174 fix: those four are written
// only by publishConsentVersion (via PublishVersion), never accepted from a
// caller — accepting them would let a caller mint text already in force,
// naming a publisher who never published.
type createVersionInput struct {
	ID         string                              `json:"id"`
	Scope      mwanachamacustody.ConsentScope      `json:"scope"`
	Version    string                              `json:"version"`
	Language   mwanachamacustody.Language          `json:"language"`
	Clauses    []mwanachamacustody.Clause          `json:"clauses"`
	Effect     map[string]mwanachamacustody.Effect `json:"effect"`
	Translated bool                                `json:"translated"`
	ReleaseRef string                              `json:"release_ref"`
}

// createConsentVersion handles POST /consent/versions — mints a new,
// unpublished version row. A mounting process gates this behind whatever
// operator capability it uses for consent administration; this handler
// carries no gate of its own.
func createConsentVersion(repo mwanachamacustody.ConsentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in createVersionInput
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		v := mwanachamacustody.TextVersion{
			ID:         in.ID,
			Scope:      in.Scope,
			Version:    in.Version,
			Language:   in.Language,
			Clauses:    in.Clauses,
			Effect:     in.Effect,
			Translated: in.Translated,
			ReleaseRef: in.ReleaseRef,
		}
		out, err := repo.CreateVersion(r.Context(), v)
		if err != nil {
			writeConsentErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// publishConsentVersion handles POST /consent/versions/{versionID}/publish.
// publishedBy is the authenticated caller, never a body field — an act that
// creates authority names who did it — so this handler needs [Identity]
// even though it carries no self-only check of its own.
func publishConsentVersion(repo mwanachamacustody.ConsentRepository, identity Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		publishedBy := identity.CallerID(r)
		out, err := repo.PublishVersion(r.Context(), r.PathValue("versionID"), publishedBy, time.Now().UTC())
		if err != nil {
			writeConsentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func writeConsentErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, mwanachamacustody.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, mwanachamacustody.ErrAlreadyPublished):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}
