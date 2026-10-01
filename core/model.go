package core

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const MaxTasks = 50

// Config is a versioned plugin-owned document, never a host settings document.
type Config struct {
	SchemaVersion int    `json:"schema_version"`
	Tasks         []Task `json:"tasks"`
}
type Task struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	SiteID       string `json:"site_id"`
	DownloaderID string `json:"downloader_id"`
	BrushMinutes int    `json:"brush_minutes"`
	CheckMinutes int    `json:"check_minutes"`
	Rules        Rules  `json:"rules"`
}
type Rules struct {
	FreeOnly  bool   `json:"free_only"`
	ExcludeHR bool   `json:"exclude_hr"`
	Include   string `json:"include"`
	Exclude   string `json:"exclude"`
	MinBytes  int64  `json:"min_bytes"`
	MaxBytes  int64  `json:"max_bytes"`
}
type Candidate struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SizeBytes int64  `json:"size_bytes"`
	Free      *bool  `json:"free"`
	HR        *bool  `json:"hr"`
}
type Decision struct {
	CandidateID string `json:"candidate_id"`
	Accepted    bool   `json:"accepted"`
	Reason      string `json:"reason"`
}

func DefaultConfig() Config { return Config{SchemaVersion: 1, Tasks: []Task{}} }
func (c Config) Validate() error {
	if c.SchemaVersion != 1 {
		return errors.New("unsupported schema_version")
	}
	if len(c.Tasks) > MaxTasks {
		return errors.New("at most 50 tasks are supported")
	}
	seen := map[string]bool{}
	for _, t := range c.Tasks {
		if !regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`).MatchString(t.ID) || seen[t.ID] {
			return errors.New("task ID is invalid or duplicated")
		}
		seen[t.ID] = true
		if strings.TrimSpace(t.Name) == "" || len(t.Name) > 160 {
			return errors.New("task name is required (max 160 bytes)")
		}
		if len(t.SiteID) > 128 || len(t.DownloaderID) > 128 {
			return errors.New("adapter reference is too long")
		}
		if t.Enabled && (t.SiteID == "" || t.DownloaderID == "") {
			return errors.New("enabled task requires site and downloader references")
		}
		if t.BrushMinutes < 1 || t.BrushMinutes > 1440 || t.CheckMinutes < 1 || t.CheckMinutes > 1440 {
			return errors.New("interval must be 1–1440 minutes")
		}
		if err := t.Rules.Validate(); err != nil {
			return fmt.Errorf("task %s: %w", t.ID, err)
		}
	}
	return nil
}
func (r Rules) Validate() error {
	if r.MinBytes < 0 || r.MaxBytes < 0 || (r.MaxBytes > 0 && r.MaxBytes < r.MinBytes) {
		return errors.New("invalid size range")
	}
	for _, pattern := range []string{r.Include, r.Exclude} {
		if len(pattern) > 512 {
			return errors.New("regex exceeds 512 bytes")
		}
		if _, err := regexp.Compile("(?i)" + pattern); err != nil {
			return errors.New("invalid Go RE2 regular expression")
		}
	}
	return nil
}

// Evaluate never treats missing promotion or H&R data as a positive match.
func Evaluate(r Rules, candidates []Candidate) ([]Decision, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if len(candidates) > 100 {
		return nil, errors.New("preview accepts at most 100 candidates")
	}
	include, _ := regexp.Compile("(?i)" + r.Include)
	exclude, _ := regexp.Compile("(?i)" + r.Exclude)
	out := make([]Decision, 0, len(candidates))
	seen := map[string]bool{}
	for _, c := range candidates {
		reason := "matched"
		switch {
		case c.ID == "" || len(c.ID) > 128 || c.Title == "" || len(c.Title) > 2048 || c.SizeBytes < 0:
			reason = "invalid_candidate"
		case seen[c.ID]:
			reason = "duplicate"
		case r.FreeOnly && c.Free == nil:
			reason = "promotion_unknown"
		case r.FreeOnly && !*c.Free:
			reason = "not_free"
		case r.ExcludeHR && c.HR == nil:
			reason = "hr_unknown"
		case r.ExcludeHR && *c.HR:
			reason = "hit_and_run"
		case r.Include != "" && !include.MatchString(c.Title):
			reason = "include_mismatch"
		case r.Exclude != "" && exclude.MatchString(c.Title):
			reason = "excluded"
		case c.SizeBytes < r.MinBytes || (r.MaxBytes > 0 && c.SizeBytes > r.MaxBytes):
			reason = "size_out_of_range"
		}
		seen[c.ID] = true
		out = append(out, Decision{c.ID, reason == "matched", reason})
	}
	return out, nil
}
