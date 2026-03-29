package api

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const (
	zipCacheSubDir = "zip_cache"
)

var (
	zipExtractGroup sync.Map // key: cacheDir, value: *sync.Mutex
)

func (g *GkillServerAPI) HandleBrowseZipContents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.BrowseZipContentsRequest{}
	response := &req_res.BrowseZipContentsResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse browse zip contents response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidBrowseZipContentsRequestDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse browse zip contents request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidBrowseZipContentsRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
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
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// IDFKyouを検索
	idfKyou, err := findIDFKyouByID(r.Context(), repositories, request.TargetID)
	if err != nil || idfKyou == nil {
		if err != nil {
			err = fmt.Errorf("error at find idf kyou by id = %s: %w", request.TargetID, err)
		} else {
			err = fmt.Errorf("idf kyou not found id = %s", request.TargetID)
		}
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.BrowseZipContentsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ZIPファイルかチェック
	if !idfKyou.IsZip {
		err = fmt.Errorf("target idf kyou is not a zip file id = %s", request.TargetID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.BrowseZipContentsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 物理ファイルパスを解決
	zipFilePath := idfKyou.ContentPath
	if zipFilePath == "" {
		err = fmt.Errorf("content path is empty for idf kyou id = %s", request.TargetID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.BrowseZipContentsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// キャッシュディレクトリを決定
	repName := idfKyou.RepName
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(zipFilePath)))
	cacheRootDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, zipCacheSubDir, repName))
	cacheDir := filepath.Join(cacheRootDir, hash)

	// singleflight的に一度だけ展開
	extractErr := extractZipOnce(zipFilePath, cacheDir)
	if extractErr != nil {
		err = fmt.Errorf("error at extract zip file %s: %w", zipFilePath, extractErr)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.BrowseZipContentsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// キャッシュディレクトリをwalkしてエントリを構築
	entries, err := buildZipEntries(cacheDir, repName, hash)
	if err != nil {
		err = fmt.Errorf("error at build zip entries from cache dir %s: %w", cacheDir, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.BrowseZipContentsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_BROWSE_ZIP_CONTENTS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Entries = entries
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.BrowseZipContentsSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_BROWSE_ZIP_CONTENTS_MESSAGE"}),
	})
}

func findIDFKyouByID(ctx context.Context, repositories *reps.GkillRepositories, targetID string) (*reps.IDFKyou, error) {
	idfKyou, err := repositories.IDFKyouReps.GetIDFKyou(ctx, targetID, nil)
	if err != nil {
		return nil, fmt.Errorf("error at get idf kyou id = %s: %w", targetID, err)
	}
	return idfKyou, nil
}

func extractZipOnce(zipFilePath string, cacheDir string) error {
	// 同じcacheDirへの同時展開を防止するmutex
	muVal, _ := zipExtractGroup.LoadOrStore(cacheDir, &sync.Mutex{})
	mu := muVal.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// 既にキャッシュが存在する場合はスキップ
	if _, err := os.Stat(cacheDir); err == nil {
		return nil
	}

	return extractZip(zipFilePath, cacheDir)
}

func extractZip(zipFilePath string, cacheDir string) error {
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("error at open zip file %s: %w", zipFilePath, err)
	}
	defer reader.Close()

	// 一時ディレクトリに展開してからリネーム（原子的展開）
	tmpDir := cacheDir + ".tmp"
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return fmt.Errorf("error at create tmp dir %s: %w", tmpDir, err)
	}

	for _, f := range reader.File {
		// シンボリックリンクはスキップ
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}

		// ファイル名のエンコーディング変換
		// ZIP仕様: Flags bit 11 (0x800) が立っていればUTF-8、そうでなければレガシーエンコーディング
		entryName := decodeZipEntryName(f)

		// パストラバーサル防止
		name := filepath.FromSlash(entryName)
		name = filepath.Clean(name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}

		destPath := filepath.Join(tmpDir, name)
		// destPathがtmpDir配下であることを確認
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(tmpDir)+string(os.PathSeparator)) && filepath.Clean(destPath) != filepath.Clean(tmpDir) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
				return fmt.Errorf("error at create dir %s: %w", destPath, err)
			}
			continue
		}

		// 親ディレクトリを作成
		if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
			return fmt.Errorf("error at create parent dir for %s: %w", destPath, err)
		}

		if err := extractZipFile(f, destPath); err != nil {
			return err
		}
	}

	// 原子的にリネーム
	os.RemoveAll(cacheDir)
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		return fmt.Errorf("error at rename tmp dir to cache dir: %w", err)
	}

	return nil
}

// decodeZipEntryName はZIPエントリのファイル名をUTF-8に変換する。
// ZIP仕様ではFlags bit 11 (0x800) が立っていればUTF-8。
// そうでない場合、日本語環境ではShift_JIS (CP932) が使われることが多いため、
// UTF-8でなければShift_JISとしてデコードを試みる。
func decodeZipEntryName(f *zip.File) string {
	name := f.Name

	// UTF-8フラグが立っている場合はそのまま
	if f.Flags&0x800 != 0 {
		return name
	}

	// 既にvalid UTF-8ならそのまま
	if utf8.ValidString(name) {
		// pure ASCIIや既にUTF-8なケース
		return name
	}

	// Shift_JIS → UTF-8 変換を試みる
	decoded, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), name)
	if err != nil {
		return name // 変換失敗時は元のまま
	}
	return decoded
}

func extractZipFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("error at open zip entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error at create file %s: %w", destPath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	if err != nil {
		return fmt.Errorf("error at write file %s: %w", destPath, err)
	}

	return nil
}

func buildZipEntries(cacheDir string, repName string, hash string) ([]*req_res.ZipEntry, error) {
	var entries []*req_res.ZipEntry

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// キャッシュディレクトリ自体はスキップ
		if path == cacheDir {
			return nil
		}

		rel, err := filepath.Rel(cacheDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		// ファイルURLを構築
		segments := strings.Split(rel, "/")
		escapedSegments := make([]string, len(segments))
		for i, seg := range segments {
			escapedSegments[i] = url.PathEscape(seg)
		}
		fileURL := "/zip_cache/" + url.PathEscape(repName) + "/" + hash + "/" + strings.Join(escapedSegments, "/")

		entry := &req_res.ZipEntry{
			Path:    rel,
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			IsImage: reps.IsImagePublic(rel),
			FileURL: fileURL,
		}
		entries = append(entries, entry)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (g *GkillServerAPI) HandleZipCacheFileServe(w http.ResponseWriter, r *http.Request) {
	// クッキーを見て認証する
	sessionIDCookie, err := r.Cookie("gkill_session_id")
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle zip cache file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
		return
	}
	sessionID := sessionIDCookie.Value

	// アカウントを取得
	account, _, err := g.getAccountFromSessionID(r.Context(), sessionID, "")
	if account == nil || err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle zip cache file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "finish", "error", err)
		return
	}

	// zip_cacheディレクトリからファイルを配信
	cacheRootDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, zipCacheSubDir))
	http.StripPrefix("/zip_cache/", http.FileServer(http.Dir(cacheRootDir))).ServeHTTP(w, r)
}
