package storage

import (
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// CallSite records a single call from one symbol to another within a file.
type CallSite struct {
	ID             int64
	CallerSymbolID int64
	CalleeName     string
	FileID         int64
	Line           int64
}

// InsertCallSites inserts a batch of call sites within a single transaction.
func (s *Store) InsertCallSites(sites []CallSite) error {
	if len(sites) == 0 {
		return nil
	}
	endFn := sqlitex.Transaction(s.conn)
	var err error
	defer endFn(&err)

	for _, cs := range sites {
		err = sqlitex.Execute(s.conn,
			`INSERT INTO call_sites (caller_symbol_id, callee_name, file_id, line)
			 VALUES (:caller_symbol_id, :callee_name, :file_id, :line)`,
			&sqlitex.ExecOptions{
				Named: map[string]any{
					":caller_symbol_id": cs.CallerSymbolID,
					":callee_name":      cs.CalleeName,
					":file_id":          cs.FileID,
					":line":             cs.Line,
				},
			})
		if err != nil {
			err = fmt.Errorf("storage: insert call site %s: %w", cs.CalleeName, err)
			return err
		}
	}
	return nil
}

// DeleteCallSitesForFile deletes all call sites associated with a file.
func (s *Store) DeleteCallSitesForFile(fileID int64) error {
	err := sqlitex.Execute(s.conn,
		`DELETE FROM call_sites WHERE file_id = :file_id`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":file_id": fileID},
		})
	if err != nil {
		return fmt.Errorf("storage: delete call sites for file %d: %w", fileID, err)
	}
	return nil
}

// GetCallees returns the distinct names of symbols called by callerSymbolID.
func (s *Store) GetCallees(callerSymbolID int64) ([]string, error) {
	var names []string
	err := sqlitex.Execute(s.conn,
		`SELECT DISTINCT callee_name FROM call_sites
		 WHERE caller_symbol_id = :id
		 ORDER BY callee_name`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":id": callerSymbolID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				names = append(names, stmt.ColumnText(0))
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: get callees of %d: %w", callerSymbolID, err)
	}
	return names, nil
}

// GetCallers returns the distinct names of symbols that call calleeName.
func (s *Store) GetCallers(calleeName string) ([]string, error) {
	var names []string
	err := sqlitex.Execute(s.conn,
		`SELECT DISTINCT sy.name
		 FROM call_sites cs
		 JOIN symbols sy ON sy.id = cs.caller_symbol_id
		 WHERE cs.callee_name = :callee_name
		 ORDER BY sy.name`,
		&sqlitex.ExecOptions{
			Named: map[string]any{":callee_name": calleeName},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				names = append(names, stmt.ColumnText(0))
				return nil
			},
		})
	if err != nil {
		return nil, fmt.Errorf("storage: get callers of %s: %w", calleeName, err)
	}
	return names, nil
}
