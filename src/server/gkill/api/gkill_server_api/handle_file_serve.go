package gkill_server_api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
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
		sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), sharedID)
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

	// StripPrefixしてIDFサーバのハンドラにわたす
	rootAddress := "/files/" + targetRepName
	http.StripPrefix(rootAddress, http.HandlerFunc(targetIDFRep.HandleFileServe)).ServeHTTP(w, r)
}
