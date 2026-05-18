package storage

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// StoreEmbedding persists a float32 embedding vector for a symbol.
func (s *Store) StoreEmbedding(symbolID int64, vec []float32) error {
	err := sqlitex.Execute(s.conn,
		`INSERT OR REPLACE INTO symbols_embeddings (symbol_id, embedding) VALUES (:id, :emb)`,
		&sqlitex.ExecOptions{
			Named: map[string]any{
				":id":  symbolID,
				":emb": encodeVec(vec),
			},
		})
	if err != nil {
		return fmt.Errorf("storage: store embedding for symbol %d: %w", symbolID, err)
	}
	return nil
}

// vecCandidate is used for in-memory cosine ranking.
type vecCandidate struct {
	symbolID int64
	vec      []float32
}

// SearchSimilar returns up to limit symbols ranked by cosine similarity to vec.
// All embeddings are loaded into memory and ranked in Go.
func (s *Store) SearchSimilar(vec []float32, limit int) ([]SearchResult, error) {
	// Load all (symbol_id, embedding) pairs.
	var candidates []vecCandidate
	err := sqlitex.Execute(s.conn,
		`SELECT symbol_id, embedding FROM symbols_embeddings`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				id := stmt.ColumnInt64(0)
				raw := make([]byte, stmt.ColumnLen(1))
				stmt.ColumnBytes(1, raw)
				candidates = append(candidates, vecCandidate{
					symbolID: id,
					vec:      decodeVec(raw),
				})
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: load embeddings: %w", err)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// Rank by cosine similarity.
	type scored struct {
		id    int64
		score float64
	}
	scores := make([]scored, len(candidates))
	for i, c := range candidates {
		scores[i] = scored{id: c.symbolID, score: cosine(vec, c.vec)}
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if limit > 0 && len(scores) > limit {
		scores = scores[:limit]
	}

	// Fetch full symbol details for each top result.
	results := make([]SearchResult, 0, len(scores))
	for _, sc := range scores {
		var r *SearchResult
		err := sqlitex.Execute(s.conn,
			`SELECT s.id, s.file_id, s.repo_id, s.name, s.kind, s.signature, s.doc,
			        s.line_start, s.line_end, f.path
			 FROM symbols s
			 JOIN files f ON f.id = s.file_id
			 WHERE s.id = :id`,
			&sqlitex.ExecOptions{
				Named: map[string]any{":id": sc.id},
				ResultFunc: func(stmt *sqlite.Stmt) error {
					r = &SearchResult{
						Symbol: Symbol{
							ID:        stmt.ColumnInt64(0),
							FileID:    stmt.ColumnInt64(1),
							RepoID:    stmt.ColumnInt64(2),
							Name:      stmt.ColumnText(3),
							Kind:      stmt.ColumnText(4),
							Signature: stmt.ColumnText(5),
							Doc:       stmt.ColumnText(6),
							LineStart: stmt.ColumnInt64(7),
							LineEnd:   stmt.ColumnInt64(8),
						},
						FilePath: stmt.ColumnText(9),
						Score:    sc.score,
					}
					return nil
				},
			})
		if err != nil {
			return nil, fmt.Errorf("storage: fetch symbol %d: %w", sc.id, err)
		}
		if r != nil {
			results = append(results, *r)
		}
	}
	return results, nil
}

// GetAllSymbols returns every symbol in the database with its ID.
func (s *Store) GetAllSymbols() ([]Symbol, error) {
	var symbols []Symbol
	err := sqlitex.Execute(s.conn,
		`SELECT id, file_id, repo_id, name, kind, signature, doc, line_start, line_end
		 FROM symbols ORDER BY id`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				symbols = append(symbols, Symbol{
					ID:        stmt.ColumnInt64(0),
					FileID:    stmt.ColumnInt64(1),
					RepoID:    stmt.ColumnInt64(2),
					Name:      stmt.ColumnText(3),
					Kind:      stmt.ColumnText(4),
					Signature: stmt.ColumnText(5),
					Doc:       stmt.ColumnText(6),
					LineStart: stmt.ColumnInt64(7),
					LineEnd:   stmt.ColumnInt64(8),
				})
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: get all symbols: %w", err)
	}
	return symbols, nil
}

func encodeVec(vec []float32) []byte {
	b := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b
}

func decodeVec(b []byte) []float32 {
	vec := make([]float32, len(b)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return vec
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
