package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// geminiProductMark は製品の判定に使う語。Takeout は製品名の中でここだけ訳さない。
const geminiProductMark = "gemini"

// geminiAppURLMark は会話 URL の目印。HTML 形式の判定にも使う。
const geminiAppURLMark = "gemini.google.com/app/"

// htmlSniffBytes は HTML 形式かどうかを見るために先頭から読む量。
const htmlSniffBytes = 64 << 10

// classifyEntries は走査で見つかったエントリを、読む JSON と設定画面に出す問題に分ける。
//
// JSON が Gemini のものかは中身で確かめるので（readActivityEntry）、ここでは拡張子で分けるだけ。
// HTML は読まないが、**Gemini のアクティビティらしい HTML を見つけたら問題として出す**。
// 黙って0件にすると「プラグインが壊れている」と読まれてしまう（ADR-0310 と同じ考え方）。
// 中身を見るのは、Takeout の "Gemini" フォルダに Gems と予定アクションの空 HTML が
// 必ず入っていて（実物は中身が <div></div> だけ）、それで警告を出したくないため。
func classifyEntries(set *sdk.SourceSet) (jsonEntries []sdk.SourceEntry, problems []sdk.SourceProblem) {
	if set == nil {
		return nil, nil
	}
	for _, entry := range set.Entries() {
		switch strings.ToLower(path.Ext(entry.EntryName)) {
		case ".json":
			jsonEntries = append(jsonEntries, entry)
		case ".html":
			if looksLikeGeminiActivityHTML(entry) {
				problems = append(problems, sdk.SourceProblem{
					Kind:    sdk.ProblemExtractedFolder,
					Path:    entry.Path,
					Message: htmlFormatMessage,
				})
			}
		}
	}
	slices.SortFunc(jsonEntries, func(a, b sdk.SourceEntry) int { return cmp.Compare(a.Path, b.Path) })
	slices.SortFunc(problems, func(a, b sdk.SourceProblem) int { return cmp.Compare(a.Path, b.Path) })
	return jsonEntries, problems
}

// looksLikeGeminiActivityHTML は HTML のエントリが Gemini のアクティビティかどうかを先頭だけ見て判定する。
func looksLikeGeminiActivityHTML(entry sdk.SourceEntry) bool {
	head, err := entry.ReadHead(htmlSniffBytes)
	if err != nil {
		return false
	}
	return strings.Contains(string(head), geminiAppURLMark)
}

// loadExchanges はエントリのアクティビティを全部読み、1往復ずつに直して返す。
//
// 同じ往復が複数の書き出しに入っていたら、新しい書き出し（exportRanks の値が小さいほう）を残す。
// Takeout は同じフォルダに世代が並ぶのが普通で、Gemini のアクティビティは
// 「その時点までの全履歴」が毎回まるごと入るため、放っておくと同じ往復が世代のぶんだけ並ぶ。
func loadExchanges(entries []sdk.SourceEntry, exportRanks map[string]int) ([]geminiExchange, error) {
	type ranked struct {
		exchange geminiExchange
		rank     int
	}
	byKey := map[string]ranked{}
	// ordinals は「同じ会話 ID・同じ時刻」の往復に書き出しの中で振る連番。
	// 書き出しにはメッセージ ID が無いので、Kyou ID は (会話 ID, 時刻, 役割, 連番) から作る。
	// 実物の書き出しでは (会話 ID, 時刻) は一意なので、ふつうは全部 0 になる。
	// **連番は世代ごとに数える。** 全体で数えると、2つ目の書き出しを足しただけで
	// 同じ往復に別の番号が付き、ID が総入れ替えになる。
	ordinals := map[string]int{}
	read := 0

	for _, entry := range entries {
		records, isGemini, err := readActivityEntry(entry)
		if err != nil {
			return nil, err
		}
		if !isGemini {
			// 別の製品のマイアクティビティ。問題ではないので黙って飛ばす。
			continue
		}
		read++
		rank, ok := exportRanks[entry.ExportID]
		if !ok {
			rank = len(exportRanks)
		}
		for i := range records {
			exchange, ok := exchangeOf(&records[i])
			if !ok {
				continue
			}
			seen := entry.ExportID + "\u0000" + exchange.ConvID + "\u0000" + exchange.TimeKey
			exchange.Ordinal = ordinals[seen]
			ordinals[seen]++

			key := exchange.ConvID + "\u0000" + exchange.TimeKey + "\u0000" + strconv.Itoa(exchange.Ordinal)
			if previous, exist := byKey[key]; exist && previous.rank <= rank {
				// 同じ往復が古い書き出しにもある。新しいほうを残す。
				continue
			}
			byKey[key] = ranked{exchange: exchange, rank: rank}
		}
	}
	if read == 0 {
		return nil, nil
	}

	exchanges := make([]geminiExchange, 0, len(byKey))
	for _, r := range byKey {
		exchanges = append(exchanges, r.exchange)
	}
	// 時刻昇順。同着は会話 ID と連番で決める（マップの順に左右されないため）。
	slices.SortFunc(exchanges, func(a, b geminiExchange) int {
		if c := cmp.Compare(a.TimeUnix, b.TimeUnix); c != 0 {
			return c
		}
		if c := cmp.Compare(a.ConvID, b.ConvID); c != 0 {
			return c
		}
		return cmp.Compare(a.Ordinal, b.Ordinal)
	})
	return exchanges, nil
}

// readActivityEntry は ZIP のエントリ1つを伸長ストリームのまま読む。展開はしない。
// 先頭の1件を見て Gemini のアクティビティでなければ、そこで読むのをやめる。
func readActivityEntry(entry sdk.SourceEntry) (records []geminiActivity, isGemini bool, err error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, false, fmt.Errorf("%s の読み込みに失敗しました: %w", entry.Path, err)
	}
	defer func() { _ = reader.Close() }()

	records, isGemini, err = decodeActivityArray(reader)
	if err != nil {
		return nil, false, fmt.Errorf("%s のパースに失敗しました: %w", entry.Path, err)
	}
	return records, isGemini, nil
}

// decodeActivityArray はトップレベルが配列の JSON を要素ごとにデコードする。
//
// 配列を丸ごと Decode しない。json.Decoder は値1つを内部バッファに全部載せてから
// デコードするので、丸ごとだと生バイトぶんのメモリが構造体の上に余計に要る。
// 要素ごとなら1レコードぶんで済む。
//
// 先頭の1件が Gemini のものでなければ isGemini=false を返してそこでやめる
// （マイアクティビティには製品ごとに同じ名前のファイルが並ぶ）。
func decodeActivityArray(reader io.Reader) (records []geminiActivity, isGemini bool, err error) {
	decoder := json.NewDecoder(reader)
	if err := expectDelim(decoder, '['); err != nil {
		return nil, false, err
	}
	for decoder.More() {
		var record geminiActivity
		if err := decoder.Decode(&record); err != nil {
			return nil, false, err
		}
		if len(records) == 0 && !isGeminiRecord(&record) {
			return nil, false, nil
		}
		records = append(records, record)
	}
	if err := expectDelim(decoder, ']'); err != nil {
		return nil, false, err
	}
	// 空配列は「Gemini のものではない」と区別がつかないが、どちらでも0件なので同じ。
	return records, len(records) != 0, nil
}

// expectDelim は次のトークンが want の区切りであることを確かめる。
func expectDelim(decoder *json.Decoder, want json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := token.(json.Delim); !ok || delim != want {
		return fmt.Errorf("unexpected token %v (want %v)", token, want)
	}
	return nil
}

// isGeminiRecord はレコードが Gemini アプリのものかどうかを判定する。
// header / products / activityControls は訳されるが "Gemini" の部分だけは残る。
func isGeminiRecord(record *geminiActivity) bool {
	if containsGeminiMark(record.Header) {
		return true
	}
	if slices.ContainsFunc(record.Products, containsGeminiMark) {
		return true
	}
	if slices.ContainsFunc(record.ActivityControls, containsGeminiMark) {
		return true
	}
	// 製品名が無くても会話 URL があれば Gemini と分かる。
	for _, detail := range record.Details {
		if strings.Contains(detail.URL, geminiAppURLMark) {
			return true
		}
	}
	return false
}

func containsGeminiMark(s string) bool {
	return strings.Contains(strings.ToLower(s), geminiProductMark)
}

// exchangeOf は1レコードを1往復に直す。
// プロンプトも返答も無いレコード（「アシスタント機能を使用」など）は捨てる。
func exchangeOf(record *geminiActivity) (geminiExchange, bool) {
	at, err := time.Parse(time.RFC3339Nano, record.Time)
	if err != nil {
		return geminiExchange{}, false
	}

	prompt := promptOf(record.Title)
	reply := replyOf(record)
	if prompt == "" && reply == "" {
		return geminiExchange{}, false
	}

	return geminiExchange{
		ConvID:      convIDOf(record),
		TimeUnix:    at.Unix(),
		TimeKey:     at.UTC().Format(time.RFC3339Nano),
		Prompt:      prompt,
		Reply:       reply,
		Attachments: attachmentsOf(record),
	}, true
}

// promptOf は title からプロンプト本文を取り出す。
//
// title は "<接頭辞>: <プロンプト>" で、接頭辞はロケールで変わる（日本語は "送信したメッセージ: "）。
// 英語は "Prompted <プロンプト>" のようにコロンを使わない形もある。
// どちらにも当たらない title（「アシスタント機能を使用」/ "Used Gemini Apps" など）は
// プロンプトではないので空を返す —— ここを「title をそのままプロンプトにする」と、
// 操作の記録が人間の発言として一覧に並んでしまう。
func promptOf(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	// "<接頭辞>: <本文>"。接頭辞が長すぎるものは、コロンを含む本文そのものとみなして採らない。
	if index := strings.Index(title, ": "); index > 0 && len([]rune(title[:index])) <= maxTitlePrefixRunes {
		return strings.TrimSpace(title[index+2:])
	}
	if rest, ok := cutPrefixFold(title, "Prompted "); ok {
		return strings.TrimSpace(rest)
	}
	return ""
}

// maxTitlePrefixRunes は title の接頭辞として認める長さ。
// これを超える位置のコロンは本文の一部とみなす。
const maxTitlePrefixRunes = 24

// cutPrefixFold は大小を無視した接頭辞の切り落とし。
func cutPrefixFold(s string, prefix string) (string, bool) {
	if len(s) < len(prefix) {
		return s, false
	}
	if !strings.EqualFold(s[:len(prefix)], prefix) {
		return s, false
	}
	return s[len(prefix):], true
}

// replyOf は safeHtmlItem の HTML を順に連結して平文にする。
// 1レコードに2つ以上入ることがある（実物にもある）ので、全部つなぐ。
func replyOf(record *geminiActivity) string {
	parts := make([]string, 0, len(record.SafeHTMLItem))
	for _, item := range record.SafeHTMLItem {
		text := htmlToText(item.HTML)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n\n")
}

// convIDOf は details の会話 URL から会話 ID を取り出す。取れなければ空。
// subtitles 側の URL は注記の参照先（ノートブックなど）なので見ない。
func convIDOf(record *geminiActivity) string {
	for _, detail := range record.Details {
		if id := conversationIDFromURL(detail.URL); id != "" {
			return id
		}
		if id := conversationIDFromURL(detail.Name); id != "" {
			return id
		}
	}
	return ""
}

// conversationIDFromURL は "https://gemini.google.com/app/<id>" から <id> を取り出す。
// "/app/c/<id>" の形にも当たる。
func conversationIDFromURL(url string) string {
	_, rest, found := strings.Cut(url, geminiAppURLMark)
	if !found {
		return ""
	}
	if trimmed, ok := strings.CutPrefix(rest, "c/"); ok {
		rest = trimmed
	}
	if cut := strings.IndexAny(rest, "/?#"); cut >= 0 {
		rest = rest[:cut]
	}
	return strings.TrimSpace(rest)
}

// attachmentsOf は添付ファイル名と注記を集める。
// プラグインはファイルを配らないので、名前を本文の折りたたみに並べるだけ。
func attachmentsOf(record *geminiActivity) []string {
	items := make([]string, 0, len(record.AttachedFiles)+len(record.Subtitles)+1)
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		items = append(items, s)
	}
	for _, name := range record.AttachedFiles {
		add(name)
	}
	add(record.ImageFile)
	for _, subtitle := range record.Subtitles {
		add(subtitle.Name)
		// 注記の参照先がファイル名なら足す（URL は足さない）。
		if !strings.Contains(subtitle.URL, "://") {
			add(subtitle.URL)
		}
	}
	return items
}
