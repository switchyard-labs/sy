# Agent usage

A coding agent should be able to drive Switchyard without scraping human
output:

```sh
sy auth status --json
sy repo list --json
sy repo view demo-basic --json
sy work list --json --status open
sy pr list --json
sy queue list --json --status blocked
sy attention --json
sy workflow runs --json
sy api GET /api/work
```

Rules for agents:

- For typed API commands, pass `--json` and read the result; never parse the human table.
  Local/auth commands have exceptions documented in [the JSON contract](json.md).
- Check exit codes: 3 = auth, 4 = conflict/stale, 5 = unavailable.
- The low-level `sy api <METHOD> <path> --body '...'` escape hatch covers
  endpoints `sy` does not wrap yet.
