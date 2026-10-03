# Examples

## Human

```sh
sy auth login --host http://45.79.189.46
sy repo list
sy repo clone demo-basic
sy work list --status open
sy work view wk_123
sy pr list
sy queue view iq_123
sy attention
sy attempt run wk_456
sy status
```

## Agent

```sh
sy repo view demo-basic --json
sy work list --json --status open
sy attention --json
sy queue list --json --status blocked
sy workflow runs --json
sy api GET /api/roles --json
```
