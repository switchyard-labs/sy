# Authentication

Switchyard currently exposes **cookie sessions** only (no durable personal
access tokens / device flow yet). `sy` stores the session token per host in a
0600 config file and retains the session cookie returned by that host. HTTPS production and local
HTTP development are supported; server-side Secure/host-only policy is preserved.

## Login

```sh
sy auth login --host https://switchyard.cx
```

Prompts for username and password (password read without echo). Prefer this
over `--password` (which leaks into shell history). For scripts/agents:

```sh
# Feed username then password from a secret manager, not a literal password.
sy auth login --host https://switchyard.cx --password-stdin
```

## Status / switch / logout

```sh
sy auth status            # active host + user
sy auth status --json
sy auth switch <host>     # multiple self-hosted instances
sy auth logout
```

## Future requirement

Switchyard should add a durable **API token / device login** flow so `sy` (and
agents) do not rely on cookie sessions. Documented for the Switchyard team in
`docs/switchyard-api-feedback.md`.
