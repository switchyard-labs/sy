# Authentication

Switchyard currently exposes **cookie sessions** only (no durable personal
access tokens / device flow yet). `sy` stores the session token per host in a
0600 config file and sends it as the `switchyard_session` cookie.

## Login

```sh
sy auth login --host http://45.79.189.46
```

Prompts for username and password (password read without echo). Prefer this
over `--password` (which leaks into shell history). For scripts/agents:

```sh
printf 'alice\npassword123\n' | sy auth login --host http://45.79.189.46 --password-stdin
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
