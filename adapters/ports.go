package adapters

import (
	"context"
	"dian115-brushflow/core"
	"dian115-brushflow/host"
)

type Broker interface {
	Do(host.Request) (host.Response, error)
}
type Site interface {
	Candidates(context.Context) ([]core.Candidate, error)
}
type Torrent struct {
	DownloaderID    string   `json:"downloader_id"`
	Hash            string   `json:"hash"`
	Name            string   `json:"name"`
	Tags            []string `json:"tags"`
	UploadedBytes   int64    `json:"uploaded_bytes"`
	DownloadedBytes int64    `json:"downloaded_bytes"`
	TotalBytes      int64    `json:"total_bytes"`
	AmountLeft      int64    `json:"amount_left"`
	Ratio           float64  `json:"ratio"`
	AddedAt         int64    `json:"added_at"`
	CompletedAt     int64    `json:"completed_at"`
	LastActivity    int64    `json:"last_activity"`
}
type Downloader interface {
	List(context.Context, string) ([]Torrent, error)
	Add(context.Context, core.Task, core.Candidate, string, string) error
	Remove(context.Context, string, bool, string) error
}
