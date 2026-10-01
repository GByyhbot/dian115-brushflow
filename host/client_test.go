package host

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
)

type fakeCaller struct {
	response Response
	request  Request
}

func (f *fakeCaller) Call(method string, in any, out any) error {
	if method != "host.call" {
		return errors.New("unexpected method")
	}
	f.request = in.(Request)
	*(out.(*Response)) = f.response
	return nil
}
func TestStorageEnvelopeAndCAS(t *testing.T) {
	f := &fakeCaller{response: Response{Status: 200, Headers: map[string][]string{"ETag": {`"pkv_2"`}}, BodyBase64: base64.RawStdEncoding.EncodeToString([]byte(`{"data":{"key":"config","value":{"n":2},"revision":"pkv_2"}}`))}}
	c := Client{Caller: f}
	var out struct {
		N int `json:"n"`
	}
	rev, found, err := c.Read("config", &out)
	if err != nil || !found || out.N != 2 || rev != `"pkv_2"` {
		t.Fatal(rev, found, out, err)
	}
	f.response.Status = 412
	if err = c.Write("config", out, rev, "stable-idempotency-key"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if f.request.Headers["if-match"] != rev || f.request.Headers["idempotency-key"] != "stable-idempotency-key" {
		t.Fatal(f.request)
	}
	b, err := base64.RawStdEncoding.DecodeString(f.request.BodyBase64)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(b, &body) != nil || body["value"] == nil {
		t.Fatal("missing value wrapper")
	}
}
func TestReadErrorsAreNotEmptyConfig(t *testing.T) {
	for _, res := range []Response{{Status: 500}, {Status: 200, BodyBase64: "invalid"}, {Status: 200, BodyBase64: base64.RawStdEncoding.EncodeToString([]byte(`{"data":{"value":{}}}`))}, {Status: 200, Headers: map[string][]string{"x-dian115-body-truncated": {"true"}}}} {
		c := Client{Caller: &fakeCaller{response: res}}
		var v map[string]any
		if _, _, err := c.Read("config", &v); err == nil {
			t.Fatal("read failure silently accepted")
		}
	}
}
