package message

import (
	"regexp"
	"strings"
)

// 伏せたあとに置くプレースホルダ。
//
// src/tools/verify_docs.mjs の checkPersonalInfo が「置き換え先」として案内している表記に
// 合わせてあります。伏せた文字列がそのまま資料やコミットメッセージへ引き写されても、
// 機械検査に落ちない形にしておくためです。
const (
	redactedUserName    = "〈ユーザー名〉"
	redactedMailAddress = "〈メールアドレス〉"
)

// windowsUserProfilePathRe は Windows のユーザープロファイル配下のパスに当たります。
// 捕獲群1が `C:\Users\` 相当、捕獲群2がユーザー名の区間です。
// 区切りは `\` と `/` の両方、ドライブレターと Users の綴りは大小を問いません。
//
// 捕獲群2が空白で始まる場合に当てないのは、`C:\Users\ 配下を確認してください` のような
// 一般的な案内文まで潰さないためです。逆に区間の途中の空白は含めます
// （`C:\Users\名 姓` のような空白入りのプロファイル名を、姓だけ残して伏せ損なわないため）。
var windowsUserProfilePathRe = regexp.MustCompile(`(?i)([a-z]:[\\/]+users[\\/]+)([^\s\\/:*?"'<>|][^\\/:*?"'<>|\r\n\t]*)`)

// posixHomePathRe は POSIX のホームディレクトリ配下のパスに当たります。
// 捕獲群の意味は windowsUserProfilePathRe と同じです。
var posixHomePathRe = regexp.MustCompile(`(?i)(/(?:home|users)/)([^\s\\/:*?"'<>|][^\\/:*?"'<>|\r\n\t]*)`)

// mailAddressRe はメールアドレスに当たります。
var mailAddressRe = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}`)

// exampleMailDomain は伏せずに残すドメイン。資料やテストの例示に使う値です。
const exampleMailDomain = "@example."

// RedactEnvironmentSpecific はAPIレスポンスへ載せる自由文から端末固有の情報
// （ホームディレクトリのユーザー名・メールアドレス）を伏せます。
//
// **プラグインの診断文や err.Error() のように、外から入ってきた文字列をレスポンスへ
// 載せるときは必ずここを通すこと。** レスポンスはMCP経由でAIへ渡り、AIがそれを資料や
// コミットメッセージへ引き写すと、verify_docs の checkPersonalInfo が防いでいる
// 個人情報の混入がそのまま成立します。プラグインは別リポジトリの成果物なので、
// 「診断文に実パスを書かない」を書き手側の約束にはできません。
//
// パスの**形は残します**。`C:\Windows\System32\config\systemprofile` のように
// ユーザー名を含まないホームは素通しになるので、LocalSystem 起動でホームが化ける事故の
// 診断は従来どおり成立します。
//
// サーバのコンソールログには適用しません。あちらは端末の中に閉じた人間の診断チャネルで、
// リポジトリへ入る経路がありません。
func RedactEnvironmentSpecific(s string) string {
	if s == "" {
		return s
	}
	s = redactUserNameSegment(windowsUserProfilePathRe, s)
	s = redactUserNameSegment(posixHomePathRe, s)
	s = mailAddressRe.ReplaceAllStringFunc(s, func(matched string) string {
		if strings.Contains(matched, exampleMailDomain) {
			return matched
		}
		return redactedMailAddress
	})
	return s
}

// redactUserNameSegment は捕獲群2（ユーザー名の区間）だけをプレースホルダへ置き換えます。
// 捕獲群1（`C:\Users\` 相当）は元の綴りのまま残します。
func redactUserNameSegment(re *regexp.Regexp, s string) string {
	return re.ReplaceAllStringFunc(s, func(matched string) string {
		groups := re.FindStringSubmatch(matched)
		if len(groups) != 3 {
			return matched
		}
		// 二重適用で `〈〈ユーザー名〉〉` にしない。
		// レスポンスを組み立てる過程で2回通ることがある。
		if strings.HasPrefix(groups[2], "〈") {
			return matched
		}
		return groups[1] + redactedUserName
	})
}
