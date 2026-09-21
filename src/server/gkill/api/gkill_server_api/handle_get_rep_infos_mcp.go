package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

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
// UnWrap（ADR-0101 の許容用途。MatchReps へ入れる用途とは別物）です。
// ファイルパスは返しません。プラグインは rep_types で絞れないため対応表(plugins)を
// 別枠で返し、canonical_rep_types には find.KyouRepTypes の11値をそのまま載せます。
// 経緯: 指摘 A1/A3「rep_types の語彙がどのAPIからも取得できず総当たりでしか判明しない」。
func (g *GkillServerAPI) HandleGetRepInfosMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetRepInfosMCPRequest{}
	response := &req_res.GetRepInfosMCPResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close request body", "error", fmt.Sprintf("%q", err))
		}
	}()
	defer func() {
		writeErrorStatus(r.Context(), w, response.Errors)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get rep infos mcp response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse get rep infos mcp response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetRepInfosMCPResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REP_INFOS_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get rep infos mcp request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse get rep infos mcp request from json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetRepInfosMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REP_INFOS_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// wrapNoAuth なので認証は自前。3段（アカウント→端末→リポジトリ）は
	// wrapAuthRepos と同じ処理なので共通ヘルパへ寄せてある。
	userID, _, repositories, gkillError := g.resolveSelfAuthContext(
		r.Context(), request.SessionID, request.LocaleName, "FAILED_GET_REP_INFOS_MESSAGE")
	if gkillError != nil {
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
	// 「一覧にはあるが書けない」を呼び出し側が書く前に判別できるようにする。
	writeTargets := repositories.WriteTargetRepNames(r.Context())
	for _, repType := range find.KyouRepTypes {
		for _, rep := range api.RepsOfKyouRepType(repositories, repType) {
			leafReps, err := rep.UnWrap()
			if err != nil {
				err = fmt.Errorf("error at unwrap rep for rep infos user id = %s: %w", userID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error at unwrap rep for rep infos user id", "error", fmt.Sprintf("%q", err))
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
				_, useToWrite := writeTargets[repName]
				repInfo := req_res.RepInfoMCPDTO{
					RepName:    repName,
					RepType:    repType,
					UseToWrite: useToWrite,
				}
				// 索引を持つ rep だけ鮮度を出す。「置いたのに0件」が
				// 取り込み待ちなのか本当に無いのかを、呼び出し側が判断できるようにする。
				if reporter, ok := leafRep.(interface {
					IndexUpdatedAt(ctx context.Context) (time.Time, error)
				}); ok {
					if indexedAt, err := reporter.IndexUpdatedAt(r.Context()); err == nil && !indexedAt.IsZero() {
						repInfo.IndexedAt = indexedAt.In(time.Local).Format(time.RFC3339)
					}
				}
				response.RepInfos = append(response.RepInfos, repInfo)
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
	// use_to_write は RepInfos と同じ集合（WriteTargetRepNames。Tag / Text / Notification / GPSLog の
	// Write rep も含む）で判定する。歴代端末ぶん並ぶ格納先のうち、今の書き込み先を1行で引けるようにする。
	appendAttachedDataRep := func(dataKind string, repNames []string) {
		for _, repName := range repNames {
			if repName == "" {
				continue
			}
			_, useToWrite := writeTargets[repName]
			response.AttachedDataReps = append(response.AttachedDataReps, req_res.AttachedDataRepInfoMCPDTO{
				RepName:    repName,
				DataKind:   dataKind,
				UseToWrite: useToWrite,
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

	// Plugins は「query.reps / data_types へ渡せる値」の対応表なので、
	// Kyouを1件も出さないプラグイン（emits_kyou=false。GPSログ専用など）は入れない。
	// manifestのrep_name/data_typeは必須項目なので値は入っているが、
	// そのプラグインのKyouは存在しないため、渡してもエラーも警告も無く0件になる。
	// GPSログの供給元としては AttachedDataReps に data_kind="gpslog" で載っており、
	// そちらが「query.repsの値ではない」と明示されている枠（ADR-0607）。
	// 1本のプラグインが複数の rep 名を申告する（get_rep_name の rep_names。zip の Git リポジトリを
	// 束ねるプラグインなど）ときは、その名前1つにつき1行にする。query.reps に渡せる値は
	// 申告された名前であって manifest の rep_name ではないので、manifest 名の行は作らない
	// （まだ1件も取り込んでおらず申告が空なら、行も無い。渡しても0件になる値を載せない）。
	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		if !manifest.EmitsKyouOrDefault() {
			continue
		}
		repNames, err := pluginRep.GetRepNames(r.Context())
		if err != nil {
			repNames = []string{manifest.RepName}
		}
		for _, repName := range repNames {
			response.Plugins = append(response.Plugins, req_res.PluginRepInfoMCPDTO{
				RepName:    repName,
				DataType:   manifest.DataType,
				PluginName: manifest.Name,
			})
		}
	}
	slices.SortFunc(response.Plugins, func(a, b req_res.PluginRepInfoMCPDTO) int {
		return strings.Compare(a.RepName, b.RepName)
	})

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetRepInfosMCPSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_REP_INFOS_MESSAGE"}),
	})
}
