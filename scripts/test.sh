#!/usr/bin/env bash

set -e

cd "$(dirname "$0")/.."

echo "=> Running test"
go test ./... "$@"