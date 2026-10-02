package app

import (
	"context"
	"crypto/sha256"
	"dian115-brushflow/adapters"
	"dian115-brushflow/core"
	"dian115-brushflow/host"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type RuntimeState struct {
	Tasks   map[string]*TaskRuntime `json:"tasks"`
	History []RunReport             `json:"history"`
}
type TaskRuntime struct {
	LastBrush string                    `json:"last_brush,omitempty"`
	LastCheck string                    `json:"last_check,omitempty"`
	LastError string                    `json:"last_error,omitempty"`
	Records   map[string]*TorrentRecord `json:"records"`
}
type TorrentRecord struct {
	CandidateID  string `json:"candidate_id"`
	CandidateTag string `json:"candidate_tag"`
	Title        string `json:"title"`
	Hash         string `json:"hash,omitempty"`
	AddedAt      string `json:"added_at"`
	FreeUntil    string `json:"free_until,omitempty"`
	Status       string `json:"status"`
	DeleteReason string `json:"delete_reason,omitempty"`
}
type RunReport struct {
	At      string `json:"at"`
	TaskID  string `json:"task_id"`
	Kind    string `json:"kind"`
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
	Error   string `json:"error,omitempty"`
}

func defaultRuntime() RuntimeState {
	return RuntimeState{Tasks: map[string]*TaskRuntime{}, History: []RunReport{}}
}
func (s Service) runtimeState() (RuntimeState, string, error) {
	r := defaultRuntime()
	rev, found, err := s.Store.Read("runtime", &r)
	if err != nil {
		return r, "", err
	}
	if !found {
		return r, "", nil
	}
	if r.Tasks == nil {
		r.Tasks = map[string]*TaskRuntime{}
	}
	return r, rev, nil
}
func ensureTaskRuntime(r *RuntimeState, id string) *TaskRuntime {
	v := r.Tasks[id]
	if v == nil {
		v = &TaskRuntime{Records: map[string]*TorrentRecord{}}
		r.Tasks[id] = v
	}
	if v.Records == nil {
		v.Records = map[string]*TorrentRecord{}
	}
	return v
}
func taskTag(id string) string { return "brushflow-" + id }
func candidateTag(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "bfid-" + hex.EncodeToString(sum[:8])
}
func shortTitle(value string) string {
	r := []rune(value)
	if len(r) > 160 {
		return string(r[:160])
	}
	return value
}
func hasTag(tags []string, want string) bool {
	for _, v := range tags {
		if v == want {
			return true
		}
	}
	return false
}

func (s Service) resolve(config core.Config, taskID string) (core.Task, core.SiteConfig, core.DownloaderConfig, error) {
	var task core.Task
	found := false
	for _, v := range config.Tasks {
		if v.ID == taskID {
			task = v
			found = true
			break
		}
	}
	if !found {
		return task, core.SiteConfig{}, core.DownloaderConfig{}, errors.New("task not found")
	}
	var site core.SiteConfig
	found = false
	for _, v := range config.Sites {
		if v.ID == task.SiteID {
			site = v
			found = true
			break
		}
	}
	if !found {
		return task, site, core.DownloaderConfig{}, errors.New("site not found")
	}
	var downloader core.DownloaderConfig
	found = false
	for _, v := range config.Downloaders {
		if v.ID == task.DownloaderID {
			downloader = v
			found = true
			break
		}
	}
	if !found {
		return task, site, downloader, errors.New("downloader not found")
	}
	return task, site, downloader, nil
}
func (s Service) saveRuntime(r RuntimeState, rev, key string) error {
	if len(r.History) > 50 {
		r.History = r.History[len(r.History)-50:]
	}
	type oldRecord struct {
		task      *TaskRuntime
		id, added string
	}
	total := 0
	deleted := make([]oldRecord, 0)
	for _, task := range r.Tasks {
		for id, record := range task.Records {
			total++
			if record.Status == "deleted" {
				deleted = append(deleted, oldRecord{task, id, record.AddedAt})
			}
		}
	}
	sort.Slice(deleted, func(i, j int) bool { return deleted[i].added < deleted[j].added })
	for total > 150 && len(deleted) > 0 {
		item := deleted[0]
		deleted = deleted[1:]
		delete(item.task.Records, item.id)
		total--
	}
	if total > 150 {
		return errors.New("active task record limit reached")
	}
	return s.Store.Write("runtime", r, rev, key)
}
func (s Service) brush(ctx context.Context, taskID, invocation string) (RunReport, error) {
	report := RunReport{At: time.Now().UTC().Format(time.RFC3339Nano), TaskID: taskID, Kind: "brush"}
	config, _, err := s.config()
	if err != nil {
		return report, err
	}
	task, siteCfg, downCfg, err := s.resolve(config, taskID)
	if err != nil {
		return report, err
	}
	site := adapters.JSONFeed{Broker: s.Broker, Config: siteCfg}
	down := adapters.QBittorrent{Broker: s.Broker, Config: downCfg}
	candidates, err := site.Candidates(ctx)
	if err != nil {
		return report, err
	}
	torrents, err := down.List(ctx, taskTag(task.ID))
	if err != nil {
		return report, err
	}
	r, rev, err := s.runtimeState()
	if err != nil {
		return report, err
	}
	tr := ensureTaskRuntime(&r, task.ID)
	decisions, err := core.Evaluate(task.Rules, candidates)
	if err != nil {
		return report, err
	}
	accepted := map[string]bool{}
	for _, d := range decisions {
		accepted[d.CandidateID] = d.Accepted
	}
	active := len(torrents)
	for _, candidate := range candidates {
		if !accepted[candidate.ID] || tr.Records[candidate.ID] != nil {
			continue
		}
		if task.MaxActive > 0 && active >= task.MaxActive {
			break
		}
		tag := candidateTag(candidate.ID)
		if strings.HasPrefix(candidate.DownloadURL, "http://") || strings.HasPrefix(candidate.DownloadURL, "https://") {
			response, fetchErr := s.Broker.Do(host.Request{Method: "GET", Path: candidate.DownloadURL, CredentialRef: siteCfg.CredentialRef})
			if fetchErr != nil {
				return report, fetchErr
			}
			if response.Status != 200 {
				return report, fmt.Errorf("torrent download HTTP %d", response.Status)
			}
			content, fetchErr := response.BodyBytes()
			if fetchErr != nil {
				return report, fetchErr
			}
			if len(content) == 0 || len(content) > 4<<20 {
				return report, errors.New("torrent file must be 1 byte–4 MiB")
			}
			candidate.TorrentContent = content
		}
		if err = down.Add(ctx, task, candidate, taskTag(task.ID), tag); err != nil {
			return report, err
		}
		tr.Records[candidate.ID] = &TorrentRecord{CandidateID: candidate.ID, CandidateTag: tag, Title: shortTitle(candidate.Title), AddedAt: time.Now().UTC().Format(time.RFC3339Nano), FreeUntil: candidate.FreeUntil, Status: "added"}
		report.Added++
		active++
	}
	tr.LastBrush = report.At
	tr.LastError = ""
	r.History = append(r.History, report)
	if err = s.saveRuntime(r, rev, "runtime-brush-"+invocation); err != nil {
		return report, err
	}
	if report.Added > 0 {
		s.notify(task, fmt.Sprintf("任务 %s 新增 %d 个种子", task.Name, report.Added), invocation)
	}
	return report, nil
}
func deletionReason(r core.Rules, t adapters.Torrent, record *TorrentRecord, now time.Time) string {
	if record.FreeUntil != "" && r.DeleteOnFreeEnd && t.AmountLeft > 0 {
		if end, err := time.Parse(time.RFC3339, record.FreeUntil); err == nil && now.After(end) {
			return "免费促销已结束且未完成"
		}
	}
	if t.AmountLeft == 0 {
		if r.SeedMinutes > 0 && t.CompletedAt > 0 && now.Unix()-t.CompletedAt >= int64(r.SeedMinutes*60) {
			return "达到做种时间"
		}
		if r.SeedRatio > 0 && t.Ratio >= r.SeedRatio {
			return "达到分享率"
		}
		if r.UploadedBytes > 0 && t.UploadedBytes >= r.UploadedBytes {
			return "达到上传量"
		}
	}
	if t.AmountLeft > 0 && r.DownloadMinutes > 0 && t.AddedAt > 0 && now.Unix()-t.AddedAt >= int64(r.DownloadMinutes*60) {
		return "下载超时"
	}
	return ""
}
func (s Service) check(ctx context.Context, taskID, invocation string) (RunReport, error) {
	report := RunReport{At: time.Now().UTC().Format(time.RFC3339Nano), TaskID: taskID, Kind: "check"}
	config, _, err := s.config()
	if err != nil {
		return report, err
	}
	task, _, downCfg, err := s.resolve(config, taskID)
	if err != nil {
		return report, err
	}
	down := adapters.QBittorrent{Broker: s.Broker, Config: downCfg}
	torrents, err := down.List(ctx, taskTag(task.ID))
	if err != nil {
		return report, err
	}
	r, rev, err := s.runtimeState()
	if err != nil {
		return report, err
	}
	tr := ensureTaskRuntime(&r, task.ID)
	now := time.Now()
	for _, torrent := range torrents {
		for _, record := range tr.Records {
			if !hasTag(torrent.Tags, record.CandidateTag) {
				continue
			}
			record.Hash = torrent.Hash
			if record.Status == "deleted" {
				continue
			}
			reason := deletionReason(task.Rules, torrent, record, now)
			if reason == "" {
				record.Status = "active"
				continue
			}
			key := "delete-" + invocation + "-" + torrent.Hash
			if err = down.Remove(ctx, torrent.Hash, task.DeleteFiles, key); err != nil {
				return report, err
			}
			record.Status = "deleted"
			record.DeleteReason = reason
			report.Deleted++
		}
	}
	tr.LastCheck = report.At
	tr.LastError = ""
	r.History = append(r.History, report)
	if err = s.saveRuntime(r, rev, "runtime-check-"+invocation); err != nil {
		return report, err
	}
	if report.Deleted > 0 {
		s.notify(task, fmt.Sprintf("任务 %s 删除 %d 个种子", task.Name, report.Deleted), invocation)
	}
	return report, nil
}
func (s Service) notify(task core.Task, message, key string) {
	if !task.Notify {
		return
	}
	body, _ := json.Marshal(map[string]any{"level": "info", "title": "站点刷流", "body": message, "dedupe_key": key})
	_, _ = s.Broker.Do(host.Request{Method: "POST", Path: "/api/notifications/plugin", Headers: map[string]string{"content-type": "application/json", "idempotency-key": "notify-" + key}, BodyBase64: host.EncodeBody(body)})
}
func parseTime(value string) time.Time { v, _ := time.Parse(time.RFC3339Nano, value); return v }
func due(last string, minutes int, now time.Time) bool {
	return last == "" || now.Sub(parseTime(last)) >= time.Duration(minutes)*time.Minute
}
func (s Service) runDue(ctx context.Context, now time.Time) error {
	config, _, err := s.config()
	if err != nil {
		return err
	}
	runtime, _, err := s.runtimeState()
	if err != nil {
		return err
	}
	var failures []string
	for _, task := range config.Tasks {
		if !task.Enabled {
			continue
		}
		tr := ensureTaskRuntime(&runtime, task.ID)
		stamp := fmt.Sprintf("resident-%s-%d", task.ID, now.Unix()/60)
		if due(tr.LastBrush, task.BrushMinutes, now) {
			if _, e := s.brush(ctx, task.ID, stamp+"-brush"); e != nil {
				failures = append(failures, task.Name+": brush: "+e.Error())
				s.notify(task, "任务 "+task.Name+" 选种失败："+e.Error(), stamp+"-brush-error")
			}
		}
		if due(tr.LastCheck, task.CheckMinutes, now) {
			if _, e := s.check(ctx, task.ID, stamp+"-check"); e != nil {
				failures = append(failures, task.Name+": check: "+e.Error())
				s.notify(task, "任务 "+task.Name+" 检查失败："+e.Error(), stamp+"-check-error")
			}
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}
