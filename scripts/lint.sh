#!/usr/bin/env bash

set -e

cd "$(dirname "$0")/.."

echo "=> Running go build"
go build ./.

echo "=> Checking tests compile"
go test -run=^$ ./...