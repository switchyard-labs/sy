# sy ↔ Switchyard server compatibility

| Feature | Status | Switchyard API |
| --- | --- | --- |
| repo list | supported | `GET /api/repos` |
| repo view | supported | `GET /api/repos/{name}` (+ canonical metadata when registered) |
| repo clone (remote) | supported | `GET /api/repos/{name}` → remote |
| repo clone (token) | **server gap** | no clone-token endpoint (see api-feedback) |
| work list/view/create/close | supported | `/api/work…` |
| work comments | supported | `GET/POST /api/work/{id}/comments` |
| attempt list | supported (derived) | work detail `attempts[]` (list omits attempts) |
| attempt view | supported (derived) | work detail scan |
| attempt run | supported | `POST /api/attempts/{id}/run` |
| pr list/view/checks | supported | `/api/prs…` |
| pr findings | supported | `GET /api/findings` |
| queue list/view/requeue | supported | `/api/queue…` |
| attention | supported | `GET /api/attention` |
| workflow list/view/run/runs | supported | `/api/workflows…`, `/api/workflow_runs…` |
| agent roles/executions | supported | `/api/roles`, `/api/executions` |
| org list/view | supported | `/api/orgs…` |
| api escape hatch | supported | any authenticated endpoint |
| Actions / CI | **waiting on server API** | no `actions*` routes yet |
| durable API tokens | **waiting on server API** | cookie sessions only |

Documented gaps: `docs/switchyard-api-feedback.md`.
