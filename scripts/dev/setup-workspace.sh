#!/usr/bin/env bash
set -euo pipefail

root="${KM_WS_ROOT:-}"
if [[ -z "${root}" ]]; then
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  root="$(cd "${here}/.." && pwd)"
fi

sdk="${root}/kilometers-plugins-sdk"
cli="${root}/kilometers-cli"
plugins="${root}/kilometers-cli-plugins"

for d in "${sdk}" "${cli}" "${plugins}"; do
  if [[ ! -d "${d}" ]]; then
    echo "Missing repo: ${d}" >&2
    echo "Set KM_WS_ROOT to a directory that contains all three repos, or place them under a common parent." >&2
    exit 1
  fi
done

cd "${root}"

if [[ ! -f go.work ]]; then
  go work init ./kilometers-plugins-sdk ./kilometers-cli ./kilometers-cli-plugins
fi

go work use ./kilometers-plugins-sdk ./kilometers-cli ./kilometers-cli-plugins
go work sync

echo "Workspace ready at: ${root}/go.work"
go env GOWORK


