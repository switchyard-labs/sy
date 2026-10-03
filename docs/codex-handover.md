# sy — Codex handover / reviewer map

Independent review entry point for the `sy` Switchyard CLI. Read this, the
README, and `docs/` before judging. Treat prior `sy` work as fallible.

## State

- Current SHA: see `git rev-parse HEAD`. October 4 review commits are local and unpushed.
- Remote: `git@github.com:switchyard-labs/sy.git` (`main`).
- No Switchyard / brochure / Linode / reference-project changes were made by
  this project. `sy` is a client of the Switchyard API only.

## Command inventory

```
auth        login status logout switch
repo        list view clone
actions     list view run cancel rerun logs (--follow)
work        list view create comment close
attempt     list view run
pr          list view checks findings
queue       list view requeue
attention   (list) view
workflow    list view run runs
agent       roles executions execution
org         list view
api         [METHOD] path --body
status browse config( list get set-host )
version completion( bash zsh fish )
```

## Architecture

- `cmd/sy` — thin entry (exit-code mapping, auth-expiry hint, ldflags).
- `internal/commands` — Cobra commands; each produces a domain result rendered
  by the shared renderer (human table / stable `--json`).
- `internal/api` — typed HTTP client: cookie auth, JSON envelopes, request IDs,
  typed errors (`ErrUnauthorized/Forbidden/NotFound/Conflict/Stale/Unavailable`),
  transient GET/HEAD retry (429/502/503/504, bounded; mutations never retried).
- `internal/config` — XDG config, 0600, `hosts[]` + `active_host`,
  `SY_CONFIG_DIR` override.
- `internal/output` — human/JSON renderers, TTY-aware color/status symbols.
- `internal/gitutil` — safe git subprocesses, owner/repo detection, credential
  hygiene.

## Auth/config model

- Cookie sessions only (Switchyard has no PAT/device flow yet). Login stores the
  session token per host in the 0600 config; token never printed.
- `--host` override, `auth switch`, `config get` shows `•••••` for the token.
- 401 surfaces `run: sy auth login --host <url>`.

## Credential-safety model

- `sy repo clone` resolves a registered canonical repository and requests a
  short-lived read credential from Switchyard. The exact HTTPS remote is validated;
  Git receives the capability through process environment configuration, never argv,
  a credential file, or `.git/config`. Credential helpers and redirects are disabled.
- No Artifacts token supplied by the user is required for normal cloning.

## Repo-context behavior

- `DetectOwnerRepo` handles Artifacts / GitHub / scp-style remotes; context is
  derived from `origin` (tests cover no-origin, malformed, Artifacts). No
  silent guessing on ambiguity.

## Exit codes (contract)

`0` success · `1` remote/operation · `2` invalid · `3` auth · `4` conflict/stale ·
`5` unavailable. `sy` never returns 0 on a failed API request.

## JSON guarantees

- `--json` emits pure, stable domain results on stdout; errors go to stderr;
  timestamps exact; booleans/numbers typed. Presentation strings never appear.

## Live dogfood result

October 4 isolated updated Switchyard: **11/11 PASS** using the existing integration
harness. A real scoped clone, checkout context, Actions list/detail/captured-log follow,
and direct Attempts listing also passed. This is isolated certification; the updated
client has not yet been certified against the final deployed Linode binary.

## Cross-build matrix (verified)

`linux/amd64` · `linux/arm64` · `darwin/amd64` · `darwin/arm64` · `windows/amd64`
via `scripts/build-release.sh` (ldflags version metadata; `CGO_ENABLED=0`).

## Remaining gaps

Actions, scoped clone credentials, top-level Attempts and typed upstream 429 errors
are implemented against the updated Switchyard contracts. PAT/device auth remains
unavailable. See `docs/codex-review.md` for the independent verdict and practical
limitations, and Switchyard `docs/plan/sy-api-handoff.md` for server contracts.
The documented invalid-input exit code still needs consistent wiring across commands.
Cross-builds prove compilation, not native macOS/Windows execution.

## Integration / release instructions

- Live harness: `SY_PASSWORD=… scripts/integration.sh` (opt-in; never in unit tests).
- Release: `./scripts/build-release.sh` → `dist/`.
- Unit tests: `go test ./...`; static: `go vet ./...`.

## What to review (independently)

CLI architecture · Cobra command structure · API client · error mapping ·
JSON stability · exit-code contract · credential redaction · clone credential
safety · config atomicity/perms · repo remote parsing · cross-platform
assumptions · retry/idempotency semantics · noninteractive behavior · live
integration harness · release script · server API feedback.

**Central question:** does `sy` feel like a **domain-aware Switchyard CLI
comparable in quality to `gh`**, or are there places where it still feels like
`curl` wrapped in Cobra? Be specific either way. Do not solve server gaps
(Actions, clone-token, PAT auth, upstream 429 mapping, top-level Attempts, raw
attention, cookie-only auth) with compatibility hacks inside the CLI.