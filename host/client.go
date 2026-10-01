package host

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Caller interface{ Call(string, any, any) error }
type Request struct {
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	Headers       map[string]string `json:"headers,omitempty"`
	BodyBase64    string            `json:"body_base64,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty"`
}
type Response struct {
	Status     int                 `json:"status"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

func (r Response) Header(name string) string {
	for k, v := range r.Headers {
		if strings.EqualFold(k, name) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
func (r Response) Decode(out any) error {
	b, err := base64.RawStdEncoding.DecodeString(r.BodyBase64)
	if err != nil {
		b, err = base64.StdEncoding.DecodeString(r.BodyBase64)
	}
	if err != nil {
		return errors.New("invalid broker body encoding")
	}
	return json.Unmarshal(b, out)
}

type Client struct{ Caller Caller }

func (c Client) Do(req Request) (Response, error) {
	var res Response
	err := c.Caller.Call("host.call", req, &res)
	if err == nil && res.Header("x-dian115-body-truncated") == "true" {
		err = errors.New("broker response truncated")
	}
	return res, err
}

var ErrConflict = errors.New("configuration changed; refresh before saving")

func (c Client) Read(key string, out any) (string, bool, error) {
	res, err := c.Do(Request{Method: "GET", Path: "/api/plugin-runtime/storage/" + key})
	if err != nil {
		return "", false, err
	}
	if res.Status == 404 {
		return "", false, nil
	}
	if res.Status != 200 {
		return "", false, fmt.Errorf("storage read HTTP %d", res.Status)
	}
	var body struct {
		Data struct {
			Value json.RawMessage `json:"value"`
		} `json:"data"`
	}
	if err = res.Decode(&body); err != nil {
		return "", false, err
	}
	if len(body.Data.Value) == 0 {
		return "", false, errors.New("storage value missing")
	}
	if err = json.Unmarshal(body.Data.Value, out); err != nil {
		return "", false, err
	}
	etag := res.Header("etag")
	if etag == "" {
		return "", false, errors.New("storage ETag missing")
	}
	return etag, true, nil
}
func (c Client) Write(key string, value any, etag, idempotency string) error {
	body, err := json.Marshal(map[string]any{"value": value})
	if err != nil {
		return err
	}
	if len(body) > 64<<10 {
		return errors.New("storage document exceeds 64 KiB")
	}
	headers := map[string]string{"content-type": "application/json", "idempotency-key": idempotency}
	if etag != "" {
		headers["if-match"] = etag
	}
	res, err := c.Do(Request{Method: "PUT", Path: "/api/plugin-runtime/storage/" + key, Headers: headers, BodyBase64: base64.RawStdEncoding.EncodeToString(body)})
	if err != nil {
		return err
	}
	if res.Status == 409 || res.Status == 412 {
		return ErrConflict
	}
	if res.Status < 200 || res.Status >= 300 {
		return fmt.Errorf("storage write HTTP %d", res.Status)
	}
	return nil
}
