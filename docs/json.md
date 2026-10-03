# `--json` contract for agents

`--json` is a stable, machine-readable contract. Presentation strings are never
included. Every command's JSON output is the domain result (e.g. `items` arrays
for lists, the item object for views).

```sh
sy repo list --json
sy work list --json --status open
sy pr list --json
sy queue list --json --status blocked
sy attention --json
sy workflow runs --json
sy agent executions --json
```

Exit codes are stable (see README): an agent can rely on non-zero exit meaning
the operation failed, and code 4 meaning conflict/stale.
