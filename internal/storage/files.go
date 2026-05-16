package storage

import (
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// File represents an indexed source file.
type File struct {
	ID     int64
	RepoID int64
	Path   string
	Hash   string // SHA-1 of file content
	Mtime  int64  // unix timestamp
}

// UpsertFile inserts or replaces a file record and returns its ID.
func (s *Store) UpsertFile(file File) (int64, error) {
	err := sqlitex.Execute(s.conn,
		`INSERT INTO files (repo_id, path, hash, mtime)
		 VALUES (:repo_id, :path, :hash, :mtime)
		 ON CONFLICT(repo_id, path) DO UPDATE SET
		   hash=excluded.hash, mtime=excluded.mtime`,
		&sqlitex.ExecOptions{
			Named: map[string]any{
				":repo_id": file.RepoID,
				":path":    file.Path,
				":hash":    file.Hash,
				":mtime":   file.Mtime,
			},
		})
	if err != nil {
		return 0, fmt.Errorf("storage: upsert file %s: %w", file.Path, err)
	}
	return s.conn.LastInsertRowID(), nil
}

// FileEntry pairs a file path with its repo metadata.
type FileEntry struct {
	FileID   int64
	RepoID   int64
	RepoName string
	RepoPath string
	FilePath string // relative to repo root
}

// ListAllFiles returns all files with their repo metadata.
func (s *Store) ListAllFiles() ([]FileEntry, error) {
	var files []FileEntry
	err := sqlitex.Execute(s.conn,
		`SELECT f.id, f.repo_id, r.name, r.path, f.path
		 FROM files f JOIN repos r ON r.id = f.repo_id
		 ORDER BY r.name, f.path`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				files = append(files, FileEntry{
					FileID:   stmt.ColumnInt64(0),
					RepoID:   stmt.ColumnInt64(1),
					RepoName: stmt.ColumnText(2),
					RepoPath: stmt.ColumnText(3),
					FilePath: stmt.ColumnText(4),
				})
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: list all files: %w", err)
	}
	return files, nil
}

// GetFile returns a file by repo ID and path, or nil if not found.
func (s *Store) GetFile(repoID int64, path string) (*File, error) {
	var file *File
	err := sqlitex.Execute(s.conn,
		`SELECT id, repo_id, path, hash, mtime FROM files
		 WHERE repo_id = :repo_id AND path = :path`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":repo_id": repoID, ":path": path},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				file = &File{
					ID:     stmt.ColumnInt64(0),
					RepoID: stmt.ColumnInt64(1),
					Path:   stmt.ColumnText(2),
					Hash:   stmt.ColumnText(3),
					Mtime:  stmt.ColumnInt64(4),
				}
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: get file %s: %w", path, err)
	}
	return file, nil
}
