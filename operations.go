package mwanachamacustody

import (
	_ "embed"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

//go:embed custody.operations.json
var operationsJSON []byte

var loadOperations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(operationsJSON)
})

func Operations() (*dispatch.Spec, error) { return loadOperations() }
