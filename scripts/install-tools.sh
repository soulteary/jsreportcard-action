#!/usr/bin/env bash
set -euo pipefail
: "${RUNNER_TEMP:?}"
: "${GITHUB_OUTPUT:?}"
tool_dir="$(mktemp -d "$RUNNER_TEMP/jsreportcard-tools.XXXXXX")"
cp "$GITHUB_ACTION_PATH/tools/package.json" "$GITHUB_ACTION_PATH/tools/package-lock.json" "$tool_dir/"
npm ci --prefix "$tool_dir" --ignore-scripts --no-audit --no-fund
printf 'directory=%s\n' "$tool_dir" >> "$GITHUB_OUTPUT"
