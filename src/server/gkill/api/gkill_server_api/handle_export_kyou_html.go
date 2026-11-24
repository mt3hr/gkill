package gkill_server_api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleExportKyouHTML は rykv の画面を1枚の HTML として組み立てて返します。
//
// POST /api/export_kyou_html（wrapAuthRepos）
// req_res.ExportKyouHTMLRequest / req_res.ExportKyouHTMLResponse
//
// 列ごとに検索を実行し、記録と型別データ・付随データを和集合で1回だけ持ちます
// （列をまたいで同じ記録が出るのは普通なので、列ごとに本体を持つと素直に膨らむ）。
// 収集は共有ページと同じ collectKyouBundle を通ります。
//
// 出力 HTML は通信をしません（地図だけ例外）。
// **google_map_api_key は焼き込みません。** 落とすと利用者のキーが共有先へ漏れます。
func (g *GkillServerAPI) HandleExportKyouHTML(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ExportKyouHTMLRequest{}
	response := &req_res.ExportKyouHTMLResponse{}

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
			err = fmt.Errorf("error at parse export rykv html response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse export rykv html response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidExportKyouHTMLResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse export rykv html request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse export rykv html request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidExportKyouHTMLRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if gkillErr := validateExportKyouHTMLRequest(r.Context(), request); gkillErr != nil {
		response.Errors = append(response.Errors, gkillErr)
		return
	}

	// ガワは先に読む。無ければ（裸の go build）収集を始める前に落とす
	template, err := readExportTemplate()
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error at read export template", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.ExportKyouHTMLTemplateNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil || applicationConfig == nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at get applicationConfig user id", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	// **焼き込む前に必ず空へ潰す。** 落とすと利用者のキーが共有先へ漏れる
	applicationConfig.GoogleMapAPIKey = ""

	// サイドバーの木から、チェックされていなかったノードを取り除く。
	// **表示を止めるだけでは足りない** —— 木には出力に含めなかったタグ名・rep 名まで
	// 入っていて、ファイルの中に残っていれば読まれる
	columnQueries := make([]find.FindQuery, 0, len(request.Columns))
	for i := range request.Columns {
		columnQueries = append(columnQueries, request.Columns[i].Query)
	}
	allowedNames := collectExportAllowedNames(columnQueries)
	if request.Screen == req_res.ExportScreenMi {
		// 板ツリーは mi の出力だけ刈る。rykv は板ツリーを使わないので、
		// 刈る根拠（どの板を出したか）が無い
		allowedNames.MiBoards = collectExportAllowedMiBoards(columnQueries)
	}
	if err := pruneExportApplicationConfigStructs(applicationConfig, allowedNames); err != nil {
		err = fmt.Errorf("error at prune export application config structs: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at prune export application config structs", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.ExportKyouHTMLRenderError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	fileNameBase := exportFileNameBase(request.Title)
	var sidecar *exportSidecar
	if request.OutputFormat == req_res.ExportOutputFormatSidecar {
		sidecar = newExportSidecar(fileNameBase)
	}

	payload, gkillErr := g.buildExportPayload(r.Context(), request, userID, device, applicationConfig, sidecar)
	if gkillErr != nil {
		response.Errors = append(response.Errors, gkillErr)
		return
	}

	rendered, err := renderExportHTML(template, payload, nil)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error at render export html", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.ExportKyouHTMLRenderError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.OutputFormat = request.OutputFormat
	if sidecar != nil {
		zipBytes, err := buildExportZip(fileNameBase, rendered, sidecar)
		if err != nil {
			slog.Log(r.Context(), gkill_log.Debug, "error at build export zip", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.ExportKyouHTMLRenderError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_KYOU_HTML_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.FileName = fileNameBase + ".zip"
		response.ZipBase64 = base64.StdEncoding.EncodeToString(zipBytes)
	} else {
		response.FileName = fileNameBase + ".html"
		response.HTML = rendered
	}
	response.Warnings = payload.Warnings
	// 予算や読み込みの失敗で落としたものがあれば、そのぶん欠けている
	response.Partial = len(payload.Warnings) != 0
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.ExportKyouHTMLSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_EXPORT_KYOU_HTML_MESSAGE"}),
	})
}

// validateExportKyouHTMLRequest は選べない組み合わせを弾く。
func validateExportKyouHTMLRequest(ctx context.Context, request *req_res.ExportKyouHTMLRequest) *message.GkillError {
	invalid := func(reason string) *message.GkillError {
		err := fmt.Errorf("error at validate export kyou html request: %s", reason)
		return bundleError(ctx, request.LocaleName, message.InvalidExportKyouHTMLOptionError,
			"FAILED_EXPORT_KYOU_HTML_MESSAGE", "error at validate export kyou html request", err)
	}

	// 空文字は rykv として扱う（印を足す前のクライアントが送ってこない）。
	// 知らない値は弾く —— 出力先のページは知らない印を rykv へ寄せるので、
	// 通してしまうと「mi を出したのに rykv として開く」が静かに起きる
	switch request.Screen {
	case "", req_res.ExportScreenRykv:
		request.Screen = req_res.ExportScreenRykv
	case req_res.ExportScreenMi:
	default:
		return invalid("unknown screen " + request.Screen)
	}

	if len(request.Columns) == 0 {
		return invalid("no columns")
	}
	for _, column := range request.Columns {
		if column.IsNull() {
			return invalid("nil column")
		}
	}

	switch request.MediaMode {
	case req_res.ExportMediaModeThumbnail, req_res.ExportMediaModeFull, req_res.ExportMediaModeNone:
	default:
		return invalid("unknown media_mode " + request.MediaMode)
	}

	switch request.OutputFormat {
	case req_res.ExportOutputFormatSingle:
	case req_res.ExportOutputFormatSidecar:
		// 同名フォルダはメディアの入れ物。メディアを埋め込まないなら空になるので意味が無い
		if request.MediaMode == req_res.ExportMediaModeNone {
			return invalid("output_format sidecar with media_mode none")
		}
	default:
		return invalid("unknown output_format " + request.OutputFormat)
	}

	return nil
}

// buildExportPayload は列ごとに収集して、焼き込む JSON を組み立てる。
//
// 記録と付随データは和集合で1回だけ持ち、列は ID の参照で持つ。
func (g *GkillServerAPI) buildExportPayload(ctx context.Context, request *req_res.ExportKyouHTMLRequest, userID string, device string, applicationConfig *user_config.ApplicationConfig, sidecar *exportSidecar) (*exportedKyouPayload, *message.GkillError) {
	payload := &exportedKyouPayload{
		SchemaVersion:         exportedPayloadSchemaVersion,
		Screen:                request.Screen,
		Title:                 request.Title,
		LocaleName:            request.LocaleName,
		WithDnote:             request.IncludeDnote,
		WithMap:               request.IncludeMap,
		WithSidebar:           request.IncludeSidebar,
		Warnings:              []string{},
		ApplicationConfig:     applicationConfig,
		Columns:               []exportedColumn{},
		Kyous:                 []reps.Kyou{},
		Kmemos:                []reps.Kmemo{},
		KCs:                   []reps.KC{},
		TimeIss:               []reps.TimeIs{},
		Mis:                   []reps.Mi{},
		Nlogs:                 []reps.Nlog{},
		Lantanas:              []reps.Lantana{},
		URLogs:                []reps.URLog{},
		IDFKyous:              []reps.IDFKyou{},
		ReKyous:               []reps.ReKyou{},
		MiReKyous:             []reps.MiReKyou{},
		GitCommitLogs:         []reps.GitCommitLog{},
		GPSLogs:               []reps.GPSLog{},
		AttachedTags:          []reps.Tag{},
		AttachedTexts:         []reps.Text{},
		AttachedNotifications: []reps.Notification{},
		AttachedTimeIss:       []reps.TimeIs{},
		AttachedTimeIsKyous:   []reps.Kyou{},
		PluginHTML:            map[string]string{},
		Media:                 map[string]exportedMediaEntry{},
		IDFTextContents:       map[string]exportedIDFTextContent{},
	}

	merged := newExportMerger()
	for i := range request.Columns {
		column := &request.Columns[i]
		bundle, gkillErr := g.collectKyouBundle(ctx, &kyouBundleOptions{
			UserID:               userID,
			Device:               device,
			LocaleName:           request.LocaleName,
			FindQuery:            &column.Query,
			ViewType:             "kyou",
			IncludeLocations:     request.IncludeGPSLog,
			IncludeTags:          request.IncludeAttachedTags,
			IncludeTexts:         request.IncludeAttachedTexts,
			IncludeTimeIss:       request.IncludeAttachedTimeIss,
			IncludeNotifications: request.IncludeAttachedNotifications,
		})
		if gkillErr != nil {
			return nil, gkillErr
		}

		kyouIDs := make([]string, 0, len(bundle.Kyous))
		for _, kyou := range bundle.Kyous {
			kyouIDs = append(kyouIDs, kyou.ID)
		}
		// 焼き込むのは**送られてきた JSON そのまま**。型に落とすと query_id が消える
		bakedQuery, err := column.BakedQueryJSON()
		if err != nil {
			return nil, bundleError(ctx, request.LocaleName, message.ExportKyouHTMLRenderError,
				"FAILED_EXPORT_KYOU_HTML_MESSAGE", "error at bake export column query", err)
		}
		payload.Columns = append(payload.Columns, exportedColumn{Query: bakedQuery, KyouIDs: kyouIDs})
		merged.add(bundle)
	}
	merged.writeTo(payload)

	// リポスト（ReKyou / MiReKyou）が指している記録そのものを足す。
	// **これが無いと出力先でリポストの行が空で出る**（画面は参照先を get_kyou で引くが、
	// 出力 HTML の API は焼き込んだ配列しか見ない）。列の検索条件には引っかからないので、
	// 検索とは別に ID で引く
	if gkillErr := g.collectExportReKyouTargets(ctx, &kyouBundleOptions{
		UserID:               userID,
		Device:               device,
		LocaleName:           request.LocaleName,
		ViewType:             "kyou",
		IncludeLocations:     false,
		IncludeTags:          request.IncludeAttachedTags,
		IncludeTexts:         request.IncludeAttachedTexts,
		IncludeTimeIss:       request.IncludeAttachedTimeIss,
		IncludeNotifications: request.IncludeAttachedNotifications,
	}, merged, payload); gkillErr != nil {
		return nil, gkillErr
	}

	// メディアと IDF の本文。記録を集め終えてから、焼いた記録のぶんだけ読む
	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		return nil, bundleError(ctx, request.LocaleName, message.RepositoriesGetError,
			"FAILED_EXPORT_KYOU_HTML_MESSAGE", "error at get repositories user id", err)
	}
	mediaWarnings, err := g.collectExportMedia(ctx, request, repositories, payload, sidecar)
	if err != nil {
		err = fmt.Errorf("error at collect export media: %w", err)
		return nil, bundleError(ctx, request.LocaleName, message.ExportKyouHTMLRenderError,
			"FAILED_EXPORT_KYOU_HTML_MESSAGE", "error at collect export media", err)
	}
	payload.Warnings = append(payload.Warnings, mediaWarnings...)

	// プラグインの本文（出力時に選んだときだけ）。
	// 同じプラグインへの問い合わせは直列化されるので、ここは最後に回す
	payload.Warnings = append(payload.Warnings,
		g.collectExportPluginHTML(ctx, request, userID, payload)...)

	return payload, nil
}
