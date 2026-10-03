# Configuration

Config follows XDG on Linux (`~/.config/switchyard/config.json` or
`$XDG_CONFIG_HOME/switchyard/config.json`), written with 0600 permissions.

```json
{
  "hosts": {
    "45.79.189.46": { "token": "sess_...", "user": "alice" }
  },
  "active_host": "45.79.189.46"
}
```

- `--host <url>` overrides the active host for one command.
- `sy auth switch <host>` changes the active host.
- Tokens are never printed by normal commands.

The demo host `http://45.79.189.46` is **not** hardcoded anywhere; it only
appears in your own config after login.
