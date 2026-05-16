package storage

import (
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Repo represents an indexed repository.
type Repo struct {
	ID        int64
	Path      string
	Name      string
	Lang      string
	IndexedAt int64 // unix timestamp
}

// RepoStats extends Repo with file and symbol counts.
type RepoStats struct {
	Repo
	Files   int64
	Symbols int64
}

// UpsertRepo inserts or replaces a repo record and returns its ID.
func (s *Store) UpsertRepo(repo Repo) (int64, error) {
	err := sqlitex.Execute(s.conn,
		`INSERT INTO repos (path, name, lang, indexed_at)
		 VALUES (:path, :name, :lang, :indexed_at)
		 ON CONFLICT(path) DO UPDATE SET
		   name=excluded.name, lang=excluded.lang, indexed_at=excluded.indexed_at`,
		&sqlitex.ExecOptions{
			Named: map[string]any{
				":path":       repo.Path,
				":name":       repo.Name,
				":lang":       repo.Lang,
				":indexed_at": repo.IndexedAt,
			},
		})
	if err != nil {
		return 0, fmt.Errorf("storage: upsert repo %s: %w", repo.Path, err)
	}
	return s.conn.LastInsertRowID(), nil
}

// GetRepoByPath returns a repo by its filesystem path, or nil if not found.
func (s *Store) GetRepoByPath(path string) (*Repo, error) {
	var repo *Repo
	err := sqlitex.Execute(s.conn,
		`SELECT id, path, name, lang, indexed_at FROM repos WHERE path = :path`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":path": path},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				repo = &Repo{
					ID:        stmt.ColumnInt64(0),
					Path:      stmt.ColumnText(1),
					Name:      stmt.ColumnText(2),
					Lang:      stmt.ColumnText(3),
					IndexedAt: stmt.ColumnInt64(4),
				}
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: get repo %s: %w", path, err)
	}
	return repo, nil
}

// ListRepos returns all repos with their file and symbol counts.
func (s *Store) ListRepos() ([]RepoStats, error) {
	var repos []RepoStats
	err := sqlitex.Execute(s.conn,
		`SELECT r.id, r.path, r.name, r.lang, r.indexed_at,
		        COUNT(DISTINCT f.id) AS files,
		        COUNT(DISTINCT sy.id) AS symbols
		 FROM repos r
		 LEFT JOIN files f  ON f.repo_id  = r.id
		 LEFT JOIN symbols sy ON sy.repo_id = r.id
		 GROUP BY r.id
		 ORDER BY r.name`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				repos = append(repos, RepoStats{
					Repo: Repo{
						ID:        stmt.ColumnInt64(0),
						Path:      stmt.ColumnText(1),
						Name:      stmt.ColumnText(2),
						Lang:      stmt.ColumnText(3),
						IndexedAt: stmt.ColumnInt64(4),
					},
					Files:   stmt.ColumnInt64(5),
					Symbols: stmt.ColumnInt64(6),
				})
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: list repos: %w", err)
	}
	return repos, nil
}
