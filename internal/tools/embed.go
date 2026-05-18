package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/brainer.sh/atlas/internal/embeddings"
	"github.com/brainer.sh/atlas/internal/storage"
)

const embedBatchSize = 32

// EmbedAll embeds every symbol in store and persists the vectors.
// Symbols are embedded as "<kind> <name> <signature> <doc>" text.
func EmbedAll(ctx context.Context, store *storage.Store, embedder embeddings.Embedder) error {
	symbols, err := store.GetAllSymbols()
	if err != nil {
		return fmt.Errorf("tools/embed: get symbols: %w", err)
	}
	if len(symbols) == 0 {
		return nil
	}

	for i := 0; i < len(symbols); i += embedBatchSize {
		end := i + embedBatchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		batch := symbols[i:end]

		texts := make([]string, len(batch))
		for j, s := range batch {
			texts[j] = symbolText(s)
		}

		vecs, err := embedder.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("tools/embed: embed batch: %w", err)
		}

		for j, s := range batch {
			if err := store.StoreEmbedding(s.ID, vecs[j]); err != nil {
				return fmt.Errorf("tools/embed: store embedding for %s: %w", s.Name, err)
			}
		}
	}
	return nil
}

// symbolText builds the text representation used for embedding.
func symbolText(s storage.Symbol) string {
	parts := []string{s.Kind, s.Name}
	if s.Signature != "" {
		parts = append(parts, s.Signature)
	}
	if s.Doc != "" {
		parts = append(parts, s.Doc)
	}
	return strings.Join(parts, " ")
}
