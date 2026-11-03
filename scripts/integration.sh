#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cd "$repo"
go build -o "$tmp/jsreportcard-cli" ./cmd/jsreportcard-cli
node scripts/integration.cjs "$tmp/jsreportcard-cli" "$repo/tools"
