package app

import (
	"dian115-brushflow/core"
	"dian115-brushflow/host"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type memoryStore struct {
	docs      map[string][]byte
	revisions map[string]string
	count     int
	writes    int
}

func newStore() *memoryStore {
	return &memoryStore{docs: map[string][]byte{}, revisions: map[string]string{}}
}
func (m *memoryStore) Read(k string, out any) (string, bool, error) {
	b, ok := m.docs[k]
	if !ok {
		return "", false, nil
	}
	return m.revisions[k], true, json.Unmarshal(b, out)
}
func (m *memoryStore) Write(k string, v any, rev, id string) error {
	if rev != "" && rev != m.revisions[k] {
		return host.ErrConflict
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.count++
	m.writes++
	m.docs[k] = b
	m.revisions[k] = fmt.Sprintf(`"pkv_%d"`, m.count)
	return nil
}
func action(t *testing.T, s Service, id string, input any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"id": id, "input": input})
	in := Input{}
	in.Envelope.Op = "action"
	in.Envelope.InvocationID = "test-invocation"
	in.Envelope.Payload = b
	out, err := s.Invoke(in)
	if err != nil {
		t.Fatal(err)
	}
	return out.(map[string]any)
}
func TestConfigCASAndReload(t *testing.T) {
	store := newStore()
	s := Service{Store: store}
	c := core.DefaultConfig()
	c.Tasks = []core.Task{{ID: "one", Name: "First", BrushMinutes: 10, CheckMinutes: 5}}
	out := action(t, s, "save-config", map[string]any{"config": c, "revision": ""})
	if out["status"] != "succeeded" {
		t.Fatal(out)
	}
	// A different runtime instance must see the persisted document.
	reloaded, rev, err := (Service{Store: store}).config()
	if err != nil || len(reloaded.Tasks) != 1 || rev == "" {
		t.Fatalf("reload failed: %+v %s %v", reloaded, rev, err)
	}
	c.Tasks[0].Name = "stale overwrite"
	out = action(t, s, "save-config", map[string]any{"config": c, "revision": ""})
	if out["status"] != "failed" || store.writes != 1 {
		t.Fatal("stale config overwrote storage")
	}
}
func TestPreviewAndExecutionNeverMutate(t *testing.T) {
	store := newStore()
	s := Service{Store: store}
	c := core.DefaultConfig()
	c.Tasks = []core.Task{{ID: "one", Name: "First", BrushMinutes: 10, CheckMinutes: 5, Rules: core.Rules{FreeOnly: true}}}
	action(t, s, "save-config", map[string]any{"config": c, "revision": ""})
	before := store.writes
	out := action(t, s, "preview", map[string]any{"task_id": "one", "candidates": []core.Candidate{{ID: "x", Title: "Movie", SizeBytes: 1}}})
	if out["status"] != "succeeded" || out["dry_run"] != true {
		t.Fatal(out)
	}
	decisions := out["decisions"].([]core.Decision)
	if decisions[0].Reason != "promotion_unknown" {
		t.Fatal(decisions)
	}
	for _, id := range []string{"run", "check"} {
		if got := action(t, s, id, nil); got["status"] != "failed" {
			t.Fatal(got)
		}
	}
	if store.writes != before {
		t.Fatal("preview or disabled execution mutated storage")
	}
}
func TestStateETagAndHeartbeat(t *testing.T) {
	store := newStore()
	s := Service{Store: store}
	a, err := s.state([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	etag := a.(map[string]any)["etag"]
	b, _ := json.Marshal(map[string]any{"if_none_match": etag})
	same, err := s.state(b)
	if err != nil || same.(map[string]any)["not_modified"] != true {
		t.Fatal(same, err)
	}
	if err = s.Tick(time.Unix(1000, 0)); err != nil {
		t.Fatal(err)
	}
	changed, err := s.state(b)
	if err != nil || changed.(map[string]any)["etag"] == etag {
		t.Fatal("heartbeat did not change ETag")
	}
	if string(store.docs["heartbeat"]) == "" {
		t.Fatal("heartbeat missing")
	}
}
