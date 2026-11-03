# JavaScript / TypeScript Report Card

[![JS/TS Report Card example](.github/jsreportcard-example/badge.svg)](.github/jsreportcard-example/report.md)
[![CI](https://github.com/soulteary/jsreportcard-action/actions/workflows/ci.yml/badge.svg)](https://github.com/soulteary/jsreportcard-action/actions/workflows/ci.yml)

A **Go CLI + Node.js analyzer + GitHub Action** for local JavaScript and TypeScript
quality reports. Uses ESLint, typescript-eslint, Prettier and the TypeScript
Compiler API to generate JSON, Markdown and self-contained SVG badges. No hosted
analysis service is required. The badge above comes from the actual bundled
[TypeScript example](testdata/clean), not a rating of this Go implementation.

## GitHub Action

```yaml
name: JS/TS quality
on: [push, pull_request, workflow_dispatch]
permissions:
  contents: read
jobs:
  report:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v6
      - uses: soulteary/jsreportcard-action@v1
        with:
          mode: standard
          include: src
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: jsreportcard
          path: .jsreportcard/
          include-hidden-files: true
```

The Action installs its pinned toolkit into a temporary directory using `npm ci
--ignore-scripts`, builds the Go CLI, then analyzes the checkout. It does not
install your project or execute its build/tests. Reports and the GitHub Step
Summary are written before quality gates are enforced. `commit` defaults to
`false`; opt-in report commits require `contents: write` and only run on branch
push events, never pull requests.

For typed analysis, prepare dependencies and generated declarations yourself:

```yaml
- run: npm ci --ignore-scripts
- uses: soulteary/jsreportcard-action@v1
  with:
    mode: typed
    include: src
    tsconfigs: tsconfig.json
```

Projects requiring install scripts, code generation or dependent-package builds
must run those steps explicitly in an appropriate trusted workflow. See the
[copyable example](examples/jsreportcard.yml).

### Inputs and outputs

- `directory`: project root relative to workspace; default `.`.
- `mode`: `standard` (default) or `typed`.
- `badge_label`: `js-ts` (default), `js`, or `ts`; display label only.
- `config_mode`: `builtin` (default) or `project`.
- `node_version`: analyzer runtime; default `22`.
- `include`: optional comma-separated source directories/files; defaults to JSON
  configuration, or the project root when not configured.
- `tsconfigs`: optional comma-separated tsconfig paths; defaults to configuration,
  or `tsconfig.json`.
- `config`: strict JSON configuration path relative to the project.
- `output_dir`: report directory; default `.jsreportcard` relative to the project.
- `baseline`: previous complete compatible JSON report.
- `fail_on`: `none`, `errors`, `new-errors` (default), or `findings`.
- `threshold`: minimum score; `0` disables the score gate.
- `commit`: commit the three report files on branch push events; default `false`.

Outputs: `status`, `grade`, `score`, `analysis_complete`, `new_errors`, `report`,
`json`, `badge`. Paths are absolute; incomplete analysis has an empty score.
Invalid invocations can fail before producing reports. Action mode/config-mode
inputs override JSON values. Pin an exact version or commit for repeatability.

### Badge labels

Choose `badge_label: js-ts`, `badge_label: js`, or `badge_label: ts` to display
**JS/TS Report**, **JS Report**, or **TS Report**. Analysis mode and incomplete
status remain visible. This option only changes the label; it does not filter
source files, change scores or invalidate baselines. The selected value is
recorded as `badge_label` in `report.json`.

For example, use `badge_label: ts` with your TypeScript project:

```yaml
- uses: soulteary/jsreportcard-action@main
  with:
    mode: typed
    badge_label: ts
    include: src
```

To show separate JS and TS grades, run the Action twice with the corresponding
source directories and distinct output directories. For a repository with
`javascript/` and `typescript/` directories:

```yaml
- uses: soulteary/jsreportcard-action@main
  with:
    badge_label: js
    include: javascript
    output_dir: .js-report
- uses: soulteary/jsreportcard-action@main
  with:
    mode: typed
    badge_label: ts
    include: typescript
    tsconfigs: typescript/tsconfig.json
    output_dir: .ts-report
```

Prepare TypeScript dependencies as described above. Commit or publish both SVG
files to display them in your README. Each grade reflects that run's `include`
scope; selecting a label alone does not produce a language-specific grade.
This option is currently available on `main`; it will be included in the next release.

## CLI

Build with Go 1.23+, or extract a prebuilt release archive. The archive includes
`tools/package.json` and `tools/package-lock.json`; install those dependencies
next to the binary before use:

```sh
npm ci --prefix tools --ignore-scripts
./jsreportcard-cli --directory /path/to/project --include src
./jsreportcard-cli --directory /path/to/project --mode typed --include src --badge-label ts
```

From a source checkout:

```sh
npm ci --prefix tools --ignore-scripts
go build -o dist/jsreportcard-cli ./cmd/jsreportcard-cli
./dist/jsreportcard-cli --directory testdata/clean --mode typed \
  --tool-dir "$PWD/tools" --json
```

`--tool-dir` is relative to the project root unless absolute. If omitted, the CLI
looks only for the packaged `tools` next to its executable. It never silently
uses the analyzed project's own toolkit directory. Supply `--tool-dir` explicitly
when running from a source checkout. Node must be installed separately.

Exit codes: **0** complete and policy passed; **1** quality gate failed;
**2** incomplete analysis, invalid input or execution failure. `--json` changes
stdout only. `--fail-on none` does not hide execution errors or disable an
explicit nonzero `--threshold`.

The Action supports Ubuntu. CLI archives cover Linux/macOS/Windows on amd64/arm64;
Windows users should run `jsreportcard-cli.exe` and install the same toolkit.

## Profiles and trust

Analysis depth and configuration source are independent:

- **standard:** syntax-level ESLint, formatting and cyclomatic complexity.
- **typed:** standard checks plus TypeScript diagnostics and selected type-aware
  rules for promises and await.
- **builtin:** fixed toolkit rules; does not load project ESLint or Prettier JS
  configuration, plugins or NODE_OPTIONS/NODE_PATH preloads.
- **project:** explicitly loads the project's flat ESLint configuration and
  Prettier configuration/plugins. These files execute JavaScript and must be
  trusted. This is environment separation, not a sandbox for hostile configs.

Project dependencies and analyzer dependencies are separate. Initial validation
uses ESLint 9.39.1, typescript-eslint 8.46.2, TypeScript 5.9.3 and Prettier 3.6.2.
The lockfile fixes transitive toolkit dependencies. Project typed mode requires
its installed TypeScript version to match the toolkit version; mismatches fail
explicitly. Legacy `.eslintrc` configurations are not automatically migrated.

Builtin ESLint uses JS recommended rules, TS syntax parsing, JSX reference rules
and a complexity limit (default 10). TS files disable core `no-undef` and use the
TypeScript-aware unused-variable rule. `environment` can be `node`, `browser` or
`universal` (default). The latter permits globals from both environments; choose
one explicitly for stricter platform checks. Inline suppressions are honored and
counted. Project mode preserves native ESLint error/warning levels.

Typed ESLint uses explicit project paths to support custom tsconfig names. The
TypeScript Compiler API follows `extends` and `references`, checks every source
project, and never invokes `emit` or a build command. It does not create JS,
declarations or `.tsbuildinfo` files. Referenced projects may still require
prebuilt declarations; prepare them before running the action.

Supported source extensions: `.js`, `.jsx`, `.mjs`, `.cjs`, `.ts`, `.tsx`, `.mts`,
`.cts`. Declaration files are used as type context but are not standalone report
source units. Vue/Svelte/Astro processors, Yarn PnP and framework build pipelines
are not supported automatically in v1. npm and pnpm projects with ordinary
`node_modules` can supply their installed dependencies.

## Scope and completeness

Configure with [.jsreportcard.example.json](.jsreportcard.example.json). Includes
and exclusions are literal paths/components, not globs. Source symlinks are not
traversed; generated directories and `node_modules` are excluded by default.

The report lists actual analyzed files and unit counts. Typed mode requires every
selected source file to appear in a TypeScript Program; files outside tsconfigs,
a root solution with no source projects, missing dependencies, parser failures,
or configuration failures yield **incomplete**, never A+. JS files in typed mode
need `allowJs` plus `checkJs` (or a file-level `@ts-check`). Files with
`@ts-nocheck` and programs with `noCheck` cannot count as type-checked. Errors outside the selected
source (including declarations) also prevent a misleading complete result.

Align `include` with the source area you want to check. In project mode, selected
files ignored by native lint/format configuration cause an explicit completeness
error; they do not silently count as passing. Libraries, test folders and build
scripts may need different project configurations.

## Grades and baselines

Scoring v1 is a severity ceiling, not a clean-file percentage:

- No findings: 100 / A+.
- Formatting only: at most 90 / A.
- Other lint warnings: at most 85 / B.
- Complexity warnings: at most 80 / B.
- Errors: at most 59 / F.

Builtin mode classifies known correctness errors (such as undefined names,
invalid assignments, unreachable statements and unhandled promises) separately
from style warnings. Original tool severity is preserved as `tool_severity`.
TypeScript errors fail the gate. This coarse grade is a summary, not a security
certification or ranking across different configurations. Tools keep their
sources; repeated findings cannot reduce the score more than their category cap.

```sh
./jsreportcard-cli --tool-dir /path/to/tools --baseline baseline.json
```

Baselines must match the mode/configuration, exact Node/tool versions, toolkit
lockfile, project manifests/lockfiles and detected native config contents.
Fingerprints use rule, severity, file, message and normalized source line;
occurrence counts catch a second identical finding. Line shifts alone are
usually tolerated. Code and commit IDs are not included in the compatibility
hash, so normal edits can be compared. Renames or source-line changes can count
as new findings. Baselines are not applied to incomplete runs. Arbitrary JS configs can depend on environment or imported
helpers; keep those stable when comparing project-mode baselines.

## Development and release

```sh
make test
npm ci --prefix tools --ignore-scripts
make integration
scripts/update-badge-example.sh
```

CI covers clean JS/TS/TSX, ESM/CJS, formatting, syntax/type errors, missing
packages, source coverage, baseline duplicates, config execution isolation,
preload isolation and monorepo references. The README badge is generated from a
real typed fixture and uploaded as a CI artifact.

Tags `vX.Y.Z` run full CI and create a release draft with six binary/toolkit
archives and SHA-256 checksums. Publish it with the Marketplace option, then
promote `v1`. See [.github/MARKETPLACE.md](.github/MARKETPLACE.md).

Knip, dependency vulnerability audits, coverage imports and alternate engines
(Biome/Oxlint) are reserved for later adapters, not silently included in v1 scores.

Apache-2.0; see [LICENSE](LICENSE). Report/badge design is shared with
[Go Report Card](https://github.com/soulteary/goreportcard-action),
[C++ Report Card](https://github.com/soulteary/cppreportcard-action) and
[Python Report Card](https://github.com/soulteary/pyreportcard-action).
