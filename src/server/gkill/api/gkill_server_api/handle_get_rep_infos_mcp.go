package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetRepInfosMCP は、MCPサーバ向けにrepの構造化一覧と rep_types の正準語彙を返します。
//
// POST /api/get_rep_infos_mcp（wrapNoAuth）
// req_res.GetRepInfosMCPRequest / req_res.GetRepInfosMCPResponse
//
// wrapNoAuth登録ですが、ハンドラ内でSessionIDからアカウントを解決するので未認証では使えません
// （get_kyous_mcp と同じ形）。
// rep_infos はKyouを供給するrepだけを (rep_name, rep_type) で列挙します。rep名の取得は
// ラッパを UnWrap した leaf の GetRepName が正本で、これは「名前列挙のためだけ」の
// UnWrap（ADR-0001 の許容用途。MatchReps へ入れる用途とは別物）です。
// ファイルパスは返しません。プラグインは rep_types で絞れないため対応表(plugins)を
// 別枠で返し、canonical_rep_types には find.KyouRepTypes の11値をそのまま載せます。
// 経緯: 外部監査 A1/A3「rep_types の語彙がどのAPIからも取得できず総当たりでしか判明しない」。
func (g *GkillServerAPI) HandleGetRepInfosMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetRepInfosMCPRequest{}
	response := &req_res.GetRepInfosMCPResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		writeErrorStatus(w, response.Errors)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get rep infos mcp response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetRepInfosMCPResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REP_INFOS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get rep infos mcp request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetRepInfosMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REP_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError = &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REP_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 正準値ごとにrep群を歩き、(rep_name, rep_type) を重複排除しつつ集める。
	// キャッシュONのとき型コレクションはキャッシュrep1つに畳まれているので、
	// UnWrap で leaf に降りてから GetRepName する（GetAllRepNames と同じ流儀）。
	type repInfoKey struct {
		repName string
		repType string
	}
	seen := map[repInfoKey]struct{}{}
	for _, repType := range find.KyouRepTypes {
		for _, rep := range api.RepsOfKyouRepType(repositories, repType) {
			leafReps, err := rep.UnWrap()
			if err != nil {
				err = fmt.Errorf("error at unwrap rep for rep infos user id = %s: %w", userID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
				continue
			}
			for _, leafRep := range leafReps {
				repName, err := leafRep.GetRepName(r.Context())
				if err != nil || repName == "" {
					continue
				}
				key := repInfoKey{repName: repName, repType: repType}
				if _, exist := seen[key]; exist {
					continue
				}
				seen[key] = struct{}{}
				response.RepInfos = append(response.RepInfos, req_res.RepInfoMCPDTO{
					RepName: repName,
					RepType: repType,
				})
			}
		}
	}
	slices.SortFunc(response.RepInfos, func(a, b req_res.RepInfoMCPDTO) int {
		if c := strings.Compare(a.RepType, b.RepType); c != 0 {
			return c
		}
		return strings.Compare(a.RepName, b.RepName)
	})

	// 付随データ（タグ・テキスト・通知・GPSログ）の格納先。
	// これらは Kyou を1件も生まないので Reps には入っておらず、
	// RepInfos にも GetAllRepNames にも出てこない。だが add_tag / add_text の
	// 書き込み先はここなので、書く前に知れないと「どこへ書かれるのか」が分からない。
	// **RepInfos へ混ぜないこと** ―― 混ぜると query.reps へ渡されて静かに0件になる。
	appendAttachedDataRep := func(dataKind string, repNames []string) {
		for _, repName := range repNames {
			if repName == "" {
				continue
			}
			response.AttachedDataReps = append(response.AttachedDataReps, req_res.AttachedDataRepInfoMCPDTO{
				RepName:  repName,
				DataKind: dataKind,
			})
		}
	}
	if tagReps, err := repositories.TagReps.UnWrapTyped(); err == nil {
		names := make([]string, 0, len(tagReps))
		for _, rep := range tagReps {
			if name, err := rep.GetRepName(r.Context()); err == nil {
				names = append(names, name)
			}
		}
		appendAttachedDataRep("tag", names)
	}
	if textReps, err := repositories.TextReps.UnWrapTyped(); err == nil {
		names := make([]string, 0, len(textReps))
		for _, rep := range textReps {
			if name, err := rep.GetRepName(r.Context()); err == nil {
				names = append(names, name)
			}
		}
		appendAttachedDataRep("text", names)
	}
	if notificationReps, err := repositories.NotificationReps.UnWrapTyped(); err == nil {
		names := make([]string, 0, len(notificationReps))
		for _, rep := range notificationReps {
			if name, err := rep.GetRepName(r.Context()); err == nil {
				names = append(names, name)
			}
		}
		appendAttachedDataRep("notification", names)
	}
	{
		names := make([]string, 0, len(repositories.GPSLogReps))
		for _, rep := range repositories.GPSLogReps {
			if name, err := rep.GetRepName(r.Context()); err == nil {
				names = append(names, name)
			}
		}
		appendAttachedDataRep("gpslog", names)
	}
	slices.SortFunc(response.AttachedDataReps, func(a, b req_res.AttachedDataRepInfoMCPDTO) int {
		if c := strings.Compare(a.DataKind, b.DataKind); c != 0 {
			return c
		}
		return strings.Compare(a.RepName, b.RepName)
	})

	response.CanonicalRepTypes = append(response.CanonicalRepTypes, find.KyouRepTypes...)

	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		response.Plugins = append(response.Plugins, req_res.PluginRepInfoMCPDTO{
			RepName:    manifest.RepName,
			DataType:   manifest.DataType,
			PluginName: manifest.Name,
		})
	}
	slices.SortFunc(response.Plugins, func(a, b req_res.PluginRepInfoMCPDTO) int {
		return strings.Compare(a.RepName, b.RepName)
	})

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetRepInfosMCPSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_REP_INFOS_MESSAGE"}),
	})
}
