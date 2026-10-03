# sy — Codex handover / reviewer map

Independent review entry point for the `sy` Switchyard CLI. Read this, the
README, and `docs/` before judging. Treat prior `sy` work as fallible.

## State

- Current SHA: see `git rev-parse HEAD` (pushed; HEAD == origin/main).
- Remote: `git@github.com:switchyard-labs/sy.git` (`main`).
- No Switchyard / brochure / Linode / reference-project changes were made by
  this project. `sy` is a client of the Switchyard API only.

## Command inventory

```
auth        login status logout switch
repo        list view clone
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

- `sy repo clone` fetches the remote from the API, then supplies any Git
  credential via a **temporary `GIT_CONFIG_GLOBAL` file (0600)**: token never in
  argv and never in `.git/config` (proven by `TestCredentialNotInGitConfig`).
- Fallback sources `SY_GIT_TOKEN` / `--token` are marked development/advanced;
  the final UX is a server clone-token endpoint (blocked).

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

Integration harness (`scripts/integration.sh`, client-only, `sy-dogfood-*`):
**10/11 PASS**. The one failure — `repo list` intermittently — is the **live
Switchyard server being rate-limited upstream by Cloudflare Artifacts** (the
server maps upstream 429 to misleading 404/502; its own logs confirm). `sy`
correctly records this rather than teaching the CLI that 404/502 mean
"rate-limited". Codex should review the server's upstream error mapping.

## Cross-build matrix (verified)

`linux/amd64` · `linux/arm64` · `darwin/amd64` · `darwin/arm64` · `windows/amd64`
via `scripts/build-release.sh` (ldflags version metadata; `CGO_ENABLED=0`).

## Known server blockers (do NOT "fix" inside `sy`)

| Blocker | Status | Pointer |
| --- | --- | --- |
| Actions/CI API | waiting on Switchyard routes | `docs/switchyard-api-feedback.md` |
| clone-token endpoint | server gap | `docs/switchyard-api-feedback.md` |
| durable PAT / device auth | server gap | `docs/switchyard-api-feedback.md` |
| clean top-level Attempts listing | server gap (work list omits attempts) | `docs/switchyard-api-feedback.md` |
| Artifacts throttle → misleading 404/502 | SERVER BUG | `docs/switchyard-api-feedback.md` |

Full categorized feedback: `docs/switchyard-api-feedback.md`.
Server compatibility: `docs/server-compatibility.md`.

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