package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetIDFKyouByRelativePath は、基準となるIDFKyou (TargetID) と
// そのファイル内に記載された相対パス (RelativePath) から、
// 同一Rep内の対象ファイルのIDFKyou IDを解決して返す。
//
// POST /api/get_idf_kyou_by_relative_path（wrapNoAuth）
// req_res.GetIDFKyouByRelativePathRequest / req_res.GetIDFKyouByRelativePathResponse
//
// wrapNoAuth登録だが、ハンドラ内でSessionIDからアカウントを解決するので未認証では使えない。
// 相対パスはrep外を指すものと絶対パスを拒否する。
// 見つからない場合はKyouIDを空文字で返す（エラーにはしない）。
func (g *GkillServerAPI) HandleGetIDFKyouByRelativePath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetIDFKyouByRelativePathRequest{}
	response := &req_res.GetIDFKyouByRelativePathResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get idf kyou by relative path response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetIDFKyouByRelativePathRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get idf kyou by relative path request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetIDFKyouByRelativePathRequestDataError,
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
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 基準となるIDFKyouを取得
	idfKyou, err := findIDFKyouByID(r.Context(), repositories, request.TargetID)
	if err != nil || idfKyou == nil {
		if err != nil {
			err = fmt.Errorf("error at find idf kyou by id = %s: %w", request.TargetID, err)
		} else {
			err = fmt.Errorf("idf kyou not found id = %s", request.TargetID)
		}
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouByRelativePathError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 相対パスを基準ファイルのディレクトリ基準で解決
	resolvedTargetFile, err := resolveIDFRelativePath(idfKyou.TargetFile, request.RelativePath)
	if err != nil {
		err = fmt.Errorf("error at resolve idf relative path %s: %w", request.RelativePath, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetIDFKyouByRelativePathRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 同一Rep内のIDFRepから対象ファイルのIDFKyouを逆引きする
	idfRepImpls, err := repositories.IDFKyouReps.UnWrapTyped()
	if err != nil {
		err = fmt.Errorf("error at unwrap idf kyou reps: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouByRelativePathError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	for _, idfRep := range idfRepImpls {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouByRelativePathError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if repName != idfKyou.RepName {
			continue
		}
		targetIDFKyou, err := idfRep.GetIDFKyouByTargetFile(r.Context(), resolvedTargetFile)
		if err != nil {
			err = fmt.Errorf("error at get idf kyou by target file %s: %w", resolvedTargetFile, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouByRelativePathError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if targetIDFKyou != nil {
			response.KyouID = targetIDFKyou.ID
			break
		}
	}
	// 見つからなかった場合はKyouID空文字のまま正常応答
}

// toSlashAnyOS は区切り文字を / に揃える。
//
// filepath.ToSlash はコンパイル先OSの区切り文字しか変換しない。
// Linux上では no-op なので `docs\a.md` がそのまま残り、
// path.Dir が "." を返してしまう。
// Markdownを書いた環境と読む環境でOSが違いうるため、両方の区切りを見る。
func toSlashAnyOS(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// isAbsAnyOS はPOSIX / Windows どちらの絶対パス表記かを判定する。
//
// filepath.IsAbs と filepath.VolumeName はコンパイル先OSの規則しか見ないので、
// Linux上では `C:\windows\system32` が相対パス扱いになり拒否されなかった。
// gkillはlinux_amd64/arm64やAndroidでも動くため、実行OSに依存しない判定にする。
// toSlashAnyOS を通したあとに呼ぶ前提で、UNCパス(`\\server\share`)は
// `//server/share` になるので先頭 / の判定で拾える。
func isAbsAnyOS(p string) bool {
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, "/") {
		return true
	}
	// ドライブレター表記 (C: / C:/foo)
	if len(p) >= 2 && p[1] == ':' {
		c := p[0]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}
	return false
}

// resolveIDFRelativePath は、基準ファイル (rep内相対パス) のディレクトリを基準に
// 相対パスを解決し、rep内相対パス (スラッシュ区切り) を返す。
// rep外を指すパス・絶対パスはエラーを返す。
func resolveIDFRelativePath(currentTargetFile string, relativePath string) (string, error) {
	rel := relativePath

	// フラグメント・クエリを除去
	if idx := strings.IndexAny(rel, "#?"); idx >= 0 {
		rel = rel[:idx]
	}

	// URLエンコードを解除（Markdown内のリンクはエンコードされていることがある）
	if unescaped, err := url.PathUnescape(rel); err == nil {
		rel = unescaped
	}

	if rel == "" {
		return "", fmt.Errorf("relative path is empty")
	}

	rel = toSlashAnyOS(rel)

	// 絶対パスは拒否
	if isAbsAnyOS(rel) {
		return "", fmt.Errorf("absolute path is not allowed: %s", relativePath)
	}

	// 基準ファイルのディレクトリを基準に解決
	dir := path.Dir(toSlashAnyOS(currentTargetFile))
	resolved := path.Join(dir, rel)

	// rep外へのパストラバーサルを拒否
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("path traversal is not allowed: %s", relativePath)
	}

	return resolved, nil
}
