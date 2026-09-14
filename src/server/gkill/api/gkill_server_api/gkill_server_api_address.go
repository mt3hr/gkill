package gkill_server_api

// 編集前に読む: .claude/skills/gkill-go-backend/SKILL.md（この領域の不変条件の正本）
//
// HTTP API のルート表。パス・HTTPメソッド・認証区分・無認証ボディ上限・ハンドラを
// 1ルート1行で持ち、本番（serve.go）とテストハーネス（gkill_server_api_test.go の
// setupTestRouter）の両方が registerAPIRoutes でこの表をそのまま登録する。
//
// かつてはアドレス定義（この構造体）・serve.go の HandleFunc・テストハーネスの部分コピー・
// gkill-api.ts の4箇所に同じ表が手書きされていて、「アドレス定義はあるがハンドラ未登録で
// 実行時404」の残骸が2件、テストハーネスには本番と違うラッパーで登録された経路が4本あった
// （ADR-0709）。表を1つにすれば「定義があるのに登録が無い」は構造的に起きない。
// TypeScript 側（gkill-api.ts）はこの表をソース走査した突き合わせテスト
// （gkill-api.test.ts「endpoint address parity with Go」）で守る。
//
// 行の書式は機械可読の契約でもある: `{Path: "...", Method: "...", Auth: ..., Body: ..., Handler: g.HandleXxx},`
// を1行で書く。Method は http.MethodPost ではなく文字列リテラルにする（verify_docs と
// TypeScript のテストが正規表現で読むため）。守るテストは api_routes_test.go。

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// apiAuthKind はルートの認証区分。auth_middleware.go の3ラッパーと1対1に対応する。
type apiAuthKind int

const (
	// authNone は wrapNoAuth / wrapNoAuthCapped。ミドルウェアでの認証なし（filterLocalOnly は通る）。
	// ボディを読む経路は Body に上限を指定すること（validateAPIRoutes が拒否する）。
	authNone apiAuthKind = iota
	// authSession は wrapAuth。セッション認証のみ（AuthContext.Repositories は nil）。
	authSession
	// authSessionRepos は wrapAuthRepos。セッション認証＋リポジトリ読み込み。
	authSessionRepos
)

// apiBodyCap は無認証経路（authNone）のボディ上限。認証ミドルウェアを通らないため
// readAuthBody の32MB上限が効かず、ここで経路別に掛ける（2026-08-30 監査 F-002）。
// authNone 以外の経路では無視される（認証ミドルウェアが maxAuthBodyBytes で切る）。
type apiBodyCap int

const (
	// bodyNone は上限なし。ボディを読まない経路（GET の配信系）にだけ許される。
	bodyNone apiBodyCap = iota
	// bodyAuth は maxAuthBodyBytes / noAuthBodyReadTimeout（認証系と同じ 32MB）。
	bodyAuth
	// bodyUpload は maxUploadBodyBytes / uploadBodyReadTimeout（ファイルアップロード 2 本だけ）。
	bodyUpload
)

// apiRoute は HTTP API の1ルート。
type apiRoute struct {
	// Path は "/api/" で始まる完全一致のパス。
	Path string
	// Method は "POST" か "GET"。
	Method string
	// Auth は認証区分。
	Auth apiAuthKind
	// Body は無認証経路のボディ上限。authNone 以外では bodyNone のまま。
	Body apiBodyCap
	// Handler は g.HandleXxx のメソッド値。
	Handler http.HandlerFunc
}

// serviceWorkerJSPath は Web Push 用 Service Worker の配信パス。API ではなく
// serve.go が PathPrefix で静的配信するので、apiRoutes の表には載せない。
const serviceWorkerJSPath = "/serviceWorker.js"

// apiRoutes は全 HTTP API ルートの表を返す。ハンドラを足したらここへ1行足す
// （足し忘れは api_routes_test.go の TestAPIRoutes_EveryHandlerIsRouted が落とす）。
func (g *GkillServerAPI) apiRoutes() []apiRoute {
	return []apiRoute{
		// --- authNone（wrapNoAuth / wrapNoAuthCapped。認証ミドルウェアを通らない） ---
		// ボディを読む経路は bodyAuth / bodyUpload で経路別の上限と読み取り期限を掛ける。
		// bodyNone のまま残してよいのはボディを読まない経路だけ。
		{Path: "/api/login", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleLogin},
		{Path: "/api/logout", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleLogout},
		{Path: "/api/reset_password", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleResetPassword},
		{Path: "/api/set_new_password", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleSetNewPassword},
		{Path: "/api/get_shared_kyous", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetSharedKyous},
		{Path: "/api/urlog_bookmarklet", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleURLogBookmarkletAddress},
		{Path: "/api/urlog_bookmarklet_page", Method: "GET", Auth: authNone, Body: bodyNone, Handler: g.HandleURLogBookmarkletPage},
		{Path: "/api/get_kyous_mcp", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetKyousMCP},
		{Path: "/api/get_rep_infos_mcp", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetRepInfosMCP},
		{Path: "/api/upload_files", Method: "POST", Auth: authNone, Body: bodyUpload, Handler: g.HandleUploadFiles},
		{Path: "/api/upload_gpslog_files", Method: "POST", Auth: authNone, Body: bodyUpload, Handler: g.HandleUploadGPSLogFiles},
		{Path: "/api/browse_zip_contents", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleBrowseZipContents},
		{Path: "/api/get_idf_kyou_by_relative_path", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetIDFKyouByRelativePath},

		// --- authSession（wrapAuth。セッション認証のみ、リポジトリなし） ---
		{Path: "/api/get_application_config", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetApplicationConfig},
		{Path: "/api/get_server_configs", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetServerConfigs},
		{Path: "/api/update_application_config", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleUpdateApplicationConfig},
		{Path: "/api/update_account_status", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleUpdateAccountStatus},
		{Path: "/api/update_user_reps", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleUpdateUserReps},
		{Path: "/api/update_server_configs", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleUpdateServerConfigs},
		{Path: "/api/add_user", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleAddAccount},
		{Path: "/api/generate_tls_file", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGenerateTLSFile},
		{Path: "/api/get_gkill_notification_public_key", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetGkillNotificationPublicKey},
		{Path: "/api/register_gkill_notification", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleRegisterGkillNotification},
		{Path: "/api/open_directory", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleOpenDirectory},
		{Path: "/api/open_file", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleOpenFile},
		{Path: "/api/reload_repositories", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleReloadRepositories},
		{Path: "/api/get_updated_datas_by_time", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetUpdatedDatasByTime},
		{Path: "/api/update_cache", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleUpdateCache},
		// プラグイン関連
		{Path: "/api/get_plugin_list", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetPluginList},
		{Path: "/api/get_plugin_content_html", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetPluginContentHTML},
		{Path: "/api/get_plugin_config_html", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleGetPluginConfigHTML},
		{Path: "/api/post_plugin_config", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandlePostPluginConfig},

		// --- authSessionRepos（wrapAuthRepos。セッション認証＋リポジトリ） ---
		// 追加
		{Path: "/api/add_tag", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddTag},
		{Path: "/api/add_text", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddText},
		{Path: "/api/add_gkill_notification", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddNotification},
		{Path: "/api/add_kmemo", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddKmemo},
		{Path: "/api/add_kc", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddKC},
		{Path: "/api/add_urlog", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddURLog},
		{Path: "/api/add_nlog", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddNlog},
		{Path: "/api/add_timeis", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddTimeis},
		{Path: "/api/add_mi", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddMi},
		{Path: "/api/add_lantana", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddLantana},
		{Path: "/api/add_rekyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddRekyou},
		{Path: "/api/add_mirekyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddMiReKyou},
		// 更新
		{Path: "/api/update_tag", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateTag},
		{Path: "/api/update_text", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateText},
		{Path: "/api/update_gkill_notification", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateNotification},
		{Path: "/api/update_kmemo", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateKmemo},
		{Path: "/api/update_kc", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateKC},
		{Path: "/api/update_urlog", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateURLog},
		{Path: "/api/update_nlog", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateNlog},
		{Path: "/api/update_timeis", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateTimeis},
		{Path: "/api/update_lantana", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateLantana},
		{Path: "/api/update_idf_kyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateIDFKyou},
		{Path: "/api/update_mi", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateMi},
		{Path: "/api/update_rekyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateRekyou},
		{Path: "/api/update_mirekyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateMiReKyou},
		// 取得
		{Path: "/api/get_kyous", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetKyous},
		{Path: "/api/get_kyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetKyou},
		{Path: "/api/get_kmemo", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetKmemo},
		{Path: "/api/get_kc", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetKC},
		{Path: "/api/get_urlog", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetURLog},
		{Path: "/api/get_nlog", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetNlog},
		{Path: "/api/get_timeis", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetTimeis},
		{Path: "/api/get_mi", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetMi},
		{Path: "/api/get_lantana", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetLantana},
		{Path: "/api/get_rekyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetRekyou},
		{Path: "/api/get_mirekyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetMiReKyou},
		{Path: "/api/get_rekyous_by_target_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetReKyousByTargetID},
		{Path: "/api/get_mirekyous_by_target_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetMiReKyousByTargetID},
		{Path: "/api/get_git_commit_log", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetGitCommitLog},
		{Path: "/api/get_idf_kyou", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetIDFKyou},
		{Path: "/api/get_mi_board_list", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetMiBoardList},
		{Path: "/api/get_all_tag_names", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetAllTagNames},
		{Path: "/api/get_all_rep_names", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetAllRepNames},
		{Path: "/api/get_tags_by_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetTagsByTargetID},
		{Path: "/api/get_tag_histories_by_tag_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetTagHistoriesByTagID},
		{Path: "/api/get_texts_by_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetTextsByTargetID},
		{Path: "/api/get_gkill_notifications_by_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetNotificationsByTargetID},
		{Path: "/api/get_text_histories_by_text_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetTextHistoriesByTextID},
		{Path: "/api/get_gkill_notification_histories_by_notification_id", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetNotificationHistoriesByNotificationID},
		{Path: "/api/get_gps_log", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetGPSLog},
		// 共有
		{Path: "/api/add_share_kyou_list_info", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddShareKyouListInfo},
		{Path: "/api/update_share_kyou_list_info", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleUpdateShareKyouListInfo},
		{Path: "/api/get_share_kyou_list_infos", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetShareKyouListInfos},
		{Path: "/api/delete_share_kyou_list_infos", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleDeleteShareKyouListInfos},
		// リポジトリとトランザクション
		{Path: "/api/get_repositories", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetRepositories},
		{Path: "/api/commit_tx", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleCommitTx},
		{Path: "/api/discard_tx", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleDiscardTX},
		{Path: "/api/submit_kftl_text", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleSubmitKFTLText},
	}
}

// wrapRoute は認証区分とボディ上限から auth_middleware.go のラッパーを選ぶ。
func (g *GkillServerAPI) wrapRoute(rt apiRoute) http.HandlerFunc {
	switch rt.Auth {
	case authSession:
		return g.wrapAuth(rt.Handler)
	case authSessionRepos:
		return g.wrapAuthRepos(rt.Handler)
	default:
		switch rt.Body {
		case bodyAuth:
			return g.wrapNoAuthCapped(rt.Handler, maxAuthBodyBytes, noAuthBodyReadTimeout)
		case bodyUpload:
			return g.wrapNoAuthCapped(rt.Handler, maxUploadBodyBytes, uploadBodyReadTimeout)
		default:
			return g.wrapNoAuth(rt.Handler)
		}
	}
}

// validateAPIRoutes は表の形を検査する。起動時（Serve）とテストの両方で呼ぶ。
// 重複・"/api/" 以外・不明なメソッド・ハンドラ無し・ボディを読む無認証経路に上限が無い、を拒否する。
func validateAPIRoutes(routes []apiRoute) error {
	seen := map[string]bool{}
	for _, rt := range routes {
		if !strings.HasPrefix(rt.Path, "/api/") || len(rt.Path) == len("/api/") {
			return fmt.Errorf("api route %q: path must start with /api/", rt.Path)
		}
		if rt.Method != http.MethodPost && rt.Method != http.MethodGet {
			return fmt.Errorf("api route %q: unsupported method %q", rt.Path, rt.Method)
		}
		key := rt.Method + " " + rt.Path
		if seen[key] {
			return fmt.Errorf("api route %q: duplicated", key)
		}
		seen[key] = true
		if rt.Handler == nil {
			return fmt.Errorf("api route %q: handler is nil", key)
		}
		// 無認証で POST を受ける経路は認証ミドルウェアの32MB上限を通らない。
		// 上限なしで登録すると未認証の無制限ボディがそのままヒープへ載る（監査 F-002）。
		if rt.Auth == authNone && rt.Method == http.MethodPost && rt.Body == bodyNone {
			return fmt.Errorf("api route %q: no-auth POST route must set a body cap", key)
		}
		if rt.Auth != authNone && rt.Body != bodyNone {
			return fmt.Errorf("api route %q: body cap is only for no-auth routes", key)
		}
	}
	return nil
}

// registerAPIRoutes は表の全ルートをルータへ登録する。本番の Serve とテストハーネスの
// 両方がこれを呼ぶ（テストだけ別のラッパーで登録する、を起こさないため）。
func (g *GkillServerAPI) registerAPIRoutes(router *mux.Router) error {
	routes := g.apiRoutes()
	if err := validateAPIRoutes(routes); err != nil {
		return err
	}
	for _, rt := range routes {
		router.HandleFunc(rt.Path, g.wrapRoute(rt)).Methods(rt.Method)
	}
	return nil
}
