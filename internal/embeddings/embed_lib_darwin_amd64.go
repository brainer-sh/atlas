//go:build with_embeddings && darwin && amd64

package embeddings

import _ "embed"

//go:embed lib/darwin_amd64/libonnxruntime.dylib
var nativeLib []byte
