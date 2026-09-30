package mwanachamacustody

import (
	"context"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

const (
	maxActLogPage     = 500
	maxCustodyLogPage = 500
)

// ScopeResolver walks a structure's descendant closure. It is an externally
// supplied fact: this module has no structure hierarchy of its own.
type ScopeResolver interface {
	Subtree(ctx context.Context, structureID string) ([]string, error)
}

type CustodyManager struct {
	custody  *CustodyStore
	export   *ExportStore
	contact  *ContactStore
	consent  *ConsentStore
	resolver ScopeResolver
}

func NewCustodyManager(db *gorm.DB, s *spec.Spec, clock Clock, resolver ScopeResolver) (*CustodyManager, error) {
	custody, err := NewCustodyStore(db, s, clock)
	if err != nil {
		return nil, err
	}
	export, err := NewExportStore(db, s, clock)
	if err != nil {
		return nil, err
	}
	contact, err := NewContactStore(db, s, clock)
	if err != nil {
		return nil, err
	}
	consent, err := NewConsentStore(db, s)
	if err != nil {
		return nil, err
	}
	return &CustodyManager{
		custody:  custody,
		export:   export,
		contact:  contact,
		consent:  consent,
		resolver: resolver,
	}, nil
}

func (m *CustodyManager) Custody() models.CustodyRepository { return m.custody }
func (m *CustodyManager) Export() models.ExportRepository   { return m.export }
func (m *CustodyManager) Contact() models.ContactRepository { return m.contact }
func (m *CustodyManager) Consent() models.ConsentRepository { return m.consent }

// actLogScope turns the requested scope into the structure list the store
// filters on. An unknown scope is refused rather than silently narrowed.
func (m *CustodyManager) actLogScope(ctx context.Context, structureID, scope string) ([]string, error) {
	switch scope {
	case "", "structure":
		return []string{structureID}, nil
	case "subtree":
		if m.resolver == nil {
			return []string{structureID}, nil
		}
		return m.resolver.Subtree(ctx, structureID)
	default:
		return nil, ErrUnknownScope
	}
}

func parseLimit(raw string, max int) (int, error) {
	if raw == "" {
		return max, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, ErrInvalidLimit
	}
	return min(n, max), nil
}

func parseSince(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if !isInstant(raw) {
		return "", ErrInvalidSince
	}
	return raw, nil
}

func (m *CustodyManager) ListStructureActs(ctx context.Context, structureID, scope, class, limit string) ([]map[string]any, error) {
	if structureID == "" {
		return nil, ErrStructureIDRequired
	}
	structures, err := m.actLogScope(ctx, structureID, scope)
	if err != nil {
		return nil, err
	}
	if class != "" && !m.custody.ValidClass(models.ActClass(class)) {
		return nil, fmt.Errorf("%w: %s", ErrUnknownActClass, class)
	}
	n, err := parseLimit(limit, maxActLogPage)
	if err != nil {
		return nil, err
	}
	entries, err := m.custody.ListActs(ctx, structures, models.ActClass(class), n)
	if err != nil {
		return nil, err
	}
	return actLogJSON(entries), nil
}

func (m *CustodyManager) CountStructureActs(ctx context.Context, structureID, scope, since string) (int, map[string]int, error) {
	if structureID == "" {
		return 0, nil, ErrStructureIDRequired
	}
	structures, err := m.actLogScope(ctx, structureID, scope)
	if err != nil {
		return 0, nil, err
	}
	from, err := parseSince(since)
	if err != nil {
		return 0, nil, err
	}
	counts, err := m.custody.CountActsByClass(ctx, structures, from)
	if err != nil {
		return 0, nil, err
	}
	classes := m.custody.Classes()
	byClass := make(map[string]int, len(classes))
	total := 0
	for _, c := range classes {
		n := counts[c]
		byClass[string(c)] = n
		total += n
	}
	return total, byClass, nil
}

func (m *CustodyManager) ListCustodyEvents(ctx context.Context, chip, limit string) ([]map[string]any, error) {
	if chip != "" && !m.custody.ValidChip(models.EventChip(chip)) {
		return nil, fmt.Errorf("%w: %s", ErrUnknownEventChip, chip)
	}
	n, err := parseLimit(limit, maxCustodyLogPage)
	if err != nil {
		return nil, err
	}
	entries, err := m.custody.ListEvents(ctx, models.EventChip(chip), n)
	if err != nil {
		return nil, err
	}
	return custodyLogJSON(entries), nil
}

func (m *CustodyManager) CountCustodyEvents(ctx context.Context, since string) (int, map[string]int, error) {
	from, err := parseSince(since)
	if err != nil {
		return 0, nil, err
	}
	counts, err := m.custody.CountEventsByChip(ctx, from)
	if err != nil {
		return 0, nil, err
	}
	chips := m.custody.Chips()
	byChip := make(map[string]int, len(chips))
	total := 0
	for _, c := range chips {
		n := counts[c]
		byChip[string(c)] = n
		total += n
	}
	return total, byChip, nil
}

func (m *CustodyManager) GetExportJob(ctx context.Context, jobID string) (models.Job, error) {
	return m.export.Get(ctx, jobID)
}

func (m *CustodyManager) ListOrganizationExports(ctx context.Context) ([]models.Job, error) {
	return m.export.ListOrganization(ctx, 0)
}

func (m *CustodyManager) ListActorExports(ctx context.Context, actorID, callerID string) ([]models.Job, error) {
	if callerID != actorID {
		return nil, ErrNotSelf
	}
	return m.export.ListForActor(ctx, actorID, 0)
}

func (m *CustodyManager) ListContactReads(ctx context.Context, actorID, callerID string) ([]models.Read, error) {
	if callerID != actorID {
		return nil, ErrNotSelf
	}
	return m.contact.ListForSubject(ctx, actorID)
}

func (m *CustodyManager) GetConsentVersionInForce(ctx context.Context, scope, language string) (models.TextVersion, error) {
	return m.consent.GetInForce(ctx, models.ConsentScope(scope), models.Language(language))
}

func (m *CustodyManager) GetConsentVersion(ctx context.Context, versionID string) (models.TextVersion, error) {
	return m.consent.GetVersion(ctx, versionID)
}

// ConsentVersionInput is the wire shape a version is created from.
// PublishedAt, PublishedBy, SupersededAt and CopiedMechanicsID are absent on
// purpose: they are written only by a publish, and accepting them would let a
// caller mint text already in force, naming a publisher who never published.
type ConsentVersionInput struct {
	ID         string                   `json:"id"`
	Scope      models.ConsentScope      `json:"scope"`
	Version    string                   `json:"version"`
	Language   models.Language          `json:"language"`
	Clauses    []models.Clause          `json:"clauses"`
	Effect     map[string]models.Effect `json:"effect"`
	Translated bool                     `json:"translated"`
	ReleaseRef string                   `json:"release_ref"`
}

func (m *CustodyManager) CreateConsentVersion(ctx context.Context, in ConsentVersionInput) (models.TextVersion, error) {
	return m.consent.CreateVersion(ctx, models.TextVersion{
		ID:         in.ID,
		Scope:      in.Scope,
		Version:    in.Version,
		Language:   in.Language,
		Clauses:    in.Clauses,
		Effect:     in.Effect,
		Translated: in.Translated,
		ReleaseRef: in.ReleaseRef,
	})
}

func (m *CustodyManager) PublishConsentVersion(ctx context.Context, versionID, publishedBy string) (models.TextVersion, error) {
	return m.consent.PublishVersion(ctx, versionID, publishedBy, models.Now())
}

// actLogJSON renders entries for the wire rather than tagging the domain
// struct: ActorID empty means the timer, a real state a client draws
// differently from a missing field, and a struct tag would make the two move
// together by accident.
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
