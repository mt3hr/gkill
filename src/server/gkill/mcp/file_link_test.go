package mcp

// リモート向け file-URL 配信経路の検査 (Part 3):
//   - FileLinkStore (mint/resolve/expiry)
//   - BuildToolResult がリモートクライアントへ file_url を注入すること
//   - HttpTransport.HandleFileServe がトークンでバイト列を配ること

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// FileLinkStore
// ---------------------------------------------------------------------------
func TestFileLinkStore(t *testing.T) {
	t.Run("mint then resolve returns the original file info", func(t *testing.T) {
		store := NewFileLinkStore(0)
		token := store.Mint(FileLink{
			GkillSessionID: "sess-1",
			RepName:        "Files",
			FileName:       "a/b.png",
			IsImage:        true,
		})
		expectTrue(t, len(token) > 16, "token too short: %q", token)

		link, ok := store.Resolve(token)
		expectTrue(t, ok, "token did not resolve")
		expectEqual(t, fileLinkObj(link), obj(
			"gkillSessionId", "sess-1",
			"repName", "Files",
			"fileName", "a/b.png",
			"isImage", true,
		))
	})

	t.Run("unknown token resolves to null", func(t *testing.T) {
		store := NewFileLinkStore(0)
		_, ok := store.Resolve("does-not-exist")
		expectTrue(t, !ok, "unknown token resolved")
	})

	t.Run("expired token resolves to null and is dropped", func(t *testing.T) {
		store := NewFileLinkStore(0)
		token := store.MintWithTTL(FileLink{GkillSessionID: "s", RepName: "r", FileName: "f", IsImage: false}, -time.Millisecond)
		_, ok := store.Resolve(token)
		expectTrue(t, !ok, "expired token resolved")
		// resolve dropped the expired entry
		expectTrue(t, !store.Has(token), "expired entry was kept")
	})

	t.Run("distinct tokens do not collide", func(t *testing.T) {
		store := NewFileLinkStore(0)
		t1 := store.Mint(FileLink{GkillSessionID: "s1", RepName: "r1", FileName: "f1", IsImage: false})
		t2 := store.Mint(FileLink{GkillSessionID: "s2", RepName: "r2", FileName: "f2", IsImage: true})
		expectTrue(t, t1 != t2, "tokens collide")
		l1, _ := store.Resolve(t1)
		l2, _ := store.Resolve(t2)
		expectEqual(t, l1.RepName, "r1")
		expectEqual(t, l2.RepName, "r2")
	})
}

func fileLinkObj(link FileLink) *jsonobj.Object {
	return obj("gkillSessionId", link.GkillSessionID, "repName", link.RepName, "fileName", link.FileName, "isImage", link.IsImage)
}

// ---------------------------------------------------------------------------
// buildToolResult — file_url injection for remote clients
// ---------------------------------------------------------------------------
func TestBuildToolResultFileURLInjection(t *testing.T) {
	type fixture struct {
		server *Server
		store  *FileLinkStore
	}
	setup := func() fixture {
		server := NewReadServer(&mockClient{defaultLocale: "ja"}, nil)
		store := NewFileLinkStore(0)
		server.IsLocalTransport = false
		server.FileLinkContext = &FileLinkContext{PublicBaseURL: "https://mcp.example.test", Store: store}
		server.CurrentSessionID = "sess-xyz"
		return fixture{server, store}
	}

	// include_file_urls:true のときハンドラが立てる印（ADR-0630）。無ければリモートでも鋳造しない。
	idfResult := func(extra *jsonobj.Object, requested bool) *jsonobj.Object {
		payloadEntry := obj(
			"kind", "idf",
			"rep_name", "Files",
			"file_name", "photo.png",
			"is_image", true,
			"file_path", `C:\Users\user\gkill\photo.png`,
		).Merge(extra)
		payload := obj("kyous", arr(obj("data_type", "idf", "payload", payloadEntry)))
		if requested {
			payload.SetMeta(MintFileLinksMark, true)
		}
		return payload
	}
	payloadOf := func(t *testing.T, result *jsonobj.Object) *jsonobj.Object {
		t.Helper()
		return objAt(t, arrAt(t, result, "structuredContent", "kyous")[0], "payload")
	}

	t.Run("remote client without include_file_urls gets no file_url, and file_path is still removed", func(t *testing.T) {
		f := setup()
		result := f.server.BuildToolResult("gkill_get_kyous", idfResult(obj(), false), false, nil)
		p := payloadOf(t, result)

		expectTrue(t, !p.Has("file_path"), "file_path kept")
		expectTrue(t, !p.Has("file_url"), "file_url minted")
		expectTrue(t, !p.Has("file_url_expires_at"), "file_url_expires_at set")
		expectTrue(t, f.store.Len() == 0, "%d links minted", f.store.Len())
		// 印は meta なので JSON には出ない
		mustNotContain(t, jsonobj.MarshalString(result.Value("structuredContent")), "mint_file_links")
	})

	t.Run("image payload gets a thumbnail file_url plus a full-size file_url_full, and file_path is removed", func(t *testing.T) {
		f := setup()
		result := f.server.BuildToolResult("gkill_get_kyous", idfResult(obj(), true), false, nil)
		p := payloadOf(t, result)
		expires, err := time.Parse(time.RFC3339Nano, strAt(t, p, "file_url_expires_at"))
		expectNoError(t, err)
		expectTrue(t, expires.After(time.Now()), "expiry is not in the future")

		expectTrue(t, !p.Has("file_path"), "file_path kept")
		mustMatch(t, strAt(t, p, "file_url"), `^https://mcp\.example\.test/files/[0-9a-f]+\?thumb=1024x1024$`)
		mustMatch(t, strAt(t, p, "file_url_full"), `^https://mcp\.example\.test/files/[0-9a-f]+$`)

		// the minted token resolves back to the file, carrying the OAuth session
		full, err := url.Parse(strAt(t, p, "file_url_full"))
		expectNoError(t, err)
		parts := strings.Split(full.Path, "/")
		token := parts[len(parts)-1]
		link, ok := f.store.Resolve(token)
		expectTrue(t, ok, "token did not resolve")
		expectContains(t, fileLinkObj(link), obj(
			"gkillSessionId", "sess-xyz",
			"repName", "Files",
			"fileName", "photo.png",
			"isImage", true,
		))
	})

	t.Run("non-image payload gets a single original file_url and no thumbnail", func(t *testing.T) {
		f := setup()
		result := f.server.BuildToolResult(
			"gkill_get_kyous",
			idfResult(obj("is_image", false, "file_name", "doc.pdf"), true),
			false,
			nil,
		)
		p := payloadOf(t, result)

		expectTrue(t, !p.Has("file_path"), "file_path kept")
		mustMatch(t, strAt(t, p, "file_url"), `^https://mcp\.example\.test/files/[0-9a-f]+$`)
		expectTrue(t, !p.Has("file_url_full"), "file_url_full set for a non-image")
	})

	t.Run("local (stdio) client keeps file_path and gets no file_url", func(t *testing.T) {
		f := setup()
		f.server.IsLocalTransport = true
		result := f.server.BuildToolResult("gkill_get_kyous", idfResult(obj(), true), false, nil)
		p := payloadOf(t, result)

		expectEqual(t, p.Value("file_path"), `C:\Users\user\gkill\photo.png`)
		expectTrue(t, !p.Has("file_url"), "file_url minted for a local client")
	})

	t.Run("remote client without a fileLinkContext falls back to stripping file_path", func(t *testing.T) {
		f := setup()
		f.server.FileLinkContext = nil
		result := f.server.BuildToolResult("gkill_get_kyous", idfResult(obj(), true), false, nil)
		p := payloadOf(t, result)

		expectTrue(t, !p.Has("file_path"), "file_path kept")
		expectTrue(t, !p.Has("file_url"), "file_url minted without a context")
	})
}

// ---------------------------------------------------------------------------
// HttpTransport.handleFileServe — public byte delivery
// ---------------------------------------------------------------------------
func TestHttpTransportHandleFileServe(t *testing.T) {
	type fixture struct {
		transport *HttpTransport
		client    *mockClient
	}
	setup := func(t *testing.T) fixture {
		t.Helper()
		client := &mockClient{}
		fakeServer := &Server{Client: client, Log: nil}
		// HttpTransport は3サーバ共通になり、スコープとファイル配信ルートの有無を options で受ける。
		// ここはファイル配信そのものの検査なので EnableFileLinks を立てる
		transport := mustTransport(t, fakeServer, 0, &OAuthServer{Issuer: "https://mcp.example.test"},
			HttpTransportOptions{Scope: "gkill:read", EnableFileLinks: true})
		return fixture{transport, client}
	}
	newReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/files/x", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		return req
	}
	lastFetch := func(t *testing.T, client *mockClient) apiCall {
		t.Helper()
		expectTrue(t, len(client.fetchCalls) > 0, "fetchFile was not called")
		return client.fetchCalls[len(client.fetchCalls)-1]
	}

	t.Run("valid token delivers the bytes gkill returns", func(t *testing.T) {
		f := setup(t)
		f.client.fetchFile = func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte{1, 2, 3, 4}, ContentType: "image/png"}, nil
		}
		token := f.transport.FileLinkStore.Mint(FileLink{
			GkillSessionID: "sess-1",
			RepName:        "Files",
			FileName:       "photo.png",
			IsImage:        true,
		})
		res := httptest.NewRecorder()

		f.transport.HandleFileServe(res, newReq(), token, url.Values{})

		expectEqual(t, res.Code, 200)
		expectEqual(t, res.Header().Get("Content-Type"), "image/png")
		expectEqual(t, res.Header().Get("Content-Length"), "4")
		expectEqual(t, res.Body.String(), string([]byte{1, 2, 3, 4}))
		// fetched from gkill with the session bound to the token
		call := lastFetch(t, f.client)
		expectEqual(t, call.Pathname, "/files/Files/photo.png")
		expectEqual(t, call.SID, "sess-1")
	})

	t.Run("thumb query is forwarded to gkill only for images and only in WxH form", func(t *testing.T) {
		f := setup(t)
		f.client.fetchFile = func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte{0}, ContentType: "image/jpeg"}, nil
		}
		token := f.transport.FileLinkStore.Mint(FileLink{GkillSessionID: "s", RepName: "Files", FileName: "photo.png", IsImage: true})

		f.transport.HandleFileServe(httptest.NewRecorder(), newReq(), token, url.Values{"thumb": {"1024x1024"}})
		call := lastFetch(t, f.client)
		expectEqual(t, call.Pathname, "/files/Files/photo.png?thumb=1024x1024")
		expectEqual(t, call.SID, "s")

		// malformed thumb is ignored (not forwarded)
		f.transport.HandleFileServe(httptest.NewRecorder(), newReq(), token, url.Values{"thumb": {"1024; rm -rf"}})
		call = lastFetch(t, f.client)
		expectEqual(t, call.Pathname, "/files/Files/photo.png")
		expectEqual(t, call.SID, "s")
	})

	t.Run("thumb query is ignored for non-image files", func(t *testing.T) {
		f := setup(t)
		f.client.fetchFile = func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte{0}, ContentType: "application/pdf"}, nil
		}
		token := f.transport.FileLinkStore.Mint(FileLink{GkillSessionID: "s", RepName: "Files", FileName: "doc.pdf", IsImage: false})

		f.transport.HandleFileServe(httptest.NewRecorder(), newReq(), token, url.Values{"thumb": {"1024x1024"}})
		call := lastFetch(t, f.client)
		expectEqual(t, call.Pathname, "/files/Files/doc.pdf")
		expectEqual(t, call.SID, "s")
	})

	t.Run("unknown or expired token returns 404 without touching gkill", func(t *testing.T) {
		f := setup(t)
		res := httptest.NewRecorder()
		f.transport.HandleFileServe(res, newReq(), "bogus-token", url.Values{})
		expectEqual(t, res.Code, 404)
		expectTrue(t, len(f.client.fetchCalls) == 0, "fetchFile was called")
	})

	t.Run("gkill fetch failure surfaces as 502", func(t *testing.T) {
		f := setup(t)
		f.client.fetchFile = func(_ string, _ string) (*FileResponse, error) { return nil, errors.New("backend down") }
		token := f.transport.FileLinkStore.Mint(FileLink{GkillSessionID: "s", RepName: "Files", FileName: "photo.png", IsImage: true})
		res := httptest.NewRecorder()

		f.transport.HandleFileServe(res, newReq(), token, url.Values{})
		expectEqual(t, res.Code, 502)
	})
}
