package mcp

// gkill_upload_files（ADR-0635）。golden（偽 gkill 経由）は data_base64 の経路と入口の検査を固定する。
// ここはファイルシステムとトランスポートに依る経路（file_path・stdio だけの受理・上限）と、
// rep の既定解決・「見つからない」の包み直しを mock で固定する。

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// uploadRepInfos は get_rep_infos_mcp の応答（file rep は書き込み先1つ + 読み取りだけ1つ）。
func uploadRepInfos() *jsonobj.Object {
	return obj("rep_infos", arr(
		obj("rep_name", "Kmemo_testpc", "rep_type", "kmemo", "use_to_write", true),
		obj("rep_name", "Files_old", "rep_type", "directory", "use_to_write", false),
		obj("rep_name", "Files_testpc", "rep_type", "directory", "use_to_write", true),
	))
}

func uploadCtx(client *mockClient, local bool) *CallContext {
	return &CallContext{Client: client, SID: "sid", UserID: "testuser", AppName: "gkill_mcp_write", IsLocalTransport: local}
}

// uploadMock は rep_infos と upload_files に答え、upload_files へ送った本文を控える。
func uploadMock(sent **jsonobj.Object) *mockClient {
	return &mockClient{callApi: func(pathname string, body *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
		switch pathname {
		case "/api/get_rep_infos_mcp":
			return uploadRepInfos(), nil
		case "/api/upload_files":
			*sent = body
			return obj("uploaded_kyous", arr(obj("id", "u1", "data_type", "idf"))), nil
		}
		return obj(), nil
	}}
}

func TestUploadFiles_DefaultRepAndCreateApp(t *testing.T) {
	var sent *jsonobj.Object
	client := uploadMock(&sent)
	payload, err := HandleWriteToolCall(uploadCtx(client, false), "gkill_upload_files", obj(
		"files", arr(obj("file_name", "a.txt", "data_base64", "YW\nJj\r\n", "related_time", "2026-01-01T00:00:00+09:00")),
	))
	expectNoError(t, err)
	// 画面と同じく、書き込み先の file rep（use_to_write）を選ぶ。読み取りだけの file rep は選ばない
	expectEqual(t, client.pathsCalled(), []string{"/api/get_rep_infos_mcp", "/api/upload_files"})
	expectEqual(t, sent.Value("target_rep_name"), "Files_testpc")
	expectEqual(t, sent.Value("conflict_behavior"), "rename")
	// 画面からのアップロード（"gkill"）と区別できるよう、MCP のアプリ名を刻ませる
	expectEqual(t, sent.Value("create_app"), "gkill_mcp_write")
	file := objAt(t, arrAt(t, sent, "files")[0])
	expectEqual(t, file.Value("data_base64"), "YWJj") // 改行は落とす
	expectEqual(t, file.Value("last_modified"), "2026-01-01T00:00:00+09:00")
	expectEqual(t, payload.Value("target_rep_name"), "Files_testpc")
	expectEqual(t, len(arrAt(t, payload, "uploaded_kyous")), 1)
}

func TestUploadFiles_NoWritableFileRep(t *testing.T) {
	client := &mockClient{callApi: func(pathname string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
		return obj("rep_infos", arr(obj("rep_name", "Files_old", "rep_type", "directory", "use_to_write", false))), nil
	}}
	_, err := HandleWriteToolCall(uploadCtx(client, false), "gkill_upload_files", obj(
		"files", arr(obj("file_name", "a.txt", "data_base64", "YQ==")),
	))
	expectErrorContains(t, err, "no writable file repository")
	expectErrorContains(t, err, "Files_old")
	// 送り先が決まらないうちは upload_files を呼ばない
	expectEqual(t, client.pathsCalled(), []string{"/api/get_rep_infos_mcp"})
}

func TestUploadFiles_UnknownRepNamesTheCandidates(t *testing.T) {
	client := &mockClient{callApi: func(pathname string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
		if pathname == "/api/upload_files" {
			return nil, NewGkillApiError("API error at /api/upload_files: ERR000151: failed [kind=not_found]",
				obj("errors", arr(obj("error_code", "ERR000151", "error_kind", "not_found"))))
		}
		return uploadRepInfos(), nil
	}}
	_, err := HandleWriteToolCall(uploadCtx(client, false), "gkill_upload_files", obj(
		"files", arr(obj("file_name", "a.txt", "data_base64", "YQ==")),
		"target_rep_name", "Nowhere",
	))
	expectErrorContains(t, err, `target_rep_name "Nowhere" is not a file repository`)
	expectErrorContains(t, err, "Omit target_rep_name to use the writable one (Files_testpc)")
	expectErrorContains(t, err, "File repositories: Files_old, Files_testpc")
}

func TestUploadFiles_StrictBase64(t *testing.T) {
	// "YR==" は "a" と同じ1バイトになるが、末尾の余剰ビットが0でない（Strict でだけ弾ける）
	_, err := NormalizeUploadFilesArgs(obj("files", arr(obj("file_name", "a.txt", "data_base64", "YR=="))))
	expectErrorContains(t, err, "files[0].data_base64")
	_, err = NormalizeUploadFilesArgs(obj("files", arr(obj("file_name", "a.txt", "data_base64", "data:text/plain;base64"))))
	expectErrorContains(t, err, `has no ","`)
}

func TestUploadFiles_FileNameMustBeBare(t *testing.T) {
	_, err := NormalizeUploadFilesArgs(obj("files", arr(obj("file_name", "sub/a.txt", "data_base64", "YQ=="))))
	expectErrorContains(t, err, "bare file name")
}

func TestUploadFiles_FilePathOverStdio(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "photo.png")
	content := []byte("\x89PNG fake")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	// テストの固定時刻（切りのよい値）。画面の lastModified と同じく、関連時刻の既定はファイルの更新日時
	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	var sent *jsonobj.Object
	client := uploadMock(&sent)
	_, err := HandleWriteToolCall(uploadCtx(client, true), "gkill_upload_files", obj(
		"files", arr(obj("file_path", filePath)),
		"target_rep_name", "Files_testpc",
		"conflict_behavior", "override",
	))
	expectNoError(t, err)
	file := objAt(t, arrAt(t, sent, "files")[0])
	expectEqual(t, file.Value("file_name"), "photo.png")
	expectEqual(t, file.Value("data_base64"), base64.StdEncoding.EncodeToString(content))
	expectEqual(t, file.Value("last_modified"), "2026-01-01T00:00:00.000Z")
	expectEqual(t, sent.Value("conflict_behavior"), "override")
	// target_rep_name を渡したら rep の解決で往復しない
	expectEqual(t, client.pathsCalled(), []string{"/api/upload_files"})
}

func TestUploadFiles_FilePathIsRefusedOverHTTP(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var sent *jsonobj.Object
	client := uploadMock(&sent)
	_, err := HandleWriteToolCall(uploadCtx(client, false), "gkill_upload_files", obj(
		"files", arr(obj("file_path", filePath)),
	))
	expectErrorContains(t, err, "files[0].file_path")
	expectErrorContains(t, err, "only on a stdio")
	// リモートの AI に MCP サーバの手元のファイルを読ませない: gkill へは何も送らない
	expectEqual(t, len(client.calls), 0)
}

// TestUploadFiles_FilePathFollowsTheServersTransport は、手で組んだ CallContext ではなく Server.HandleMessage
// （実際の接続と同じ入口）を通して、file_path の可否がサーバのトランスポートで決まることを固定する。
// 書き込み側の CallContext に IsLocalTransport を渡し忘れていて、本物の stdio でも HTTP 扱いで断っていた
// （手で組んだ ctx のテストは通っていた）。
func TestUploadFiles_FilePathFollowsTheServersTransport(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	message := obj("jsonrpc", "2.0", "id", 1, "method", "tools/call", "params", obj(
		"name", "gkill_upload_files",
		"arguments", obj("files", arr(obj("file_path", filePath)), "target_rep_name", "Files_testpc"),
	))
	for _, tc := range []struct {
		name    string
		local   bool
		isError bool
	}{
		{name: "stdio は読んで送る", local: true, isError: false},
		{name: "HTTP は断る", local: false, isError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sent *jsonobj.Object
			client := uploadMock(&sent)
			client.userID = "testuser"
			for _, server := range []*Server{NewWriteServer(client, nil), NewReadWriteServer(client, nil)} {
				server.IsLocalTransport = tc.local
				result := objAt(t, serverMessage(t, server, message), "result")
				expectEqual(t, result.Value("isError") == true, tc.isError)
			}
			expectEqual(t, sent != nil, !tc.isError)
		})
	}
}

func TestUploadFiles_FilePathChecks(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "存在しない", path: filepath.Join(dir, "missing.txt"), want: "does not exist"},
		{name: "ディレクトリ", path: dir, want: "is a directory"},
		{name: "相対パス", path: "relative.txt", want: "must be an absolute path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sent *jsonobj.Object
			client := uploadMock(&sent)
			_, err := HandleWriteToolCall(uploadCtx(client, true), "gkill_upload_files", obj("files", arr(obj("file_path", tc.path))))
			expectErrorContains(t, err, tc.want)
			expectEqual(t, len(client.calls), 0)
		})
	}
}

func TestUploadFiles_TotalSizeLimitIsCheckedBeforeReading(t *testing.T) {
	saved := maxUploadRequestBytes
	maxUploadRequestBytes = uploadRequestOverheadBytes + 8
	t.Cleanup(func() { maxUploadRequestBytes = saved })

	filePath := filepath.Join(t.TempDir(), "big.bin")
	if err := os.WriteFile(filePath, make([]byte, 12), 0o600); err != nil { // base64 で 16 バイト
		t.Fatal(err)
	}
	var sent *jsonobj.Object
	client := uploadMock(&sent)
	_, err := HandleWriteToolCall(uploadCtx(client, true), "gkill_upload_files", obj("files", arr(obj("file_path", filePath))))
	expectErrorContains(t, err, "Split the files across several gkill_upload_files calls")
	expectEqual(t, len(client.calls), 0)
}

func TestUploadFiles_RepNameListIsCapped(t *testing.T) {
	names := []string{}
	for i := range maxListedRepNames + 3 {
		names = append(names, "Files_"+itoa(i))
	}
	joined := joinRepNames(names)
	mustContain(t, joined, "Files_18, Files_19 and 3 more")
	mustNotContain(t, joined, "Files_20")
	expectEqual(t, joinRepNames([]string{"a", "b"}), "a, b")
}
