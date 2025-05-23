#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

if grep -r -l -E '(:|-) \{\{\s*[^}]+\s*\}\}$$' --include '*.yaml' --include '*.yml'; then
  sed -i -E "s/(:|-) \{\{\s*([^}]+?)\s*\}\}$/\1 \'{{ \2 }}\'/g" $( grep -r -l -E '(:|-) \{\{\s*[^}]+\s*\}\}$' --include '*.yaml' --include '*.yml' )
  if grep -r -l -E '(:|-) \{\{\s*[^}]+\s*\}\}$' --include '*.yaml' --include '*.yml'; then
    # ([^}]+?) should make the group non-greedy, and this works in other regexp implementations, but not for sed...
    # so, we can just squash trailing spaces explicitly so we don't add extra ones on every round-trip
    sed -i -E "s/\s+\}\}$/ }}/g" $( grep -r -l -E '(:|-) \{\{\s*[^}]+\s*\}\}$' --include '*.yaml' --include '*.yml' )
  fi
fi