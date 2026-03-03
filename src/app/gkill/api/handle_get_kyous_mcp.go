package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleGetKyousMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyousMCPRequest{}
	response := &req_res.GetKyousMCPResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous mcp response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousMCPResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous mcp request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// デフォルト設定
	if request.Limit <= 0 {
		request.Limit = 50
	}
	if request.MaxSizeMB <= 0 {
		request.MaxSizeMB = 1.0
	}
	if request.Query == nil {
		request.Query = &find.FindQuery{}
	}
	request.Query.OnlyLatestData = true

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

	// Kyou一覧を取得
	allKyous, gkillErrors, err := g.FindFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, request.Query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous mcp: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	// related_time 降順ソート
	sort.Slice(allKyous, func(i, j int) bool {
		return allKyous[i].RelatedTime.After(allKyous[j].RelatedTime)
	})

	totalCount := len(allKyous)

	// カーソル適用
	startIdx := 0
	if request.Cursor != "" {
		cursorTime, parseErr := time.Parse(time.RFC3339, request.Cursor)
		if parseErr == nil {
			found := false
			for i, kyou := range allKyous {
				if kyou.RelatedTime.Before(cursorTime) {
					startIdx = i
					found = true
					break
				}
			}
			if !found {
				startIdx = len(allKyous)
			}
		}
	}

	batch := allKyous[startIdx:]

	// リポジトリを取得
	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError = &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 候補IDを収集
	candidateCount := request.Limit
	if candidateCount > len(batch) {
		candidateCount = len(batch)
	}
	candidateIDs := make([]string, 0, candidateCount)
	for i := 0; i < candidateCount; i++ {
		candidateIDs = append(candidateIDs, batch[i].ID)
	}

	// 候補ID用クエリを作成
	findQueryForBatch := &find.FindQuery{
		UseIDs:         true,
		IDs:            candidateIDs,
		OnlyLatestData: true,
	}

	// 各型の詳細データを一括取得してマップを構築
	kmemoMap := map[string]reps.Kmemo{}
	if kmemos, kmemoErr := repositories.KmemoReps.FindKmemo(r.Context(), findQueryForBatch); kmemoErr == nil {
		for _, k := range kmemos {
			kmemoMap[k.ID] = k
		}
	}

	kcMap := map[string]reps.KC{}
	if kcs, kcErr := repositories.KCReps.FindKC(r.Context(), findQueryForBatch); kcErr == nil {
		for _, k := range kcs {
			kcMap[k.ID] = k
		}
	}

	timeIsMap := map[string]reps.TimeIs{}
	if timeiss, timeIsErr := repositories.TimeIsReps.FindTimeIs(r.Context(), findQueryForBatch); timeIsErr == nil {
		for _, t := range timeiss {
			timeIsMap[t.ID] = t
		}
	}

	nlogMap := map[string]reps.Nlog{}
	if nlogs, nlogErr := repositories.NlogReps.FindNlog(r.Context(), findQueryForBatch); nlogErr == nil {
		for _, n := range nlogs {
			nlogMap[n.ID] = n
		}
	}

	lantanaMap := map[string]reps.Lantana{}
	if lantanas, lantanaErr := repositories.LantanaReps.FindLantana(r.Context(), findQueryForBatch); lantanaErr == nil {
		for _, l := range lantanas {
			lantanaMap[l.ID] = l
		}
	}

	urlogMap := map[string]reps.URLog{}
	if urlogs, urlogErr := repositories.URLogReps.FindURLog(r.Context(), findQueryForBatch); urlogErr == nil {
		for _, u := range urlogs {
			urlogMap[u.ID] = u
		}
	}

	idfKyouMap := map[string]reps.IDFKyou{}
	if idfKyous, idfErr := repositories.IDFKyouReps.FindIDFKyou(r.Context(), findQueryForBatch); idfErr == nil {
		for _, idfk := range idfKyous {
			idfKyouMap[idfk.ID] = idfk
		}
	}

	gitCommitLogMap := map[string]reps.GitCommitLog{}
	if gitCommitLogs, gitErr := repositories.GitCommitLogReps.FindGitCommitLog(r.Context(), findQueryForBatch); gitErr == nil {
		for _, gcl := range gitCommitLogs {
			gitCommitLogMap[gcl.ID] = gcl
		}
	}

	miMap := map[string]reps.Mi{}
	if mis, miErr := repositories.MiReps.FindMi(r.Context(), findQueryForBatch); miErr == nil {
		for _, m := range mis {
			miMap[m.ID] = m
		}
	}

	// DTO構築ループ（サイズ監視）
	maxBytes := int64(request.MaxSizeMB * 1024 * 1024)
	runningSize := int64(0)
	resultDTOs := make([]req_res.KyouMCPDTO, 0, candidateCount)

	for i := 0; i < candidateCount; i++ {
		kyou := batch[i]

		// タグ取得
		tags, _ := repositories.TagReps.GetTagsByTargetID(r.Context(), kyou.ID)
		tagStrings := make([]string, 0, len(tags))
		for _, tag := range tags {
			tagStrings = append(tagStrings, tag.Tag)
		}

		// テキスト取得
		texts, _ := repositories.TextReps.GetTextsByTargetID(r.Context(), kyou.ID)
		textStrings := make([]string, 0, len(texts))
		for _, text := range texts {
			textStrings = append(textStrings, text.Text)
		}

		// 通知取得
		notifications, _ := repositories.NotificationReps.GetNotificationsByTargetID(r.Context(), kyou.ID)
		notificationDTOs := make([]req_res.NotificationMCPDTO, 0, len(notifications))
		for _, n := range notifications {
			notificationDTOs = append(notificationDTOs, req_res.NotificationMCPDTO{
				Content:          n.Content,
				NotificationTime: n.NotificationTime,
				IsNotificated:    n.IsNotificated,
			})
		}

		// ペイロード構築
		var payload interface{}
		switch kyou.DataType {
		case "kmemo":
			if k, ok := kmemoMap[kyou.ID]; ok {
				payload = req_res.KmemoPayloadMCPDTO{
					Kind:    "kmemo",
					Content: k.Content,
				}
			}
		case "kc":
			if k, ok := kcMap[kyou.ID]; ok {
				payload = req_res.KCPayloadMCPDTO{
					Kind:     "kc",
					Title:    k.Title,
					NumValue: k.NumValue,
				}
			}
		case "timeis":
			if t, ok := timeIsMap[kyou.ID]; ok {
				payload = req_res.TimeIsPayloadMCPDTO{
					Kind:      "timeis",
					Title:     t.Title,
					StartTime: t.StartTime,
					EndTime:   t.EndTime,
				}
			}
		case "nlog":
			if n, ok := nlogMap[kyou.ID]; ok {
				payload = req_res.NlogPayloadMCPDTO{
					Kind:   "nlog",
					Title:  n.Title,
					Shop:   n.Shop,
					Amount: n.Amount,
				}
			}
		case "lantana":
			if l, ok := lantanaMap[kyou.ID]; ok {
				payload = req_res.LantanaPayloadMCPDTO{
					Kind: "lantana",
					Mood: l.Mood,
				}
			}
		case "urlog":
			if u, ok := urlogMap[kyou.ID]; ok {
				payload = req_res.URLogPayloadMCPDTO{
					Kind:  "urlog",
					Title: u.Title,
					URL:   u.URL,
				}
			}
		case "idf":
			if idfk, ok := idfKyouMap[kyou.ID]; ok {
				payload = req_res.IDFPayloadMCPDTO{
					Kind:     "idf",
					FileName: idfk.TargetFile,
				}
			}
		case "git_commit_log":
			if gcl, ok := gitCommitLogMap[kyou.ID]; ok {
				payload = req_res.GitPayloadMCPDTO{
					Kind:          "git_commit_log",
					CommitMessage: gcl.CommitMessage,
					Addition:      gcl.Addition,
					Deletion:      gcl.Deletion,
				}
			}
		case "mi":
			if m, ok := miMap[kyou.ID]; ok {
				payload = req_res.MiPayloadMCPDTO{
					Kind:              "mi",
					Title:             m.Title,
					IsChecked:         m.IsChecked,
					BoardName:         m.BoardName,
					LimitTime:         m.LimitTime,
					EstimateStartTime: m.EstimateStartTime,
					EstimateEndTime:   m.EstimateEndTime,
				}
			}
		}

		dto := req_res.KyouMCPDTO{
			DataType:      kyou.DataType,
			RelatedTime:   kyou.RelatedTime.In(time.Local),
			Tags:          tagStrings,
			Texts:         textStrings,
			Notifications: notificationDTOs,
			Payload:       payload,
		}

		dtoJSON, marshalErr := json.Marshal(dto)
		if marshalErr != nil {
			continue
		}

		if runningSize+int64(len(dtoJSON)) > maxBytes {
			break
		}
		runningSize += int64(len(dtoJSON))
		resultDTOs = append(resultDTOs, dto)
	}

	returnedCount := len(resultDTOs)
	hasMore := (startIdx + returnedCount) < totalCount
	nextCursor := ""
	if hasMore && returnedCount > 0 {
		nextCursor = resultDTOs[returnedCount-1].RelatedTime.Format(time.RFC3339)
	}

	response.Kyous = resultDTOs
	response.TotalCount = totalCount
	response.ReturnedCount = returnedCount
	response.HasMore = hasMore
	response.NextCursor = nextCursor
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyousMCPSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOUS_MESSAGE"}),
	})
}
