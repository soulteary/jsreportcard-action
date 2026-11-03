# Security

Builtin mode uses trusted toolkit packages and parses selected source; it does
not import project modules, run package scripts or load project JavaScript config.
NODE_OPTIONS and NODE_PATH are removed from analyzer subprocesses.

Project mode executes ESLint/Prettier configuration and plugins. Those are code,
not inert settings. Run untrusted pull requests with read-only credentials in
isolated CI jobs; never use pull_request_target to execute attacker-controlled
configuration with privileged credentials. Dependency installation and generated
build artifacts are explicit caller responsibilities.

Report suspected vulnerabilities privately using GitHub's repository security
reporting feature when enabled. Do not include secrets in public issues.
