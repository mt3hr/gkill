package reps

// FillURLogFieldSkipping の抑止フラグの回帰テスト。
//
// MCP の gkill_add_urlog は fetch_metadata:false / fetch_favicon:false を
// SkipFetchMetadata / SkipFetchFavicon へ写して「外向き通信なしのブックマーク登録」を
// 約束する（MCPレビュー）。配線が外れても目の前ではエラーにならず、
// 「抑止したはずなのに対象サイトと favicon サービスへ通信が飛ぶ」という
// 静かな約束破りになるので、取得口ごと差し替えて「呼ばれないこと」を固定する。
//
// 取得口は2つとも差し替え可能なパッケージ変数（fetchFaviconBytes / fetchPageBody。
// ur_log_favicon_test.go と同じ流儀）。パッケージ変数を書き換えるので t.Parallel() は使わない。

import (
	"fmt"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
)

func fillSkipTestServerConfig() *server_config.ServerConfig {
	return &server_config.ServerConfig{
		URLogTimeout:   5 * time.Second,
		URLogUserAgent: "gkill-test",
	}
}

// overridePageBodyFetch はページ本文の取得口を差し替える。
func overridePageBodyFetch(t *testing.T, impl func(targeturl string, timeout time.Duration, useragent string, enableProxy bool, proxyURL string) ([]byte, error)) {
	t.Helper()
	orig := fetchPageBody
	fetchPageBody = impl
	t.Cleanup(func() { fetchPageBody = orig })
}

// overrideFaviconFetchFunc は favicon の取得口をホスト名ベースのまま差し替える。
func overrideFaviconFetchFunc(t *testing.T, impl func(hostname string) ([]byte, error)) {
	t.Helper()
	orig := fetchFaviconBytes
	fetchFaviconBytes = impl
	t.Cleanup(func() { fetchFaviconBytes = orig })
}

// 両方 skip なら外向き取得は1回も走らず、ID と RelatedTime だけが補完される。
func TestURLogFillSkipping_SkipsAllOutboundFetches(t *testing.T) {
	overridePageBodyFetch(t, func(targeturl string, _ time.Duration, _ string, _ bool, _ string) ([]byte, error) {
		t.Errorf("skip_fetch_metadata なのにページ本文の取得が飛んだ: url=%q", targeturl)
		return nil, fmt.Errorf("unexpected page body request")
	})
	overrideFaviconFetchFunc(t, func(hostname string) ([]byte, error) {
		t.Errorf("skip_fetch_favicon なのに favicon の取得が飛んだ: hostname=%q", hostname)
		return nil, fmt.Errorf("unexpected favicon request")
	})

	urlog := &URLog{URL: "https://example.com/page"}
	if err := urlog.FillURLogFieldSkipping(fillSkipTestServerConfig(), nil, true, true); err != nil {
		t.Fatalf("FillURLogFieldSkipping failed: %v", err)
	}

	// 抑止と無関係に補完される2つ（IDが無いと登録できない）
	if urlog.ID == "" {
		t.Error("skip 指定で ID が補完されていない")
	}
	if urlog.RelatedTime.IsZero() {
		t.Error("skip 指定で RelatedTime が補完されていない")
	}
	// 取得由来の4つは空のまま
	if urlog.Title != "" || urlog.Description != "" || urlog.FaviconImage != "" || urlog.ThumbnailImage != "" {
		t.Errorf("skip 指定なのに取得由来のフィールドが埋まっている: title=%q description=%q favicon=%dchars thumbnail=%dchars",
			urlog.Title, urlog.Description, len(urlog.FaviconImage), len(urlog.ThumbnailImage))
	}
}

// favicon だけ skip してもページ本文由来の補完（Title・Description）は従来どおり動く。
// フラグ2つが独立であることの固定。
func TestURLogFillSkipping_FaviconOnlySkipStillFillsMetadata(t *testing.T) {
	overridePageBodyFetch(t, func(_ string, _ time.Duration, _ string, _ bool, _ string) ([]byte, error) {
		return []byte(`<html><head><title>Example Title</title><meta name="description" content="Example Description"></head><body></body></html>`), nil
	})
	overrideFaviconFetchFunc(t, func(hostname string) ([]byte, error) {
		t.Errorf("skip_fetch_favicon なのに favicon の取得が飛んだ: hostname=%q", hostname)
		return nil, fmt.Errorf("unexpected favicon request")
	})

	urlog := &URLog{URL: "https://example.com/page"}
	if err := urlog.FillURLogFieldSkipping(fillSkipTestServerConfig(), nil, false, true); err != nil {
		t.Fatalf("FillURLogFieldSkipping failed: %v", err)
	}

	if urlog.Title != "Example Title" {
		t.Errorf("Title がページ本文から補完されていない: %q", urlog.Title)
	}
	if urlog.FaviconImage != "" {
		t.Errorf("skip したはずの favicon が埋まっている: %d chars", len(urlog.FaviconImage))
	}
}

// 互換ラッパ FillURLogField は従来どおり両方の取得を行う（skip の既定が false であること）。
func TestURLogFillField_CompatWrapperStillFetchesBoth(t *testing.T) {
	pageBodyCalls := 0
	faviconCalls := 0
	overridePageBodyFetch(t, func(_ string, _ time.Duration, _ string, _ bool, _ string) ([]byte, error) {
		pageBodyCalls++
		return nil, fmt.Errorf("fetch failed (test)")
	})
	overrideFaviconFetchFunc(t, func(_ string) ([]byte, error) {
		faviconCalls++
		return nil, fmt.Errorf("fetch failed (test)")
	})

	urlog := &URLog{URL: "https://example.com/page"}
	if err := urlog.FillURLogField(fillSkipTestServerConfig(), nil); err != nil {
		t.Fatalf("FillURLogField failed: %v", err)
	}

	if pageBodyCalls != 1 {
		t.Errorf("互換ラッパでページ本文の取得が走っていない: calls=%d", pageBodyCalls)
	}
	if faviconCalls != 1 {
		t.Errorf("互換ラッパで favicon の取得が走っていない: calls=%d", faviconCalls)
	}
}
