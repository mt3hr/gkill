package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetIDFFilePath は、RepNameとFileName (rep内相対パス) から
// IDFファイルの絶対パスを解決して返す。
//
// 絶対パスは同一マシン上のクライアントにしか意味がなく、
// かつ外部に漏らしたくない情報なので、リクエスト元がlocalhostのときだけ返す。
// 対象ファイルが見つからない場合はFilePathを空文字・Existsをfalseで返す（エラーにはしない）。
func (g *GkillServerAPI) HandleGetIDFFilePath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetIDFFilePathRequest{}
	response := &req_res.GetIDFFilePathResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get idf file path response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetIDFFilePathRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get idf file path request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetIDFFilePathRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
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

	// 絶対パスは同一マシンのクライアントにしか意味がないので、ローカルリクエスト以外には返さない
	if !isLocalRequest(r) {
		err = fmt.Errorf("get idf file path from non local request: remote addr = %s", r.RemoteAddr)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFFilePathNotLocalRequestError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_IDF_FILE_PATH_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfRepImpls, err := repositories.IDFKyouReps.UnWrapTyped()
	if err != nil {
		err = fmt.Errorf("error at unwrap idf kyou reps: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFFilePathError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 指定Rep内で対象ファイルのIDFKyouを引く。
	// DBに登録済みのファイルしか引けないので、パストラバーサルで任意のパスを解決させることはできない。
	for _, idfRep := range idfRepImpls {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFFilePathError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if repName != request.RepName {
			continue
		}
		targetIDFKyou, err := idfRep.GetIDFKyouByTargetFile(r.Context(), request.FileName)
		if err != nil {
			err = fmt.Errorf("error at get idf kyou by target file %s: %w", request.FileName, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFFilePathError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if targetIDFKyou != nil {
			response.FilePath = targetIDFKyou.ContentPath
			response.Exists = true
			break
		}
	}
	// 見つからなかった場合はFilePath空文字・Exists falseのまま正常応答
}
