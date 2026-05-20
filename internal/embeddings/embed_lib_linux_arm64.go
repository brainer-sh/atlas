//go:build with_embeddings && linux && arm64

package embeddings

import _ "embed"

//go:embed lib/linux_arm64/libonnxruntime.so
var nativeLib []byte
