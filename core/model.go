package core

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const MaxTasks = 50

// Config is a versioned plugin-owned document, never a host settings document.
type Config struct {
	SchemaVersion int                `json:"schema_version"`
	Sites         []SiteConfig       `json:"sites"`
	Downloaders   []DownloaderConfig `json:"downloaders"`
	Tasks         []Task             `json:"tasks"`
}
type SiteConfig struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FeedURL       string `json:"feed_url"`
	CredentialRef string `json:"credential_ref"`
}
type DownloaderConfig struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	BaseURL       string `json:"base_url"`
	CredentialRef string `json:"credential_ref"`
	SavePath      string `json:"save_path"`
	Category      string `json:"category"`
}
type Task struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	SiteID       string `json:"site_id"`
	DownloaderID string `json:"downloader_id"`
	BrushMinutes int    `json:"brush_minutes"`
	CheckMinutes int    `json:"check_minutes"`
	MaxActive    int    `json:"max_active"`
	Notify       bool   `json:"notify"`
	DeleteFiles  bool   `json:"delete_files"`
	Rules        Rules  `json:"rules"`
}
type Rules struct {
	FreeOnly        bool    `json:"free_only"`
	ExcludeHR       bool    `json:"exclude_hr"`
	Include         string  `json:"include"`
	Exclude         string  `json:"exclude"`
	MinBytes        int64   `json:"min_bytes"`
	MaxBytes        int64   `json:"max_bytes"`
	MaxSeeders      int     `json:"max_seeders"`
	MaxAgeMinutes   int     `json:"max_age_minutes"`
	SeedMinutes     int     `json:"seed_minutes"`
	SeedRatio       float64 `json:"seed_ratio"`
	UploadedBytes   int64   `json:"uploaded_bytes"`
	DownloadMinutes int     `json:"download_minutes"`
	DeleteOnFreeEnd bool    `json:"delete_on_free_end"`
}
type Candidate struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	SizeBytes      int64  `json:"size_bytes"`
	Free           *bool  `json:"free"`
	HR             *bool  `json:"hr"`
	DownloadURL    string `json:"download_url"`
	PublishedAt    string `json:"published_at"`
	Seeders        *int   `json:"seeders"`
	FreeUntil      string `json:"free_until"`
	TorrentContent []byte `json:"-"`
}
type Decision struct {
	CandidateID string `json:"candidate_id"`
	Accepted    bool   `json:"accepted"`
	Reason      string `json:"reason"`
}

func DefaultConfig() Config {
	return Config{SchemaVersion: 1, Sites: []SiteConfig{}, Downloaders: []DownloaderConfig{}, Tasks: []Task{}}
}
func (c Config) Validate() error {
	if c.SchemaVersion != 1 {
		return errors.New("unsupported schema_version")
	}
	if len(c.Tasks) > MaxTasks {
		return errors.New("at most 50 tasks are supported")
	}
	seen := map[string]bool{}
	for _, site := range c.Sites {
		if err := validateAdapter(site.ID, site.Name, site.FeedURL); err != nil {
			return fmt.Errorf("site: %w", err)
		}
		if seen["site:"+site.ID] {
			return errors.New("site ID is duplicated")
		}
		seen["site:"+site.ID] = true
		if len(site.CredentialRef) > 200 {
			return errors.New("site credential reference is too long")
		}
	}
	for _, downloader := range c.Downloaders {
		if err := validateAdapter(downloader.ID, downloader.Name, downloader.BaseURL); err != nil {
			return fmt.Errorf("downloader: %w", err)
		}
		if seen["downloader:"+downloader.ID] {
			return errors.New("downloader ID is duplicated")
		}
		seen["downloader:"+downloader.ID] = true
		if len(downloader.CredentialRef) > 200 || len(downloader.SavePath) > 500 || len(downloader.Category) > 200 {
			return errors.New("downloader field is too long")
		}
	}
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
		if t.MaxActive < 0 || t.MaxActive > 1000 {
			return errors.New("max_active must be 0–1000")
		}
		if t.Enabled && (!seen["site:"+t.SiteID] || !seen["downloader:"+t.DownloaderID]) {
			return errors.New("enabled task references an unknown adapter")
		}
		if err := t.Rules.Validate(); err != nil {
			return fmt.Errorf("task %s: %w", t.ID, err)
		}
	}
	return nil
}
func validateAdapter(id, name, endpoint string) error {
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`).MatchString(id) {
		return errors.New("adapter ID is invalid")
	}
	if strings.TrimSpace(name) == "" || len(name) > 160 {
		return errors.New("adapter name is required")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || len(endpoint) > 2000 || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("adapter endpoint must be an HTTP(S) URL without userinfo or fragment")
	}
	return nil
}
func (r Rules) Validate() error {
	if r.MinBytes < 0 || r.MaxBytes < 0 || (r.MaxBytes > 0 && r.MaxBytes < r.MinBytes) {
		return errors.New("invalid size range")
	}
	if r.MaxSeeders < 0 || r.MaxAgeMinutes < 0 || r.SeedMinutes < 0 || r.SeedRatio < 0 || r.UploadedBytes < 0 || r.DownloadMinutes < 0 {
		return errors.New("numeric rule cannot be negative")
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
		case r.MaxSeeders > 0 && c.Seeders == nil:
			reason = "seeders_unknown"
		case r.MaxSeeders > 0 && *c.Seeders > r.MaxSeeders:
			reason = "too_many_seeders"
		case r.MaxAgeMinutes > 0 && candidateAgeExceeded(c.PublishedAt, r.MaxAgeMinutes):
			reason = "too_old_or_unknown"
		}
		seen[c.ID] = true
		out = append(out, Decision{c.ID, reason == "matched", reason})
	}
	return out, nil
}

func candidateAgeExceeded(value string, max int) bool {
	if value == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return true
	}
	age := time.Since(t)
	if age < 0 {
		return false
	}
	return int(age.Minutes()) > max
}
