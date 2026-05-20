// Package storage handles SQLite persistence for repos, files, and symbols.
package storage

import (
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const schema = `
CREATE TABLE IF NOT EXISTS repos (
    id         INTEGER PRIMARY KEY,
    path       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    lang       TEXT NOT NULL,
    indexed_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
    id      INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL REFERENCES repos(id),
    path    TEXT NOT NULL,
    hash    TEXT NOT NULL,
    mtime   INTEGER NOT NULL,
    UNIQUE(repo_id, path)
);

CREATE TABLE IF NOT EXISTS symbols (
    id         INTEGER PRIMARY KEY,
    file_id    INTEGER NOT NULL REFERENCES files(id),
    repo_id    INTEGER NOT NULL REFERENCES repos(id),
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL,
    signature  TEXT,
    doc        TEXT,
    line_start INTEGER NOT NULL,
    line_end   INTEGER NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
    name, signature, doc,
    content=symbols,
    content_rowid=id
);

CREATE TRIGGER IF NOT EXISTS symbols_ai AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, name, signature, doc)
    VALUES (new.id, new.name, new.signature, new.doc);
END;

CREATE TRIGGER IF NOT EXISTS symbols_ad AFTER DELETE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid, name, signature, doc)
    VALUES ('delete', old.id, old.name, old.signature, old.doc);
END;

CREATE TRIGGER IF NOT EXISTS symbols_au AFTER UPDATE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid, name, signature, doc)
    VALUES ('delete', old.id, old.name, old.signature, old.doc);
    INSERT INTO symbols_fts(rowid, name, signature, doc)
    VALUES (new.id, new.name, new.signature, new.doc);
END;

CREATE TABLE IF NOT EXISTS symbols_embeddings (
    symbol_id INTEGER PRIMARY KEY REFERENCES symbols(id),
    embedding BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS call_sites (
    id               INTEGER PRIMARY KEY,
    caller_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
    callee_name      TEXT NOT NULL,
    file_id          INTEGER NOT NULL REFERENCES files(id),
    line             INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS call_sites_caller ON call_sites(caller_symbol_id);
CREATE INDEX IF NOT EXISTS call_sites_callee ON call_sites(callee_name);
`

// Store handles all database operations for Atlas.
type Store struct {
	conn *sqlite.Conn
}

// Open opens (or creates) a SQLite database at path and applies the schema.
func Open(path string) (*Store, error) {
	conn, err := sqlite.OpenConn(path, sqlite.OpenReadWrite, sqlite.OpenCreate)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
	s := &Store{conn: conn}
	if err := sqlitex.ExecScript(conn, schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("storage: apply schema: %w", err)
	}
	return s, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.conn.Close()
}
