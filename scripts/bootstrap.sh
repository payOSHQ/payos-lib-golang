#!/usr/bin/env bash

set -e

cd "$(dirname "$0")/.."

if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go first."
    exit 1
fi

echo "=> Install dependencies"
go mod tidy -e

