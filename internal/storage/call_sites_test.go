package storage

import (
	"testing"
)

func seedCallSiteFixture(t *testing.T, s *Store) (callerID, calleeFileID int64) {
	t.Helper()
	repoID, _ := s.UpsertRepo(Repo{Path: "/tmp/r", Name: "r", Lang: "go", IndexedAt: 1})
	fileID, _ := s.UpsertFile(File{RepoID: repoID, Path: "main.go", Hash: "x", Mtime: 1})

	syms := []Symbol{
		{FileID: fileID, RepoID: repoID, Name: "Run", Kind: "function", LineStart: 1, LineEnd: 5},
		{FileID: fileID, RepoID: repoID, Name: "Add", Kind: "function", LineStart: 7, LineEnd: 10},
	}
	if err := s.InsertSymbols(syms); err != nil {
		t.Fatalf("InsertSymbols: %v", err)
	}

	inserted, err := s.GetSymbolsByFileID(fileID)
	if err != nil || len(inserted) < 2 {
		t.Fatalf("GetSymbolsByFileID: got %d symbols, err=%v", len(inserted), err)
	}
	// Run is first (line 1), Add is second (line 7).
	var runID int64
	for _, sym := range inserted {
		if sym.Name == "Run" {
			runID = sym.ID
		}
	}
	return runID, fileID
}

func TestInsertAndGetCallSites(t *testing.T) {
	s := openTestStore(t)
	callerID, fileID := seedCallSiteFixture(t, s)

	sites := []CallSite{
		{CallerSymbolID: callerID, CalleeName: "Add", FileID: fileID, Line: 3},
	}
	if err := s.InsertCallSites(sites); err != nil {
		t.Fatalf("InsertCallSites: %v", err)
	}

	callees, err := s.GetCallees(callerID)
	if err != nil {
		t.Fatalf("GetCallees: %v", err)
	}
	if len(callees) != 1 || callees[0] != "Add" {
		t.Errorf("GetCallees = %v, want [Add]", callees)
	}

	callers, err := s.GetCallers("Add")
	if err != nil {
		t.Fatalf("GetCallers: %v", err)
	}
	if len(callers) != 1 || callers[0] != "Run" {
		t.Errorf("GetCallers = %v, want [Run]", callers)
	}
}

func TestGetCallees_empty(t *testing.T) {
	s := openTestStore(t)
	callerID, fileID := seedCallSiteFixture(t, s)
	_ = fileID

	callees, err := s.GetCallees(callerID)
	if err != nil {
		t.Fatalf("GetCallees: %v", err)
	}
	if len(callees) != 0 {
		t.Errorf("GetCallees = %v, want []", callees)
	}
}

func TestGetCallers_empty(t *testing.T) {
	s := openTestStore(t)

	callers, err := s.GetCallers("NonExistent")
	if err != nil {
		t.Fatalf("GetCallers: %v", err)
	}
	if len(callers) != 0 {
		t.Errorf("GetCallers = %v, want []", callers)
	}
}

func TestDeleteCallSitesForFile(t *testing.T) {
	s := openTestStore(t)
	callerID, fileID := seedCallSiteFixture(t, s)

	sites := []CallSite{
		{CallerSymbolID: callerID, CalleeName: "Add", FileID: fileID, Line: 3},
	}
	if err := s.InsertCallSites(sites); err != nil {
		t.Fatalf("InsertCallSites: %v", err)
	}

	if err := s.DeleteCallSitesForFile(fileID); err != nil {
		t.Fatalf("DeleteCallSitesForFile: %v", err)
	}

	callees, err := s.GetCallees(callerID)
	if err != nil {
		t.Fatalf("GetCallees: %v", err)
	}
	if len(callees) != 0 {
		t.Errorf("GetCallees after delete = %v, want []", callees)
	}
}

func TestInsertCallSites_empty(t *testing.T) {
	s := openTestStore(t)
	if err := s.InsertCallSites(nil); err != nil {
		t.Errorf("InsertCallSites(nil) error: %v", err)
	}
	if err := s.InsertCallSites([]CallSite{}); err != nil {
		t.Errorf("InsertCallSites([]) error: %v", err)
	}
}
