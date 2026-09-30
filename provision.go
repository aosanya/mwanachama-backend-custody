package mwanachamacustody

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func Provision(db *gorm.DB, s *spec.Spec) error {
	if err := createSequences(db, s); err != nil {
		return err
	}
	return spec.Migrate(db, s)
}

func createSequences(db *gorm.DB, s *spec.Spec) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	for _, role := range []string{RoleStructureAct, RoleEvent} {
		if _, ok := s.ByRole(role); !ok {
			continue
		}
		name := sequenceFor(role)
		if err := db.Exec("CREATE SEQUENCE IF NOT EXISTS " + name).Error; err != nil {
			return fmt.Errorf("custody: provision: %s: %w", name, err)
		}
	}
	return nil
}
