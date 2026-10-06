# Repository context

`sy` derives `owner/repo` from the current Git checkout's `origin` remote, for commands that resolve repository context, such as `sy pr list`.
`sy work list` remains account-wide; its current `--repo` flag does not
actually narrow results. See the [command inventory](command-inventory.md).

Supported remote shapes:

- Artifacts: `https://<account>.artifacts.cloudflare.net/git/<ns>/<repo>.git`
- GitHub-style: `https://github.com/owner/repo.git`
- scp-style: `git@github.com:owner/repo.git`

`sy repo view` / `sy repo clone` with no argument use the current checkout;
passing `owner/repo` explicitly overrides it.
