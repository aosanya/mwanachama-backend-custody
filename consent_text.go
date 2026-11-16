package mwanachamacustody

import (
	"context"
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

const (
	MaxConsentTextLength   = 2000
	maxConsentTextAttempts = 20
	consentTextBackoffBase = 2 * time.Millisecond
	consentTextBackoffCap  = 50 * time.Millisecond
	consentTextClauseKey   = "consent"
)

type ConsentVersionRow struct {
	models.TextVersion
	InForce bool `json:"in_force"`
}

type ConsentTextInput struct {
	Text     string              `json:"text"`
	Scope    models.ConsentScope `json:"scope"`
	Language models.Language     `json:"language"`
}

func (m *CustodyManager) ListConsentVersions(ctx context.Context, scope, language string) ([]ConsentVersionRow, error) {
	sheetScope, err := consentScopeOrDefault(scope)
	if err != nil {
		return nil, err
	}
	versions, err := m.consent.ListVersions(ctx, sheetScope, consentLanguageOrDefault(language))
	if err != nil {
		return nil, err
	}
	rows := make([]ConsentVersionRow, 0, len(versions))
	for _, v := range versions {
		v.PublishedBy = ""
		rows = append(rows, ConsentVersionRow{TextVersion: v, InForce: v.PublishedAt != nil && v.SupersededAt == nil})
	}
	return rows, nil
}

func (m *CustodyManager) GetInForceConsent(ctx context.Context, scope, language string) (models.TextVersion, error) {
	sheetScope, err := consentScopeOrDefault(scope)
	if err != nil {
		return models.TextVersion{}, err
	}
	return m.GetConsentVersionInForce(ctx, string(sheetScope), string(consentLanguageOrDefault(language)))
}

func (m *CustodyManager) PublishConsentText(ctx context.Context, in ConsentTextInput, publishedBy string) (models.TextVersion, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return models.TextVersion{}, invalidValue("consent text: text is required")
	}
	if utf8.RuneCountInString(text) > MaxConsentTextLength {
		return models.TextVersion{}, invalidValue("consent text: text is longer than %d characters", MaxConsentTextLength)
	}
	scope, err := consentScopeOrDefault(string(in.Scope))
	if err != nil {
		return models.TextVersion{}, err
	}
	language := consentLanguageOrDefault(string(in.Language))
	m.publishText.Lock()
	defer m.publishText.Unlock()
	var draft models.TextVersion
	for attempt := 0; ; attempt++ {
		draft, err = m.draftNextConsentVersion(ctx, scope, language, text)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrConflict) || attempt+1 >= maxConsentTextAttempts {
			return models.TextVersion{}, err
		}
		if err := waitBeforeRetry(ctx, attempt); err != nil {
			return models.TextVersion{}, err
		}
	}
	published, err := m.consent.PublishVersion(ctx, draft.ID, publishedBy, models.Now())
	if err != nil {
		_ = m.consent.DeleteDraft(context.WithoutCancel(ctx), draft.ID)
		return models.TextVersion{}, err
	}
	published.PublishedBy = ""
	return published, nil
}

func waitBeforeRetry(ctx context.Context, attempt int) error {
	ceiling := consentTextBackoffBase << min(attempt, 6)
	if ceiling > consentTextBackoffCap {
		ceiling = consentTextBackoffCap
	}
	timer := time.NewTimer(rand.N(ceiling))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *CustodyManager) draftNextConsentVersion(ctx context.Context, scope models.ConsentScope, language models.Language, text string) (models.TextVersion, error) {
	existing, err := m.consent.ListVersions(ctx, scope, language)
	if err != nil {
		return models.TextVersion{}, err
	}
	return m.consent.CreateVersion(ctx, models.TextVersion{
		Scope:    scope,
		Version:  "v" + strconv.Itoa(highestVersionNumber(existing)+1),
		Language: language,
		Clauses:  []models.Clause{{Key: consentTextClauseKey, Text: text}},
	})
}

func highestVersionNumber(versions []models.TextVersion) int {
	highest := 0
	for _, v := range versions {
		n, err := strconv.Atoi(strings.TrimPrefix(v.Version, "v"))
		if err == nil && strings.HasPrefix(v.Version, "v") && n > highest {
			highest = n
		}
	}
	return highest
}

func consentScopeOrDefault(scope string) (models.ConsentScope, error) {
	switch models.ConsentScope(scope) {
	case "":
		return models.ScopeMechanics, nil
	case models.ScopeMechanics, models.ScopeAppendix, models.ScopePublicSheet:
		return models.ConsentScope(scope), nil
	}
	return "", invalidValue("consent: scope is %q, which is not one of mechanics, appendix, public_sheet", scope)
}

func consentLanguageOrDefault(language string) models.Language {
	if language == "" {
		return models.LanguageEnglish
	}
	return models.Language(language)
}
