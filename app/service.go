package app

import (
	"crypto/sha256"
	"dian115-brushflow/core"
	"dian115-brushflow/host"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Store interface {
	Read(string, any) (string, bool, error)
	Write(string, any, string, string) error
}
type Service struct{ Store Store }
type Input struct {
	Envelope struct {
		Op           string          `json:"op"`
		InvocationID string          `json:"invocation_id"`
		Payload      json.RawMessage `json:"payload"`
	} `json:"envelope"`
}
type Heartbeat struct {
	At           string `json:"at"`
	EnabledTasks int    `json:"enabled_tasks"`
	Status       string `json:"status"`
}

func (s Service) config() (core.Config, string, error) {
	c := core.DefaultConfig()
	rev, found, err := s.Store.Read("config", &c)
	if err != nil {
		return c, "", err
	}
	if !found {
		return core.DefaultConfig(), "", nil
	}
	if err = c.Validate(); err != nil {
		return c, "", err
	}
	return c, rev, nil
}
func (s Service) Invoke(in Input) (any, error) {
	if in.Envelope.InvocationID == "" {
		return nil, errors.New("invocation_id required")
	}
	switch in.Envelope.Op {
	case "state":
		return s.state(in.Envelope.Payload)
	case "action":
		return s.action(in.Envelope.InvocationID, in.Envelope.Payload)
	case "job":
		return map[string]any{"status": "skipped", "message": "No jobs declared; resident only reports readiness in v0.1"}, nil
	case "event":
		return map[string]any{"handled": false}, nil
	default:
		return nil, errors.New("unsupported operation")
	}
}
func (s Service) state(raw json.RawMessage) (any, error) {
	var p struct {
		IfNoneMatch string `json:"if_none_match"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	c, rev, err := s.config()
	if err != nil {
		return nil, err
	}
	var beat Heartbeat
	_, _, err = s.Store.Read("heartbeat", &beat)
	if err != nil {
		return nil, err
	}
	state := map[string]any{"config": c, "config_revision": rev, "heartbeat": beat, "mode": "preview-only", "live_execution": false, "version": "0.1.0"}
	b, _ := json.Marshal(state)
	hash := sha256.Sum256(b)
	version := hex.EncodeToString(hash[:])
	etag := `"` + version + `"`
	if p.IfNoneMatch == etag {
		return map[string]any{"not_modified": true, "etag": etag}, nil
	}
	return map[string]any{"state_version": version, "etag": etag, "state": state}, nil
}
func failed(err error) map[string]any {
	return map[string]any{"status": "failed", "message": err.Error()}
}
func (s Service) action(id string, raw json.RawMessage) (any, error) {
	if len(raw) > 64<<10 {
		return failed(errors.New("action input exceeds 64 KiB")), nil
	}
	var p struct {
		ID    string          `json:"id"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	switch p.ID {
	case "save-config":
		var v struct {
			Config   core.Config `json:"config"`
			Revision string      `json:"revision"`
		}
		if err := json.Unmarshal(p.Input, &v); err != nil {
			return failed(err), nil
		}
		if err := v.Config.Validate(); err != nil {
			return failed(err), nil
		}
		_, rev, err := s.config()
		if err != nil {
			return failed(err), nil
		}
		if rev != v.Revision {
			return failed(host.ErrConflict), nil
		}
		digest := sha256.Sum256([]byte(id))
		key := "save-config-" + hex.EncodeToString(digest[:])
		if err = s.Store.Write("config", v.Config, rev, key); err != nil {
			return failed(err), nil
		}
		return map[string]any{"status": "succeeded", "message": "配置已保存；初版仅支持规则预览"}, nil
	case "preview":
		var v struct {
			TaskID     string           `json:"task_id"`
			Candidates []core.Candidate `json:"candidates"`
		}
		if err := json.Unmarshal(p.Input, &v); err != nil {
			return failed(err), nil
		}
		c, _, err := s.config()
		if err != nil {
			return failed(err), nil
		}
		for _, t := range c.Tasks {
			if t.ID == v.TaskID {
				decisions, err := core.Evaluate(t.Rules, v.Candidates)
				if err != nil {
					return failed(err), nil
				}
				return map[string]any{"status": "succeeded", "dry_run": true, "decisions": decisions}, nil
			}
		}
		return failed(errors.New("task not found; save configuration first")), nil
	case "run", "check", "delete":
		return map[string]any{"status": "skipped", "code": "adapter_unavailable", "message": "初步架构未接通下载器，不执行下载或删种"}, nil
	default:
		return failed(errors.New("unknown action")), nil
	}
}

// Tick only records readiness. It must never pretend to have brushed or checked torrents.
func (s Service) Tick(now time.Time) error {
	c, _, err := s.config()
	if err != nil {
		return err
	}
	n := 0
	for _, t := range c.Tasks {
		if t.Enabled {
			n++
		}
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	return s.Store.Write("heartbeat", Heartbeat{stamp, n, "adapter_unavailable"}, "", fmt.Sprintf("heartbeat-%s", stamp))
}
