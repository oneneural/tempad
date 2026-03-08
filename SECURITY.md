# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in TEMPAD, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, email **hello@oneneural.ai** with:

1. A description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## Scope

TEMPAD runs locally on your machine. Security-relevant areas include:

- **API key handling** — Linear API keys are stored in config files and environment variables. TEMPAD never transmits keys to any endpoint other than the configured tracker API.
- **Workspace isolation** — Path traversal is prevented via `filepath.Rel()` containment checks.
- **Hook execution** — Shell hooks run with the user's permissions via `bash -lc`. Only execute hooks from trusted WORKFLOW.md files.
- **HTTP server** — When enabled (`--port`), the server binds to `127.0.0.1` only (loopback). It is not accessible from the network.
- **Process groups** — Agent subprocesses run in isolated process groups for clean termination.

## Best Practices

- Use `$ENV_VAR` references for API keys instead of hardcoding them in config files
- Review WORKFLOW.md hook scripts before running TEMPAD on untrusted repositories
- Keep TEMPAD and its dependencies up to date
