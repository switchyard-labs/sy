# Verified sy command inventory — 2026-10-06

Audited against runtime source checkpoint `28947e5`, built help and command source.
Compatibility fix is the only implementation change in this documentation pass.
See [public CLI guide](https://docs.switchyard.cx/docs/cli.html).

All network commands require a configured host. Valid authentication and server
permissions are required for protected reads and mutations; public repository
reads can use a token-free configured profile. Config/switch/browse are local,
but current bootstrap requires an active configured profile. Help/version do not.

| Group | Subcommands | Purpose / mutation | JSON |
| --- | --- | --- | --- |
| auth | login, status, logout, switch | Login/session and local profile changes; status is read-only. | login/status only |
| repo | list, view, clone, archive | Repository reads; clone/archive write local files. | Yes |
| work | list, view, create, comment, close | Work reads and mutations. create currently sends title/kind only; body flags are not transmitted. | Yes |
| attempt | list, view, run | Reads; run starts implementer execution on an existing Attempt. | Yes |
| pr | list, view, checks, findings | PR inspection only; no create/enqueue/review commands. | Yes |
| proposal | list, view, create, update, comment, links, link, accept, defer, reject, complete, reopen, work, generate | Optional intake; writes/decisions and proposer execution. | Yes |
| queue | list, view, requeue | Reads; explicit requeue mutation. | Yes |
| attention | list, view (default: list) | Decision packets; read-only. | Yes |
| workflow | list, view, runs, run | Reads; run starts a configured workflow. | Yes |
| agent | roles, executions, execution | Role and execution inspection only. | Yes |
| actions | list, view, run, rerun, cancel, logs | Run reads and dispatch/cancel mutations. | Yes; logs NDJSON |
| pages | status, deployments, view, configure, deploy, promote, rollback | Site reads, config/build/production changes. | Yes |
| release | list, view, create, upload, download, publish | Release reads/draft writes/publication; local downloads. | Yes |
| org | list, view | Organisation inspection only. | Yes |
| api | [METHOD] PATH | Raw HTTP request; effect/auth depends on endpoint/method. | Server body |
| status | — | Authenticated host/user and checkout identity; read-only. | Yes |
| browse | — | Opens a local browser using configured host. | No result |
| config | list, get, set-host | Local profile inspection/selection; requires existing active profile to bootstrap. | list only |
| version | — | Local build information. | Human text currently |
| help / completion | help [command]; completion bash/fish/powershell/zsh | Built-in help and shell-script generation. | Not JSON results |

## Known implementation/documentation drift (not expanded into fixes)

- No typed Attempt creation, PR creation/enqueue, review or resolver command.
- `pages deployments`, not `pages list`; `pages configure`, not `pages config`.
- `version --json` emits human text because pre-run does not initialize its renderer.
- `config get/set-host --json` emit no result; auth switch/logout print text.
- Work create accepts body/body-file flags but sends only title/kind; examples
  intentionally omit those flags. Work/Attempt/queue lists do not all automatically
  scope to checkout. Status reports identity, not health/open-work counts.
- ExitInvalid=2 is declared but not emitted by the current mapper. Raw API non-2xx
  responses return 1, unlike typed 401/403->3, 409->4, 429/5xx->5.
- Raw API is server-body output, Actions JSON logs are NDJSON; command-specific
  view envelopes differ. No universal JSON schema for every command.
- HTTPS cookie compatibility was fixed separately; production server hardening
  was not altered. Host config now optionally retains the returned cookie name;
  old token-only profiles infer the transport-appropriate name, never both.

## Verification

Every top-level/group/leaf help path inspected (plus four shell-completion paths).
Public documentation command lines are checked against built help/flag syntax,
without executing production mutations. Full sy tests and vet cover the separately
committed HTTP/TLS cookie fix and existing command fixtures. Read-only live probes
and responsive/link checks are recorded in the final task report.

`work list --repo` currently does not narrow results (its predicate always
returns true). Work lists are account-wide; do not use this flag as a scope
guarantee. This documentation pass leaves that implementation drift unchanged.
