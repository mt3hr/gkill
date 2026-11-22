package gkill_server_api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleExportRykvHTML は rykv の画面を1枚の HTML として組み立てて返します。
//
// POST /api/export_rykv_html（wrapAuthRepos）
// req_res.ExportRykvHTMLRequest / req_res.ExportRykvHTMLResponse
//
// 列ごとに検索を実行し、記録と型別データ・付随データを和集合で1回だけ持ちます
// （列をまたいで同じ記録が出るのは普通なので、列ごとに本体を持つと素直に膨らむ）。
// 収集は共有ページと同じ collectKyouBundle を通ります。
//
// 出力 HTML は通信をしません（地図だけ例外）。
// **google_map_api_key は焼き込みません。** 落とすと利用者のキーが共有先へ漏れます。
func (g *GkillServerAPI) HandleExportRykvHTML(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ExportRykvHTMLRequest{}
	response := &req_res.ExportRykvHTMLResponse{}

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
				ErrorCode:    message.InvalidExportRykvHTMLResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_RYKV_HTML_MESSAGE"}),
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
			ErrorCode:    message.InvalidExportRykvHTMLRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_RYKV_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if gkillErr := validateExportRykvHTMLRequest(r.Context(), request); gkillErr != nil {
		response.Errors = append(response.Errors, gkillErr)
		return
	}

	// ガワは先に読む。無ければ（裸の go build）収集を始める前に落とす
	template, err := readExportTemplate()
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error at read export template", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.ExportRykvHTMLTemplateNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_RYKV_HTML_MESSAGE"}),
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
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_RYKV_HTML_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	// **焼き込む前に必ず空へ潰す。** 落とすと利用者のキーが共有先へ漏れる
	applicationConfig.GoogleMapAPIKey = ""

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
			ErrorCode:    message.ExportRykvHTMLRenderError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_RYKV_HTML_MESSAGE"}),
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
				ErrorCode:    message.ExportRykvHTMLRenderError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_EXPORT_RYKV_HTML_MESSAGE"}),
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
		MessageCode: message.ExportRykvHTMLSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_EXPORT_RYKV_HTML_MESSAGE"}),
	})
}

// validateExportRykvHTMLRequest は選べない組み合わせを弾く。
func validateExportRykvHTMLRequest(ctx context.Context, request *req_res.ExportRykvHTMLRequest) *message.GkillError {
	invalid := func(reason string) *message.GkillError {
		err := fmt.Errorf("error at validate export rykv html request: %s", reason)
		return bundleError(ctx, request.LocaleName, message.InvalidExportRykvHTMLOptionError,
			"FAILED_EXPORT_RYKV_HTML_MESSAGE", "error at validate export rykv html request", err)
	}

	if len(request.Columns) == 0 {
		return invalid("no columns")
	}
	for _, column := range request.Columns {
		if column == nil {
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
func (g *GkillServerAPI) buildExportPayload(ctx context.Context, request *req_res.ExportRykvHTMLRequest, userID string, device string, applicationConfig *user_config.ApplicationConfig, sidecar *exportSidecar) (*exportedKyouPayload, *message.GkillError) {
	payload := &exportedKyouPayload{
		SchemaVersion:         exportedPayloadSchemaVersion,
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
	for _, query := range request.Columns {
		bundle, gkillErr := g.collectKyouBundle(ctx, &kyouBundleOptions{
			UserID:               userID,
			Device:               device,
			LocaleName:           request.LocaleName,
			FindQuery:            query,
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
		payload.Columns = append(payload.Columns, exportedColumn{Query: query, KyouIDs: kyouIDs})
		merged.add(bundle)
	}
	merged.writeTo(payload)

	// メディアと IDF の本文。記録を集め終えてから、焼いた記録のぶんだけ読む
	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		return nil, bundleError(ctx, request.LocaleName, message.RepositoriesGetError,
			"FAILED_EXPORT_RYKV_HTML_MESSAGE", "error at get repositories user id", err)
	}
	mediaWarnings, err := g.collectExportMedia(ctx, request, repositories, payload, sidecar)
	if err != nil {
		err = fmt.Errorf("error at collect export media: %w", err)
		return nil, bundleError(ctx, request.LocaleName, message.ExportRykvHTMLRenderError,
			"FAILED_EXPORT_RYKV_HTML_MESSAGE", "error at collect export media", err)
	}
	payload.Warnings = append(payload.Warnings, mediaWarnings...)

	return payload, nil
}
