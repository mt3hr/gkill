package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// HandleGetPluginList はログイン中の利用者が使えるプラグインの一覧を返します。
//
// POST /api/get_plugin_list（wrapAuth）
// req_res.GetPluginListRequest / req_res.GetPluginListResponse
//
// 各要素はプラグインのmanifestの内容に、プロセスが生きているか（IsAlive）を添えたものです。
// プラグインが1つも無い場合はPluginsをnilのまま正常応答します。
func (g *GkillServerAPI) HandleGetPluginList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetPluginListRequest{}
	response := &req_res.GetPluginListResponse{}

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
			err = fmt.Errorf("error at encode get plugin list response: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "error at encode get plugin list response", "error", fmt.Sprintf("%q", err))
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode get plugin list request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at decode get plugin list request", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetPluginListRequestDataError,
			ErrorMessage: "プラグイン一覧取得リクエストのパースに失敗しました",
		})
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID

	pm := g.GkillDAOManager.GetPluginManager(userID)
	for _, pluginRepo := range pm.GetPluginRepositories() {
		manifest := pluginRepo.GetManifest()
		info := req_res.PluginInfo{
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			DataType:    manifest.DataType,
			RepName:     manifest.RepName,
			// emits_kyou=false のとき DataType / RepName は検索に使える値ではない。
			// 役割（Kyouを出すのか、GPSログだけなのか）を呼び出し側が判別できるよう
			// manifest の宣言をそのまま返す（新しい語彙は作らない）。
			EmitsKyou: manifest.EmitsKyouOrDefault(),
			Provides:  providedKindStrings(manifest),
			IsAlive:   pluginRepo.IsAlive(r.Context()),
			// 受動情報。IsAlive(ping)と違い副作用なし
			ProcessRunning: pluginRepo.ProcessRunning(),
			// stderr末尾。「is_alive=trueなのに0件」の診断用（外部監査 D2）。
			// プラグインは別リポジトリの成果物で、診断のためにホームディレクトリや
			// 読み取り元の絶対パスを書く（書いてよい）。出口で伏せるのはこちらの責務。
			LastError: message.RedactEnvironmentSpecific(pluginRepo.LastStderr()),
		}
		// provides宣言のあるプラグインは索引統計（鮮度・件数・時刻範囲）も返す（外部監査 D1）
		if typedIndex := pluginRepo.TypedIndex(); typedIndex != nil {
			stats := typedIndex.Stats()
			statsDTO := &req_res.PluginTypedIndexStatsMCPDTO{
				OK:    stats.OK,
				State: stats.State,
				// 索引構築の失敗理由にはプラグインディレクトリの絶対パスが乗る
				// （起動失敗のエラーがそのまま文字列化されるため）。last_error と同じく伏せる。
				LastBuildError: message.RedactEnvironmentSpecific(stats.LastBuildError),
				RecordCount:    stats.RecordCount,
				Truncated:      stats.Truncated,
				BuiltAt:        stats.BuiltAt.In(time.Local).Format(time.RFC3339),
			}
			if !stats.LastAttemptAt.IsZero() {
				statsDTO.LastAttemptAt = stats.LastAttemptAt.In(time.Local).Format(time.RFC3339)
			}
			if !stats.Oldest.IsZero() {
				statsDTO.Oldest = stats.Oldest.In(time.Local).Format(time.RFC3339)
			}
			if !stats.Newest.IsZero() {
				statsDTO.Newest = stats.Newest.In(time.Local).Format(time.RFC3339)
			}
			info.TypedIndex = statsDTO
		}
		// GPSログを提供するプラグインの取り込み状況。型別索引とは別枠で返す
		// （GPSログはKyouではないので型別索引には1件も載らない）。
		if reporter, ok := pluginRepo.(interface {
			GPSIndexStats() *reps.GPSIndexStats
		}); ok {
			if gpsStats := reporter.GPSIndexStats(); gpsStats != nil {
				gpsDTO := &req_res.PluginGPSIndexStatsMCPDTO{
					PointCount: gpsStats.PointCount,
					FetchedAt:  gpsStats.FetchedAt.In(time.Local).Format(time.RFC3339),
				}
				if !gpsStats.Oldest.IsZero() {
					gpsDTO.Oldest = gpsStats.Oldest.In(time.Local).Format(time.RFC3339)
				}
				if !gpsStats.Newest.IsZero() {
					gpsDTO.Newest = gpsStats.Newest.In(time.Local).Format(time.RFC3339)
				}
				info.GPSIndex = gpsDTO
			}
		}
		response.Plugins = append(response.Plugins, info)
	}
}

// providedKindStrings は manifest の provides を文字列スライスにする。
// 宣言が無ければ nil を返し、JSON では omitempty でキーごと消える。
func providedKindStrings(manifest gkill_plugin.PluginManifest) []string {
	if len(manifest.Provides) == 0 {
		return nil
	}
	kinds := make([]string, 0, len(manifest.Provides))
	for _, kind := range manifest.Provides {
		kinds = append(kinds, string(kind))
	}
	return kinds
}
