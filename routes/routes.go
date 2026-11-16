package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

type Route = httpwire.Route

type Mount = dispatch.Mount

var Sentinels = map[string]error{
	"ErrNotFound":            mwanachamacustody.ErrNotFound,
	"ErrConsentRequired":     mwanachamacustody.ErrConsentRequired,
	"ErrConsentWithdrawn":    mwanachamacustody.ErrConsentWithdrawn,
	"ErrAlreadyPublished":    mwanachamacustody.ErrAlreadyPublished,
	"ErrUnknownScope":        mwanachamacustody.ErrUnknownScope,
	"ErrUnknownActClass":     mwanachamacustody.ErrUnknownActClass,
	"ErrUnknownEventChip":    mwanachamacustody.ErrUnknownEventChip,
	"ErrInvalidLimit":        mwanachamacustody.ErrInvalidLimit,
	"ErrInvalidSince":        mwanachamacustody.ErrInvalidSince,
	"ErrStructureIDRequired": mwanachamacustody.ErrStructureIDRequired,
	"ErrNotSelf":             mwanachamacustody.ErrNotSelf,
	"ErrNoMechanicsInForce":  mwanachamacustody.ErrNoMechanicsInForce,
}

var AnonymousActions = []string{}

var Table = dispatch.NewTable(mwanachamacustody.Operations(), Sentinels, AnonymousActions...)

func Build(cm *mwanachamacustody.CustodyManager) ([]Route, error) { return Table.Build(cm, Mount{}) }

func BuildFor(cm *mwanachamacustody.CustodyManager, m Mount) ([]Route, error) {
	return Table.Build(cm, m)
}

func RoutesFor(cm *mwanachamacustody.CustodyManager, m Mount) []Route { return Table.Routes(cm, m) }

func Split(cm *mwanachamacustody.CustodyManager, m Mount) dispatch.Split { return Table.Split(cm, m) }
