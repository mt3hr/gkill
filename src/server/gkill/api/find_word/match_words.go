// Package find_word はワード検索（FindQuery.Words / NotWords / WordsAnd）の Go 側の照合規則。
//
// 編集前に読む: .claude/skills/gkill-find-query/SKILL.md（この領域の不変条件の正本）
//
// ここに置く理由: 同じ規則を gkill 本体（dao/reps の IDF・git・プラグイン型別アダプタ）と
// プラグイン SDK（plugin/sdk の Query.MatchText）の両方が使う。
// SDK は本体のパッケージを引き込まない方針なので、標準ライブラリ以外に依存しない葉として分けてある。
// api/find に置かないのは、あちらが gkill_log → gkill_options を引くため。
//
// SQLite で検索するリポジトリは dao/sqlite3impl.GenerateFindSQLCommon が生成する WHERE 句で
// 同じ判定をしている。**片方だけを変えるとリポジトリ種別によって検索結果が食い違う**ので、
// 意味を変えるときは必ず両方（と sqlite3impl_util_test.go の実 SQLite 検査）を揃えること。
//
// 規則:
//   - 部分一致・大小無視（呼び出し側で text / id / 語を小文字化してから渡す。LowerWords を使う）
//   - 肯定語は「text に含む OR id が語で始まる」。ID は前方一致だけ。
//     部分一致にすると `1` / `a` / `cafe` のような hex だけの短い語が UUID に偶然含まれ、
//     無関係な記録が「ランダムに」出る（1文字なら ~86%、3文字なら ~0.7% の UUID が当たる）。
//     前方一致なら UUID 丸ごとの貼り付けと git の短縮ハッシュはそのまま引ける。
//   - 除外語は「text に含まない」だけ。ID は見ない（同じ理由で `-1` が無関係な記録を消していた）。
//   - id が空文字なら ID 照合をしない（find.FindQuery.WordsSkipIDMatch と対。除外語を肯定語として
//     再検索する内部クエリ用）。
//   - 肯定語が空のときは肯定条件なしとして扱い、AND・OR どちらでも通す。
//     除外語はいずれか1語でも含まれていれば不一致。
package find_word

import "strings"

// MatchLoweredWords は検索対象テキストがキーワード条件を満たすか判定します。
//
// loweredText・loweredID・loweredWords・loweredNotWords はいずれも呼び出し元で小文字化済みであること
// （LowerWords / strings.ToLower）。規則はパッケージコメントを参照。
//
// 以前はこの判定が各リポジトリに直接書かれており、除外語のループが
// match = strings.Contains(...) と代入になっていたため、
// 「除外語を含まない」行まで不一致として落としていた。
// 除外語を1語でも指定すると全件0件になるのがその症状。
// 同様に OR 検索の分岐が match = false から始まっていたため、
// 除外語だけを指定した検索（肯定語が空）も必ず0件になっていた。
func MatchLoweredWords(loweredText string, loweredID string, loweredWords []string, loweredNotWords []string, wordsAnd bool) bool {
	if wordsAnd {
		for _, word := range loweredWords {
			if !matchLoweredWord(loweredText, loweredID, word) {
				return false
			}
		}
	} else if len(loweredWords) != 0 {
		matched := false
		for _, word := range loweredWords {
			if matchLoweredWord(loweredText, loweredID, word) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	for _, notWord := range loweredNotWords {
		if strings.Contains(loweredText, notWord) {
			return false
		}
	}
	return true
}

// matchLoweredWord は肯定語1語の判定。text に含むか、id が語で始まるか。
func matchLoweredWord(loweredText string, loweredID string, loweredWord string) bool {
	if strings.Contains(loweredText, loweredWord) {
		return true
	}
	return loweredID != "" && strings.HasPrefix(loweredID, loweredWord)
}

// LowerWords は検索語を小文字化した新しいスライスを返します。空なら nil。
//
// query は全リポジトリで共有されているので、スライスの中身を直接書き換えると
// 並列に走っている他リポジトリの検索語まで小文字化して壊してしまう。必ず複製すること。
func LowerWords(words []string) []string {
	if len(words) == 0 {
		return nil
	}
	lowered := make([]string, len(words))
	for i, word := range words {
		lowered[i] = strings.ToLower(word)
	}
	return lowered
}

// NormalizeWords は検索語の前後の空白（全角スペース U+3000 を含む）を落とし、空になった語を捨てた
// 新しいスライスを返します。nil は nil のまま、非nil は空になっても非nil のまま返します
// （FindQuery の「nil=フィルタ未使用 / 非nil空=語なし」の区別を壊さないため）。
//
// 空文字の語を通すと、SQL では LIKE '%%' が全件に一致し、除外語なら全件が消える。
// Go 側でも strings.Contains(x, "") は常に真。画面のパーサは空語を作らないが、
// MCP や API の直叩きでは届くので、サーバ側の入口（FindQuery.WithNormalizedWords）とプラグイン SDK の
// 両方でこれを通す。
func NormalizeWords(words []string) []string {
	if words == nil {
		return nil
	}
	normalized := make([]string, 0, len(words))
	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
