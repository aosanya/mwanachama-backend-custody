#!/usr/bin/env bash
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
hub="${1:-$repo/../projects/hub-steward/code/wakala-api/modules/custody}"
module="${2:-github.com/aosanya/mwanachama-backend-custody}"
shared="${3:-github.com/aosanya/mwanachama-backend-shared}"
hub_module="${4:-wakalaserver/modules/custody}"
hub_shared="${5:-wakalaserver/modules/shared}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

copy_tree() {
  local from="$1" to="$2"
  mkdir -p "$to"
  (cd "$from" && find . \( -name '*.go' -o -name '*.json' \) \
    -not -path './cmd/*' -not -path './documentation/*' -not -path './scripts/*' \
    -not -path './.git/*' -print0 | tar -cf - --null -T -) | tar -xf - -C "$to"
}

copy_tree "$repo" "$work/backend"
copy_tree "$hub" "$work/hub"

find "$work/hub" -name '*.go' -print0 | HUB_SHARED="$hub_shared" SHARED="$shared" HUB_MODULE="$hub_module" MODULE="$module" xargs -0 perl -pi -e '
  s{"\Q$ENV{HUB_SHARED}\E}{"$ENV{SHARED}}g;
  s{"\Q$ENV{HUB_MODULE}\E}{"$ENV{MODULE}}g;
  s{^package custody}{package mwanachamacustody}m;
  s{\bcustody "github}{mwanachamacustody "github}g;
  s{(?<![.\w])custody\.(?=[A-Z])}{mwanachamacustody.}g;
  s{"custody: }{"mwanachamacustody: }g;
  s{`custody: }{`mwanachamacustody: }g;
'

(cd "$repo" && go run ./scripts/hubdiff "$work/backend")
(cd "$repo" && go run ./scripts/hubdiff "$work/hub")

cd "$work"
diff -r backend hub | sed "s#^diff -r backend/#diff #"
