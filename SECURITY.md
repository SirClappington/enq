# Security Policy

## Supported Versions

We actively support security fixes on the latest release line.

Version	Supported
main	✅
Latest vX.Y.Z	✅
Older releases	❌

If you’re pinned to an older version, please upgrade to the latest patch/minor as soon as possible.

## Reporting a Vulnerability

Please do not open public GitHub issues for security problems.

Use one of these private channels:

GitHub Private Advisory: Security → Advisories → Create security advisory

Include (when possible):

A clear description of the issue and impact

A minimal PoC (commands, config, or API calls)

Affected version(s) and environment (OS/arch, Docker/Go versions, DB version)

Any relevant logs (redact secrets)

We’ll acknowledge your report within 3 business days.

## Coordinated Disclosure

Our standard process:

1. Triage (≤3 business days). Confirm the issue and assess severity (CVSS-like).

2. Fix window (typically 7–21 days). We prepare a patch, tests, and release notes. Complex issues may need more time; we’ll keep you updated.

3. Release. We publish a patched release and backport to the supported line(s).

4. Disclosure. We credit reporters (optional) and publish details once users have a reasonable upgrade window.

If you require an embargo period or have a coordinated release timeline, tell us up front.

## Scope

In scope:

- enq-api (HTTP service)

- enq-scheduler (workers)

- Repository build/CI artifacts that ship these binaries (Docker images)

## Out of scope:

- Third-party services and managed databases

- General DoS via resource exhaustion in small dev/test environments

- Vulnerabilities requiring privileged local access or unsupported configs

## What We Consider Security-Relevant

- AuthN/AuthZ bypasses (when/if introduced)

- Job tampering, execution escalation, or lease/locking bypass

- Persistent injection in logs/metrics that leads to code execution

- Supply-chain risks in published containers (e.g., malicious files, leaked secrets)

- Sensitive info exposure (credentials, tokens) via logs or API responses

- Protocol & API request smuggling that causes unintended job runs

Not security issues by themselves:

- Lack of rate limiting or global throttling knobs

- Non-default, unsafe deployments (e.g., exposing DB directly to the internet)

- Best-practice hardening requests (we still welcome them as enhancements)

## Submitting a PoC Safely

- Provide minimal steps and sanitized data.

- If sharing container images, do not include real secrets.

- For logs, scrub tokens, connection strings, and IDs.

- If the PoC requires a public demo server, coordinate a short-lived test window.

## Dependencies & Supply Chain

- We track dependencies via Go modules and container base images.

- If you found a vulnerability in a dependency that affects Enq:

  - Report it here and upstream if known.

  - Include the vulnerable package + version and a link to the upstream advisory if available.

- Container images include OCI metadata; we aim to publish SBOMs in future releases.

## Hardening Guidance (Quick Notes)

- Run Enq behind a reverse proxy or API gateway.

- Use network policies/security groups to restrict DB access to Enq only.

- Set strong credentials for the database; prefer TLS where available.

- Keep images updated; prefer pinned tags (e.g., vX.Y.Z) over latest.

- Limit container privileges (no root if possible) and use read-only FS where you can.

# Credits & Bounties

We currently do not run a paid bug bounty. With consent, we will credit reporters in release notes and SECURITY.md.