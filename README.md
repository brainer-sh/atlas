# Atlas

Atlas is a Go MCP server that lets you learn the architecture of a codebase through
a conversational session with any agent (Claude, OpenCode, etc.).

It indexes a local repo (Go, C, C++) via tree-sitter, stores symbols in SQLite, and
exposes MCP tools to search, explore, and visualize the codebase.

## Install

**From source (requires Go 1.21+):**

```sh
git clone https://github.com/brainer-sh/atlas
cd atlas
make install
```

**With go install:**

```sh
go install github.com/brainer.sh/atlas/cmd/atlas@latest
```

## Quickstart

```sh
# Index a repository
atlas index /path/to/your/repo

# Start the MCP server
atlas serve
```

## Usage

```sh
atlas index <path>     # index a repository, creates ~/.atlas/<name>.db
atlas reindex <path>   # re-index modified files only
atlas list             # list all indexed repositories
atlas serve            # start the MCP server (stdio)
atlas search <query>   # debug search without the agent
atlas --version        # print version
atlas --help           # show this help
```

## MCP Config

Add Atlas to your agent's MCP config. Index your repos first, then start the server.

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "atlas": {
      "command": "atlas",
      "args": ["serve"]
    }
  }
}
```

**OpenCode** (`~/.config/opencode/config.json`):

```json
{
  "mcp": {
    "atlas": {
      "command": "atlas",
      "args": ["serve"]
    }
  }
}
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `index_repo` | Index a repository |
| `reindex` | Re-index modified files |
| `search` | Full-text symbol search |
| `explore` | Symbol details with callers/callees |
| `get_map` | Mermaid architecture diagram |
| `list_repos` | List indexed repositories |

## Supported Languages

- Go
- C
- C++

## License

MIT
