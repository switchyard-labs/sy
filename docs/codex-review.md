# Independent Codex review and server integration

2026-10-04. Reviewed independently from DeepSeek's handover. No project pushes.

Verdict: a useful domain CLI with a sound Cobra/shared-client/output foundation,
but the original implementation had correctness/security issues that prevented
calling it a polished gh peer. Fixed HTTPS host downgrade on save, untyped
string server errors, ignored Retry-After, cancellation-insensitive retry sleep,
unnecessary sleep after final retry, unsafe credential redirects and non-atomic
config writes. Mutation requests still never retry automatically. Existing
scheme-less host profiles remain legacy HTTP profiles; re-login with an explicit
HTTPS URL to preserve transport unambiguously.

Repository listing/view now uses canonical registered metadata. ReadOnly is not
visibility. Flat names resolve only when unique; owner/repo is recommended.
Clone now requests short-lived read capability from Switchyard and uses Git's
transient environment config scoped to the HTTPS remote. No token argv,
credential helper, redirect, global credential file or persisted header. Clone
JSON contains only repository/directory/remote. Local `.git/switchyard.json`
contains canonical owner/repo so provider namespace names do not break checkout
context. Credentials are not copied to the checkout metadata.

Added `actions list/view/run/cancel/rerun/logs`, with `--repo/-R` or checkout
context. Run requires exact SHA or PR; rerun accepts `--failed`; cancellation is
explicit. Logs follows captured step output until terminal run, drains pages,
and Ctrl-C never cancels the Action. `--json` logs emits one JSON object per line
(NDJSON with run_id/step/line). No fake shell or raw terminal control parsing.
Direct paginated `/api/attempts` replaces the first30 Work scan and suppressed
per-Work failures. No compatibility guesses for absent PAT/device auth.

Validation: full Go tests and vet; context cancellation/429/redirect tests;
source-specific credential boundary tests; log pagination/terminal/cancellation
tests. Real scoped clone of `alice/railway` succeeded, with no extraHeader in
`.git/config`. Real Action log follow yielded `NO_PRIVILEGED_CREDENTIALS` and
exited0. Actions list9 runs, view real exact-SHA run, Attempts list1 and repository
list2 passed against isolated18127. Existing integration harness: **11/11 PASS**
against isolated18127, using only `sy-dogfood-*` Work state. No Linode mutation
for these tests. Live production11/11 and cross-build rerun remain separate gates.

Remaining review items: original path parsing/argument validation and invalid
request exit2 coverage should be broadened; color/TTY output is intentionally
simple; auth login now emits a credential-free JSON result; existing config
commands use legacy host assumptions; terminal canceled/skipped steps without captures are skipped. Actions list API is currently an unpaginated
server list. CLI reads expose honest server errors rather than pretending native
Worker Preview deployment or PAT/device capabilities exist.
