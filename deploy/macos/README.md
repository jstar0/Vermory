# macOS User Service

This profile installs Vermory as a per-user `launchd` service. It requires no
root privileges and keeps PostgreSQL, service state, logs, and client runtimes
on the target Mac rather than on the invoking workstation.

Requirements:

- PostgreSQL 18 listening on the local Unix socket;
- a Darwin arm64 or amd64 Vermory binary;
- an active graphical user launchd domain.

Install or update:

```bash
VERMORY_DATABASE_NAME=vermory \
VERMORY_TENANT_ID=local \
VERMORY_LISTEN=127.0.0.1:8787 \
VERMORY_PROVIDER=external \
  ./deploy/macos/install-user-service.sh /path/to/vermory
```

The installer:

- copies the binary to `~/.local/bin/vermory`;
- creates the database if needed and applies migrations;
- installs `~/Library/LaunchAgents/org.vermory.web-chat.plist`;
- stores logs under `~/Library/Logs/Vermory`;
- verifies the real loopback Web Chat endpoint before returning success.

The default service intentionally listens only on loopback. Remote clients
should use SSH stdio or an operator-controlled SSH tunnel instead of exposing
the unauthenticated local Web Chat profile to a LAN or public network.

## OpenClaw Gateway

After installing dependencies and building `integrations/openclaw`, install the
official OpenClaw Gateway with the Vermory lifecycle plugin:

```bash
./deploy/macos/install-openclaw-service.sh /path/to/integrations/openclaw
```

The installer keeps OpenClaw state, config, workspace, package caches, runtime
inspection, and health evidence under `~/Library/Application Support/Vermory`.
It calls OpenClaw's official Gateway service installer rather than maintaining
a second custom Gateway plist. Re-running the installer merges Vermory's required
settings into the existing OpenClaw config, preserves Gateway authentication,
CLI backends, channels, and unrelated plugins, and keeps the config mode at
`0600`. The Gateway and Vermory API remain loopback-only.

## Grok CLI Runtime

Install the official signed Grok binary and an existing authenticated
`auth.json` into an isolated Vermory-owned runtime on the target Mac:

```bash
./deploy/macos/install-grok-runtime.sh /path/to/grok /path/to/auth.json
./deploy/macos/install-openclaw-service.sh /path/to/integrations/openclaw
```

The Grok installer stores the binary, mutable auth state, and runtime home under
`~/Library/Application Support/Vermory/grok`. `grok-vermory` supplies the
isolated home, protected auth, proxy, and disabled ambient memory environment
for Vermory's provider adapter, which owns its own no-tool arguments.
`grok-vermory-isolated` additionally enforces one-turn, no-plan, no-subagent,
no-web, and no-tool arguments for OpenClaw. Reinstalling updates the binary but
preserves the target host's refreshed auth file unless
`VERMORY_GROK_REPLACE_AUTH=1` is explicitly set. The OpenClaw installer detects
the wrapper and registers the isolated `grok-cli` backend without replacing
other backend configuration.

If the target host requires a local HTTP proxy, pass an unauthenticated loopback
URL when installing. Non-loopback and credential-bearing proxy URLs are rejected:

```bash
VERMORY_GROK_PROXY_URL=http://127.0.0.1:6152 \
  ./deploy/macos/install-grok-runtime.sh /path/to/grok /path/to/auth.json
```
