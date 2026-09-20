package mcp

// GkillClient のテスト用の HTTP モック。vitest の vi.mock("undici") + fetch.mockResolvedValue に相当する。
// http.Client の Transport を差し替えるので、実際のポートは開かない。

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fetchCall struct {
	URL    string
	Method string
	Header http.Header
	Body   string
}

type fetchMock struct {
	calls []fetchCall
	impl  func(callCount int, call fetchCall) (*http.Response, error)
}

func (m *fetchMock) RoundTrip(req *http.Request) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		body = string(raw)
	}
	call := fetchCall{URL: req.URL.String(), Method: req.Method, Header: req.Header.Clone(), Body: body}
	m.calls = append(m.calls, call)
	if m.impl == nil {
		return nil, errors.New("fetch is not mocked")
	}
	return m.impl(len(m.calls), call)
}

// installFetchMock はクライアントの Transport をモックに差し替える。
func installFetchMock(client *GkillClient) *fetchMock {
	m := &fetchMock{}
	client.HTTPClient.Transport = m
	return m
}

func httpJSON(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func httpRaw(status int, contentType string, body []byte) *http.Response {
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(string(body)))}
}

// mockFetchOk は fetch.mockResolvedValue({ok:true, json: () => body})。
func (m *fetchMock) mockFetchOk(body string) {
	m.impl = func(_ int, _ fetchCall) (*http.Response, error) { return httpJSON(200, body), nil }
}

// mockFetchError は fetch.mockResolvedValue({ok:false, status, json: () => body})。
func (m *fetchMock) mockFetchError(status int, body string) {
	m.impl = func(_ int, _ fetchCall) (*http.Response, error) { return httpJSON(status, body), nil }
}

// mockFetchReject は fetch.mockRejectedValue(err)。
func (m *fetchMock) mockFetchReject(err error) {
	m.impl = func(_ int, _ fetchCall) (*http.Response, error) { return nil, err }
}

// clearGkillEnv は gkill 関連の環境変数を全部空にする（beforeEach の delete process.env.X）。
func clearGkillEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GKILL_BASE_URL",
		"GKILL_USER",
		"GKILL_PASSWORD_SHA256",
		"GKILL_PASSWORD",
		"GKILL_LOCALE",
		"GKILL_SESSION_ID",
		"GKILL_INSECURE",
		"GKILL_FETCH_TIMEOUT_MS",
	} {
		t.Setenv(key, "")
	}
}
