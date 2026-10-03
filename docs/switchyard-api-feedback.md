# Current resolution (2026-10-04)

Clone credentials, Actions, corrected upstream429/Retry-After and top-level
paginated Attempts are implemented and consumed by the reviewed CLI.
PAT/device login remains absent. See `codex-review.md` and Switchyard
`docs/plan/sy-api-handoff.md` for current contracts. The original feedback below
is historical and must not be read as the current server capability list.

# Switchyard API feedback (surfaced by `sy`)

Documented for the Switchyard team (Codex). `sy` is an API dogfood client.
Findings are classified: **BLOCKS SY FEATURE / POOR DX / NICE API IMPROVEMENT /
SERVER BUG**.

## BLOCKS SY FEATURE
- **Clone-token endpoint missing** — `sy repo clone` can obtain the remote but
  not a scoped Git credential. Suggested: `POST /api/repos/{owner}/{repo}/clone-token`.
  Workaround: `SY_GIT_TOKEN`.
- **Durable API token / device login missing** — cookie sessions only; `sy`
  and headless agents rely on them. Suggested: token minting + device login.
- **Actions/CI API absent** — no `actions*` routes yet; `sy actions` is
  designed but unimplemented until the server API lands.

## SERVER BUG
- **Artifacts throttling mapped to 404** — upstream 429 rate limits surface as
  `repo_not_found` (404), which misleads clients into thinking the repo is
  missing. `sy` must NOT treat 404 as a rate-limit hint; this needs a server
  fix (e.g. 503 + Retry-After).

## POOR DX
- **`GET /api/repos` mixes fixtures + lacks owner/repo/visibility** — `sy repo
  list` shows the Artifacts namespace as owner; demo vs fixture debris is
  indistinguishable.
- **Canonical owner/repo metadata optional/empty** — repos must be explicitly
  registered; the legacy flat API is the only reliable source today.
- **Attention summaries too raw** — e.g. `integration blocked:
  semantic_conflict`; `sy` humanizes what it can, but the server should expose
  structured `what/why/options`.
- **Work list omits attempts** — `sy attempt list` must fetch per-work detail
  (bounded scan); the list should include attempts.
- **Run response lacks adapter/runner** — `POST /api/attempts/{id}/run` does not
  identify which runner executed.

## NICE API IMPROVEMENT
- `GET /api/attempts` and `GET /api/attempts/{id}` top-level endpoints.
- PR comments API (Work comments exist; PR comments do not).

| Endpoint | Problem | Why it matters to a CLI | Suggested improvement | Workaround |
| --- | --- | --- | --- | --- |
| `GET /api/repos` | Returns all repos including CP/PX fixtures; no owner/repo or visibility on most items; `registered_at` empty for unregistered | `sy repo list` can't distinguish demo from debris; owner column is the Artifacts namespace | Expose canonical owner/repo metadata for every repo; a `visibility` field; optionally filter fixtures | Filter by `registered` |
| `GET /api/repos/{name}` | No clone credential is returned; no token-minting endpoint exists | `sy repo clone` cannot obtain a scoped Git token | Add `POST /api/repos/{owner}/{repo}/clone-token` (short TTL, scope read/write) or return a token on `GET` | `SY_GIT_TOKEN` env |
| `GET /api/repositories` | Empty unless repos are explicitly registered; no self-registration | Canonical owner/repo namespace is optional, so `sy` falls back to the legacy flat API | Auto-register repos on create; document the registration contract |
| `GET /api/repos/{name}` | Throttled Artifacts calls surface as `repo_not_found` (404) | `sy` maps 404 to "not found" but the cause is rate limiting (429 upstream) | Distinguish upstream throttle/error from genuine not-found (e.g. 503 with retry-after) | Retry, or note `--host` infra load |
| `GET /api/attention` | Summary strings are sometimes raw internal state (e.g. `integration blocked: semantic_conflict`) | `sy attention` wants a human story: what happened, why me, options | Structured attention items with `what/why/options` fields |
| Auth | Cookie-session only; no PAT/device flow | `sy` (and agents) rely on cookies; no durable token for headless hosts | Add API token minting + device login |
| `POST /api/attempts/{id}/run` | Runs the deterministic implementer; no client hint of which runner | CLI needs to surface adapter/execution clearly | Return the runner/adapter in the run response |
