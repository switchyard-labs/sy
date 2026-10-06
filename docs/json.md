# `--json` contract for agents

`--json` is a stable, machine-readable contract. Presentation strings are never
included. Most typed commands' JSON output is the domain result (e.g. `items` arrays
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

Current exceptions: version still emits human text; config get/set-host emit no
JSON result; auth switch/logout print text, and browse opens a browser. Actions
logs use NDJSON; api returns the server body. Exit code 2 is reserved but not
emitted by the current mapper; raw API non-2xx errors return 1. See the
[verified inventory](command-inventory.md) before assuming a universal schema.
