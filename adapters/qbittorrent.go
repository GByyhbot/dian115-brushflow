package adapters

import (
	"bytes"
	"context"
	"dian115-brushflow/core"
	"dian115-brushflow/host"
	"fmt"
	"mime/multipart"
	"net/url"
	"strings"
)

type QBittorrent struct {
	Broker Broker
	Config core.DownloaderConfig
}
type qbTorrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	Tags         string  `json:"tags"`
	Uploaded     int64   `json:"uploaded"`
	Downloaded   int64   `json:"downloaded"`
	TotalSize    int64   `json:"total_size"`
	AmountLeft   int64   `json:"amount_left"`
	Ratio        float64 `json:"ratio"`
	AddedOn      int64   `json:"added_on"`
	CompletionOn int64   `json:"completion_on"`
	LastActivity int64   `json:"last_activity"`
}

func (q QBittorrent) endpoint(path string) string {
	return strings.TrimRight(q.Config.BaseURL, "/") + path
}
func (q QBittorrent) List(ctx context.Context, tag string) ([]Torrent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := q.endpoint("/api/v2/torrents/info?tag=" + url.QueryEscape(tag))
	res, err := q.Broker.Do(host.Request{Method: "GET", Path: path, Headers: map[string]string{"accept": "application/json"}, CredentialRef: q.Config.CredentialRef})
	if err != nil {
		return nil, err
	}
	if res.Status != 200 {
		return nil, fmt.Errorf("qBittorrent list HTTP %d", res.Status)
	}
	var raw []qbTorrent
	if err = res.Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Torrent, 0, len(raw))
	for _, v := range raw {
		tags := []string{}
		for _, tag := range strings.Split(v.Tags, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				tags = append(tags, tag)
			}
		}
		out = append(out, Torrent{q.Config.ID, v.Hash, v.Name, tags, v.Uploaded, v.Downloaded, v.TotalSize, v.AmountLeft, v.Ratio, v.AddedOn, v.CompletionOn, v.LastActivity})
	}
	return out, nil
}
func (q QBittorrent) Add(ctx context.Context, task core.Task, candidate core.Candidate, taskTag, candidateTag string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var bodyBytes []byte
	contentType := "application/x-www-form-urlencoded"
	if len(candidate.TorrentContent) > 0 {
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		part, err := writer.CreateFormFile("torrents", candidate.ID+".torrent")
		if err != nil {
			return err
		}
		if _, err = part.Write(candidate.TorrentContent); err != nil {
			return err
		}
		_ = writer.WriteField("tags", taskTag+","+candidateTag)
		if q.Config.SavePath != "" {
			_ = writer.WriteField("savepath", q.Config.SavePath)
		}
		if q.Config.Category != "" {
			_ = writer.WriteField("category", q.Config.Category)
		}
		if err = writer.Close(); err != nil {
			return err
		}
		bodyBytes = buffer.Bytes()
		contentType = writer.FormDataContentType()
	} else {
		form := url.Values{}
		form.Set("urls", candidate.DownloadURL)
		form.Set("tags", taskTag+","+candidateTag)
		if q.Config.SavePath != "" {
			form.Set("savepath", q.Config.SavePath)
		}
		if q.Config.Category != "" {
			form.Set("category", q.Config.Category)
		}
		bodyBytes = []byte(form.Encode())
	}
	res, err := q.Broker.Do(host.Request{Method: "POST", Path: q.endpoint("/api/v2/torrents/add"), Headers: map[string]string{"content-type": contentType}, BodyBase64: host.EncodeBody(bodyBytes), CredentialRef: q.Config.CredentialRef})
	if err != nil {
		return err
	}
	if res.Status != 200 {
		return fmt.Errorf("qBittorrent add HTTP %d", res.Status)
	}
	body, err := res.BodyBytes()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(body)) != "Ok." {
		return fmt.Errorf("qBittorrent rejected torrent: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
func (q QBittorrent) Remove(ctx context.Context, hash string, deleteFiles bool, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	form := url.Values{"hashes": {hash}, "deleteFiles": {fmt.Sprintf("%t", deleteFiles)}}
	res, err := q.Broker.Do(host.Request{Method: "POST", Path: q.endpoint("/api/v2/torrents/delete"), Headers: map[string]string{"content-type": "application/x-www-form-urlencoded"}, BodyBase64: host.EncodeBody([]byte(form.Encode())), CredentialRef: q.Config.CredentialRef})
	if err != nil {
		return err
	}
	if res.Status != 200 {
		return fmt.Errorf("qBittorrent delete HTTP %d", res.Status)
	}
	return nil
}
