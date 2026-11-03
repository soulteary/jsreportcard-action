# Changelog

## 1.1.1

- Raise total Go statement coverage to 97.75% with deterministic unit tests for success, failure and boundary cases.
- Enforce a 95% coverage floor across all production Go packages in CI, retaining real-analyzer integration checks.

## 1.1.0

- Add `badge_label` / `--badge-label` to select JS/TS, JS or TS badge labels without changing analysis or baseline compatibility.

## 1.0.1

- Capitalize report badge labels and refresh the README badge example.
- Capitalize displayed analysis modes and incomplete status without changing machine-readable report fields.

## 1.0.0

- Add the Go CLI, isolated Node toolkit and Ubuntu GitHub Action.
- Integrate ESLint, typescript-eslint, Prettier and TypeScript Compiler API diagnostics.
- Separate standard/typed analysis from builtin/project configuration trust.
- Follow tsconfig references and reject incomplete selected-source coverage.
- Generate JSON, Markdown and compact SVG reports with baseline-aware quality gates.
- Add real JS/TS/TSX, module-system, isolation and monorepo tests.
- Publish six-platform binary archives with locked analyzer toolkit manifests.
