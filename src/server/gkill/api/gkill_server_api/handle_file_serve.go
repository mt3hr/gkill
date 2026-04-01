package gkill_server_api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

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
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
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
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
			return
		}
		userID = account.UserID
	} else if sharedID != "" {
		sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), sharedID)
		if err != nil || sharedKyouInfo == nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
			return
		}
		userID = sharedKyouInfo.UserID
	} else {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
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
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
		return
	}
	for _, idfRep := range idfRepImpls {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
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
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
		return
	}

	// StripPrefixしてIDFサーバのハンドラにわたす
	rootAddress := "/files/" + targetRepName
	http.StripPrefix(rootAddress, http.HandlerFunc(targetIDFRep.HandleFileServe)).ServeHTTP(w, r)
}
