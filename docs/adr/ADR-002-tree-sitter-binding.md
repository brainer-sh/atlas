# ADR-002 - Go tree-sitter Binding

**Date:** May 2026
**Status:** Accepted

---

## Context

Atlas uses tree-sitter to parse source files and extract symbols (functions, types,
interfaces, etc.) with their line numbers. The choice of Go binding determines
long-term maintainability and grammar availability.

## Options Considered

### `github.com/smacker/go-tree-sitter`
- Unofficial binding, historically the most used in the Go ecosystem
- Many examples and reference projects
- Maintenance has slowed since 2023
- Grammars are bundled in the repo

### `github.com/tree-sitter/go-tree-sitter`
- Official binding maintained by the tree-sitter team
- More recent, with a cleaner API
- Aligned with tree-sitter core releases
- Grammars available via official repos (`tree-sitter/tree-sitter-go`, etc.)
- Fewer Go examples in the wild but complete official documentation

## Decision

**`github.com/tree-sitter/go-tree-sitter`**

## Rationale

The official binding guarantees compatibility with future tree-sitter core versions.
Atlas supports Go, C and C++. All three grammars have official repos maintained by
the tree-sitter team. Project longevity outweighs the quantity of available examples.

## Consequences

- Grammars are imported separately: `tree-sitter/tree-sitter-go`,
  `tree-sitter/tree-sitter-c`, `tree-sitter/tree-sitter-cpp`
- tree-sitter queries (`.scm` files) are written per language
  in `internal/indexer/<lang>/queries/`
- The binding uses cgo internally. This is a documented exception, isolated
  in `internal/indexer/`.
