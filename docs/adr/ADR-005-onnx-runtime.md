# ADR-005 - ONNX Runtime and Distribution

**Date:** May 2026
**Status:** Accepted

---

## Context

Atlas embeds an ONNX model (`all-MiniLM-L6-v2`, see ADR-004) for semantic search.
A Go runtime capable of executing this model is needed, along with a strategy to
distribute the native onnxruntime lib with the binary.

## Options Considered

### `knights-analytics/hugot`
- High-level abstraction oriented toward HuggingFace
- Handles tokenizer and model together
- Based on onnxruntime underneath, carrying the same native lib constraint
- More opaque, less control over inputs/outputs

### `yalue/onnxruntime_go`
- Direct and minimal wrapper around the C onnxruntime API
- Full control over inputs/outputs
- Native onnxruntime lib to provide separately or embed
- Well maintained, used in production

### External embeddings server (ollama, llamafile)
- Zero cgo in Atlas
- Requires an external dependency installed by the user, which is incompatible
  with the binary autonomy principle

## Decision

**`yalue/onnxruntime_go` with onnxruntime lib embedded via `go:embed`**

## Rationale

User experience is the priority: a single binary to download, zero setup.
`yalue/onnxruntime_go` allows embedding the native lib and extracting it to a
temporary directory at startup. This is an established pattern for Go tools with
native dependencies.

## Consequences

### cgo Exception
`internal/embeddings` is the only exception to the project's zero-cgo rule.
This exception is isolated and all other packages remain pure Go.

### Native Lib per Platform
The onnxruntime lib varies by OS and architecture:
- `linux/amd64` -> `libonnxruntime.so`
- `darwin/arm64` -> `libonnxruntime.dylib` (Apple Silicon)
- `darwin/amd64` -> `libonnxruntime.dylib` (Intel Mac)
- `windows/amd64` -> `onnxruntime.dll`

### Matrix Build
The Makefile defines one target per platform. Binaries are distributed via GitHub Releases:

```
atlas-linux-amd64
atlas-darwin-arm64
atlas-darwin-amd64
atlas-windows-amd64.exe
```

### Startup Extraction
On first launch, `internal/embeddings` extracts the native lib to `~/.atlas/lib/`
and loads it dynamically. Subsequent launches reuse the extracted lib.

### Embedded Files Structure
```
internal/embeddings/
  model/
    all-MiniLM-L6-v2.onnx    (gitignored, downloaded at build)
    tokenizer.json             (gitignored, downloaded at build)
  lib/
    linux_amd64/libonnxruntime.so
    darwin_arm64/libonnxruntime.dylib
    darwin_amd64/libonnxruntime.dylib
    windows_amd64/onnxruntime.dll
  embed.go                    (go:embed directives)
```

### Build Script
A `scripts/download-deps.sh` script downloads the model and onnxruntime libs for
each platform before compilation. Run it once after `git clone`.
