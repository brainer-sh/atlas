#!/usr/bin/env bash
# download-deps.sh - Download model and onnxruntime native lib for atlas.
# Run once before building with -tags with_embeddings.
set -euo pipefail

ATLAS_DIR="${HOME}/.atlas"
MODELS_DIR="${ATLAS_DIR}/models"
LIB_DIR="${ATLAS_DIR}/lib"
ORT_VERSION="1.21.0"

mkdir -p "${MODELS_DIR}" "${LIB_DIR}"

dl() {
    local url="$1" dst="$2"
    if [ -f "${dst}" ]; then
        echo "  already present: ${dst##*/}"
        return
    fi
    echo "  downloading ${dst##*/}..."
    curl -fsSL -o "${dst}" "${url}"
}

echo "==> model"
dl "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx" \
   "${MODELS_DIR}/all-MiniLM-L6-v2.onnx"

echo "==> tokenizer"
dl "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json" \
   "${MODELS_DIR}/tokenizer.json"

echo "==> onnxruntime native lib"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${OS}/${ARCH}" in
    linux/x86_64)
        ORT_ARCHIVE="onnxruntime-linux-x64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.so"
        ;;
    linux/aarch64)
        ORT_ARCHIVE="onnxruntime-linux-aarch64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.so"
        ;;
    darwin/arm64)
        ORT_ARCHIVE="onnxruntime-osx-arm64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.dylib"
        ;;
    darwin/x86_64)
        ORT_ARCHIVE="onnxruntime-osx-x86_64-${ORT_VERSION}.tgz"
        LIB_NAME="libonnxruntime.dylib"
        ;;
    *)
        echo "unsupported platform: ${OS}/${ARCH}" >&2
        exit 1
        ;;
esac

LIB_DST="${LIB_DIR}/${LIB_NAME}"
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
