package adapters

import (
	"context"
	"dian115-brushflow/core"
	"dian115-brushflow/host"
	"errors"
	"fmt"
	"strings"
)

// JSONFeed response: {"items":[Candidate...]}; DIAN115 injects credentials.
type JSONFeed struct {
	Broker Broker
	Config core.SiteConfig
}

func (s JSONFeed) Candidates(ctx context.Context) ([]core.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := s.Broker.Do(host.Request{Method: "GET", Path: s.Config.FeedURL, Headers: map[string]string{"accept": "application/json"}, CredentialRef: s.Config.CredentialRef})
	if err != nil {
		return nil, err
	}
	if res.Status != 200 {
		return nil, fmt.Errorf("site feed HTTP %d", res.Status)
	}
	var body struct {
		Items []core.Candidate `json:"items"`
	}
	if err = res.Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Items) > 500 {
		return nil, errors.New("site feed exceeds 500 candidates")
	}
	for _, item := range body.Items {
		if len(item.DownloadURL) > 4096 || !(strings.HasPrefix(item.DownloadURL, "magnet:?") || strings.HasPrefix(item.DownloadURL, "http://") || strings.HasPrefix(item.DownloadURL, "https://")) {
			return nil, errors.New("site candidate download_url must be magnet or HTTP(S)")
		}
	}
	return body.Items, nil
}
