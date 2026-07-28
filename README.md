# runtz-mcp

A **hosted, docs-only [Model Context Protocol](https://modelcontextprotocol.io)
server for [runtz](https://runtz.dev)**, the open source DevSecOps scans platform.

It lets any MCP-capable AI client — **Claude**, **Codex**, **Gemini**, and others —
read the runtz documentation directly from a chat or agent loop.
**Nothing to install**: point your client at `https://mcp.runtz.dev/mcp` and
you're done.

- **Zero-install.** runtz runs and maintains this server for you. There is no
  local binary to build or configure.
- **Zero dependencies.** Implemented with only the Go standard library
  (JSON-RPC 2.0 over the MCP streamable HTTP transport).
- **Docs-only, on purpose.** This server only ever answers documentation
  questions. Run DevSecOps scans with the [runtz CLI](https://github.com/runtz-dev/runtz-cli)
  directly — scans need your code on the machine running them, which a shared
  hosted server never has.

## Tools

| Tool | Purpose |
| --- | --- |
| `runtz_docs_list` | List the documentation pages with summaries |
| `runtz_docs_read` | Read a full page by slug |
| `runtz_docs_search` | Search the docs and return snippets |

Each documentation page is also exposed as an MCP **resource**
(`runtz-docs://<slug>`).

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
      "type": "http",
      "url": "https://mcp.runtz.dev/mcp"
    }
  }
}
```

Pair this with the matching [runtz skill](../runtz-skills) for your assistant so
it knows *when* and *how* to use these tools.

## Self-hosting

You don't need to — `mcp.runtz.dev` is always up — but the chart is public if
you want to run your own copy (e.g. an air-gapped environment):

```bash
helm repo add runtz https://helm.runtz.dev
helm install runtz-mcp runtz/runtz-mcp \
  --set ingress.enabled=true \
  --set 'ingress.hosts[0].host=mcp.example.com' \
  --set 'ingress.hosts[0].paths[0].path=/' \
  --set 'ingress.hosts[0].paths[0].pathType=Prefix'
```

## Development

```bash
make build      # build into ./bin
make run        # run the HTTP server locally on :8080
make test       # run tests
make vet        # static checks
```

The codebase is small and layered:

```
cmd/runtz-mcp        entrypoint, env wiring, HTTP transport
internal/mcp         dependency-free JSON-RPC 2.0 / MCP server (HTTP)
internal/runtz       embedded docs
internal/tools       doc tool + resource registration
```

### Why not the official Go SDK?

The official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)
is excellent and a fine choice. This server deliberately uses only the standard
library so the initial open source drop builds anywhere with no module downloads
and a trivially auditable surface. Swapping in the SDK later is straightforward:
the tool definitions in `internal/tools` map directly onto `mcp.AddTool`.

## License

MIT — see [LICENSE](./LICENSE).
