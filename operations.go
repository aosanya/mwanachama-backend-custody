package mwanachamacustody

import _ "embed"

//go:embed custody.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
