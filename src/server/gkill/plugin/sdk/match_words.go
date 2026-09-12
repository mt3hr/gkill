package sdk

import (
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api/find_word"
)

// MatchText は text（と id）が検索条件の Words / NotWords / WordsAnd を満たすかを返す。
//
// gkill 本体はプラグインが返した Kyou のワードを再判定しない（本文は gkill に無い）ので、
// FindKyous ハンドラの中でこの判定を通したものだけを返すこと。**自前のループを書かない。**
// 規則は本体の各リポジトリ（SQL の LIKE / Go の判定）と同じで、正本は api/find_word:
//   - 部分一致・大小無視
//   - 肯定語は「text に含む OR id が語で始まる」（ID は前方一致だけ。UUID 丸ごとの貼り付けが引ける）
//   - 除外語は「text に含まない」だけ（ID は見ない）
//   - WordsAnd が真なら全語、偽ならいずれか1語。Words / NotWords とも空なら常に真
//   - 前後の空白を落として空になった語は無視する（古い gkill は入口で捨てないので SDK 側でも捨てる）
//
// text はそのプラグインが「検索対象にしたい文字列」を連結したもの（本文・タイトル・数値など）。
// 複数フィールドをつなぐときは "\n" や "\x00" で区切り、境界をまたいだ語が当たらないようにする。
// id は返す Kyou の ID。
//
// 一覧をループで判定するときは、語の正規化と小文字化を毎回やり直さないよう Matcher を使う。
func (q Query) MatchText(text string, id string) bool {
	return q.Matcher().MatchText(text, id)
}

// Matcher は Query のワード条件を正規化・小文字化して保持し、複数の text を判定するのに使う。
// 意味は Query.MatchText と同じ。
type Matcher struct {
	words    []string
	notWords []string
	wordsAnd bool
}

// Matcher はこの Query のワード条件から Matcher を作る。
func (q Query) Matcher() Matcher {
	return Matcher{
		words:    find_word.LowerWords(find_word.NormalizeWords(q.Words)),
		notWords: find_word.LowerWords(find_word.NormalizeWords(q.NotWords)),
		wordsAnd: q.WordsAnd,
	}
}

// HasWords は肯定語か除外語が1語でもあるか（＝判定が結果を変えうるか）を返す。
// 語が無ければ MatchText は常に真なので、本文を読み出す前の枝刈りに使える。
func (m Matcher) HasWords() bool {
	return len(m.words) != 0 || len(m.notWords) != 0
}

// MatchText は Query.MatchText と同じ判定。
func (m Matcher) MatchText(text string, id string) bool {
	if !m.HasWords() {
		return true
	}
	return find_word.MatchLoweredWords(strings.ToLower(text), strings.ToLower(id), m.words, m.notWords, m.wordsAnd)
}
