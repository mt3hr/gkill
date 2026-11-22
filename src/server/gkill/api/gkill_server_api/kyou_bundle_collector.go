package gkill_server_api

// Kyou と付随データをまとめて集める処理。
// /api/get_shared_kyous（共有ページ）と「HTML で出力」の両方がここを通る。
//
// **コピーして2実装にしないこと。** ここには写し漏らすと静かに壊れるものが固まっている
// （とくに collectKyouBundle の中の IDs の扱い。コメントを参照）。

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// kyouBundleOptions は収集の指定。
type kyouBundleOptions struct {
	// UserID / Device は引くリポジトリの持ち主。
	// 共有では共有設定に記録されたもの、出力ではセッションのもの
	UserID string
	Device string

	// LocaleName はエラーメッセージの言語
	LocaleName string

	// FindQuery は収集の起点。**中身は書き換えない**（値コピーして使う）
	FindQuery *find.FindQuery

	// ViewType が "mi" のときは Mi 以外の型別データを引かない
	ViewType string

	// 付随データを含めるか。偽の項目は空のまま返る
	IncludeLocations bool
	IncludeTags      bool
	IncludeTexts     bool
	IncludeTimeIss   bool
}

// kyouBundle は集めた結果。どのスライスも nil にはならない。
type kyouBundle struct {
	Kyous []reps.Kyou
	Mis   []reps.Mi

	Kmemos        []reps.Kmemo
	KCs           []reps.KC
	TimeIss       []reps.TimeIs
	Nlogs         []reps.Nlog
	Lantanas      []reps.Lantana
	URLogs        []reps.URLog
	IDFKyous      []reps.IDFKyou
	ReKyous       []reps.ReKyou
	GitCommitLogs []reps.GitCommitLog

	GPSLogs []reps.GPSLog

	AttachedTags        []reps.Tag
	AttachedTexts       []reps.Text
	AttachedTimeIss     []reps.TimeIs
	AttachedTimeIsKyous []reps.Kyou
}

// bundleError はログを1行出して GkillError を作る。
// エラーコード・メッセージID・ログ文言は場面ごとに違うので呼び出し側から受ける
// （切り出し前のハンドラが出していたものをそのまま保つため）。
func bundleError(ctx context.Context, localeName string, errorCode string, messageID string, logMessage string, err error) *message.GkillError {
	slog.Log(ctx, gkill_log.Debug, logMessage, "error", fmt.Sprintf("%q", err))
	return &message.GkillError{
		ErrorCode:    errorCode,
		ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: messageID}),
		Cause:        err,
	}
}

// collectKyouBundle は FindQuery を実行して Kyou と型別データ・付随データを集める。
//
// 守ること:
//   - Kyou 一覧は最新版のみ（OnlyLatestData = true）を引き、型別データ・付随データは
//     そのIDに対して版を絞らずに引く
//   - **検索結果が0件のとき、実体取得の IDs は nil ではなく空スライスにする。**
//     nil は「ID で絞らない」の意味なので、実体取得（FindKmemo 等）がタグ・打刻・
//     実行中といった条件を解釈しないぶん素通しになり、共有していない記録の本文まで返る。
//     /api/get_shared_kyous は wrapNoAuth（共有IDを知っていれば誰でも叩ける）なので、
//     これがそのまま第三者への漏洩になる
func (g *GkillServerAPI) collectKyouBundle(ctx context.Context, opts *kyouBundleOptions) (*kyouBundle, *message.GkillError) {
	userID := opts.UserID
	device := opts.Device
	localeName := opts.LocaleName

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		return nil, bundleError(ctx, localeName, message.RepositoriesGetError,
			"FAILED_GET_KYOU_MESSAGE", "error at get repositories user id", err)
	}

	// 呼び出し側の FindQuery を壊さないよう値コピーしてから設定を足す
	findQueryValue := *opts.FindQuery
	findQuery := &findQueryValue
	findQuery.OnlyLatestData = true

	bundle := &kyouBundle{
		Kyous:               []reps.Kyou{},
		Mis:                 []reps.Mi{},
		Kmemos:              []reps.Kmemo{},
		KCs:                 []reps.KC{},
		TimeIss:             []reps.TimeIs{},
		Nlogs:               []reps.Nlog{},
		Lantanas:            []reps.Lantana{},
		URLogs:              []reps.URLog{},
		IDFKyous:            []reps.IDFKyou{},
		ReKyous:             []reps.ReKyou{},
		GitCommitLogs:       []reps.GitCommitLog{},
		GPSLogs:             []reps.GPSLog{},
		AttachedTags:        []reps.Tag{},
		AttachedTexts:       []reps.Text{},
		AttachedTimeIss:     []reps.TimeIs{},
		AttachedTimeIsKyous: []reps.Kyou{},
	}

	// Kyou
	findFilter := &api.FindFilter{}
	kyous, _, err := findFilter.FindKyous(ctx, userID, device, g.GkillDAOManager, findQuery)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		return nil, bundleError(ctx, localeName, message.FindKyousShareKyouError,
			"FAILED_GET_KYOU_MESSAGE", "error at find Kyous user id", err)
	}
	bundle.Kyous = kyous

	findQueryValueForKyouInstances := *findQuery
	findQueryForKyouInstances := &findQueryValueForKyouInstances
	findQueryForKyouInstances.IncludeCreateMi = true
	findQueryForKyouInstances.IncludeStartMi = true
	findQueryForKyouInstances.IncludeCheckMi = true
	findQueryForKyouInstances.IncludeEndMi = true
	findQueryForKyouInstances.IncludeLimitMi = true
	findQueryForKyouInstances.IncludeEndTimeIs = true
	if len(kyous) == 0 {
		// 検索結果なし → 実体も1件も返さない。
		// ここは「明示的に0件指定」である非nilの空スライスでなければならない（上の doc コメント）
		findQueryForKyouInstances.IDs = []string{}
	} else {
		ids := make([]string, 0, len(kyous))
		for _, kyou := range kyous {
			ids = append(ids, kyou.ID)
		}
		findQueryForKyouInstances.IDs = ids
	}
	findQueryForKyouInstances.OnlyLatestData = false

	// Mi
	mis, err := repositories.MiReps.FindMi(ctx, findQueryForKyouInstances)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		return nil, bundleError(ctx, localeName, message.FindKyousShareKyouError,
			"FAILED_GET_KYOU_MESSAGE", "error at find Kyous user id", err)
	}
	bundle.Mis = mis

	// GPSLogs
	if opts.IncludeLocations {
		gpsLogs, err := repositories.GPSLogReps.GetGPSLogs(ctx, findQuery.CalendarStartTime, findQuery.CalendarEndTime)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			return nil, bundleError(ctx, localeName, message.FindKyousShareKyouError,
				"FAILED_GET_KYOU_MESSAGE", "error at find Kyous user id", err)
		}
		bundle.GPSLogs = gpsLogs
	}

	if opts.ViewType != "mi" {
		if gkillErr := g.collectTypedDatas(ctx, opts, repositories, findQueryForKyouInstances, bundle); gkillErr != nil {
			return nil, gkillErr
		}
	}

	if gkillErr := g.collectAttachedDatas(ctx, opts, repositories, findQuery, bundle); gkillErr != nil {
		return nil, gkillErr
	}

	return bundle, nil
}

// collectTypedDatas は型別データを引く。ViewType が "mi" のときは呼ばれない。
func (g *GkillServerAPI) collectTypedDatas(ctx context.Context, opts *kyouBundleOptions, repositories *reps.GkillRepositories, findQueryForKyouInstances *find.FindQuery, bundle *kyouBundle) *message.GkillError {
	userID := opts.UserID
	device := opts.Device
	localeName := opts.LocaleName

	// エラーの出かたは全部同じなので、1つにまとめる
	fail := func(err error) *message.GkillError {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		return bundleError(ctx, localeName, message.FindKyousShareKyouError,
			"FAILED_GET_KYOU_MESSAGE", "error at find Kyous user id", err)
	}

	// Kmemo
	kmemos, err := repositories.KmemoReps.FindKmemo(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.Kmemos = kmemos

	// KC
	kcs, err := repositories.KCReps.FindKC(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.KCs = kcs

	// TimeIs
	timeiss, err := repositories.TimeIsReps.FindTimeIs(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.TimeIss = timeiss

	// Nlog
	nlogs, err := repositories.NlogReps.FindNlog(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.Nlogs = nlogs

	// Lantana
	lantanas, err := repositories.LantanaReps.FindLantana(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.Lantanas = lantanas

	// URLog
	urlogs, err := repositories.URLogReps.FindURLog(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.URLogs = urlogs

	// IDFKyou
	idfKyous, err := repositories.IDFKyouReps.FindIDFKyou(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.IDFKyous = idfKyous

	// ReKyou
	rekyous, err := repositories.ReKyouReps.FindReKyou(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.ReKyous = rekyous

	// GitCommitLog
	gitCommitLogs, err := repositories.GitCommitLogReps.FindGitCommitLog(ctx, findQueryForKyouInstances)
	if err != nil {
		return fail(err)
	}
	bundle.GitCommitLogs = gitCommitLogs

	return nil
}

// collectAttachedDatas はタグ・付随テキスト・付随 TimeIs を引く。
func (g *GkillServerAPI) collectAttachedDatas(ctx context.Context, opts *kyouBundleOptions, repositories *reps.GkillRepositories, findQuery *find.FindQuery, bundle *kyouBundle) *message.GkillError {
	userID := opts.UserID
	device := opts.Device
	localeName := opts.LocaleName

	// AttachedTag
	if opts.IncludeTags {
		tagSet := map[string]reps.Tag{}
		for _, kyou := range bundle.Kyous {
			tagsRelatedID, err := repositories.GetTagsByTargetID(ctx, kyou.ID)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				return bundleError(ctx, localeName, message.FindTagsShareKyouError,
					"FAILED_GET_TAGS_BY_TARGET_ID_MESSAGE", "error at find tags user id", err)
			}
			for _, tag := range tagsRelatedID {
				tagSet[tag.ID] = tag
			}
		}
		for _, tag := range tagSet {
			bundle.AttachedTags = append(bundle.AttachedTags, tag)
		}
	}

	// AttachedText
	if opts.IncludeTexts {
		textSet := map[string]reps.Text{}
		for _, kyou := range bundle.Kyous {
			textsRelatedID, err := repositories.GetTextsByTargetID(ctx, kyou.ID)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				return bundleError(ctx, localeName, message.FindTextsShareKyouError,
					"FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE", "error at find tags user id", err)
			}
			for _, text := range textsRelatedID {
				textSet[text.ID] = text
			}
		}
		for _, text := range textSet {
			bundle.AttachedTexts = append(bundle.AttachedTexts, text)
		}
	}

	// AttachedTimeIs
	if opts.IncludeTimeIss {
		if gkillErr := g.collectAttachedTimeIss(ctx, opts, repositories, findQuery, bundle); gkillErr != nil {
			return gkillErr
		}
	}

	return nil
}

// collectAttachedTimeIss は「その記録の時刻に実行中だった打刻」を集める。
//
// 期間に重なる打刻のほか、期間の両端で実行中だったものを PlayingTime で引く
// （開始が期間より前の打刻は期間の検索では出てこない）。
func (g *GkillServerAPI) collectAttachedTimeIss(ctx context.Context, opts *kyouBundleOptions, repositories *reps.GkillRepositories, findQuery *find.FindQuery, bundle *kyouBundle) *message.GkillError {
	userID := opts.UserID
	device := opts.Device
	localeName := opts.LocaleName

	attachedTimeIsKyousMap := map[string]reps.Kyou{}
	attachedTimeIssMap := map[string]reps.TimeIs{}
	queries := []find.FindQuery{}

	timeisQueryValue := *findQuery
	timeisQuery := timeisQueryValue
	timeisQuery.RepTypes = []string{"timeis"}
	timeisQuery.OnlyLatestData = true
	queries = append(queries, timeisQuery)

	if timeisQuery.CalendarStartTime != nil {
		timeisPlayingHeadQuery := find.FindQuery{}
		// 浅いコピー由来のポインタ共有を避けるため値コピーで新規に作る
		playingHeadTime := *timeisQuery.CalendarStartTime
		timeisPlayingHeadQuery.PlayingTime = &playingHeadTime
		queries = append(queries, timeisPlayingHeadQuery)
	}

	if timeisQuery.CalendarEndTime != nil {
		timeisPlayingHipQuery := find.FindQuery{}
		playingHipTime := *timeisQuery.CalendarEndTime
		timeisPlayingHipQuery.PlayingTime = &playingHipTime
		queries = append(queries, timeisPlayingHipQuery)
	}

	for _, query := range queries {
		findFilter := &api.FindFilter{}
		matchPlayingKyous, _, err := findFilter.FindKyous(ctx, userID, device, g.GkillDAOManager, &query)
		if err != nil {
			err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
			return bundleError(ctx, localeName, message.FindTextsShareKyouError,
				"FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE", "error at find tags user id", err)
		}
		for _, timeisKyou := range matchPlayingKyous {
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
			playingTimeIss, err := repositories.TimeIsReps.FindTimeIs(ctx, &query)
			if err != nil {
				err = fmt.Errorf("error at find playing timeis user id = %s device = %s: %w", userID, device, err)
				return bundleError(ctx, localeName, message.FindTextsShareKyouError,
					"FAILED_GET_TIMEIS_MESSAGE", "error at find playing timeis user id", err)
			}
			// 削除済みは Kyou 検索と同じ規則で落とす(ADR-0612)。ここは FindFilter を
			// 通らない直叩きなので、落とさないと「終了していない削除済みの打刻」が
			// 開始以降のすべての記録に *実行中* として永久に付き続ける。
			for _, timeis := range livePlayingTimeIsCandidates(playingTimeIss) {
				// 同じIDの版が複数来たら新しいほうを採る。
				// 以前は先に無条件で代入していたため、直後の比較が必ず自分自身との
				// 比較になり(else も到達不能で)、版の選択が働いていなかった。
				existTimeIs, exist := attachedTimeIssMap[timeis.ID]
				if !exist || timeis.UpdateTime.After(existTimeIs.UpdateTime) {
					attachedTimeIssMap[timeis.ID] = timeis
				}
			}
		}

		for _, kyou := range attachedTimeIsKyousMap {
			bundle.AttachedTimeIsKyous = append(bundle.AttachedTimeIsKyous, kyou)
		}
		for _, timeis := range attachedTimeIssMap {
			bundle.AttachedTimeIss = append(bundle.AttachedTimeIss, timeis)
		}
	}

	return nil
}
