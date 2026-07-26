# runtz-mcp

An **offline [Model Context Protocol](https://modelcontextprotocol.io) server for
[runtz](https://runtz.dev)**, the open source DevSecOps scans platform.

It lets any MCP-capable AI client — **Claude**, **Codex**, **Gemini**, and others —
read the runtz documentation and run runtz security scans directly from a chat or
agent loop.

- **Offline-first.** The documentation is embedded in the binary; the server
  never needs network access to answer docs questions.
- **Zero dependencies.** Implemented with only the Go standard library
  (JSON-RPC 2.0 over stdio). `go build` and you are done — nothing to `go get`.
- **Secrets stay in config.** The workspace token is read from the environment
  injected by the MCP client, so it never appears in prompts, logs, or tool
  arguments.

> Written in Go to match the runtz stack (the runtz backend and CLI are Go), so
> there is one toolchain to build and ship.

## Tools

### Scans (require a configured token)

| Tool | runtz command | Purpose |
| --- | --- | --- |
| `runtz_sca` | `runtz sca` | Software Composition Analysis over a `package.json` |
| `runtz_sast` | `runtz sast` | Static analysis for secrets, weak crypto, disabled TLS, etc. |
| `runtz_host` | `runtz host` | Package CVEs on a Debian/Ubuntu host or rootfs |
| `runtz_container` | `runtz container` | Package CVEs inside a container image |
| `runtz_k8s` | `runtz k8s` | Kubernetes cluster / manifest posture findings |

### Documentation (always available, fully offline)

| Tool | Purpose |
| --- | --- |
| `runtz_docs_list` | List the documentation pages with summaries |
| `runtz_docs_read` | Read a full page by slug |
| `runtz_docs_search` | Search the docs and return snippets |

Each documentation page is also exposed as an MCP **resource**
(`runtz-docs://<slug>`).

## Install

```bash
# from this directory
make install          # installs `runtz-mcp` into your GOBIN
# or
go build -o runtz-mcp ./cmd/runtz-mcp
```

The scan tools shell out to the runtz CLI, so install it too (or point
`RUNTZ_MCP_BIN` at it):

```bash
curl -fsSL https://get.runtz.dev | bash
# or: go install github.com/runtz-dev/runtz-cli/cmd/runtz@latest
```

## Configuration

All configuration is environment-based. Set it in your MCP client's `env` block.

| Variable | Default | Description |
| --- | --- | --- |
| `RUNTZ_ENDPOINT` | hosted SaaS engine | Runtz backend endpoint |
| `RUNTZ_TOKEN` | — | Workspace token generated in the platform |
| `RUNTZ_MCP_BIN` | `runtz` | runtz CLI binary or launcher (e.g. `go run ./cmd/runtz`) |
| `RUNTZ_MCP_WORKDIR` | process cwd | Working directory scans run from |
| `RUNTZ_MCP_ALLOW_SCANS` | `true` | Set `false` for a docs-only server |
| `RUNTZ_MCP_HTTP` | — | Listen address (e.g. `:8080`) to serve the HTTP transport instead of stdio; always docs-only |

The hosted engine endpoint is `https://engine.runtz.dev`. Generate a
workspace token in the runtz platform.

## Hosted server

runtz also runs this server for you over HTTP, so you don't have to install
anything to browse the docs from an MCP client:

| Environment | URL |
| --- | --- |
| Production | `https://mcp.runtz.dev/mcp` |
| Development | `https://mcp-dev.runtz.dev/mcp` |

The hosted server is **docs-only** — it never runs scans, because scans need
your code on the machine running the CLI. Run the server locally over stdio (as
below) when you want the scan tools. To self-host the HTTP server, deploy the
chart:

```bash
helm repo add runtz https://helm.runtz.dev
helm install runtz-mcp runtz/runtz-mcp \
  --set ingress.enabled=true \
  --set 'ingress.hosts[0].host=mcp.example.com' \
  --set 'ingress.hosts[0].paths[0].path=/' \
  --set 'ingress.hosts[0].paths[0].pathType=Prefix'
```

## Client setup

Ready-to-edit configs live in [`examples/`](./examples):

- **Claude Code / Desktop** — [`examples/claude.json`](./examples/claude.json)
  (save as `.mcp.json` in a project, or merge into `~/.claude.json` /
  `claude_desktop_config.json`).
- **Codex** — [`examples/codex.toml`](./examples/codex.toml)
  (merge into `~/.codex/config.toml`).
- **Gemini CLI** — [`examples/gemini.json`](./examples/gemini.json)
  (merge into `~/.gemini/settings.json`).

Example (Claude Code `.mcp.json`):

```json
{
  "mcpServers": {
    "runtz": {
      "command": "/usr/local/bin/runtz-mcp",
      "env": {
        "RUNTZ_ENDPOINT": "https://engine.runtz.dev",
        "RUNTZ_TOKEN": "rtz_live_..."
      }
    }
  }
}
```

Pair this with the matching [runtz skill](../runtz-skills) for your assistant so
it knows *when* and *how* to use these tools.

## Development

```bash
make build      # build into ./bin
make test       # run tests
make vet        # static checks
```

The codebase is small and layered:

```
cmd/runtz-mcp        entrypoint, env wiring, stdio + HTTP transports
internal/mcp         dependency-free JSON-RPC 2.0 / MCP server (stdio + HTTP)
internal/runtz       config, scan execution, embedded docs
internal/tools       tool + resource registration
```

### Why not the official Go SDK?

The official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)
is excellent and a fine choice. This server deliberately uses only the standard
library so the initial open source drop builds anywhere with no module downloads
and a trivially auditable surface. Swapping in the SDK later is straightforward:
the tool definitions in `internal/tools` map directly onto `mcp.AddTool`.

## License

MIT — see [LICENSE](./LICENSE).
