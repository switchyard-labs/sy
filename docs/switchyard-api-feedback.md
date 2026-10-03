# Switchyard API feedback (surfaced by `sy`)

Documented for the Switchyard team (Codex). `sy` is an API dogfood client;
these are contracts that made the CLI awkward, with suggested improvements.

| Endpoint | Problem | Why it matters to a CLI | Suggested improvement | Workaround |
| --- | --- | --- | --- | --- |
| `GET /api/repos` | Returns all repos including CP/PX fixtures; no owner/repo or visibility on most items; `registered_at` empty for unregistered | `sy repo list` can't distinguish demo from debris; owner column is the Artifacts namespace | Expose canonical owner/repo metadata for every repo; a `visibility` field; optionally filter fixtures | Filter by `registered` |
| `GET /api/repos/{name}` | No clone credential is returned; no token-minting endpoint exists | `sy repo clone` cannot obtain a scoped Git token | Add `POST /api/repos/{owner}/{repo}/clone-token` (short TTL, scope read/write) or return a token on `GET` | `SY_GIT_TOKEN` env |
| `GET /api/repositories` | Empty unless repos are explicitly registered; no self-registration | Canonical owner/repo namespace is optional, so `sy` falls back to the legacy flat API | Auto-register repos on create; document the registration contract |
| `GET /api/repos/{name}` | Throttled Artifacts calls surface as `repo_not_found` (404) | `sy` maps 404 to "not found" but the cause is rate limiting (429 upstream) | Distinguish upstream throttle/error from genuine not-found (e.g. 503 with retry-after) | Retry, or note `--host` infra load |
| `GET /api/attention` | Summary strings are sometimes raw internal state (e.g. `integration blocked: semantic_conflict`) | `sy attention` wants a human story: what happened, why me, options | Structured attention items with `what/why/options` fields |
| Auth | Cookie-session only; no PAT/device flow | `sy` (and agents) rely on cookies; no durable token for headless hosts | Add API token minting + device login |
| `POST /api/attempts/{id}/run` | Runs the deterministic implementer; no client hint of which runner | CLI needs to surface adapter/execution clearly | Return the runner/adapter in the run response |
