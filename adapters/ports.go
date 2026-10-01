// Package adapters defines the boundary between strategy and external systems.
// v0.1 deliberately ships no live download/delete implementation.
package adapters

import (
	"context"
	"dian115-brushflow/core"
	"errors"
)

var ErrNotImplemented = errors.New("live adapter is not implemented in v0.1")

type Site interface {
	Candidates(context.Context, string) ([]core.Candidate, error)
}
type Torrent struct {
	DownloaderID    string
	Hash            string
	OwnerTaskID     string
	UploadedBytes   int64
	DownloadedBytes int64
	SeedingSeconds  int64
}
type Downloader interface {
	List(context.Context) ([]Torrent, error)
	Add(context.Context, core.Task, core.Candidate, string) (string, error)
	Remove(context.Context, Torrent, bool, string) error
}

// Ownership must be checked against persisted downloader ID, hash and task ID,
// not a human-readable torrent name. Implementations must reconcile timeouts.
type UnavailableDownloader struct{}

func (UnavailableDownloader) List(context.Context) ([]Torrent, error) { return nil, ErrNotImplemented }
func (UnavailableDownloader) Add(context.Context, core.Task, core.Candidate, string) (string, error) {
	return "", ErrNotImplemented
}
func (UnavailableDownloader) Remove(context.Context, Torrent, bool, string) error {
	return ErrNotImplemented
}
