# ADR-003 - Go MCP SDK

**Date:** May 2026
**Status:** Accepted

---

## Context

Atlas is an MCP server exposed via stdio. The choice of SDK determines the amount
of plumbing to maintain and conformance to the MCP protocol.

## Options Considered

### Manual stdio protocol implementation
- Zero external dependency
- Full transport control
- Requires writing and maintaining ~200-300 lines of JSON-RPC plumbing
- Risk of divergence as the MCP protocol evolves

### `github.com/mark3labs/mcp-go`
- Most adopted SDK in the Go ecosystem
- Clean abstraction covering tool registration, handshake management, and stdio transport
- Actively maintained and aligned with MCP protocol evolution
- Reduces the code surface to maintain

## Decision

**`github.com/mark3labs/mcp-go`**

## Rationale

Atlas's value is in the indexer and search, not in MCP plumbing. `mcp-go` is the
de facto standard in Go and handles the protocol reliably. Time saved is reinvested
in value-added components.

## Consequences

- External dependency explicitly assumed and documented
- Tool registration uses the `mcp-go` API in `internal/mcp/`
- `mcp-go` upgrades must be tracked as the MCP protocol evolves
