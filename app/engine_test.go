package app

import (
	"context"
	"dian115-brushflow/core"
	"dian115-brushflow/host"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type engineBroker struct {
	torrents      []map[string]any
	adds, deletes int
}

func brokerResponse(status int, value any) host.Response {
	b, _ := json.Marshal(value)
	return host.Response{Status: status, Headers: map[string][]string{}, BodyBase64: base64.RawStdEncoding.EncodeToString(b)}
}
func (b *engineBroker) Do(req host.Request) (host.Response, error) {
	switch {
	case req.Path == "https://site.example/feed":
		yes, no := true, false
		return brokerResponse(200, map[string]any{"items": []core.Candidate{{ID: "keep", Title: "Movie.1080p", SizeBytes: 100, Free: &yes, HR: &no, DownloadURL: "https://site.example/keep.torrent"}, {ID: "paid", Title: "Paid", SizeBytes: 100, Free: &no, HR: &no, DownloadURL: "magnet:?xt=urn:btih:paid"}}}), nil
	case req.Path == "https://site.example/keep.torrent":
		return host.Response{Status: 200, BodyBase64: base64.RawStdEncoding.EncodeToString([]byte("d4:infod4:name4:testee"))}, nil
	case strings.Contains(req.Path, "/api/v2/torrents/info"):
		return brokerResponse(200, b.torrents), nil
	case strings.HasSuffix(req.Path, "/api/v2/torrents/add"):
		if !strings.HasPrefix(req.Headers["content-type"], "multipart/form-data; boundary=") {
			return host.Response{Status: 400}, nil
		}
		b.adds++
		return host.Response{Status: 200, BodyBase64: base64.RawStdEncoding.EncodeToString([]byte("Ok."))}, nil
	case strings.HasSuffix(req.Path, "/api/v2/torrents/delete"):
		b.deletes++
		return host.Response{Status: 200, BodyBase64: base64.RawStdEncoding.EncodeToString([]byte("Ok."))}, nil
	default:
		t := host.Response{Status: 500}
		return t, nil
	}
}
func liveConfig() core.Config {
	return core.Config{SchemaVersion: 1, Sites: []core.SiteConfig{{ID: "site", Name: "Site", FeedURL: "https://site.example/feed"}}, Downloaders: []core.DownloaderConfig{{ID: "qb", Name: "QB", BaseURL: "http://qb:8080"}}, Tasks: []core.Task{{ID: "task", Name: "Task", Enabled: true, SiteID: "site", DownloaderID: "qb", BrushMinutes: 10, CheckMinutes: 5, MaxActive: 2, DeleteFiles: true, Rules: core.Rules{FreeOnly: true, ExcludeHR: true, SeedRatio: 1}}}}
}
func TestBrushAndProtectedDelete(t *testing.T) {
	store := newStore()
	broker := &engineBroker{}
	service := Service{Store: store, Broker: broker}
	config := liveConfig()
	if err := store.Write("config", config, "", "initial-config-key"); err != nil {
		t.Fatal(err)
	}
	report, err := service.brush(context.Background(), "task", "brush-1")
	if err != nil || report.Added != 1 || broker.adds != 1 {
		t.Fatal(report, err)
	}
	runtime, _, err := service.runtimeState()
	if err != nil {
		t.Fatal(err)
	}
	record := runtime.Tasks["task"].Records["keep"]
	if record == nil {
		t.Fatal("record missing")
	}
	broker.torrents = []map[string]any{{"hash": "abc", "name": "Movie.1080p", "tags": "brushflow-task," + record.CandidateTag, "uploaded": 100, "downloaded": 100, "total_size": 100, "amount_left": 0, "ratio": 1.2, "added_on": time.Now().Add(-time.Hour).Unix(), "completion_on": time.Now().Add(-time.Minute).Unix(), "last_activity": time.Now().Unix()}}
	report, err = service.check(context.Background(), "task", "check-1")
	if err != nil || report.Deleted != 1 || broker.deletes != 1 {
		t.Fatal(report, err)
	}
	// Same torrent without the deterministic candidate ownership tag must never be removed.
	broker.deletes = 0
	broker.torrents[0]["tags"] = "brushflow-task"
	report, err = service.check(context.Background(), "task", "check-2")
	if err != nil || report.Deleted != 0 || broker.deletes != 0 {
		t.Fatal("unowned torrent deleted", report, err)
	}
}
