package message

import (
	"strings"
	"testing"
)

// RedactEnvironmentSpecific が伏せ損なうと、プラグインの診断文に載った端末のローカル絶対パスが
// APIレスポンス経由でAIへ届き、そのまま資料やコミットメッセージへ引き写される。
// それは verify_docs の checkPersonalInfo が防いでいる混入そのもの。
// 経緯: documents/adr/0707-redact-environment-specific-strings.md
//
// テストの入力にユーザー名として `user` / `username` を使っているのは、
// 実在しそうな名前を書くとこのテストファイル自身が checkPersonalInfo に引っかかるため。
// **Go 側はこの2語を例外にしていない**ので、入力として使っても検証になる。
func TestRedactEnvironmentSpecific(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Windowsのユーザープロファイル配下はユーザー名だけ伏せる",
			in:   `fork/exec C:\Users\username\gkill\plugins\admin\p\p.exe: not found`,
			want: `fork/exec C:\Users\〈ユーザー名〉\gkill\plugins\admin\p\p.exe: not found`,
		},
		{
			name: "区切りがスラッシュでもドライブレターが小文字でも当たる",
			in:   `open c:/users/username/gkill/x.db: denied`,
			want: `open c:/users/〈ユーザー名〉/gkill/x.db: denied`,
		},
		{
			name: "空白を含むプロファイル名を姓だけ残して伏せ損なわない",
			in:   `ホーム=C:\Users\user name`,
			want: `ホーム=C:\Users\〈ユーザー名〉`,
		},
		{
			name: "POSIXのホーム配下",
			in:   `open /home/user/gkill/x.db: no such file or directory`,
			want: `open /home/〈ユーザー名〉/gkill/x.db: no such file or directory`,
		},
		{
			name: "macOSのホーム配下",
			in:   `open /Users/user/gkill/x.db: no such file or directory`,
			want: `open /Users/〈ユーザー名〉/gkill/x.db: no such file or directory`,
		},
		{
			// パスの形を残すのはこのため。伏せ方を強くするとこの診断が死ぬ。
			name: "ユーザー名を含まないホームは素通し（LocalSystem起動の診断が生きる）",
			in:   `ホーム=C:\Windows\System32\config\systemprofile`,
			want: `ホーム=C:\Windows\System32\config\systemprofile`,
		},
		{
			name: "末尾に区切りが無いパスも伏せる",
			in:   `ホーム=C:\Users\username`,
			want: `ホーム=C:\Users\〈ユーザー名〉`,
		},
		{
			name: "1つの文字列に複数出てきても全部伏せる",
			in:   "読み取り元=[C:\\Users\\username\\uguisu\\*.db]\nホーム=C:\\Users\\username\n",
			want: "読み取り元=[C:\\Users\\〈ユーザー名〉\\uguisu\\*.db]\nホーム=C:\\Users\\〈ユーザー名〉\n",
		},
		{
			name: "すでにプレースホルダなら二重に包まない",
			in:   `ホーム=C:\Users\〈ユーザー名〉\gkill`,
			want: `ホーム=C:\Users\〈ユーザー名〉\gkill`,
		},
		{
			// ここを潰すと一般的な案内文まで壊れる。
			name: "ユーザー名の区間が無い一般的な案内文は壊さない",
			in:   `C:\Users\ 配下を確認してください`,
			want: `C:\Users\ 配下を確認してください`,
		},
		{
			// 資料やテストの例示に使う値なので残す（verify_docs も @example. は許容する）。
			name: "example.のメールアドレスは残す",
			in:   `送信先: someone@example.com`,
			want: `送信先: someone@example.com`,
		},
		{
			name: "パスもメールも含まない文字列は完全に不変",
			in:   "プラグイン設定の保存に失敗しました: invalid JSON at line 3",
			want: "プラグイン設定の保存に失敗しました: invalid JSON at line 3",
		},
		{
			name: "空文字",
			in:   "",
			want: "",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := RedactEnvironmentSpecific(c.in); got != c.want {
				t.Errorf("RedactEnvironmentSpecific(%q)\n = %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

// メールアドレスの入力は文字列連結で組み立てる。
// ソースへ `名前@ドメイン` の形でそのまま書くと、このテストファイル自身が
// verify_docs の checkPersonalInfo（メールアドレスのパターン）に引っかかる。
func TestRedactEnvironmentSpecificHidesMailAddress(t *testing.T) {
	in := "問い合わせ: " + "someone" + "@" + "mail.invalid" + " まで"

	got := RedactEnvironmentSpecific(in)

	if strings.Contains(got, "mail.invalid") {
		t.Errorf("メールアドレスが残っている: %q", got)
	}
	if !strings.Contains(got, "〈メールアドレス〉") {
		t.Errorf("プレースホルダへ置き換わっていない: %q", got)
	}
}

// レスポンスを組み立てる過程で2回通ることがある。冪等でないと表示が壊れる。
func TestRedactEnvironmentSpecificIsIdempotent(t *testing.T) {
	for _, in := range []string{
		`fork/exec C:\Users\username\gkill\p.exe: not found`,
		`open /home/user/gkill/x.db: denied`,
		"問い合わせ: " + "someone" + "@" + "mail.invalid",
	} {
		once := RedactEnvironmentSpecific(in)
		twice := RedactEnvironmentSpecific(once)
		if once != twice {
			t.Errorf("2回通すと結果が変わる: 1回目=%q 2回目=%q", once, twice)
		}
	}
}
