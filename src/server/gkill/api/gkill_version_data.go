package api

import "time"

type GkillVersionData struct {
	CommitHash string    `json:"commit_hash"`
	BuildTime  time.Time `json:"build_time"`
	Version    string    `json:"version"`
	// ビルドした作業ツリーの tree hash（put_version_info.mjs が書く。無ければ空）。
	// E2E の attestation が `gkill_server version` から読み、PATH 上のバイナリが
	// 今のツリーから作られていることの証拠にする（src/tools/run_test_suite.mjs）。
	TreeHash string `json:"tree_hash"`
}
