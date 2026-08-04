package gkill_server_api

// GetDevice が重いSQLを1回しか投げないことの回帰テスト。
//
// 取得元の GetAllServerConfigs は SERVER_CONFIG への相関サブクエリを18本使い、
// 毎回約250行のSQL文字列を組み立てて PrepareContext している。
// これが認証付きAPI1本につき最低3回、/files/ の画像配信では1枚ごとに走っていた。
//
// 呼び出し回数は結果に出ないので、数えないと退行に気づけない。

import (
	"context"
	"sync"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
)

// countingServerConfigDAO は GetAllServerConfigs の呼び出し回数を数えるスタブ。
// 埋め込みが nil なので、テストで呼ばない他のメソッドを実装する必要がない。
type countingServerConfigDAO struct {
	server_config.ServerConfigDAO
	m     sync.Mutex
	calls int
}

func (c *countingServerConfigDAO) GetAllServerConfigs(_ context.Context) ([]*server_config.ServerConfig, error) {
	c.m.Lock()
	c.calls++
	c.m.Unlock()
	return []*server_config.ServerConfig{
		{Device: "other-device", EnableThisDevice: false},
		{Device: "this-device", EnableThisDevice: true},
	}, nil
}

func (c *countingServerConfigDAO) callCount() int {
	c.m.Lock()
	defer c.m.Unlock()
	return c.calls
}

func newServerAPIWithCountingDAO(dao_ *countingServerConfigDAO) *GkillServerAPI {
	return &GkillServerAPI{
		GkillDAOManager: &dao.GkillDAOManager{
			ConfigDAOs: &dao.ConfigDAOs{
				ServerConfigDAO: dao_,
			},
		},
	}
}

func TestGetDevice_QueriesOnlyOnce(t *testing.T) {
	stub := &countingServerConfigDAO{}
	g := newServerAPIWithCountingDAO(stub)

	for range 10 {
		device, err := g.GetDevice()
		if err != nil {
			t.Fatalf("GetDevice: %v", err)
		}
		if device != "this-device" {
			t.Fatalf("device = %q, want %q", device, "this-device")
		}
	}

	if stub.callCount() != 1 {
		t.Errorf("GetAllServerConfigs が %d 回呼ばれた。1回に収まるべき(相関サブクエリ18本の重いSQL)", stub.callCount())
	}
}

// 全HTTPハンドラのgoroutineから同時に呼ばれる。
// 以前は g.device へ排他無しで書いており go test -race で落ちるデータ競合だった。
func TestGetDevice_ConcurrentCallsAreSafe(t *testing.T) {
	stub := &countingServerConfigDAO{}
	g := newServerAPIWithCountingDAO(stub)

	wg := &sync.WaitGroup{}
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			device, err := g.GetDevice()
			if err != nil || device != "this-device" {
				t.Errorf("GetDevice = %q, %v", device, err)
			}
		}()
	}
	wg.Wait()

	if stub.callCount() != 1 {
		t.Errorf("並行呼び出しでも1回に収まるべき: %d 回", stub.callCount())
	}
}
