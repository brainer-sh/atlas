//go:build with_embeddings && linux && amd64

package embeddings

import _ "embed"

//go:embed lib/linux_amd64/libonnxruntime.so
var nativeLib []byte
