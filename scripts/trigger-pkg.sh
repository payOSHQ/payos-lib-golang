#!/usr/bin/env bash
set -euo pipefail

# Trigger the Go proxy (and indirectly pkg.go.dev) to fetch a specific tagged version.
# Usage: scripts/trigger-pkg.sh <version-tag>

VERSION="${1:-}" 
if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version-tag>"
  exit 1
fi

cd "$(dirname "$0")/.."

if [ ! -f go.mod ]; then
  echo "No go.mod found in repo root; skipping pkg.go.dev trigger."
  exit 0
fi

MODULE=$(awk '/^module /{print $2; exit}' go.mod || true)
if [ -z "$MODULE" ]; then
  echo "Unable to determine module path from go.mod; skipping."
  exit 0
fi

export GOPROXY="https://proxy.golang.org,direct"
echo "Triggering proxy fetch for $MODULE@$VERSION"

# Use `go list -m -json` to resolve and download the requested version which triggers
# the module proxy to fetch the tag. We tolerate non-zero exit codes but still signal
# the attempt in logs.
if go list -m -json "${MODULE}@${VERSION}" >/dev/null 2>&1; then
  echo "Successfully triggered fetch for ${MODULE}@${VERSION}"
else
  echo "Trigger command completed (exit non-zero); proxy may still be fetching." >&2
fi

echo "pkg.go.dev URL: https://pkg.go.dev/${MODULE}"
