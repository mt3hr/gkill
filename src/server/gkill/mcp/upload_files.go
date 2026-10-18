package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// gkill_upload_files（ADR-0635）。画面の「アップロード」の「ファイル」タブ（upload-file-view.vue）と同じく
// /api/upload_files を呼び、ファイルを file rep（rep_type directory）へ置いて idf の記録として登録する。
// 定義は write_tools.go（verify_docs がツール数をそこから数える）、ディスパッチは write_handlers.go。
//
// 中身の渡し方は2つで、用途で分かれる（読み取り側の gkill_get_idf_file と同じ考え方。ADR-0606）:
//   - data_base64: どのトランスポートでも使える。HTTP では MCP の本文上限（maxMcpBodyBytes）がそのまま効く。
//   - file_path: stdio だけ。MCP サーバが自分の手元のファイルを読む。HTTP で受けると、リモートの AI が
//     MCP サーバの手元の任意のファイルを gkill へ取り込み gkill_get_idf_file で読み出せてしまうので断る。

import (
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// MaxUploadFiles は1回の gkill_upload_files で送れるファイル数。
const MaxUploadFiles = 100

// maxUploadRequestBytes は gkill の /api/upload_files の本文上限（gkill_server_api/auth_middleware.go の
// maxUploadBodyBytes）の写し。MCP は gkill/api を import しないので値を写す。base64 の合計がこれを超えると
// gkill は 413 で断るが理由が本文に乗らないので、MCP が先に見積もって分け方を案内する。
// var なのはテストで縮めるため（1GB のファイルを作らずに上限の経路を通す）。
var maxUploadRequestBytes int64 = 1 << 30

// uploadRequestOverheadBytes は JSON の枠（キー名・ファイル名・時刻）のぶんの余裕。
const uploadRequestOverheadBytes = 1 << 20

// UploadConflictBehaviors は画面の「名前の衝突」の2択。merge は GPS ログ用で、ファイルでは override と
// 同じ動きになる（resolveFileName）ので、画面と同じく出さない。
var UploadConflictBehaviors = []string{"rename", "override"}

// stripBase64Body は data URL の前置きと空白を落とした base64 本体を返す。
func stripBase64Body(value string) (string, bool) {
	body := value
	if strings.HasPrefix(body, "data:") {
		comma := strings.Index(body, ",")
		if comma < 0 {
			return "", false
		}
		body = body[comma+1:]
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, body), true
}

// decodedBase64Size は標準 base64（パディングあり）を Strict で復号し、復号後のバイト数を返す。
// Strict でないと末尾の余剰ビットが違う別の文字列も通ってしまう（同じ中身に複数の表現ができる）。
func decodedBase64Size(body string) (int64, error) {
	return io.Copy(io.Discard, base64.NewDecoder(base64.StdEncoding.Strict(), strings.NewReader(body)))
}

// NormalizeUploadFilesArgs は gkill_upload_files の引数。トランスポートに依らない検査だけを行う
// （file_path の可否とファイルの読み込みは handleUploadFiles）。
func NormalizeUploadFilesArgs(args any) (*jsonobj.Object, error) {
	source, err := assertArgs(args)
	if err != nil {
		return nil, err
	}
	if err := AssertKnownKeys(source, NewStringSet("files", "target_rep_name", "conflict_behavior", "locale_name"), "", noHiddenKeys); err != nil {
		return nil, err
	}
	items, isArray := jsonobj.AsArray(source.Value("files"))
	if !isArray {
		return nil, InvalidArgument("files", "is required: an array of {file_name, data_base64} or {file_path}", source.Value("files"))
	}
	if len(items) == 0 {
		return nil, InvalidArgument("files", "must not be empty", source.Value("files"))
	}
	if len(items) > MaxUploadFiles {
		return nil, InvalidArgument("files", "must have at most "+itoa(MaxUploadFiles)+" entries; split the upload across several calls", int64(len(items)))
	}

	files := make([]any, 0, len(items))
	// 同じ名前を1回に2つ送ると、gkill は両方を「まだ無い名前」と判定して同じパスへ書く（rename が効かない）。
	seenNames := map[string]int{}
	for index, item := range items {
		prefix := "files[" + itoa(index) + "]"
		if !IsPlainObject(item) {
			return nil, InvalidArgument(prefix, "must be an object with file_name + data_base64, or file_path", item)
		}
		entry := item.(*jsonobj.Object)
		if err := AssertKnownKeys(entry, NewStringSet("file_name", "data_base64", "file_path", "related_time"), prefix, noHiddenKeys); err != nil {
			return nil, err
		}
		hasBase64 := entry.Defined("data_base64")
		hasPath := entry.Defined("file_path")
		if hasBase64 && hasPath {
			return nil, InvalidArgument(prefix+".file_path", "cannot be combined with data_base64; pass exactly one of them", entry.Value("file_path"))
		}
		if !hasBase64 && !hasPath {
			return nil, InvalidArgument(prefix+".data_base64", "is required (or file_path on a stdio connection): pass exactly one of them", entry.Value("data_base64"))
		}

		normalized := jsonobj.New()
		if hasPath {
			filePath, err := AssertTrimmedString(entry.Value("file_path"), prefix+".file_path")
			if err != nil {
				return nil, err
			}
			if !filepath.IsAbs(filePath) {
				return nil, InvalidArgument(prefix+".file_path", "must be an absolute path on the machine the MCP server runs on", filePath)
			}
			normalized.Set("file_path", filePath)
		} else {
			raw, ok := entry.Value("data_base64").(string)
			if !ok {
				return nil, InvalidArgument(prefix+".data_base64", "must be a string", entry.Value("data_base64"))
			}
			body, ok := stripBase64Body(raw)
			if !ok {
				return nil, InvalidArgument(prefix+".data_base64", `starts with "data:" but has no "," before the base64 body`, raw)
			}
			if body == "" {
				return nil, InvalidArgument(prefix+".data_base64", "must not be empty", raw)
			}
			if _, err := decodedBase64Size(body); err != nil {
				return nil, InvalidArgument(prefix+".data_base64", "is not valid standard base64 (A-Z a-z 0-9 + /, padded with =): "+err.Error(), raw)
			}
			normalized.Set("data_base64", body)
		}

		var fileName string
		if entry.Defined("file_name") {
			fileName, err = AssertTrimmedString(entry.Value("file_name"), prefix+".file_name")
			if err != nil {
				return nil, err
			}
		} else if hasPath {
			fileName = filepath.Base(jsString(normalized.Value("file_path")))
		} else {
			return nil, InvalidArgument(prefix+".file_name", "is required with data_base64 (the name the file is stored under, e.g. photo.jpg)", entry.Value("file_name"))
		}
		// gkill は区切り文字を黙って取り除いて保存する（"a/b.png" が "ab.png" になる）ので、入口で断る。
		if strings.ContainsAny(fileName, `/\`) {
			return nil, InvalidArgument(prefix+".file_name", `must be a bare file name without "/" or "\"; files always go to the top of the repository directory`, fileName)
		}
		nameKey := strings.ToLower(fileName)
		if first, seen := seenNames[nameKey]; seen {
			return nil, InvalidArgument(prefix+".file_name", "duplicates files["+itoa(first)+"].file_name; upload same-named files in separate calls (conflict_behavior applies between calls, not within one)", fileName)
		}
		seenNames[nameKey] = index
		normalized.Set("file_name", fileName)

		if entry.Defined("related_time") {
			relatedTime, err := NormalizeDateTimeString(entry.Value("related_time"), prefix+".related_time", DateTimeOptions{AllowDateOnly: true})
			if err != nil {
				return nil, err
			}
			normalized.Set("related_time", relatedTime)
		}
		files = append(files, normalized)
	}

	result := jsonobj.Obj("files", files, "target_rep_name", jsonobj.Undefined, "conflict_behavior", "rename", "locale_name", jsonobj.Undefined)
	if targetRepName, ok, err := optionalTrimmedString(source, "target_rep_name"); err != nil {
		return nil, err
	} else if ok {
		result.Set("target_rep_name", targetRepName)
	}
	if source.Defined("conflict_behavior") {
		behavior, err := AssertTrimmedString(source.Value("conflict_behavior"), "conflict_behavior")
		if err != nil {
			return nil, err
		}
		if !containsStringItem(UploadConflictBehaviors, behavior) {
			return nil, InvalidArgument("conflict_behavior", "must be one of: "+strings.Join(UploadConflictBehaviors, ", "), behavior)
		}
		result.Set("conflict_behavior", behavior)
	}
	if localeName, ok, err := optionalTrimmedString(source, "locale_name"); err != nil {
		return nil, err
	} else if ok {
		result.Set("locale_name", localeName)
	}
	return result, nil
}

// maxListedRepNames はエラー文に並べる rep 名の上限。読み取りだけの file rep は端末ごとに数十本ありうる。
const maxListedRepNames = 20

// joinRepNames は rep 名を並べる。上限を超えたぶんは件数だけ言う。
func joinRepNames(names []string) string {
	if len(names) <= maxListedRepNames {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:maxListedRepNames], ", ") + " and " + itoa(len(names)-maxListedRepNames) + " more"
}

// fileRepNames は file rep（rep_type directory）の名前を、書き込み先（use_to_write）とそれ以外に分けて返す。
func fileRepNames(ctx *CallContext, normalized *jsonobj.Object) (writable []string, all []string, err error) {
	response, err := ctx.callApi("/api/get_rep_infos_mcp", localeOnlyBody(normalized))
	if err != nil {
		return nil, nil, err
	}
	for _, item := range arrayOrEmpty(response.Value("rep_infos")) {
		info, ok := item.(*jsonobj.Object)
		if !ok || info == nil || info.Value("rep_type") != "directory" {
			continue
		}
		name := jsString(info.Value("rep_name"))
		all = append(all, name)
		if info.Value("use_to_write") == true {
			writable = append(writable, name)
		}
	}
	return writable, all, nil
}

// resolveUploadTargetRepName は target_rep_name の省略時に、画面と同じく書き込み先の file rep を選ぶ。
func resolveUploadTargetRepName(ctx *CallContext, normalized *jsonobj.Object) (string, error) {
	if normalized.Defined("target_rep_name") {
		return jsString(normalized.Value("target_rep_name")), nil
	}
	writable, all, err := fileRepNames(ctx, normalized)
	if err != nil {
		return "", err
	}
	if len(writable) == 0 {
		message := "This account has no writable file repository (rep_type directory with use_to_write), so there is nowhere to upload to. " +
			"Ask the user to enable one in gkill's settings"
		if len(all) != 0 {
			message += ", or pass target_rep_name (file repositories: " + joinRepNames(all) + ")"
		}
		return "", NewGkillApiError(message+".", nil)
	}
	return writable[0], nil
}

// describeUploadTargetRepNotFound は gkill の「アップロード先の rep が無い」（ERR000151）を、
// 指定できる名前を添えた文言で包み直す。それ以外はそのまま返す。
func describeUploadTargetRepNotFound(ctx *CallContext, normalized *jsonobj.Object, err error, targetRepName string) error {
	apiErr, ok := AsGkillApiError(err)
	if !ok || apiErr.Detail == nil {
		return err
	}
	notFound := false
	for _, entry := range arrayOrEmpty(apiErr.Detail.Value("errors")) {
		if o, isObject := entry.(*jsonobj.Object); isObject && o != nil && o.Value("error_code") == "ERR000151" {
			notFound = true
			break
		}
	}
	if !notFound {
		return err
	}
	message := "target_rep_name " + jsonobj.MarshalString(targetRepName) + " is not a file repository this account can upload to. "
	if writable, all, listErr := fileRepNames(ctx, normalized); listErr == nil {
		if len(writable) != 0 {
			message += "Omit target_rep_name to use the writable one (" + joinRepNames(writable) + "). "
		}
		if len(all) != 0 {
			message += "File repositories: " + joinRepNames(all) + ". "
		}
	}
	return NewGkillApiError(message+"Server said: "+apiErr.Message, apiErr.Detail)
}

// readUploadFilePaths は file_path の指定を読み、data_base64 と既定の related_time（更新日時）を埋める。
// 先に全部を stat して合計を見積もり、読み込みはその後（大きすぎるときに読まずに断る）。
func readUploadFilePaths(ctx *CallContext, files []any) error {
	var totalBytes int64
	type pathEntry struct {
		entry *jsonobj.Object
		path  string
		field string
	}
	paths := []pathEntry{}
	for index, item := range files {
		entry, _ := item.(*jsonobj.Object)
		prefix := "files[" + itoa(index) + "]"
		if !entry.Defined("file_path") {
			totalBytes += int64(len(jsString(entry.Value("data_base64"))))
			continue
		}
		filePath := jsString(entry.Value("file_path"))
		if !ctx.IsLocalTransport {
			return InvalidArgument(prefix+".file_path", "is accepted only on a stdio (local) connection: over HTTP the path would name a file on the "+
				"MCP server's machine, not yours. Send the bytes as data_base64 instead", filePath)
		}
		info, err := os.Stat(filePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return InvalidArgument(prefix+".file_path", "does not exist", filePath)
			}
			return InvalidArgument(prefix+".file_path", "cannot be read: "+err.Error(), filePath)
		}
		if info.IsDir() {
			return InvalidArgument(prefix+".file_path", "is a directory; pass each file in it separately", filePath)
		}
		totalBytes += int64(base64.StdEncoding.EncodedLen(int(info.Size())))
		if !entry.Defined("related_time") {
			// 画面と同じく、ファイルの更新日時を関連時刻にする（File.lastModified）。
			entry.Set("related_time", jsISOString(info.ModTime()))
		}
		paths = append(paths, pathEntry{entry: entry, path: filePath, field: prefix + ".file_path"})
	}
	if totalBytes > maxUploadRequestBytes-uploadRequestOverheadBytes {
		return NewGkillApiError(
			"The upload is "+jsString(totalBytes)+" bytes after base64 encoding, over gkill's per-request limit ("+jsString(maxUploadRequestBytes)+
				" bytes). Split the files across several gkill_upload_files calls. A single file that large has to be placed in the "+
				"repository directory by the user and picked up by an index refresh.",
			jsonobj.Obj("total_base64_bytes", totalBytes, "max_bytes", maxUploadRequestBytes),
		)
	}
	for _, p := range paths {
		data, err := os.ReadFile(p.path)
		if err != nil {
			return InvalidArgument(p.field, "cannot be read: "+err.Error(), p.path)
		}
		p.entry.Set("data_base64", base64.StdEncoding.EncodeToString(data))
	}
	return nil
}

func handleUploadFiles(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeUploadFilesArgs(args)
	if err != nil {
		return nil, err
	}
	files := arrayOrEmpty(normalized.Value("files"))
	if err := readUploadFilePaths(ctx, files); err != nil {
		return nil, err
	}
	targetRepName, err := resolveUploadTargetRepName(ctx, normalized)
	if err != nil {
		return nil, err
	}

	now := jsISOString(timeNow())
	fileData := make([]any, 0, len(files))
	for _, item := range files {
		entry, _ := item.(*jsonobj.Object)
		fileData = append(fileData, jsonobj.Obj(
			"file_name", entry.Value("file_name"),
			"data_base64", entry.Value("data_base64"),
			"last_modified", orNow(entry.Value("related_time"), now),
		))
	}
	response, err := ctx.callApi("/api/upload_files", localeBody(normalized,
		"files", fileData,
		"target_rep_name", targetRepName,
		"conflict_behavior", normalized.Value("conflict_behavior"),
		// 画面からのアップロード（"gkill"）と区別できるよう、MCP のアプリ名を刻ませる。
		"create_app", ctx.AppName,
	))
	if err != nil {
		if normalized.Defined("target_rep_name") {
			return nil, describeUploadTargetRepNotFound(ctx, normalized, err, targetRepName)
		}
		return nil, err
	}
	return jsonobj.Obj(
		"uploaded_kyous", arrayOrEmpty(response.Value("uploaded_kyous")),
		"target_rep_name", targetRepName,
		"conflict_behavior", normalized.Value("conflict_behavior"),
	), nil
}

// summarizeUploadFilesPayload は gkill_upload_files の1行要約。
func summarizeUploadFilesPayload(payload *jsonobj.Object) string {
	count := lengthOfArray(payload.Value("uploaded_kyous"))
	return "Uploaded " + itoa(count) + " file(s) to " + jsString(payload.Value("target_rep_name")) +
		" (conflict_behavior " + jsString(payload.Value("conflict_behavior")) + ")."
}
