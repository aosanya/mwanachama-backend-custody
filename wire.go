package mwanachamacustody

import (
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func actLogJSON(entries []models.StructureActLogEntry) []map[string]any {
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

func custodyLogJSON(entries []models.Entry) []map[string]any {
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
