package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
	"github.com/google/uuid"
)

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Entry is a domain action to append to the audit log.
type Entry struct {
	Action      string
	Entity_type string
	Entity_id   string
	Before      any
	After       any
}

// Store is the subset of store.Store required by the append-only recorder.
type Store interface {
	GetAuditChainTip(ctx context.Context) (last_hash string, last_seq int64, err error)
	AppendAuditLog(ctx context.Context, entry store.AuditLog) (store.AuditLog, error)
	GetAuditLogByID(ctx context.Context, id string) (store.AuditLog, error)
}

type Recorder struct {
	st       Store
	archiver Archiver
	hmacKey  []byte
	auditCh  chan<- domain.AuditEvent
	log      *slog.Logger
}

func NewRecorder(st Store, archiver Archiver, hmacSecret string, auditCh chan<- domain.AuditEvent, log *slog.Logger) *Recorder {
	if hmacSecret == "" {
		hmacSecret = "dev-audit-hmac"
	}
	if archiver == nil {
		archiver = NewFilesystemArchiver("./var/audit")
	}
	return &Recorder{st: st, archiver: archiver, hmacKey: []byte(hmacSecret), auditCh: auditCh, log: log}
}

func (r *Recorder) Record(ctx context.Context, e Entry) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		entry, err := r.appendOnce(ctx, e)
		if err == nil {
			r.push(entry)
			return nil
		}
		lastErr = err
		if err.Error() != "audit_chain_mismatch" {
			break
		}
	}
	if r.log != nil {
		r.log.Error("audit record failed", "action", e.Action, "err", lastErr)
	}
	return lastErr
}

func (r *Recorder) appendOnce(ctx context.Context, e Entry) (store.AuditLog, error) {
	prevHash, prevSeq, err := r.st.GetAuditChainTip(ctx)
	if err != nil {
		// tip table missing during early boot — treat as genesis
		prevHash, prevSeq = genesisHash, 0
	}
	if prevHash == "" {
		prevHash = genesisHash
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	seq := prevSeq + 1

	before, _ := json.Marshal(e.Before)
	after, _ := json.Marshal(e.After)
	if e.Before == nil {
		before = nil
	}
	if e.After == nil {
		after = nil
	}

	actorID := ""
	if u, ok := ctxkeys.UserFromContext(ctx); ok {
		actorID = u.ID
	}
	reqID, _ := ctx.Value(ctxkeys.RequestIDKey).(string)
	ip, _ := ctx.Value(ctxkeys.IPKey).(string)
	ua, _ := ctx.Value(ctxkeys.UserAgentKey).(string)

	canonical := canonicalPayload{
		ID:          id,
		ActorID:     actorID,
		ActionType:  e.Action,
		EntityType:  e.Entity_type,
		EntityID:    e.Entity_id,
		BeforeState: jsonRaw(before),
		AfterState:  jsonRaw(after),
		RequestID:   reqID,
		Seq:         seq,
		PrevHash:    prevHash,
		IPAddress:   ip,
		UserAgent:   ua,
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	entryHash := hashCanonical(canonical)
	sig := signHMAC(r.hmacKey, entryHash)

	key := fmt.Sprintf("audit/%04d/%02d/%s.json", now.Year(), int(now.Month()), id)
	archiveBody, _ := json.Marshal(map[string]any{
		"entry":     canonical,
		"entry_hash": entryHash,
		"signature": sig,
	})
	uri, archErr := r.archiver.Put(ctx, key, archiveBody)
	if archErr != nil && r.log != nil {
		r.log.Warn("audit archive failed", "key", key, "err", archErr)
		uri = key // still record intended location
	}

	entry := store.AuditLog{
		ID:           id,
		Actor_id:     actorID,
		Action_type:  e.Action,
		Entity_type:  e.Entity_type,
		Entity_id:    e.Entity_id,
		Before_state: before,
		After_state:  after,
		Request_id:   reqID,
		Seq:          seq,
		Prev_hash:    prevHash,
		Entry_hash:   entryHash,
		Signature:    sig,
		Archive_uri:  uri,
		IP_address:   ip,
		User_agent:   ua,
		Created_at:   now,
	}
	return r.st.AppendAuditLog(ctx, entry)
}

func (r *Recorder) push(entry store.AuditLog) {
	ws.TryPush(r.auditCh, domain.AuditEvent{
		ID:          entry.ID,
		Actor_id:    entry.Actor_id,
		Action_type: entry.Action_type,
		Entity_type: entry.Entity_type,
		Entity_id:   entry.Entity_id,
		Request_id:  entry.Request_id,
		Seq:         entry.Seq,
		Prev_hash:   entry.Prev_hash,
		Entry_hash:  entry.Entry_hash,
		Signature:   entry.Signature,
		Archive_uri: entry.Archive_uri,
		Timestamp:   entry.Created_at,
		Before:      json.RawMessage(entry.Before_state),
		After:       json.RawMessage(entry.After_state),
	}, r.log, "audit_ch full, event dropped", "audit_id", entry.ID)
}

type VerifyResult struct {
	Valid          bool   `json:"valid"`
	Broken_chain   bool   `json:"broken_chain"`
	Bad_signature  bool   `json:"bad_signature"`
	Expected_hash  string `json:"expected_hash,omitempty"`
	Expected_sig   string `json:"expected_sig,omitempty"`
}

func (r *Recorder) Verify(ctx context.Context, id string) (VerifyResult, error) {
	entry, err := r.st.GetAuditLogByID(ctx, id)
	if err != nil {
		return VerifyResult{}, err
	}
	expectedSig := signHMAC(r.hmacKey, entry.Entry_hash)
	res := VerifyResult{
		Expected_hash: entry.Entry_hash,
		Expected_sig:  expectedSig,
	}
	if !hmac.Equal([]byte(entry.Signature), []byte(expectedSig)) {
		res.Bad_signature = true
	}
	if entry.Prev_hash == "" && entry.Seq > 1 {
		res.Broken_chain = true
	}
	res.Valid = !res.Bad_signature && !res.Broken_chain
	return res, nil
}

type canonicalPayload struct {
	ID          string          `json:"id"`
	ActorID     string          `json:"actor_id"`
	ActionType  string          `json:"action_type"`
	EntityType  string          `json:"entity_type"`
	EntityID    string          `json:"entity_id"`
	BeforeState json.RawMessage `json:"before_state"`
	AfterState  json.RawMessage `json:"after_state"`
	RequestID   string          `json:"request_id"`
	Seq         int64           `json:"seq"`
	PrevHash    string          `json:"prev_hash"`
	IPAddress   string          `json:"ip_address"`
	UserAgent   string          `json:"user_agent"`
	CreatedAt   string          `json:"created_at"`
}

func jsonRaw(b []byte) json.RawMessage {
	if len(b) == 0 || string(b) == "null" {
		return json.RawMessage("null")
	}
	return json.RawMessage(b)
}

func hashCanonical(p canonicalPayload) string {
	b, _ := json.Marshal(p)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func signHMAC(key []byte, entryHash string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(entryHash))
	return hex.EncodeToString(mac.Sum(nil))
}
