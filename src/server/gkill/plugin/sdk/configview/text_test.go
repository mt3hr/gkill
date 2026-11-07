package configview

import (
	"html"
	"strings"
	"testing"
	"time"
)

func TestProse(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"そのまま":              "そのまま",
		"`code` と **bold**": "<code>code</code> と <strong>bold</strong>",
		"ワイルドカード `*` `**` `?` `[]`":   "ワイルドカード <code>*</code> <code>**</code> <code>?</code> <code>[]</code>",
		"**配列で複数指定**できます(1つなら文字列でも可)": "<strong>配列で複数指定</strong>できます(1つなら文字列でも可)",
		"対が無い ` はそのまま":                "対が無い ` はそのまま",
		"対が無い ** もそのまま":               "対が無い ** もそのまま",
		"空の `` はそのまま":                 "空の `` はそのまま",
		"<b>& `<code>` **<s>**":       "&lt;b&gt;&amp; <code>&lt;code&gt;</code> <strong>&lt;s&gt;</strong>",
		"`a` b `c`":                   "<code>a</code> b <code>c</code>",
		"**a** `**` **b**":            "<strong>a</strong> <code>**</code> <strong>b</strong>",
	}
	for in, want := range cases {
		if got := Prose(in); got != want {
			t.Errorf("Prose(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProsePlainEqualsEscapeString(t *testing.T) {
	// 記号を含まない文字列では html.EscapeString とバイト一致（エスケープを弱めていないこと）
	for _, s := range []string{`<script>alert("x")</script>`, "a & b", "'quote'", "日本語 — ダッシュ"} {
		if got, want := Prose(s), html.EscapeString(s); got != want {
			t.Errorf("Prose(%q) = %q, want %q", s, got, want)
		}
	}
}

func TestCount(t *testing.T) {
	cases := map[int]string{0: "0", 7: "7", 999: "999", 1000: "1,000", 1234567: "1,234,567", -1234: "-1,234"}
	for n, want := range cases {
		if got := Count(n); got != want {
			t.Errorf("Count(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestFormatUnixAndTime(t *testing.T) {
	if got := FormatUnix(0); got != "-" {
		t.Errorf("FormatUnix(0) = %q, want -", got)
	}
	if got := FormatUnix(-1); got != "-" {
		t.Errorf("FormatUnix(-1) = %q, want -", got)
	}
	if got := FormatTime(time.Time{}); got != "-" {
		t.Errorf("FormatTime(zero) = %q, want -", got)
	}
	at := time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)
	if got := FormatTime(at); got != "2024-01-01 10:00:00" {
		t.Errorf("FormatTime = %q", got)
	}
	if got := FormatUnix(at.Unix()); got != "2024-01-01 10:00:00" {
		t.Errorf("FormatUnix = %q", got)
	}
}

func TestSampleJSON(t *testing.T) {
	got := SampleJSON(map[string]any{"source_dirs": []string{"~/x"}, "max": 3})
	if want := "{\n  \"max\": 3,\n  \"source_dirs\": [\n    \"~/x\"\n  ]\n}"; got != want {
		t.Errorf("SampleJSON = %q, want %q", got, want)
	}
	if got := SampleJSON(map[string]any{"bad": func() {}}); got != "{}" {
		t.Errorf("整形できないときは {} を返す: %q", got)
	}
}

func TestSourceDirsNote(t *testing.T) {
	got := SourceDirsNote("マッチしたフォルダは再帰的に走査します", "キーごと省略すると既定を見ます。")
	for _, want := range []string{
		"`source_dirs` は**配列で複数指定**できます(1つなら文字列でも可)。",
		"ワイルドカード `*` `**` `?` `[]` が使えます — マッチしたフォルダは再帰的に走査します。",
		"先頭の `~` と環境変数(`$HOME` など)も展開されます",
		"絶対パスが確実です)。キーごと省略すると既定を見ます。変更は次の検索から反映されます(gkillの再起動は不要)。",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q が無い:\n%s", want, got)
		}
	}
	// Prose で描いたときに `**` が強調に化けない
	if !strings.Contains(Prose(got), "<code>**</code>") {
		t.Error("ワイルドカード ** がコードとして描かれていない")
	}
}
