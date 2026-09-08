// contact.go — HTTP routes over mwanachamacustody.ContactRepository. Only
// ListForSubject qualifies — see doc.go for why the gateway's real,
// decorated /contact-reads response cannot move here.
package routes

import (
	"errors"
	"net/http"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func listContactReads(repo mwanachamacustody.ContactRepository, identity Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID := r.PathValue("actorID")
		if identity.CallerID(r) != actorID {
			writeErr(w, http.StatusForbidden, "only the actor this record is about may read it")
			return
		}
		reads, err := repo.ListForSubject(r.Context(), actorID)
		if err != nil {
			writeContactErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, reads)
	}
}

func writeContactErr(w http.ResponseWriter, err error) {
	if errors.Is(err, mwanachamacustody.ErrNotFound) {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeErr(w, http.StatusInternalServerError, "internal error")
}
