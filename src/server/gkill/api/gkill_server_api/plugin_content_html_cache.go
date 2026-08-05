package gkill_server_api

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

const (
	// pluginContentHTMLCacheTTL はキャッシュしたHTMLを使い回す時間。
	pluginContentHTMLCacheTTL = 10 * time.Minute

	// pluginContentHTMLCacheMaxEntries はキャッシュに保持する最大件数。
	// 上限を超えたら古いものから捨てる。
	pluginContentHTMLCacheMaxEntries = 2000
)

// pluginContentHTMLEntry はキャッシュ1件ぶん。
type pluginContentHTMLEntry struct {
	html     string
	storedAt time.Time
}

// pluginContentHTMLCache はプラグインKyouの本文HTMLをサーバ側でキャッシュする。
//
// プラグインの本文はgkillに保存されておらず、要求のたびにプラグインプロセスへ
// 問い合わせる。プラグインのstdioは1本しかなく呼び出しは直列化されるので、
// 一覧の行数ぶんの要求が同時に来ると待ち行列ができる。
// 仮想スクロールで行が再利用されるたびに同じKyouのHTMLが再要求されるため、
// ここで受け止めないと同じ問い合わせを何度もプラグインへ流すことになる。
//
// singleflight は「同じKyouのHTMLを同時に要求されたときに1回だけ問い合わせる」ためのもの。
// キャッシュのTTLだけでは、初回の同時要求が全部プラグインへ抜けてしまう。
type pluginContentHTMLCache struct {
	sf singleflight.Group

	mu      sync.Mutex
	entries map[string]pluginContentHTMLEntry
	// order は挿入順。件数上限を超えたときに古いものから捨てるために持つ。
	order []string
}

func newPluginContentHTMLCache() *pluginContentHTMLCache {
	return &pluginContentHTMLCache{
		entries: map[string]pluginContentHTMLEntry{},
	}
}

// cacheKey はユーザ・リポジトリ表示名・KyouIDの組からキーを作る。
// 区切りに使う \x00 はいずれの値にも現れないので、境界の取り違えが起きない。
func pluginContentHTMLCacheKey(userID string, repName string, kyouID string) string {
	return userID + "\x00" + repName + "\x00" + kyouID
}

// GetOrFetch はキャッシュを引き、無ければ fetch を呼んで結果を覚える。
// 同じキーへの同時呼び出しでは fetch は1回しか走らない。
func (c *pluginContentHTMLCache) GetOrFetch(ctx context.Context, key string, fetch func(ctx context.Context) (string, error)) (string, error) {
	if html, ok := c.lookup(key); ok {
		return html, nil
	}

	// singleflightの共有結果を受け取る側も、自分のctxで打ち切られたら抜けられるようにする。
	// そうしないと、代表になった1本が期限切れになるまで全員が待たされる。
	ch := c.sf.DoChan(key, func() (any, error) {
		// fetchの寿命を呼び出し元1本に縛らない。
		// 縛ると、代表になったリクエストが切れただけで待っている全員が失敗する。
		html, err := fetch(context.WithoutCancel(ctx))
		if err != nil {
			return "", err
		}
		c.store(key, html)
		return html, nil
	})

	select {
	case r := <-ch:
		if r.Err != nil {
			return "", r.Err
		}
		html, _ := r.Val.(string)
		return html, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// lookup は有効期限内のキャッシュを返す。
func (c *pluginContentHTMLCache) lookup(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exist := c.entries[key]
	if !exist {
		return "", false
	}
	if time.Since(entry.storedAt) > pluginContentHTMLCacheTTL {
		delete(c.entries, key)
		return "", false
	}
	return entry.html, true
}

// store はキャッシュへ書き込む。上限を超えたぶんは古いものから捨てる。
func (c *pluginContentHTMLCache) store(key string, html string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exist := c.entries[key]; !exist {
		c.order = append(c.order, key)
	}
	c.entries[key] = pluginContentHTMLEntry{html: html, storedAt: time.Now()}

	for len(c.order) > pluginContentHTMLCacheMaxEntries {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
}

// InvalidateUser は指定ユーザぶんのキャッシュを捨てる。
// プラグインの設定変更やリポジトリ再読み込みのあとに呼ぶ。
func (c *pluginContentHTMLCache) InvalidateUser(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := userID + "\x00"
	remain := c.order[:0]
	for _, key := range c.order {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
			continue
		}
		remain = append(remain, key)
	}
	c.order = remain
}

// fetchPluginContentHTML はプラグインへ本文HTMLを問い合わせる関数を返す。
func fetchPluginContentHTML(pluginRepo reps.PluginRepository, kyouID string) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		return pluginRepo.GetContentHTML(ctx, kyouID)
	}
}
