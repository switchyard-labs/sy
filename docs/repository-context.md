# Repository context

`sy` derives `owner/repo` from the current Git checkout's `origin` remote, so
`cd repo && sy work list` works without repeating the repository name.

Supported remote shapes:

- Artifacts: `https://<account>.artifacts.cloudflare.net/git/<ns>/<repo>.git`
- GitHub-style: `https://github.com/owner/repo.git`
- scp-style: `git@github.com:owner/repo.git`

`sy repo view` / `sy repo clone` with no argument use the current checkout;
passing `owner/repo` explicitly overrides it.
