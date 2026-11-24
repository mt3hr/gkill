package main

// Google Takeout の「マイ アクティビティ」→「Gemini アプリ」の書き出し形式。
//
// 1要素 = 1往復（人間のプロンプト1つと Gemini の返答1つ）で、新しい順に並ぶ。
// ChatGPT / Claude.ai の書き出しと違い、会話単位の入れ物もメッセージ ID も無い。
// 会話のまとまりは details[].url の会話 URL からしか分からない（uuid.go で ID を導出する）。
//
// **フィールドの値はロケールで変わる**（日本語アカウントでは header が "Gemini アプリ"、
// title の接頭辞が "送信したメッセージ: "）。製品の判定は "Gemini" の部分一致で行い、
// プロンプトの切り出しは最初の ": " までを接頭辞として落とす。

// geminiActivity はマイアクティビティの1レコード。
type geminiActivity struct {
	// Header は製品名。"Gemini アプリ" / "Gemini Apps" などロケール依存。
	Header string `json:"header"`
	// Title は "<接頭辞>: <プロンプト>"。接頭辞を持たないもの（"アシスタント機能を使用" など）もある。
	Title string `json:"title"`
	// Time は RFC 3339 の UTC（ミリ秒つき）。ゾーン付きなので time.Local に依存しない。
	Time string `json:"time"`
	// Products はこのレコードを出した製品。Header と同じ値が入る。
	Products []string `json:"products"`
	// ActivityControls はアクティビティ設定の名前。"Gemini アプリ アクティビティ" など。
	ActivityControls []string `json:"activityControls"`
	// Details は会話へのリンク。name も url も同じ会話 URL が入る。
	Details []geminiLink `json:"details"`
	// SafeHTMLItem は Gemini の返答。**1レコードに2つ以上入ることがある**ので順に連結する。
	SafeHTMLItem []geminiHTMLItem `json:"safeHtmlItem"`
	// Subtitles は注記（「音声が含まれています。」「添付ファイル N 件」など）とその参照先。
	Subtitles []geminiLink `json:"subtitles"`
	// AttachedFiles は同じ ZIP フォルダに並ぶ添付ファイル名。
	AttachedFiles []string `json:"attachedFiles"`
	// ImageFile は添付画像のファイル名。
	ImageFile string `json:"imageFile"`
}

// geminiLink は details / subtitles の1要素。
type geminiLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// geminiHTMLItem は safeHtmlItem の1要素。
type geminiHTMLItem struct {
	HTML string `json:"html"`
}

// geminiExchange は1レコードから取り出した1往復。キャッシュへの取り込み単位。
type geminiExchange struct {
	// ConvID は会話 URL から取った会話 ID。取れなければ空。
	ConvID string
	// TimeUnix はレコードの時刻（UTC 秒）。
	TimeUnix int64
	// TimeKey は ID の導出に使う時刻の文字列（RFC 3339 ナノ秒）。
	TimeKey string
	// Prompt は人間のプロンプト本文。接頭辞を落としたもの。無ければ空。
	Prompt string
	// Reply は Gemini の返答。HTML から写した平文（markdown 風）。無ければ空。
	Reply string
	// Attachments は添付ファイル名と注記。
	Attachments []string
	// Ordinal は会話 ID と時刻が同じ往復が複数あるときの連番。ふつうは 0。
	// Kyou ID の材料。書き出し（世代）ごとに数えるので、書き出しを足しても番号が動かない。
	Ordinal int
}
