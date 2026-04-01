package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleGetSharedKyous(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSharedKyousRequest{}
	response := &req_res.GetSharedKyousResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get shared kyous response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiSharedTasksResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get shared kyous request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTasksRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.SharedID)
	if err != nil || sharedKyouInfo == nil {
		err = fmt.Errorf("error at get ShareKyouListInfos shared id = %s: %w", request.SharedID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiSharedTasksError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := sharedKyouInfo.UserID
	device := sharedKyouInfo.Device

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	findQuery := &find.FindQuery{}
	err = json.Unmarshal([]byte(sharedKyouInfo.FindQueryJSON), findQuery)
	if err != nil {
		err = fmt.Errorf("error at parse query json at find kyous %#v: %w", findQuery, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTaskRequest,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	findQuery.OnlyLatestData = true

	// Kyou
	findFilter := &api.FindFilter{}
	kyous, _, err := findFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, findQuery)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	useIDs := len(kyous) != 0
	findQueryValueForKyouInstances := *findQuery
	findQueryForKyouInstances := &findQueryValueForKyouInstances
	findQueryForKyouInstances.UseIDs = useIDs
	findQueryForKyouInstances.IncludeCreateMi = true
	findQueryForKyouInstances.IncludeStartMi = true
	findQueryForKyouInstances.IncludeCheckMi = true
	findQueryForKyouInstances.IncludeEndMi = true
	findQueryForKyouInstances.IncludeLimitMi = true
	findQueryForKyouInstances.IncludeEndTimeIs = true
	findQueryForKyouInstances.IDs = []string{}
	for _, kyou := range kyous {
		findQueryForKyouInstances.IDs = append(findQueryForKyouInstances.IDs, kyou.ID)
	}
	findQueryForKyouInstances.OnlyLatestData = false

	// Mi
	mis, err := repositories.MiReps.FindMi(r.Context(), findQueryForKyouInstances)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// GPSLogs
	gpsLogs := []reps.GPSLog{}
	if sharedKyouInfo.IsShareWithLocations {
		gpsLogs, err = repositories.GPSLogReps.GetGPSLogs(r.Context(), findQuery.CalendarStartDate, findQuery.CalendarEndDate)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	kmemos := []reps.Kmemo{}
	kcs := []reps.KC{}
	timeiss := []reps.TimeIs{}
	nlogs := []reps.Nlog{}
	lantanas := []reps.Lantana{}
	urlogs := []reps.URLog{}
	idfKyous := []reps.IDFKyou{}
	rekyous := []reps.ReKyou{}
	gitCommitLogs := []reps.GitCommitLog{}
	if sharedKyouInfo.ViewType != "mi" {
		// Kmemo
		kmemos, err = repositories.KmemoReps.FindKmemo(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// KC
		kcs, err = repositories.KCReps.FindKC(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// TimeIs
		timeiss, err = repositories.TimeIsReps.FindTimeIs(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// Nlog
		nlogs, err = repositories.NlogReps.FindNlog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// Lantana
		lantanas, err = repositories.LantanaReps.FindLantana(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// URLogs
		urlogs, err = repositories.URLogReps.FindURLog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// IDFKyou
		idfKyous, err = repositories.IDFKyouReps.FindIDFKyou(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// ReKyou
		rekyous, err = repositories.ReKyouReps.FindReKyou(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// GitCommitLog
		gitCommitLogs, err = repositories.GitCommitLogReps.FindGitCommitLog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// AttachedTag
	tags := []reps.Tag{}
	tagSet := map[string]reps.Tag{}
	if sharedKyouInfo.IsShareWithTags {
		for _, kyou := range kyous {
			tagsRelatedID, err := repositories.GetTagsByTargetID(r.Context(), kyou.ID)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTagsShareKyouError,
					ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAGS_BY_TARGET_ID_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			for _, tag := range tagsRelatedID {
				tagSet[tag.ID] = tag
			}
		}
		for _, tag := range tagSet {
			tags = append(tags, tag)
		}
	}

	// AttachedText
	texts := []reps.Text{}
	textSet := map[string]reps.Text{}
	if sharedKyouInfo.IsShareWithTexts {
		for _, kyou := range kyous {
			textsRelatedID, err := repositories.GetTextsByTargetID(r.Context(), kyou.ID)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTextsShareKyouError,
					ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			for _, text := range textsRelatedID {
				textSet[text.ID] = text
			}
		}
		for _, text := range textSet {
			texts = append(texts, text)
		}
	}

	// AttachedTimeIs
	attachedTimeisKyous := []reps.Kyou{}
	attachedTimeiss := []reps.TimeIs{}
	if sharedKyouInfo.IsShareWithTimeIss {

		attachedTimeIsKyousMap := map[string]reps.Kyou{}
		attachedTimeIssMap := map[string]reps.TimeIs{}
		queries := []find.FindQuery{}

		timeisQueryValue := *findQuery
		timeisQuery := timeisQueryValue
		timeisQuery.UseRepTypes = true
		timeisQuery.RepTypes = []string{"timeis"}
		timeisQuery.OnlyLatestData = true
		queries = append(queries, timeisQuery)

		if timeisQuery.UseCalendar && timeisQuery.CalendarStartDate != nil {
			timeisPlaingHeadQuery := find.FindQuery{}
			timeisPlaingHeadQuery.UsePlaing = true
			timeisPlaingHeadQuery.PlaingTime = *timeisQuery.CalendarStartDate
			queries = append(queries, timeisPlaingHeadQuery)
		}

		if timeisQuery.UseCalendar && timeisQuery.CalendarEndDate != nil {
			timeisPlaingHipQuery := find.FindQuery{}
			timeisPlaingHipQuery.UsePlaing = true
			timeisPlaingHipQuery.PlaingTime = *timeisQuery.CalendarEndDate
			queries = append(queries, timeisPlaingHipQuery)
		}

		for _, query := range queries {
			findFilter := &api.FindFilter{}
			matchPlaingKyous, _, err := findFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, &query)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTextsShareKyouError,
					ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			for _, timeisKyou := range matchPlaingKyous {
				if existKyou, exist := attachedTimeIsKyousMap[timeisKyou.ID]; exist {
					if timeisKyou.UpdateTime.After(existKyou.UpdateTime) {
						attachedTimeIsKyousMap[timeisKyou.ID] = timeisKyou
					}
				} else {
					attachedTimeIsKyousMap[timeisKyou.ID] = timeisKyou
				}
			}

			ids := []string{}
			for id := range attachedTimeIsKyousMap {
				ids = append(ids, id)
			}
			if len(ids) != 0 {
				plaingTimeIss, err := repositories.TimeIsReps.FindTimeIs(r.Context(), &query)
				if err != nil {
					err = fmt.Errorf("error at find plaing timeis user id = %s device = %s: %w", userID, device, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.FindTextsShareKyouError,
						ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
				for _, timeis := range plaingTimeIss {
					attachedTimeIssMap[timeis.ID] = timeis
					if existTimeIs, exist := attachedTimeIssMap[timeis.ID]; exist {
						if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
							attachedTimeIssMap[timeis.ID] = timeis
						}
					} else {
						attachedTimeIssMap[timeis.ID] = timeis
					}
				}
			}

			for _, kyou := range attachedTimeIsKyousMap {
				attachedTimeisKyous = append(attachedTimeisKyous, kyou)
			}
			for _, timeis := range attachedTimeIssMap {
				attachedTimeiss = append(attachedTimeiss, timeis)
			}
		}
	}

	response.Kyous = kyous
	response.Mis = mis
	response.Kmemos = kmemos
	response.KCs = kcs
	response.TimeIss = timeiss
	response.Nlogs = nlogs
	response.Lantanas = lantanas
	response.URLogs = urlogs
	response.IDFKyous = idfKyous
	response.ReKyous = rekyous
	response.GitCommitLogs = gitCommitLogs
	response.GPSLogs = gpsLogs
	response.AttachedTags = tags
	response.AttachedTexts = texts
	response.Title = sharedKyouInfo.ShareTitle
	response.ViewType = sharedKyouInfo.ViewType
	response.AttachedTimeIss = attachedTimeiss
	response.AttachedTimeIsKyous = attachedTimeisKyous
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSharedTasksSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOU_MESSAGE"}),
	})
}
