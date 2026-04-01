package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleURLogBookmarkletAddress(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	request := &req_res.URLogBookmarkletRequest{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse urlog bookmarklet request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidURLogBookmarkletRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		// response.Errors = append(response.Errors, gkillError)
		_ = gkillError
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionIDWithApplicationName(r.Context(), request.SessionID, "urlog_bookmarklet", request.LocaleName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
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
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	var imgBase64 string
	if request.ImageURL != "" {
		imgBase64, err = httpGetBase64Data(request.ImageURL)
		if err != nil {
			err = fmt.Errorf("error at http get base 64 data from %s: %w", request.ImageURL, err)
			log.Printf("err = %+v\n", err)
		}
	}
	var faviconBase64 string
	if request.FaviconURL != "" {
		faviconBase64, err = httpGetBase64Data(request.FaviconURL)
		if err != nil {
			err = fmt.Errorf("error at http get base 64 data from %s: %w", request.FaviconURL, err)
			log.Printf("err = %+v\n", err)
		}
	}

	urlog := &reps.URLog{
		IsDeleted:      false,
		ID:             GenerateNewID(),
		RelatedTime:    time.Now(),
		CreateTime:     time.Now(),
		CreateApp:      "urlog_bookmarklet",
		CreateDevice:   device,
		CreateUser:     userID,
		UpdateTime:     time.Now(),
		UpdateApp:      "urlog_bookmarklet",
		UpdateUser:     userID,
		UpdateDevice:   device,
		URL:            request.URL,
		Title:          request.Title,
		Description:    request.Description,
		FaviconImage:   faviconBase64,
		ThumbnailImage: imgBase64,
	}

	// 対象が存在する場合はエラー
	existURLog, err := repositories.URLogReps.GetURLog(r.Context(), urlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", urlog.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AlreadyExistURLogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// applicationConfigを取得
	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	urlog.FillURLogField(serverConfig, applicationConfig)

	err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), *urlog)
	if err != nil {
		err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddURLogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, urlog.ID)

	if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
		err = repositories.URLogReps[0].AddURLogInfo(r.Context(), *urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[urlog.ID] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              urlog.IsDeleted,
		TargetID:                               urlog.ID,
		DataUpdateTime:                         urlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[urlog.ID])
	if err != nil {
		err = fmt.Errorf("error at add or update latest data repository address for urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}

	// 通知する
	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}

	// 送信対象を取得する
	notificationTargets, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(r.Context(), userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}

	content := &struct {
		Content        string    `json:"content"`
		URL            string    `json:"url"`
		Time           time.Time `json:"time"`
		IsNotification bool      `json:"is_notification"`
	}{
		Content:        urlog.Title,
		URL:            "/kyou?kyou_id=" + urlog.ID,
		Time:           urlog.CreateTime,
		IsNotification: true,
	}
	contentJSONb, err := json.Marshal(content)
	if err != nil {
		err = fmt.Errorf("error at marshal webpush content: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}

	for _, notificationTarget := range notificationTargets {
		subscription := string(notificationTarget.Subscription)
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subscription), s)
		resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  currentServerConfig.GkillNotificationPublicKey,
			VAPIDPrivateKey: currentServerConfig.GkillNotificationPrivateKey,
			TTL:             0,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		if resp.Body != nil {
			defer func() {
				err := resp.Body.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(r.Context(), notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	}
}
