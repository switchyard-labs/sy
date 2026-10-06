# Configuration

Config follows XDG on Linux (`~/.config/switchyard/config.json` or
`$XDG_CONFIG_HOME/switchyard/config.json`), written with 0600 permissions.

```json
{
  "hosts": {
    "https://switchyard.cx": { "token": "REDACTED", "user": "alice" }
  },
  "active_host": "https://switchyard.cx"
}
```

- `--host <url>` overrides the active host for one command.
- `sy auth switch <host>` changes the active host.
- Tokens are never printed by normal commands.

Host profiles are created by login; no production host is hardcoded.
Use an explicit HTTPS URL for production. Local HTTP servers remain supported.
