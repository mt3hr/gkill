package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
// ツールスキーマの世代（schema_revision）。
//
// MCP のツール一覧はクライアントのセッション寿命で固定される。サーバを直しても、
// 生きているセッション（claude.ai コネクタ・ChatGPT のアプリ）には新しい一覧が届かない。
// 2026-09-14 のレビューで、ChatGPT が改名前のフィールド名を握ったまま呼び、
// 「tools/list どおりに呼んだのに未知の引数で拒否される」が実測された
// （サーバのプロセスは改名後に起動していた。古かったのはクライアント側の一覧）。
//
// AI が「自分の握っている一覧がどの世代か」を知る手段は description しか無いので、
// gkill_status の description 末尾へこの値を焼き込み、応答の schema_revision と比べさせる。
// 値はツール一覧そのものから決定的に計算する（手書きの版番号は更新し忘れる）。
// 自己参照を避けるため、計算対象から gkill_status 自身を除く。
// 詳細と却下案は documents/adr/0619-mcp-schema-revision-and-stale-tool-list.md。

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

const StatusToolName = "gkill_status"

// SchemaRevisionLength: 12桁あれば同じサーバ種別の世代の取り違えには十分（衝突確率 2^-48）。
const SchemaRevisionLength = 12

// SchemaRevisionMarkRegex は gkill_status の description 末尾に焼き込む印。
// テスト（3サーバの tools/list 同一性）はこの正規表現で剥がしてから比較する。
var SchemaRevisionMarkRegex = regexp.MustCompile(` \[schema_revision: [0-9a-f]{12}\]$`)

// ComputeSchemaRevision はツール一覧の世代を表す短いハッシュを返す。
//
// gkill_status 自身は除く（description に自分の値を焼き込むので、含めると値が定まらない）。
// 同じツール集合・同じ説明文なら同じ値。ツールを1本足す・説明文を1文字直すだけで変わる。
func ComputeSchemaRevision(tools []*jsonobj.Object) string {
	canonical := make([]any, 0, len(tools))
	for _, tool := range tools {
		if name, _ := tool.String("name"); name == StatusToolName {
			continue
		}
		canonical = append(canonical, tool)
	}
	sum := sha256.Sum256([]byte(jsonobj.MarshalString(canonical)))
	return hex.EncodeToString(sum[:])[:SchemaRevisionLength]
}

// StripSchemaRevisionMark は description から焼き込みの印を剥がす。
func StripSchemaRevisionMark(description string) string {
	return SchemaRevisionMarkRegex.ReplaceAllString(description, "")
}

// StampSchemaRevision は gkill_status の description 末尾へ世代を焼き込んだ新しい配列を返す。
//
// 静的な ReadTools は書き換えない（read / write / readwrite が同じ配列を共有しており、
// サーバごとに世代が違う）。gkill_status 以外の要素は同じオブジェクトをそのまま返す。
func StampSchemaRevision(tools []*jsonobj.Object, revision string) []*jsonobj.Object {
	out := make([]*jsonobj.Object, 0, len(tools))
	for _, tool := range tools {
		name, _ := tool.String("name")
		if name != StatusToolName {
			out = append(out, tool)
			continue
		}
		description, _ := tool.String("description")
		stamped := tool.Clone()
		stamped.Set("description", StripSchemaRevisionMark(description)+" [schema_revision: "+revision+"]")
		out = append(out, stamped)
	}
	return out
}
