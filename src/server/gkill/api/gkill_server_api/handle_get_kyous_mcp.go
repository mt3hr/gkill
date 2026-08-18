package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetKyousMCP は、MCPサーバ向けにKyouを検索し、型ごとのペイロードと
// 付随データ (タグ・テキスト・通知) を1件にまとめたDTOをページングして返します。
//
// POST /api/get_kyous_mcp（wrapNoAuth）
// req_res.GetKyousMCPRequest / req_res.GetKyousMCPResponse
//
// wrapNoAuth登録ですが、ハンドラ内でSessionIDからアカウントを解決するので未認証では使えません。
// Limitは1〜1000にクランプ (未指定は50)、MaxSizeMBの未指定は1.0、
// Queryはnilなら空のクエリに差し替え、いずれの場合も OnlyLatestData = true に上書きします。
// 並び順はRelatedTimeの降順で、Cursor (RFC3339) を渡すとその時刻より前の最初の件から再開します。
// Cursorはクエリの期間上限(CalendarEndDate)へ押し下げるため、2ページ目以降は
// 検索対象そのものが「カーソル以降」に縮みます。これに伴い TotalCount は
// 「条件に合う全件」ではなく「カーソル以降の残り件数」を指します（HasMoreの判定は変わりません）。
// DTOを1件ずつJSONにした累積サイズがMaxSizeMBを超えた時点で打ち切るため、
// ReturnedCountがLimitに満たないことがあります。
// Cursorは RFC3339 と日付のみ(YYYY-MM-DD)を受け、どちらでもないときはエラーを返します
// （黙って1ページ目に戻すとページングが終わらないため）。
// LimitとMaxSizeMBによる打ち切りは同一RelatedTimeのかたまりを割らないので、
// ReturnedCountがLimitを少し超えることがあります（割ると次ページが取りこぼすため）。
// NextCursorは RFC3339Nano で返します（秒へ切り捨てると同じ秒の内側が漏れるため）。
// URLogのサムネイル画像はAIクライアントで扱えないうえ巨大なので、DBから読む段階で外します。
// IDFペイロードのFilePathはローカルリクエストのときだけ入ります。
// Kyouのrep_nameはIncludeRepNameのときだけ載せます（全件に載せるとMaxSizeMBの打ち切りが早まるため）。
// ペイロードの分岐はDataTypeそのものではなく payloadKindOfDataType で寄せた種別で行います。
// Mi/MiReKyou/TimeIsのDataTypeは射影ごとに枝分かれする(mi_create、mirekyou_limit、
// timeis_start ...)ので、素の型名との完全一致では拾えません。
// そこにも当てはまらないKyouはプラグイン由来とみなし、rep_nameでプラグインを
// 引き当てて、本文の代わりにrep_name/kyou_idを載せたペイロードを返します
// （本文はgkill側に保存されておらず、別途コンテンツHTML取得が要るため）。
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousMCPResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous mcp request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// デフォルト設定
	const maxLimit = 1000
	if request.Limit <= 0 {
		request.Limit = 50
	} else if request.Limit > maxLimit {
		request.Limit = maxLimit
	}
	if request.MaxSizeMB <= 0 {
		request.MaxSizeMB = 1.0
	}
	if request.Query == nil {
		request.Query = &find.FindQuery{}
	}
	request.Query.OnlyLatestData = true

	// カーソルをクエリの期間上限へ押し下げる。
	//
	// ★これが無いと、ページ1枚(最大1000件)を返すためだけに毎回全期間を検索し直す。
	//   実データ(30年・56万件・376リポジトリ)では1リクエストあたり +1.8GB のメモリと
	//   数十分を要し、それがページ数(約568回)ぶん繰り返されてサーバが膨れ続ける
	//   (2026-08-16 実測)。ページングがサーバ側の仕事をまったく軽くしていなかった。
	//
	//   CalendarEndDate は RelatedTime の上限(境界を含む)としてSQLまで降りるので、
	//   ここへ落とせば検索対象そのものが「カーソル以降」に縮む。
	//   境界ちょうど(同一時刻)の件はこのあとのカーソル走査が従来どおり読み飛ばすため、
	//   返る中身は押し下げの前後で変わらない。
	//   呼び出し元が期間を指定している場合は狭いほうを採る。
	cursorTime := time.Time{}
	hasCursor := false
	if request.Cursor != "" {
		parsedCursorTime, parseErr := time.Parse(time.RFC3339, request.Cursor)
		if parseErr != nil {
			// 日付のみ(YYYY-MM-DD)も受ける。MCPサーバは日付のみを日時へ正規化してから
			// 送るが(normalization.mjsのnormalizeDateTimeString)、APIを直接叩く
			// クライアントはそのまま送ってくる。
			parsedCursorTime, parseErr = time.ParseInLocation(time.DateOnly, request.Cursor, time.Local)
		}
		if parseErr != nil {
			// ★解釈できないカーソルを黙って無視してはいけない。
			//   以前は無視して1ページ目を返していたため、呼び出し側は同じページを
			//   受け取り続け、ページングが永久に終わらなかった。
			err = fmt.Errorf("error at parse cursor %q: %w", request.Cursor, parseErr)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousMCPRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		cursorTime = parsedCursorTime
		hasCursor = true
		if request.Query.CalendarEndDate == nil || request.Query.CalendarEndDate.After(cursorTime) {
			request.Query.CalendarEndDate = &cursorTime
		}
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

	// Kyou一覧を取得
	allKyous, gkillErrors, err := g.FindFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, request.Query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous mcp: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			// 検索が失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
			// そのまま返すと errors:null + 0件 になり、呼び出し側からは
			// 「成功・該当0件」と区別が付かない。理由はEnsureNotEmptyのコメント。
			gkillErrors = message.EnsureNotEmpty(gkillErrors, message.FindKyousError,
				api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}))
		}
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	// related_time 降順ソート
	slices.SortFunc(allKyous, func(a, b reps.Kyou) int {
		return b.RelatedTime.Compare(a.RelatedTime)
	})

	totalCount := len(allKyous)

	// カーソル適用。
	// 期間上限へ押し下げ済みなので、ここで読み飛ばすのは境界(カーソルと同一時刻)ぶんだけ。
	startIdx := 0
	if hasCursor {
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

	batch := allKyous[startIdx:]

	// リポジトリを取得
	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError = &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 候補IDを収集
	// request.Limitは冒頭で[1,maxLimit]にクランプ済み。ここでは batch のインデックス範囲
	// (batch[i])を安全にするため candidateCount <= len(batch) にクランプする。
	candidateCount := request.Limit
	if candidateCount < 0 {
		candidateCount = 0
	}
	if candidateCount > len(batch) {
		candidateCount = len(batch)
	}
	if candidateCount > maxLimit {
		candidateCount = maxLimit
	}
	// ★同一RelatedTimeのかたまりの途中でページを切らない。
	//
	//   次ページは「カーソルより厳密に前」から始まるので、境界が同一時刻の連続の
	//   途中に落ちると、返しそこねた同時刻の残りが次ページからも漏れて永久に取れない。
	//   実データでは一括取り込みのIDFやFitbitの日次指標のように同時刻が並ぶため現実に起きる。
	//   ここでかたまりの終わりまで伸ばしておけば、Limitを少し超える代わりに取りこぼしが無くなる
	//   (伸びる量はそのかたまりの残り件数ぶんだけ)。
	for candidateCount > 0 && candidateCount < len(batch) &&
		batch[candidateCount].RelatedTime.Equal(batch[candidateCount-1].RelatedTime) {
		candidateCount++
	}
	candidateIDs := make([]string, 0)
	for i := range candidateCount {
		candidateIDs = append(candidateIDs, batch[i].ID)
	}

	// 候補ID用クエリを作成
	findQueryForBatch := &find.FindQuery{
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
	// URLogのサムネイルはbase64画像で、実データでは227行で90MBある(1行最大10MB)。
	// MCPの利用者はAIクライアントで画像本体を使えないため、DBから読む段階で外す。
	findQueryForURLog := *findQueryForBatch
	findQueryForURLog.ExcludeURLogThumbnailImage = true
	if urlogs, urlogErr := repositories.URLogReps.FindURLog(r.Context(), &findQueryForURLog); urlogErr == nil {
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

	// Mi / MiReKyou のFindは、5つのIncludeXxxMiがどのSQL射影を流すかのスイッチになっており、
	// 全falseだとUNIONの元が1本も無くなって必ず空を返す(mi_repository_sqlite3_impl.goの
	// len(sqlSegments)==0、mi_re_kyou_sql.goのbuildMiReKyouSQLも同じ)。
	// ここはID指定でペイロードの元データを引くだけなので、
	// 全行に当たる作成射影(CREATE_TIME IS NOT NULL)だけを立てる。
	findQueryForMi := *findQueryForBatch
	findQueryForMi.IncludeCreateMi = true

	miMap := map[string]reps.Mi{}
	if mis, miErr := repositories.MiReps.FindMi(r.Context(), &findQueryForMi); miErr == nil {
		for _, m := range mis {
			miMap[m.ID] = m
		}
	}

	miReKyouMap := map[string]reps.MiReKyou{}
	if miReKyous, miReKyouErr := repositories.MiReKyouReps.FindMiReKyou(r.Context(), &findQueryForMi); miReKyouErr == nil {
		for _, m := range miReKyous {
			miReKyouMap[m.ID] = m
		}
	}

	reKyouMap := map[string]reps.ReKyou{}
	if reKyous, reKyouErr := repositories.ReKyouReps.FindReKyou(r.Context(), findQueryForBatch); reKyouErr == nil {
		for _, rk := range reKyous {
			reKyouMap[rk.ID] = rk
		}
	}

	// プラグインをrep_name別に引けるようにする。
	// プラグインKyouの本文はgkill側に保存されないため、ペイロードには
	// コンテンツHTML取得に必要なrep_name/kyou_idを載せる。
	pluginManifestByRepName := map[string]gkill_plugin.PluginManifest{}
	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		pluginManifestByRepName[manifest.RepName] = manifest
	}

	// attached TimeIs を一括取得
	var allTimeIs []reps.TimeIs
	if request.ShouldIncludeTimeIs() {
		findAllTimeIsQuery := &find.FindQuery{OnlyLatestData: true, IncludeEndTimeIs: true}
		allTimeIs, _ = repositories.TimeIsReps.FindTimeIs(r.Context(), findAllTimeIsQuery)
	}

	// DTO構築ループ（サイズ監視）
	maxBytes := int64(request.MaxSizeMB * 1024 * 1024)
	runningSize := int64(0)
	// 容量ヒントは candidateCount(len(batch)以下にクランプ済み)を使う。
	// go/uncontrolled-allocation-size が上限ガードを認識できるよう、割り当てを
	// candidateCount <= len(batch) が成立するブランチ内に置く(この条件は常に真)。
	resultDTOs := make([]req_res.KyouMCPDTO, 0)
	if candidateCount <= len(batch) {
		resultDTOs = make([]req_res.KyouMCPDTO, 0, candidateCount)
	}
	// 直前に採用したKyouのRelatedTime。MaxSizeMBでの打ち切りが
	// 同一時刻のかたまりを割らないようにするために持つ（割ると次ページが取りこぼす）。
	lastAppendedRelatedTime := time.Time{}

	for i := range candidateCount {
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

		// attached TimeIs 取得
		var timeisDTOs []req_res.TimeIsMCPDTO
		if request.ShouldIncludeTimeIs() && len(allTimeIs) > 0 {
			for _, ti := range allTimeIs {
				inRange := kyou.RelatedTime.After(ti.StartTime)
				if ti.EndTime != nil {
					inRange = inRange && kyou.RelatedTime.Before(*ti.EndTime)
				}
				if inRange {
					tiTags, _ := repositories.TagReps.GetTagsByTargetID(r.Context(), ti.ID)
					tiTagStrings := make([]string, 0, len(tiTags))
					for _, tag := range tiTags {
						tiTagStrings = append(tiTagStrings, tag.Tag)
					}
					timeisDTOs = append(timeisDTOs, req_res.TimeIsMCPDTO{
						Title: ti.Title,
						Tags:  tiTagStrings,
					})
				}
			}
		}

		// ペイロード構築。
		// DataTypeは射影ごとに枝分かれする(mi_create, timeis_start ...)ので、
		// 種別へ寄せてから分岐する。詳細は payloadKindOfDataType を参照。
		var payload any
		switch payloadKindOfDataType(kyou.DataType) {
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
					Kind:        "urlog",
					Title:       u.Title,
					URL:         u.URL,
					Description: u.Description,
				}
			}
		case "idf":
			if idfk, ok := idfKyouMap[kyou.ID]; ok {
				mimeType := mime.TypeByExtension(filepath.Ext(idfk.TargetFile))
				repName := kyou.RepName
				// ファイル実パスは同一マシンのクライアントにしか意味がないので、ローカルリクエストのときだけ返す
				filePath := ""
				if isLocalRequest(r) {
					filePath = idfk.ContentPath
				}
				payload = req_res.IDFPayloadMCPDTO{
					Kind:     "idf",
					FileName: idfk.TargetFile,
					IsImage:  idfk.IsImage,
					IsVideo:  idfk.IsVideo,
					IsAudio:  idfk.IsAudio,
					IsZip:    idfk.IsZip,
					RepName:  repName,
					MimeType: mimeType,
					FilePath: filePath,
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
					CreateTime:        m.CreateTime,
					LimitTime:         m.LimitTime,
					EstimateStartTime: m.EstimateStartTime,
					EstimateEndTime:   m.EstimateEndTime,
				}
			}
		case "mirekyou":
			if m, ok := miReKyouMap[kyou.ID]; ok {
				payload = req_res.MiReKyouPayloadMCPDTO{
					Kind:              "mirekyou",
					TargetID:          m.TargetID,
					IsChecked:         m.IsChecked,
					BoardName:         m.BoardName,
					CreateTime:        m.CreateTime,
					LimitTime:         m.LimitTime,
					EstimateStartTime: m.EstimateStartTime,
					EstimateEndTime:   m.EstimateEndTime,
				}
			}
		case "rekyou":
			if rk, ok := reKyouMap[kyou.ID]; ok {
				payload = req_res.ReKyouPayloadMCPDTO{
					Kind:     "rekyou",
					TargetID: rk.TargetID,
				}
			}
		default:
			// 既存のdata_typeに該当しないKyouはプラグイン由来。
			// data_typeはプラグインが自由に決めるので、rep_nameで引き当てる。
			if manifest, ok := pluginManifestByRepName[kyou.RepName]; ok {
				payload = req_res.PluginPayloadMCPDTO{
					Kind:        "plugin",
					DataType:    kyou.DataType,
					RepName:     kyou.RepName,
					KyouID:      kyou.ID,
					PluginName:  manifest.Name,
					Description: manifest.Description,
				}
			}
		}

		dto := req_res.KyouMCPDTO{
			DataType:      kyou.DataType,
			RelatedTime:   kyou.RelatedTime.In(time.Local),
			Tags:          tagStrings,
			Texts:         textStrings,
			Notifications: notificationDTOs,
			TimeIs:        timeisDTOs,
			Payload:       payload,
		}
		if request.IncludeID {
			dto.ID = kyou.ID
		}
		if request.IncludeRepName {
			dto.RepName = kyou.RepName
		}

		dtoJSON, marshalErr := json.Marshal(dto)
		if marshalErr != nil {
			continue
		}

		// サイズ上限での打ち切りも、同一時刻のかたまりの途中では行わない。
		// ここで割ると、返しそこねた同時刻の記録が次ページ(カーソルより厳密に前)から
		// 漏れて永久に取れなくなる。かたまりを跨ぐまでは上限を超えても入れ続ける。
		inSameRelatedTimeGroup := len(resultDTOs) > 0 && kyou.RelatedTime.Equal(lastAppendedRelatedTime)
		if runningSize+int64(len(dtoJSON)) > maxBytes && !inSameRelatedTimeGroup {
			break
		}
		runningSize += int64(len(dtoJSON))
		resultDTOs = append(resultDTOs, dto)
		lastAppendedRelatedTime = kyou.RelatedTime
	}

	returnedCount := len(resultDTOs)
	hasMore := (startIdx + returnedCount) < totalCount
	nextCursor := ""
	if hasMore && returnedCount > 0 {
		// ★秒精度(time.RFC3339)で出してはいけない。
		//   RelatedTimeが小数秒を持つとき、秒へ切り捨てたカーソルは実際の時刻より前を指す。
		//   次ページは「カーソルより厳密に前」を取るので、切り捨てた秒の内側にある
		//   未返却の記録が丸ごと漏れる。RFC3339Nanoなら切り捨てが起きない
		//   (ISO 8601のままなので、受け取る側の time.RFC3339 パースでもそのまま読める)。
		nextCursor = resultDTOs[returnedCount-1].RelatedTime.Format(time.RFC3339Nano)
	}

	response.Kyous = resultDTOs
	response.TotalCount = totalCount
	response.ReturnedCount = returnedCount
	response.HasMore = hasMore
	response.NextCursor = nextCursor
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyousMCPSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOUS_MESSAGE"}),
	})
}

// payloadKindOfDataType は射影ごとに枝分かれしたDataTypeをペイロード種別へ寄せます。
//
// Mi / MiReKyou / TimeIs のDataTypeはリポジトリのSQLが射影名を焼き込むため
// (mi_create / mi_check / mi_limit / mi_start / mi_end、mirekyou_*、
// timeis_start / timeis_end)、素の "mi" / "timeis" とは一致しません。
// さらに FindFilter.overrideKyous がMi検索時にMiSortTypeへ合わせて付け替えます。
// 完全一致のswitchで書くと全射影がdefaultへ落ち、payloadごと消えます。
//
// mirekyou_ を mi_ より先に判定すること（接頭辞判定の順序の罠）。
// 単体取得経路は素の "mi" / "timeis" を使うので、既知の射影に当たらない値は
// そのまま返して呼び出し側のcaseに任せます。
func payloadKindOfDataType(dataType string) string {
	switch {
	case strings.HasPrefix(dataType, "mirekyou_"):
		return "mirekyou"
	case strings.HasPrefix(dataType, "mi_"):
		return "mi"
	case strings.HasPrefix(dataType, "timeis_"):
		return "timeis"
	}
	return dataType
}
