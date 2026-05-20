//go:build with_embeddings

package embeddings

import _ "embed"

//go:embed tokenizer.json
var tokenizerData []byte
