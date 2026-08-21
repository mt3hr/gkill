package gkill_server_api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// HandleFileServe は、IDFリポジトリが管理するファイル実体を配信します。
//
// GET /files/{repName}/{rep内相対パス}（PathPrefixルート・wrapNoAuth）
// req_res は使わず、ファイル本体をそのまま返します。
//
// リクエストボディを読まないので、認証はクッキーで行います。
// gkill_session_id があればそのセッションの利用者、無ければ gkill_shared_id を
// 共有IDとみなして共有設定の作成者を利用者とします
// （共有ページから画像などを表示するための経路）。どちらも解決できなければ403です。
// パスの /files/ に続くセグメントをrep名として扱い、名前が一致するIDFリポジトリへ
// StripPrefixして委譲するので、?thumb= の解釈やサムネイル・互換動画の生成は委譲先が行います。
// 一致するrepが無ければ404です。
func (g *GkillServerAPI) HandleFileServe(w http.ResponseWriter, r *http.Request) {

	sessionID := ""
	sharedID := ""

	// クッキーを見て認証する
	sessionIDCookie, err := r.Cookie("gkill_session_id")
	if err != nil {
		sharedIDCookie, err := r.Cookie("gkill_shared_id")
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
			return
		}
		sharedID = strings.ReplaceAll(sharedIDCookie.Value, "shared_id", "")
	} else {
		sessionID = sessionIDCookie.Value
	}

	// アカウントを取得
	// NGであれば403でreturn
	userID := ""
	var sharedKyouInfo *share_kyou_info.ShareKyouInfo
	if sessionID != "" {
		account, gkillError, err := g.getAccountFromSessionID(r.Context(), sessionID, "")
		if account == nil || gkillError != nil || err != nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
			return
		}
		userID = account.UserID
	} else if sharedID != "" {
		sharedKyouInfo, err = g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), sharedID)
		if err != nil || sharedKyouInfo == nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
			return
		}
		userID = sharedKyouInfo.UserID
	} else {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
		return
	}

	// リクエストPathから対象Rep名を抽出
	targetRepName := strings.SplitN(r.URL.Path, "/", 4)[2]

	// OKであればRepNameが一致するIDFRepを探す
	var targetIDFRep reps.IDFKyouRepository
	idfRepImpls, err := repositories.IDFKyouReps.UnWrapTyped()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
		return
	}
	for _, idfRep := range idfRepImpls {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
			return
		}
		if repName == targetRepName {
			targetIDFRep = idfRep
			break
		}
	}

	if targetIDFRep == nil {
		w.WriteHeader(http.StatusNotFound)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
		return
	}

	// 共有経路（sharedID）は、共有クエリの結果に含まれるファイルだけを配信する。
	// rep名一致だけで委譲すると、共有相手が同一rep内の任意相対パスを取得できてしまう。
	// セッション経路（自分のリポジトリ）はフルアクセスのままにする（ここは通らない）。
	if sharedKyouInfo != nil {
		allowed, err := g.collectSharedIDFFilePaths(r.Context(), sharedKyouInfo, repositories)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("error at collect shared idf file paths: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", fmt.Sprintf("%q", err))
			return
		}
		requestedRel := ""
		// /files/{repName}/{相対パス} の3つ目以降が相対パス。net/httpが百分率デコード済み。
		pathParts := strings.SplitN(r.URL.Path, "/", 4)
		if len(pathParts) >= 4 {
			requestedRel = pathParts[3]
		}
		if !isSharedFileAllowed(allowed, targetRepName, requestedRel) {
			w.WriteHeader(http.StatusForbidden)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", "requested file is not in the shared query result")
			return
		}
	}

	// StripPrefixしてIDFサーバのハンドラにわたす
	rootAddress := "/files/" + targetRepName
	http.StripPrefix(rootAddress, http.HandlerFunc(targetIDFRep.HandleFileServe)).ServeHTTP(w, r)
}
