# ADR-004 - Embedding Model

**Date:** May 2026
**Status:** Accepted

---

## Context

Atlas uses vector embeddings for semantic search on indexed symbols. The model must
run on CPU on modest machines, without requiring a GPU.

## Criteria

- Runs on CPU without GPU
- Lightweight (< 30MB)
- Good quality on technical text / code
- ONNX format available
- Reasonable vector dimension for sqlite-vec

## Options Considered

### `all-MiniLM-L6-v2`
- 22MB, dimension 384
- Runs on CPU in < 50ms per batch
- De facto standard for lightweight semantic search
- Official ONNX format available on HuggingFace
- Good performance on technical text (function names, signatures, docstrings)

### `nomic-embed-code`
- Specifically oriented toward code
- Heavier (~130MB), less suited for modest machines

### `text-embedding-3-small` (OpenAI API)
- Excellent quality
- Requires an API key and network connection, which is incompatible with
  Atlas's offline principle

## Decision

**`all-MiniLM-L6-v2`**

## Rationale

The model is embedded in the binary via `go:embed`, making size and CPU compatibility
hard constraints. `all-MiniLM-L6-v2` offers the best quality/size ratio for this use
case. Indexed symbols (names, signatures, docstrings) are short texts for which this
model is well suited.

## Consequences

- Model downloaded from HuggingFace at build time and bundled via `go:embed`
- Files to bundle: `all-MiniLM-L6-v2.onnx` + `tokenizer.json`
- Final binary size: ~35MB (model + tokenizer + runtime)
- Vector dimension is 384, stored in an `embedding BLOB` column in sqlite-vec
- A matrix build is required per platform (see ADR-005)
- Files are stored in `internal/embeddings/model/` (gitignored, downloaded at build)
