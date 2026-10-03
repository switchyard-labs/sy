# Environment variables

Precedence: **flag > environment > active host config > default**.

| Variable | Purpose |
| --- | --- |
| `SY_HOST` | default host (when `--host` not given and no active host) |
| `SY_CONFIG_DIR` | override the config directory (testing/CI) |
| `SY_GIT_TOKEN` | Git credential for `sy repo clone` when the server exposes none |
| `NO_COLOR` | disable ANSI color |
| `PAGER` | pager for long human output (when stdout is a TTY) |
| `TERM=dumb` | disable color/progress |
