package routes

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

// maxActLogPage and maxCustodyLogPage bound what one request may pull. A
// caller asking for more gets this many rather than an error: the log is a
// scroll, not a query API, and refusing a large `limit` would push clients
// into paging loops that hit the table harder than one capped read —
// mirrors the gateway's own constants.
const (
	maxActLogPage     = 500
	maxCustodyLogPage = 500
)

func actLogScope(resolver ScopeResolver, r *http.Request, structureID string) ([]string, error) {
	switch scope := r.URL.Query().Get("scope"); scope {
	case "", "structure":
		return []string{structureID}, nil
	case "subtree":
		if resolver == nil {
			return []string{structureID}, nil
		}
		return resolver.Subtree(r.Context(), structureID)
	default:
		return nil, errUnknownScope
	}
}

func listStructureActs(repo mwanachamacustody.CustodyRepository, resolver ScopeResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		structureID := r.PathValue("structureID")
		if structureID == "" {
			writeErr(w, http.StatusBadRequest, "structure id required")
			return
		}
		structures, err := actLogScope(resolver, r, structureID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, `scope must be "structure" or "subtree"`)
			return
		}
		class := mwanachamacustody.ActClass(r.URL.Query().Get("class"))
		if class != "" && !mwanachamacustody.IsActClass(class) {
			writeErr(w, http.StatusBadRequest, "unknown act class: "+string(class))
			return
		}
		limit, ok := parseLimit(w, r, maxActLogPage)
		if !ok {
			return
		}
		entries, err := repo.ListActs(r.Context(), structures, class, limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": actLogJSON(entries)})
	}
}

func countStructureActs(repo mwanachamacustody.CustodyRepository, resolver ScopeResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		structureID := r.PathValue("structureID")
		if structureID == "" {
			writeErr(w, http.StatusBadRequest, "structure id required")
			return
		}
		structures, err := actLogScope(resolver, r, structureID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, `scope must be "structure" or "subtree"`)
			return
		}
		since, ok := parseSince(w, r)
		if !ok {
			return
		}
		counts, err := repo.CountActsByClass(r.Context(), structures, since)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		byClass := make(map[string]int, len(mwanachamacustody.ActClasses()))
		total := 0
		for _, c := range mwanachamacustody.ActClasses() {
			n := counts[c]
			byClass[string(c)] = n
			total += n
		}
		writeJSON(w, http.StatusOK, map[string]any{"everything": total, "by_class": byClass})
	}
}

// listCustodyEvents handles GET /custody-log.
func listCustodyEvents(repo mwanachamacustody.CustodyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chip := mwanachamacustody.EventChip(r.URL.Query().Get("chip"))
		if chip != "" && !mwanachamacustody.IsEventChip(chip) {
			writeErr(w, http.StatusBadRequest, "unknown custody chip: "+string(chip))
			return
		}
		limit, ok := parseLimit(w, r, maxCustodyLogPage)
		if !ok {
			return
		}
		entries, err := repo.ListEvents(r.Context(), chip, limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": custodyLogJSON(entries)})
	}
}

// countCustodyEvents handles GET /custody-log/counts.
func countCustodyEvents(repo mwanachamacustody.CustodyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		since, ok := parseSince(w, r)
		if !ok {
			return
		}
		counts, err := repo.CountEventsByChip(r.Context(), since)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		byChip := make(map[string]int, len(mwanachamacustody.EventChips()))
		total := 0
		for _, c := range mwanachamacustody.EventChips() {
			n := counts[c]
			byChip[string(c)] = n
			total += n
		}
		writeJSON(w, http.StatusOK, map[string]any{"everything": total, "by_chip": byChip})
	}
}

func parseLimit(w http.ResponseWriter, r *http.Request, max int) (int, bool) {
	limit := max
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			writeErr(w, http.StatusBadRequest, "limit must be a positive integer")
			return 0, false
		}
		limit = min(n, max)
	}
	return limit, true
}

func parseSince(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "since must be an RFC3339 timestamp")
			return time.Time{}, false
		}
		return t, true
	}
	return time.Time{}, true
}

// actLogJSON renders entries for the wire — written out rather than tagging
// the domain struct, mirroring the gateway's own reason: ActorID empty
// *means* the timer, a real state the client draws differently from a
// missing field, and a struct tag would make the two move together by
// accident.
func actLogJSON(entries []mwanachamacustody.StructureActLogEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		row := map[string]any{
			"id":           e.ID,
			"structure_id": e.StructureID,
			"occurred_at":  e.OccurredAt,
			"act_kind":     string(e.Kind),
			"act_class":    string(e.Class),
			"subject_ref":  e.SubjectRef,
			"actor_id":     e.ActorID,
			"actor_label":  e.ActorLabel,
		}
		if e.ActorStructureID != "" {
			row["actor_structure_id"] = e.ActorStructureID
		}
		if e.SubjectID != "" {
			row["subject_id"] = e.SubjectID
		}
		if e.Detail != nil {
			row["detail"] = e.Detail
		}
		if e.EscalationLevel != nil {
			row["escalation_level"] = *e.EscalationLevel
		}
		if e.ToStructureID != "" {
			row["to_structure_id"] = e.ToStructureID
		}
		out = append(out, row)
	}
	return out
}

// custodyLogJSON renders entries for the wire, for actLogJSON's reason.
func custodyLogJSON(entries []mwanachamacustody.Entry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		row := map[string]any{
			"id":          e.ID,
			"occurred_at": e.OccurredAt,
			"event_kind":  string(e.Kind),
			"event_chip":  string(e.Chip),
			"actor_id":    e.ActorID,
			"actor_label": e.ActorLabel,
			"detail":      e.Detail,
		}
		if e.Evidence != nil {
			row["evidence"] = e.Evidence
		}
		if e.SubjectID != "" {
			row["subject_id"] = e.SubjectID
		}
		out = append(out, row)
	}
	return out
}

// errUnknownScope is a sentinel actLogScope fails with; both call sites
// above turn it into the same 400.
var errUnknownScope = errors.New("unknown scope")
