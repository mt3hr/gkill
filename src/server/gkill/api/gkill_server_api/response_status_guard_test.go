package gkill_server_api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// JSONを返すハンドラは「defer の中で response をエンコードする」形で書かれていて、
// その1箇所が唯一の書き込み点になっている。HTTPステータスはそこより前に書かないと効かない
// (本文を1バイト書いた時点でnet/httpが200を確定させる)。
//
// つまり新しいハンドラを既存のコピペで作ると、writeErrorStatus の1行だけが抜けて
// **そのエンドポイントだけ、異常時も200を返す状態に戻る**。エラーは errors 配列に
// 入っているので画面は普段どおり動き、気付く手立てが無い。
//
// そこでソースを走査して、エンコード行の直前に writeErrorStatus があることを機械的に確認する。

// canonicalEncode はJSONハンドラ共通のエンコード行。
const canonicalEncode = "err := json.NewEncoder(w).Encode(response)"

// writeErrorStatusCall は直前に来ていなければならない呼び出し。
const writeErrorStatusCall = "writeErrorStatus(w, response.Errors)"

// statusExemptHandlers はこの検査の対象外にするファイル。
//
// いずれも「GkillErrorを載せるレスポンス構造体を持たない」経路で、
// 自前で WriteHeader を呼んでステータスだけを返している。
var statusExemptHandlers = map[string]string{
	"handle_file_serve.go":                "/files/ のバイナリ配信。JSONを返さない",
	"handle_urlog_bookmarklet_address.go": "ブックマークレットからの遷移。ステータスだけを返す(ファイル冒頭に理由あり)",
	"handle_urlog_bookmarklet_page.go":    "ブックマークレット設置ページ。HTMLを返す",
}

// TestAllJSONHandlersWriteErrorStatus は、全ハンドラのエンコード行の直前に
// writeErrorStatus があることを確認する。
//
// **落ちたら、ハンドラを足したのに writeErrorStatus の1行を入れていない。**
// defer の中の json.NewEncoder(w).Encode(response) の直前へ
// writeErrorStatus(w, response.Errors) を足すこと。
func TestAllJSONHandlersWriteErrorStatus(t *testing.T) {
	files, err := filepath.Glob("handle_*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("handle_*.go が1つも見つからない（テストの置き場所が変わった可能性がある）")
	}

	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if _, exempt := statusExemptHandlers[filepath.Base(file)]; exempt {
			continue
		}

		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}

		lines := strings.Split(string(body), "\n")
		found := false
		for i, line := range lines {
			if !strings.Contains(line, canonicalEncode) {
				continue
			}
			found = true
			if i == 0 || !strings.Contains(lines[i-1], writeErrorStatusCall) {
				t.Errorf("%s:%d の %q の直前に %q が無い。"+
					"本文より先にステータスを書かないと、異常時も200で返る",
					file, i+1, canonicalEncode, writeErrorStatusCall)
			}
		}
		if !found {
			t.Errorf("%s に %q が無い。JSONハンドラの書き方を変えたなら、"+
				"このテストと statusExemptHandlers も見直すこと", file, canonicalEncode)
			continue
		}
		checked++
	}

	// 走査そのものが空振りしていないことを確かめる（グロブやファイル名の規約が変わったとき用）。
	if checked < 80 {
		t.Errorf("検査できたハンドラが %d 本しか無い。ハンドラの命名か置き場所が変わった可能性がある", checked)
	}
}

// TestStatusExemptHandlersExist は免除リストのファイルが実在することを確認する。
//
// 免除リストに存在しないファイル名が残っていると、リネームで検査を外れたハンドラを
// 見逃す。ファイルを消したり改名したりしたときにここで気付ける。
func TestStatusExemptHandlersExist(t *testing.T) {
	for file, reason := range statusExemptHandlers {
		if _, err := os.Stat(file); err != nil {
			t.Errorf("statusExemptHandlers の %s (%s) が実在しない。"+
				"リネーム・削除したなら免除リストも直すこと", file, reason)
		}
	}
}

// TestMiddlewaresWriteStatusWithJSONBody は、ハンドラより手前で打ち切る2経路が
// ステータスとJSON本文の両方を書いていることを確認する。
//
// 認証ミドルウェアとローカル限定アクセスのフィルタは、以前は本文だけ(または本文すら無しで)
// 返していた。クライアントはステータスを見ずに res.json() するので、
// 本文が空だとそこで例外になり、認証エラーが「証明書が必要です」と表示される。
func TestMiddlewaresWriteStatusWithJSONBody(t *testing.T) {
	// 直に json.NewEncoder(w).Encode(...) を書くとステータスを書き忘れられるので、
	// 必ず writeGkillErrorResponse を通す。
	rawEncode := regexp.MustCompile(`json\.NewEncoder\(w\)\.Encode\(`)

	for _, file := range []string{"auth_middleware.go", "filter_local_only.go"} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)

		if !strings.Contains(text, "writeGkillErrorResponse(") {
			t.Errorf("%s が writeGkillErrorResponse を使っていない。"+
				"ステータスと本文の両方を書くにはこのヘルパを通すこと", file)
		}
		if rawEncode.MatchString(text) {
			t.Errorf("%s が json.NewEncoder(w).Encode( を直に書いている。"+
				"writeGkillErrorResponse を使うこと（ステータスの書き忘れを防ぐため）", file)
		}
	}
}
