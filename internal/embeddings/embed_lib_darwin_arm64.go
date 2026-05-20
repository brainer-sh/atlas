//go:build with_embeddings && darwin && arm64

package embeddings

import _ "embed"

//go:embed lib/darwin_arm64/libonnxruntime.dylib
var nativeLib []byte
