package audit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/store"
)

const genesis = "0000000000000000000000000000000000000000000000000000000000000000"

type memAuditStore struct {
	tipHash string
	tipSeq  int64
	entries []store.AuditLog
}

func (m *memAuditStore) GetAuditChainTip(_ context.Context) (string, int64, error) {
	if m.tipHash == "" {
		return genesis, 0, nil
	}
	return m.tipHash, m.tipSeq, nil
}

func (m *memAuditStore) AppendAuditLog(_ context.Context, e store.AuditLog) (store.AuditLog, error) {
	prev := m.tipHash
	if prev == "" {
		prev = genesis
	}
	if e.Prev_hash != prev || e.Seq != m.tipSeq+1 {
		return store.AuditLog{}, errors.New("audit_chain_mismatch")
	}
	m.entries = append(m.entries, e)
	m.tipHash = e.Entry_hash
	m.tipSeq = e.Seq
	return e, nil
}

func (m *memAuditStore) GetAuditLogByID(_ context.Context, id string) (store.AuditLog, error) {
	for _, e := range m.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return store.AuditLog{}, errors.New("not_found")
}

func TestFilesystemArchiverPut(t *testing.T) {
	dir := t.TempDir()
	a := audit.NewFilesystemArchiver(dir)
	uri, err := a.Put(context.Background(), "audit/2026/07/entry.json", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if uri == "" {
		t.Fatal("expected uri")
	}
	path := filepath.Join(dir, filepath.FromSlash("audit/2026/07/entry.json"))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}

func TestRecorderChainAndPush(t *testing.T) {
	ms := &memAuditStore{}
	ch := make(chan domain.AuditEvent, 4)
	rec := audit.NewRecorder(ms, audit.NewFilesystemArchiver(t.TempDir()), "test-secret", ch, nil)

	if err := rec.Record(context.Background(), audit.Entry{
		Action: "user_suspended", Entity_type: "user", Entity_id: "u1",
		After: map[string]string{"status": "suspended"},
	}); err != nil {
		t.Fatalf("record1: %v", err)
	}
	if err := rec.Record(context.Background(), audit.Entry{
		Action: "user_banned", Entity_type: "user", Entity_id: "u1",
	}); err != nil {
		t.Fatalf("record2: %v", err)
	}
	if len(ms.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ms.entries))
	}
	if ms.entries[0].Seq != 1 || ms.entries[1].Seq != 2 {
		t.Fatalf("bad seq: %d %d", ms.entries[0].Seq, ms.entries[1].Seq)
	}
	if ms.entries[1].Prev_hash != ms.entries[0].Entry_hash {
		t.Fatal("chain broken")
	}
	if ms.entries[0].Signature == "" {
		t.Fatal("missing signature")
	}
	select {
	case ev := <-ch:
		if ev.Action_type != "user_suspended" {
			t.Fatalf("ws event: %v", ev.Action_type)
		}
	default:
		t.Fatal("expected audit ws event")
	}

	res, err := rec.Verify(context.Background(), ms.entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Bad_signature {
		t.Fatal("expected valid signature")
	}
}
