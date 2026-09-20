package mcp

// レスポンスのペイロード加工。3つの MCP サーバで完全に同じものを使う（旧 payload.mjs）。

import (
	"regexp"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// DefaultFileLinkThumb はリモート向け file_url の既定サムネサイズ（gkill の ?thumb=WxH に渡す。長辺上限1024）。
const DefaultFileLinkThumb = "1024x1024"

// MintFileLinksMark は「この応答には公開ファイル URL を発行してよい」の印（Object の meta キー）。
// gkill_get_kyous のハンドラが include_file_urls:true のときだけ payload に立て、
// BuildToolResult（server_base.go）が HTTP のときに見る。meta なので JSON には出ず、
// structuredContent にも混ざらない。以前は HTTP なら全応答で idf ごとに2本鋳造していた（ADR-0630）。
const MintFileLinksMark = "gkill.mint_file_links"

// ThumbQueryRegex は配信ルートが受け付けるサムネ指定の検証用。
var ThumbQueryRegex = regexp.MustCompile(`^\d{1,4}x\d{1,4}$`)

// MaxThumbSize は一辺の上限。Go 側 thumbFileServer.maxSize の写し
// (dao/reps/idf_thumb_file_server.go)。これを超えると Go はサムネを作らず
// **黙って原本を返す**ので、送る前に弾く必要がある。
const MaxThumbSize = 1024

// NormalizeMimeType は Content-Type ヘッダから "; charset=..." などのパラメータを落とし、MIME 型だけにする。
func NormalizeMimeType(contentType any) string {
	s := ""
	if contentType != nil && !jsonobj.IsUndefined(contentType) {
		s = jsString(contentType)
	}
	return strings.TrimSpace(strings.SplitN(s, ";", 2)[0])
}

// StripFilePaths は file_path を再帰的に取り除く。
// file_path はこのマシン上の絶対パス。同一マシンで動くクライアント (stdio) にしか意味がなく、
// リモートクライアントに渡すとユーザのディレクトリ構造を漏らすことになる。
func StripFilePaths(value any) any {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			StripFilePaths(item)
		}
	case *jsonobj.Object:
		if v == nil {
			return value
		}
		v.Delete("file_path")
		for _, key := range v.Keys() {
			StripFilePaths(v.Value(key))
		}
	}
	return value
}

// IsIdfPayload は idf ペイロード (rep_name + file_name を持つ) を判定する。
func IsIdfPayload(value any) bool {
	o, ok := value.(*jsonobj.Object)
	if !ok || o == nil {
		return false
	}
	_, hasRep := o.String("rep_name")
	_, hasFile := o.String("file_name")
	return hasRep && hasFile
}

// FileLinkContext は HttpTransport が置く「公開 URL の起点と保管庫」。stdio では nil。
type FileLinkContext struct {
	PublicBaseURL string
	Store         *FileLinkStore
}

// ApplyFileLinks はリモートクライアント向けに、idf ペイロードへ期限付きの公開ファイル URL を注入する。
// 実パス (file_path) は同時に取り除く。ローカルクライアント (stdio) では呼ばない。
func ApplyFileLinks(value any, ctx *FileLinkContext, gkillSessionID string) any {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			ApplyFileLinks(item, ctx, gkillSessionID)
		}
		return value
	case *jsonobj.Object:
		if v == nil {
			return value
		}
		if IsIdfPayload(v) {
			v.Delete("file_path")
			repName, _ := v.String("rep_name")
			fileName, _ := v.String("file_name")
			isImage := jsTruthy(v.Value("is_image"))
			token, expiresAt := ctx.Store.MintLink(FileLink{
				GkillSessionID: gkillSessionID,
				RepName:        repName,
				FileName:       fileName,
				IsImage:        isImage,
			}, ctx.Store.TTL())
			base := ctx.PublicBaseURL + "/files/" + token
			if isImage {
				// 既定は軽量なサムネ、原寸は file_url_full で別途取得できる
				v.Set("file_url", base+"?thumb="+DefaultFileLinkThumb)
				v.Set("file_url_full", base)
			} else {
				v.Set("file_url", base)
			}
			// 期限を添える。無いと人間へ渡したリンクがいつ切れるか誰にも分からない（2026-09-18 の実利用報告）。
			v.Set("file_url_expires_at", jsISOString(expiresAt))
			return value
		}
		for _, key := range v.Keys() {
			ApplyFileLinks(v.Value(key), ctx, gkillSessionID)
		}
	}
	return value
}

// jsISOString は Date#toISOString（UTC・ミリ秒3桁・Z）。
func jsISOString(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// SummarizeToolError はエラー応答の1行要約。
func SummarizeToolError(name string, errText string, detail *jsonobj.Object) string {
	prefix := "Tool call failed"
	if name != "" {
		prefix = name + " failed"
	}
	if detail != nil {
		if field := detail.Value("field"); jsTruthy(field) {
			return prefix + ": " + errText + " (field: " + jsString(field) + ")"
		}
	}
	return prefix + ": " + errText
}

// AppendStaleSchemaNoteToSummary は、本文の warnings に古スキーマの指摘があるとき
// 1行サマリにも印を付ける。本文の warnings を読まない経路でも気づけるようにするため。
//
// 以前は読み取りの要約器だけが持っており、書き込みの要約器には無かった。
// gkill_delete_kyou / gkill_restore_kyou の targets（後から足した非 string 型の引数）が
// まさに古スキーマで壊れる側なので、片側だけだと「同じ古さなのに読み取りでしか
// 知らされない」ことになる（ADR-0609 / ADR-0611）。
func AppendStaleSchemaNoteToSummary(summary string, payload *jsonobj.Object) string {
	if payload == nil {
		return summary
	}
	warnings, ok := payload.Array("warnings")
	if !ok {
		return summary
	}
	for _, warning := range warnings {
		if strings.Contains(jsString(warning), "tool schema snapshot looks stale") {
			return summary + " (this client's tool schema looks stale — reconnect the MCP client)"
		}
	}
	return summary
}

// summaryWarningMaxLength は 1行サマリに載せる warning の長さの上限。長文でも要点が残るよう緩めに切る
// (全文は本文の warnings[] にある)。
const summaryWarningMaxLength = 200

// AppendWarningsToSummary は本文の warnings / partial を1行サマリへ昇格させる。
// 以前は要約が件数と cursor だけで、未知タグで0件でも `No entries matched.` としか
// 出なかった。要約だけを見る利用者・モデルが「本当に0件」と誤読する
// (2026-08-30 レビュー P1)。
//
//   - partial は付随データ欠落専用の印 (M-05) のままで、意味は変えない。ここは表示だけ。
//   - warning があっても partial:false になりうる (壊れた rep の全期間 count 等) ので、
//     partial では warnings の代用にならない。両方を独立に見る。
//   - 古スキーマ警告は AppendStaleSchemaNoteToSummary が専用文言で扱うのでここでは飛ばす。
//   - 新しい部分成功フラグは足さない (ADR-0216)。
func AppendWarningsToSummary(summary string, payload *jsonobj.Object) string {
	if payload == nil {
		return summary
	}
	parts := []string{}
	if jsTruthy(payload.Value("partial")) {
		parts = append(parts, "PARTIAL: attached data may be missing for some entries.")
	}
	warnings := []string{}
	if list, ok := payload.Array("warnings"); ok {
		for _, warning := range list {
			text := jsString(warning)
			if strings.Contains(text, "tool schema snapshot looks stale") {
				continue
			}
			warnings = append(warnings, text)
		}
	}
	if len(warnings) > 0 {
		first := warnings[0]
		if jsLength(first) > summaryWarningMaxLength {
			first = jsSlice(first, 0, summaryWarningMaxLength) + "…"
		}
		more := ""
		if len(warnings) > 1 {
			more = " (+" + itoa(len(warnings)-1) + " more)"
		}
		parts = append(parts, "WARNING: "+first+more)
	}
	if len(parts) == 0 {
		return summary
	}
	return summary + " " + strings.Join(parts, " ")
}

// EntityNotFoundMessage は「1件を型別に引いたが見つからない」ときの文言。
//
// **read / write の両方から使う。** 以前は3種類に割れていて
// （read の `Entity not found: {id}`、write の親切版、update 9本の `Kmemo not found: {id}`）、
// 同じ状況で受け取る説明が呼んだツールによって違った（2026-08-25 の実利用レビュー）。
//
// 取得は型別エンドポイントなので、ID が無いのか型を取り違えたのかは
// サーバの応答からは区別できない。**区別できないことを言う**のが唯一正しい案内で、
// 「ID が存在しない」と断定してはいけない。
// 実際 data_type:"urlog" で kmemo の id を引くと、この経路へ来る。
func EntityNotFoundMessage(id string, dataType any) string {
	return "Entity not found: " + id + " (looked it up as data_type " + jsonobj.MarshalString(dataType) + "; " +
		"the lookup is per-type, so a wrong data_type looks exactly like a wrong id. " +
		// gkill_get_kyous を名指ししない: 書き込み専用サーバには載っていないので、
		// そこで出すと「案内されたツールが無い」になる（read 側は持っている）。
		"Confirm the entry's data_type, and check you are on the account that holds it " +
		`(gkill_get_application_config with fields:["user_id"]), then retry.)`
}
