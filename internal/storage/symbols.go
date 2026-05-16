package storage

import (
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Symbol represents an extracted code symbol.
type Symbol struct {
	ID        int64
	FileID    int64
	RepoID    int64
	Name      string
	Kind      string // function | method | struct | interface | type | const
	Signature string
	Doc       string
	LineStart int64
	LineEnd   int64
}

// SearchResult is a symbol match with its file path and relevance score.
type SearchResult struct {
	Symbol
	FilePath string
	Score    float64
}

// InsertSymbols inserts a batch of symbols within a single transaction.
func (s *Store) InsertSymbols(symbols []Symbol) error {
	endFn := sqlitex.Transaction(s.conn)
	var err error
	defer endFn(&err)

	for _, sym := range symbols {
		err = sqlitex.Execute(s.conn,
			`INSERT INTO symbols (file_id, repo_id, name, kind, signature, doc, line_start, line_end)
			 VALUES (:file_id, :repo_id, :name, :kind, :signature, :doc, :line_start, :line_end)`,
			&sqlitex.ExecOptions{
				Named: map[string]any{
					":file_id":    sym.FileID,
					":repo_id":    sym.RepoID,
					":name":       sym.Name,
					":kind":       sym.Kind,
					":signature":  sym.Signature,
					":doc":        sym.Doc,
					":line_start": sym.LineStart,
					":line_end":   sym.LineEnd,
				},
			})
		if err != nil {
			err = fmt.Errorf("storage: insert symbol %s: %w", sym.Name, err)
			return err
		}
	}
	return nil
}

// DeleteSymbolsForFile deletes all symbols associated with a file.
func (s *Store) DeleteSymbolsForFile(fileID int64) error {
	err := sqlitex.Execute(s.conn,
		`DELETE FROM symbols WHERE file_id = :file_id`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":file_id": fileID},
		})
	if err != nil {
		return fmt.Errorf("storage: delete symbols for file %d: %w", fileID, err)
	}
	return nil
}

// SymbolDetail extends Symbol with its file path and repo path.
type SymbolDetail struct {
	Symbol
	FilePath string
	RepoPath string
}

// GetSymbolByName returns the first symbol with an exact name match, or nil if not found.
func (s *Store) GetSymbolByName(name string) (*SymbolDetail, error) {
	var detail *SymbolDetail
	err := sqlitex.Execute(s.conn,
		`SELECT s.id, s.file_id, s.repo_id, s.name, s.kind, s.signature, s.doc,
		        s.line_start, s.line_end, f.path, r.path
		 FROM symbols s
		 JOIN files f ON f.id = s.file_id
		 JOIN repos r ON r.id = s.repo_id
		 WHERE s.name = :name
		 LIMIT 1`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":name": name},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				detail = &SymbolDetail{
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
					RepoPath: stmt.ColumnText(10),
				}
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: get symbol %s: %w", name, err)
	}
	return detail, nil
}

// Search runs a full-text search over symbols and returns ranked results.
func (s *Store) Search(query string, limit int) ([]SearchResult, error) {
	var results []SearchResult
	err := sqlitex.Execute(s.conn,
		`SELECT s.id, s.file_id, s.repo_id, s.name, s.kind, s.signature, s.doc,
		        s.line_start, s.line_end, f.path, bm25(symbols_fts) AS score
		 FROM symbols_fts
		 JOIN symbols s ON s.id = symbols_fts.rowid
		 JOIN files f   ON f.id = s.file_id
		 WHERE symbols_fts MATCH :query
		 ORDER BY score
		 LIMIT :limit`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":query": query, ":limit": limit},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				results = append(results, SearchResult{
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
					Score:    stmt.ColumnFloat(10),
				})
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: search %q: %w", query, err)
	}
	return results, nil
}
