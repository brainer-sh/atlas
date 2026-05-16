# Atlas

Atlas is a Go MCP server that lets you learn the architecture of a codebase through
a conversational session with any agent.

It indexes a local repo (Go, C, C++) via tree-sitter, stores symbols in SQLite, and
exposes MCP tools to search, explore, and visualize the codebase.

## Install

```sh
# coming in v0.1
```

## Usage

```sh
atlas index <path>      # index a repo
atlas reindex <path>    # refresh modified files only
atlas list              # list indexed repos
atlas serve             # start the MCP server (stdio)
atlas search <query>    # debug search
```

## MCP Config

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

## License

MIT
