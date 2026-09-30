package mwanachamacustody

import (
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func isInstant(s string) bool {
	_, err := models.ParseTime(s)
	return err == nil
}

var patterns = map[string]func(string) bool{
	"instant": isInstant,
}
