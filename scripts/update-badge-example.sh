#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo"
go run ./cmd/jsreportcard-cli --directory testdata/clean --mode typed \
  --tool-dir "$repo/tools" --output-dir "$repo/.github/jsreportcard-example" --fail-on findings
