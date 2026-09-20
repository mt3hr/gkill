package mcp

// README の実行可能な例と現行スキーマの同期検査 (MCPレビュー P1)。
//
// verify_docs はツール数などの件数しか守れず、README の JSON 例が
// スキーマとずれても検出できない。実際に Mi の例が include_*_mi を欠いたまま
// 「コピーすると0件」の状態で放置されていた。
// README 中の ```json ブロックは gkill_get_kyous の引数例なので、
// 実物の正規化器 (NormalizeKyouArgs) へそのまま通す。キーの改名・廃止・
// 意味変更で例が壊れたら、利用者より先にこのテストが落ちる。

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ```json フェンスの中身を出現順に取り出す。README に json フェンスで載るのは
// gkill_get_kyous の引数例だけ、という前提がこのテストの土台
// (別ツールの例を足すときはフェンス言語を変えるか、このテストへ分岐を足すこと)。
func extractJsonBlocks(markdown string) []string {
	blocks := []string{}
	fence := regexp.MustCompile("```json\r?\n([\\s\\S]*?)```")
	for _, match := range fence.FindAllStringSubmatch(markdown, -1) {
		blocks = append(blocks, match[1])
	}
	return blocks
}

func TestReadmeGetKyousExamplesNormalizeWithTheCurrentSchema(t *testing.T) {
	readme := readSourceFile(t, "README.md")
	blocks := extractJsonBlocks(readme)

	parsedBlocks := func(t *testing.T) []*jsonobj.Object {
		t.Helper()
		out := []*jsonobj.Object{}
		for _, block := range blocks {
			out = append(out, parseObj(t, block))
		}
		return out
	}

	t.Run("json ブロックが存在する (抽出の空振りをテスト成功と混同しない)", func(t *testing.T) {
		expectTrue(t, len(blocks) >= 6, "only %d json blocks", len(blocks))
	})

	t.Run("すべての例が JSON として妥当で、正規化器を通る", func(t *testing.T) {
		for index, block := range blocks {
			args, err := jsonobj.Unmarshal([]byte(block))
			if err != nil {
				t.Fatalf("README json example #%d is not valid JSON: %v", index+1, err)
			}
			if _, err := NormalizeKyouArgs(args); err != nil {
				t.Fatalf("README json example #%d does not normalize: %v", index+1, err)
			}
		}
	})

	// for_mi は include_*_mi を最低1つ要求する (全て無指定は0件+warning)。
	// README の Mi 例がこの制約を欠いたまま出荷されていたのが元指摘。
	t.Run("for_mi の例は include_*_mi を最低1つ持つ", func(t *testing.T) {
		miExamples := []*jsonobj.Object{}
		for _, args := range parsedBlocks(t) {
			if query, ok := args.Object("query"); ok {
				if forMi, _ := query.Bool("for_mi"); forMi {
					miExamples = append(miExamples, args)
				}
			}
		}
		expectTrue(t, len(miExamples) > 0, "no for_mi examples")
		projectionFlags := []string{
			"include_create_mi",
			"include_check_mi",
			"include_limit_mi",
			"include_start_mi",
			"include_end_mi",
		}
		for _, args := range miExamples {
			query, _ := args.Object("query")
			hasProjection := false
			for _, flag := range projectionFlags {
				if v, _ := query.Bool(flag); v {
					hasProjection = true
				}
			}
			expectTrue(t, hasProjection, "for_mi example lacks include_*_mi: %s", jsonobj.MarshalString(args))
		}
	})

	// ページング例の cursor は「next_cursor を verbatim で返す」教えどおり
	// v2 複合形式 ({RFC3339}::{ID}) で示す。素の ISO 日時の例は旧形式の教材になる。
	t.Run("cursor の例は v2 複合形式で示されている", func(t *testing.T) {
		cursorExamples := []string{}
		for _, args := range parsedBlocks(t) {
			if cursor, ok := args.String("cursor"); ok {
				cursorExamples = append(cursorExamples, cursor)
			}
		}
		expectTrue(t, len(cursorExamples) > 0, "no cursor examples")
		for _, cursor := range cursorExamples {
			mustContain(t, cursor, "::")
		}
	})

	// README のパラメータ表に散文で列挙された group_by の語彙は、正規化器では検証されない
	// (スキーマ enum の検査はクライアント側)。今回の修正前は url_domain だけ抜けた状態で
	// 放置されていたので、表の列挙をスキーマ enum とまるごと突き合わせる。
	t.Run("group_by の語彙列挙はスキーマ enum と一致する", func(t *testing.T) {
		match := regexp.MustCompile(`バケット集計（([^）]+)）`).FindStringSubmatch(readme)
		expectTrue(t, match != nil, "README の group_by 行 (バケット集計（…）) が見つからない")
		documented := []any{}
		for _, value := range strings.Split(match[1], "/") {
			documented = append(documented, strings.TrimSpace(value))
		}
		kyousTool := findTool(ReadTools, "gkill_get_kyous")
		expectEqual(t, documented, objAt(t, kyousTool, "inputSchema", "properties", "group_by").Value("enum"))
	})

	// mi_sort_type は「対応する include_*_mi 射影があるときだけ効く」(スキーマの説明文)。
	// include_*_mi 最低1つの検査 (上) はこの対応ズレを検出できないので、例が自分の
	// 注意書きを守っていることまで見る。
	t.Run("mi_sort_type の例は対応する include_*_mi 射影を立てている", func(t *testing.T) {
		sortToProjection := map[string]string{
			"create_time":         "include_create_mi",
			"estimate_start_time": "include_start_mi",
			"estimate_end_time":   "include_end_mi",
			"limit_time":          "include_limit_mi",
		}
		// 対応表の語彙がスキーマ enum から乖離したら、まずここで気づく。
		keys := []string{}
		for key := range sortToProjection {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		enumValues := []string{}
		for _, v := range arrAt(t, FindQuerySchema, "properties", "mi_sort_type", "enum") {
			enumValues = append(enumValues, v.(string))
		}
		sort.Strings(enumValues)
		expectEqual(t, keys, enumValues)
		sortExamples := []*jsonobj.Object{}
		for _, args := range parsedBlocks(t) {
			if query, ok := args.Object("query"); ok {
				if _, ok := query.String("mi_sort_type"); ok {
					sortExamples = append(sortExamples, args)
				}
			}
		}
		expectTrue(t, len(sortExamples) > 0, "no mi_sort_type examples")
		for _, args := range sortExamples {
			query, _ := args.Object("query")
			sortType, _ := query.String("mi_sort_type")
			projection, ok := sortToProjection[sortType]
			expectTrue(t, ok, "unknown mi_sort_type %s", sortType)
			flag, _ := query.Bool(projection)
			expectTrue(t, flag, "example with mi_sort_type:%s lacks %s:true", sortType, projection)
		}
	})
}
