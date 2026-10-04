# sy — Switchyard CLI

`sy` is the first-class command-line client for
[Switchyard](https://github.com/switchyard-labs/switchyard), the agent-native
Git collaboration platform. It is to Switchyard what `gh` is to GitHub:
pleasant for humans, deterministic for agents (`--json`), and a durable
dogfood client for the Switchyard API.

```
sy
 ↓
Switchyard API
 ↓
Switchyard control plane (Artifacts · Trestle · workers)
```

You never need to know Trestle collection ids, Cloudflare account ids, or
Artifacts namespace internals to use `sy`.

## Install / build

```sh
go build -o sy ./cmd/sy
```

## Quick start

```sh
# log in (prompts for password; never store it in shell history)
sy auth login --host http://45.79.189.46

# clone a repository (uses ordinary Git; token never lands in .git/config)
sy repo clone demo-basic

# from inside a Switchyard checkout, run commands without owner/repo
cd demo-basic
sy work list
sy pr list

# what needs you right now?
sy attention

# deterministic output for agents
sy work list --json
sy attention --json
```

## Commands

```
CORE
  auth        Authenticate with a Switchyard host
  repo        Work with repositories
  work        Work items (issues/tasks)
  pr          Pull requests
  attention   Items requiring human direction

AUTOMATION
  workflow    Durable engineering workflows
  agent       Agent roles and executions
  queue       Integration Queue

OTHER
  org         Organizations
  api         Make an authenticated API request (escape hatch)
  version     Print version
```

Run `sy <command> --help` for examples. See:

- [docs/auth.md](docs/auth.md) — authentication and host profiles
- [docs/config.md](docs/config.md) — configuration
- [docs/json.md](docs/json.md) — the `--json` contract for agents
- [docs/repository-context.md](docs/repository-context.md) — deriving owner/repo from Git
- [docs/agent-usage.md](docs/agent-usage.md) — using `sy` from a coding agent
- [docs/switchyard-api-feedback.md](docs/switchyard-api-feedback.md) — API gaps surfaced by `sy`

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | remote/operation failure |
| 2 | invalid invocation/input |
| 3 | authentication/authorization |
| 4 | conflict/stale state |
| 5 | infrastructure/unavailable |

`sy` never returns 0 when an API request failed.

## Design notes

- **Credentials**: session tokens are stored in a 0600 config file; Git
  credentials are supplied via a temporary `GIT_CONFIG_GLOBAL` file (never in
  argv, never in `.git/config`). See the credential-sanitization tests in
  `internal/gitutil`.
- **Output**: every command produces a domain result rendered by a shared
  renderer (human table / stable JSON). `--json` is a public contract.
- **Terminal**: color/spinners are TTY-only; `NO_COLOR`, `TERM=dumb`,
  `--no-color`, and `--json` force deterministic output.
## Actions and scoped clone

`sy repo clone alice/demo-basic` obtains a short-lived read credential from
Switchyard automatically. No Artifacts account token is required. Use canonical
owner/repo when names are ambiguous. Registered repository visibility comes
from Switchyard metadata.

```sh
sy actions list --repo alice/demo-basic
sy actions view RUN_ID --repo alice/demo-basic
sy actions run --repo alice/demo-basic --sha EXACT_SHA --ref refs/heads/main
sy actions rerun RUN_ID --repo alice/demo-basic --failed
sy actions cancel RUN_ID --repo alice/demo-basic
sy actions logs RUN_ID --repo alice/demo-basic --follow
```

Logs are bounded captured output; `--follow` stops after terminal completion.
Ctrl-C stops following only. `--json` logs is NDJSON (one event per line).
Detailed independent review and certification: [docs/codex-review.md](docs/codex-review.md).

Download source at a branch, tag or commit with `sy repo archive owner/repo --ref main --format zip --output source.zip` (or `--format tar.gz`). The command resolves the ref to an immutable commit first, verifies the server's commit header, and streams a bounded archive to a private temporary file. Existing output files are never overwritten. `--json` reports repository, requested ref, resolved commit, path, format and bytes.

Manage releases with `sy release --repo owner/repo list`, `view <tag>`,
`create <existing-tag> --title <title> --notes-file notes.md`, and `publish <tag>`.
Creation always produces a draft; publication verifies the tag still points to
its original commit and freezes the assets. `--prerelease` marks a prerelease.
Use `upload <tag> <file>` to attach a draft asset and
`download <tag> <asset-name> --output <path>` to retrieve it. Downloads verify
metadata size and SHA256 before publishing the local file and never overwrite
an existing path. All release commands support `--json`; upload mutations are
sent once and can be retried explicitly after checking the draft state.
