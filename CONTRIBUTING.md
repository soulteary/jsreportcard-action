# Contributing

Use Go 1.23+ and Node 22+. Run `npm ci --prefix tools --ignore-scripts`, `make test`
and `make integration`. Keep the toolkit lockfile committed.

Keep worker execution, diagnostic normalization, scoring and rendering separate.
Builtin mode must not evaluate project JavaScript config or package scripts.
Any parser/configuration/coverage failure must remain incomplete. Add regression
fixtures for new language, module-system, workspace or compiler-version support.
