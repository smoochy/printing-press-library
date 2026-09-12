package store

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestBatchRollbackReportsNoCommittedRows(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.DB().Exec(`CREATE TRIGGER reject_second_resource BEFORE INSERT ON resources WHEN NEW.id = 'second' BEGIN SELECT RAISE(ABORT, 'injected batch failure'); END`); err != nil {
		t.Fatal(err)
	}
	stored, missed, err := s.UpsertBatch("rollback_fixture", []json.RawMessage{json.RawMessage(`{"id":"first"}`), json.RawMessage(`{"id":"second"}`)})
	if err == nil || stored != 0 || missed != 0 {
		t.Fatalf("stored=%d missed=%d err=%v; want zero counts and an error", stored, missed, err)
	}
	var count int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM resources WHERE resource_type = 'rollback_fixture'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back batch left %d rows", count)
	}
}
