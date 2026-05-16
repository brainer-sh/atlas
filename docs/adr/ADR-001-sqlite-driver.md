# ADR-001 - SQLite Driver

**Date:** May 2026
**Status:** Accepted

---

## Context

Atlas stores the symbol index in SQLite. The choice of Go driver directly impacts
binary portability and build complexity.

## Options Considered

### `mattn/go-sqlite3`
- Most widespread driver in the Go ecosystem
- cgo-based, requires a C toolchain at compile time
- Well documented, many examples

### `zombiezen.com/go/sqlite`
- Pure Go driver, zero cgo
- Based on `modernc.org/sqlite`, a C->Go transpilation of SQLite
- Binary compilable without C toolchain
- Supports FTS5 and standard SQLite extensions
- Slightly lower performance (~10-15%) but negligible for Atlas

## Decision

**`zombiezen.com/go/sqlite`**

## Rationale

Atlas targets a zero system-dependency binary. Using cgo would impose a C toolchain
on the build machine and complicate cross-compilation. The slight performance loss
has no impact for an indexing and interactive search tool.

## Consequences

- Binary compiles with `go build` alone, no gcc/clang required
- Cross-compilation is simplified (`GOOS=windows go build` works)
- FTS5 is available natively
- The `embeddings` package is the only cgo exception in the project, via
  `yalue/onnxruntime_go` (see ADR-005). It is isolated and does not affect storage.
