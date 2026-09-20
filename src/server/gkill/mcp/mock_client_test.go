package mcp

// テスト用の gkill クライアント。vitest の `client: { callApi: vi.fn(...) }` に相当する。
// 呼び出しは calls に記録され、各メソッドは関数フィールドで差し替える。

import (
	"context"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

type apiCall struct {
	Pathname     string
	Body         *jsonobj.Object
	RequiresAuth bool
	SID          string
}

type mockClient struct {
	callApi       func(pathname string, body *jsonobj.Object, requiresAuth bool, sid string) (*jsonobj.Object, error)
	fetchFile     func(filePath string, sid string) (*FileResponse, error)
	login         func() (string, error)
	post          func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error)
	userID        string
	defaultLocale string
	calls         []apiCall
	fetchCalls    []apiCall
	// once は mockResolvedValueOnce / mockRejectedValueOnce のキュー（先頭から消費）。
	once []func() (*jsonobj.Object, error)
}

// mockResolvedValueOnce は次の CallApi にこの応答を返す（vitest の mockResolvedValueOnce）。
func (m *mockClient) mockResolvedValueOnce(response *jsonobj.Object) *mockClient {
	m.once = append(m.once, func() (*jsonobj.Object, error) { return response, nil })
	return m
}

// mockRejectedValueOnce は次の CallApi をこのエラーで失敗させる。
func (m *mockClient) mockRejectedValueOnce(err error) *mockClient {
	m.once = append(m.once, func() (*jsonobj.Object, error) { return nil, err })
	return m
}

// pathsCalled は CallApi の呼び出し先を順に返す。
func (m *mockClient) pathsCalled() []string {
	out := []string{}
	for _, call := range m.calls {
		out = append(out, call.Pathname)
	}
	return out
}

var _ GkillAPI = (*mockClient)(nil)

func (m *mockClient) CallApi(_ context.Context, pathname string, body *jsonobj.Object, requiresAuth bool, sid string) (*jsonobj.Object, error) {
	m.calls = append(m.calls, apiCall{Pathname: pathname, Body: body, RequiresAuth: requiresAuth, SID: sid})
	if len(m.once) > 0 {
		next := m.once[0]
		m.once = m.once[1:]
		return next()
	}
	if m.callApi == nil {
		return obj(), nil
	}
	return m.callApi(pathname, body, requiresAuth, sid)
}

func (m *mockClient) FetchFile(_ context.Context, filePath string, sid string) (*FileResponse, error) {
	m.fetchCalls = append(m.fetchCalls, apiCall{Pathname: filePath, SID: sid})
	if m.fetchFile == nil {
		return nil, Errorf("fetchFile is not mocked")
	}
	return m.fetchFile(filePath, sid)
}

func (m *mockClient) Login(_ context.Context) (string, error) {
	if m.login == nil {
		return "", Errorf("login is not mocked")
	}
	return m.login()
}

func (m *mockClient) Post(_ context.Context, pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
	if m.post == nil {
		return nil, Errorf("post is not mocked")
	}
	return m.post(pathname, body)
}

func (m *mockClient) HasErrors(response *jsonobj.Object) bool {
	return responseHasErrors(response)
}

func (m *mockClient) DefaultLocale() string {
	if m.defaultLocale == "" {
		return "ja"
	}
	return m.defaultLocale
}

func (m *mockClient) UserID() string { return m.userID }

// lastCall は直近の callApi 呼び出し。
func (m *mockClient) lastCall() apiCall {
	if len(m.calls) == 0 {
		return apiCall{}
	}
	return m.calls[len(m.calls)-1]
}

// htmlToTextObj は HtmlToTextResult を toEqual 比較用のオブジェクトにする。
func htmlToTextObj(r HtmlToTextResult) *jsonobj.Object {
	return obj("text", r.Text, "truncated", r.Truncated)
}
