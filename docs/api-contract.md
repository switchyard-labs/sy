# Switchyard API contract survey (SY0)

Recorded against the live instance (`http://45.79.189.46`) and the current
`switchyard` source (read-only). This is the contract `sy` is built against.

## Auth
- `POST /api/auth/register`, `POST /api/auth/login` (returns `switchyard_session`
  cookie), `POST /api/auth/logout`, `GET /api/auth/me` → `{authed, user}`.
- Cookie-session only (no PAT/device flow yet).

## Repositories
- Legacy flat: `GET /api/repos` → `{items:[{name, default_branch, remote,
  read_only, registered, registered_at}]}` (73 repos incl. demo + CP fixtures).
- `GET /api/repos/{name}` → `{name, default_branch, remote, source, read_only}`.
- Canonical owner/repo: `GET /api/repositories`, `POST /api/repositories/register`,
  `GET /api/repositories/{owner}/{repo}[/tree|content|refs|overview|commits|
  compare|settings]`. Empty until repos are registered.

## Work
- `GET /api/work`, `POST /api/work {title, kind}`, `GET/PATCH /api/work/{id}`,
  `GET/POST /api/work/{id}/comments`.
- Work detail includes `attempts[]` and `pull_requests[]`.

## Attempts / PRs
- `POST /api/work/{id}/attempts`, `POST /api/attempts/{id}/run|pr|review|preview|resolve`.
- `GET /api/prs`, `GET /api/prs/{id}`, `POST /api/prs/{id}/check|integrate|enqueue`.

## Integration Queue / Attention
- `GET /api/queue`, `POST /api/queue/{id}/requeue`.
- `GET /api/attention` → `{count, items:[{kind, target_id, repo, branch,
  summary, created_at}]}`.
- `GET /api/escalations`, `POST /api/escalations/{id}/decide`,
  `POST /api/attention/{kind}/{id}/escalate`.

## Workflows / Agents
- `GET/POST /api/workflows`, `POST /api/workflows/{id}/run`,
  `GET /api/workflow_runs`, `GET/POST /api/workflow_runs/{id}`.
- `GET /api/roles` → `{items:[{name, capabilities, profile}]}`,
  `GET /api/executions`.

## Orgs
- `GET/POST /api/orgs`, `GET /api/orgs/{id}`, repositories, members, teams,
  invitations, policies, audit, fleet.

## Observations
- Throttled Cloudflare Artifacts calls surface as `repo_not_found` (404) even
  when the cause is upstream 429 rate-limiting — a server mapping that misleads
  clients (see `docs/switchyard-api-feedback.md`).
- Canonical owner/repo metadata is optional (registration); the legacy flat API
  is the reliable source for repo discovery today.