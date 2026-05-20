#!/usr/bin/env bash
# download-deps.sh - Download ORT native lib, tokenizer, and ONNX model for atlas.
#
# The native ORT library and tokenizer are embedded into the binary at build time
# (go:embed). They must be downloaded into internal/embeddings/ before building
# with -tags with_embeddings.
#
# The model (~87 MB) is too large to embed and must be present at runtime in
# ~/.atlas/models/.
#
# Run once before building:
#   ./scripts/download-deps.sh
#   go build -tags with_embeddings ./...
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EMBEDDINGS_DIR="${REPO_ROOT}/internal/embeddings"

ATLAS_DIR="${HOME}/.atlas"
MODELS_DIR="${ATLAS_DIR}/models"
ORT_VERSION="1.25.0"

mkdir -p "${MODELS_DIR}"

dl() {
    local url="$1" dst="$2"
    if [ -f "${dst}" ]; then
        echo "  already present: ${dst##*/}"
        return
    fi
    echo "  downloading ${dst##*/}..."
    curl -fsSL -o "${dst}" "${url}"
}

echo "==> tokenizer (embedded in binary)"
dl "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json" \
   "${EMBEDDINGS_DIR}/tokenizer.json"

echo "==> model (runtime, ~/.atlas/models/)"
dl "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx" \
   "${MODELS_DIR}/all-MiniLM-L6-v2.onnx"

echo "==> onnxruntime native lib (embedded in binary)"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${OS}/${ARCH}" in
    linux/x86_64)
        ORT_ARCHIVE="onnxruntime-linux-x64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.so"
        LIB_SUBDIR="linux_amd64"
        ;;
    linux/aarch64)
        ORT_ARCHIVE="onnxruntime-linux-aarch64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.so"
        LIB_SUBDIR="linux_arm64"
        ;;
    darwin/arm64)
        ORT_ARCHIVE="onnxruntime-osx-arm64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.dylib"
        LIB_SUBDIR="darwin_arm64"
        ;;
    darwin/x86_64)
        ORT_ARCHIVE="onnxruntime-osx-x86_64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.dylib"
        LIB_SUBDIR="darwin_amd64"
        ;;
    *)
        echo "unsupported platform: ${OS}/${ARCH}" >&2
        exit 1
        ;;
esac

LIB_DST="${EMBEDDINGS_DIR}/lib/${LIB_SUBDIR}/${LIB_NAME}"
if [ ! -f "${LIB_DST}" ]; then
    echo "  downloading onnxruntime ${ORT_VERSION} for ${OS}/${ARCH}..."
    TMP="$(mktemp -d)"
    trap 'rm -rf "${TMP}"' EXIT
    curl -fsSL -o "${TMP}/ort.tgz" \
        "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/${ORT_ARCHIVE}"
    tar -xzf "${TMP}/ort.tgz" -C "${TMP}"
    # Copy the first matching lib (follows symlinks with -L).
    find "${TMP}" \( -name "libonnxruntime.so" -o -name "libonnxruntime.dylib" \) \
        | head -1 \
        | xargs -I{} cp -L "{}" "${LIB_DST}"
    echo "  installed: ${LIB_DST}"
else
    echo "  already present: ${LIB_NAME}"
fi

echo ""
echo "Done. Build with:  go build -tags with_embeddings ./..."
