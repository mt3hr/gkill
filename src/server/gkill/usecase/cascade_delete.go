package usecase

// 記録の連鎖削除（ADR-0636）。画面（旧 src/client/classes/cascade-delete-kyou.ts）と MCP（gkill_delete_kyou）が
// /api/delete_kyou_cascade 経由で同じこの1実装を使う。以前は画面の TypeScript だけが連鎖させており、
// MCP は対象1件だけを消していたので、付いていたタグ・テキスト・通知と、参照していたリポスト・タスク化が残った。

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// MaxCascadeDeleteDepth は参照の連鎖を辿る深さの上限。サーバの逆引きが壊れたときに無限に掘り進まないための保険で、
// 実データで届く値ではない（旧 TS 実装の max_cascade_depth と同じ）。
const MaxCascadeDeleteDepth = 32

// CascadeDeletedRecord は連鎖削除で消した1件。DataType はエンティティ名（kmemo / mi / mirekyou / tag / text /
// notification / rekyou / idf …）で、MCP の gkill_restore_kyou の targets[] にそのまま渡せる。
type CascadeDeletedRecord struct {
	ID       string
	DataType string
}

// cascadeDeleteEntityPrefixes は data_type（射影名を含む）からエンティティ名を決める表。
// mirekyou_* は "mi" で始まるので mi より先に判定する（旧 TS 実装の delete_kyou_body と同じ順）。
var cascadeDeleteEntityPrefixes = []string{"kmemo", "kc", "urlog", "nlog", "timeis", "mirekyou", "mi", "lantana", "idf", "rekyou"}

// CascadeDeleteEntityOf は連鎖削除の起点にできる data_type をエンティティ名にする。
// git_commit_log・プラグインの記録・付随データ（tag / text / notification）は起点にできないので false。
func CascadeDeleteEntityOf(dataType string) (string, bool) {
	for _, prefix := range cascadeDeleteEntityPrefixes {
		if strings.HasPrefix(dataType, prefix) {
			return prefix, true
		}
	}
	return "", false
}

// cascadeDeleteTargets は書く前に集めた削除対象。浅い順（起点に近い順）に積み、書くときは逆順に積む。
type cascadeDeleteTargets struct {
	tags          []reps.Tag
	texts         []reps.Text
	notifications []reps.Notification
	rekyous       []reps.ReKyou
	mirekyous     []reps.MiReKyou
}

// cascadeDeleteStamp は全件に同じ値で刻む削除の印。履歴のタイムスタンプがばらけないように update_time は全件で揃える。
type cascadeDeleteStamp struct {
	updateTime time.Time
	updateApp  string
	updateUser string
	device     string
}

// DeleteKyouCascade は記録1件と、それに付いているタグ・テキスト・通知、それを参照しているリポスト（ReKyou）・
// タスク化（MiReKyou）を、参照の連鎖を辿ってまとめて論理削除します。
//
// 探索（読むだけ）と書き込みを完全に分けます。参照先を消したあとでは辿れなくなる可能性があるためです。
// 探索が1つでも失敗したら何も書きません（見えていない参照元を残したまま本体だけ消すと「参照先の無いリポスト」が残る）。
// 書き込みは全件を1つの txID で temp rep に積み、1件でも積めなければ DiscardTx、全部積めたら CommitTx します。
// CommitTx は1つの SQLite トランザクションなので、**全部消えるか、何も消えないか**のどちらかです（ADR-0219 / ADR-0410）。
//
// dataType は起点の記録の種別で、射影名（mi_create / timeis_start / mirekyou_create …）でもかまいません。
// 戻り値は消したもの全部（起点・付随データ・参照元）です。
func (uc *UsecaseContext) DeleteKyouCascade(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string, dataType string, updateApp string) ([]CascadeDeletedRecord, []*message.GkillError, error) {
	entity, ok := CascadeDeleteEntityOf(dataType)
	if !ok {
		err := fmt.Errorf("data type %q cannot be the root of a cascade delete", dataType)
		slog.Log(ctx, gkill_log.Debug, "unsupported data type for cascade delete", "error", fmt.Sprintf("%q", err))
		return nil, []*message.GkillError{{
			ErrorCode:    message.DeleteKyouCascadeUnsupportedDataTypeError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
			Cause:        err,
		}}, nil
	}
	if updateApp == "" {
		updateApp = "gkill"
	}

	// 起点の最新版。型別の GetXxx(id, nil) は最新版を返す。
	root, rootUpdateTime, rootIsDeleted, err := getCascadeDeleteRoot(ctx, repositories, entity, targetID)
	if err != nil {
		err = fmt.Errorf("error at get cascade delete root user id = %s device = %s id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get cascade delete root", "error", fmt.Sprintf("%q", err))
		return nil, []*message.GkillError{{
			ErrorCode:    message.DeleteKyouCascadeError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
			Cause:        err,
		}}, nil
	}
	if root == nil {
		return nil, []*message.GkillError{{
			ErrorCode:    message.DeleteKyouCascadeTargetNotFoundError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
		}}, nil
	}
	if rootIsDeleted {
		return nil, []*message.GkillError{{
			ErrorCode:    message.DeleteKyouCascadeAlreadyDeletedError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
		}}, nil
	}

	targets, gkillErrors := uc.discoverCascadeDeleteTargets(ctx, repositories, userID, device, localeName, targetID)
	if len(gkillErrors) != 0 {
		return nil, gkillErrors, nil
	}

	stamp := cascadeDeleteStamp{
		updateTime: cascadeDeleteUpdateTime(time.Now(), rootUpdateTime, targets),
		updateApp:  updateApp,
		updateUser: userID,
		device:     device,
	}
	txID := uuid.NewString()
	gkillErrors = uc.stageCascadeDelete(ctx, repositories, userID, device, localeName, entity, root, targets, stamp, txID)
	if len(gkillErrors) != 0 {
		discardCascadeDeleteTx(ctx, repositories, txID, userID, device)
		return nil, gkillErrors, nil
	}
	if _, err := repositories.CommitTx(ctx, txID, userID, device); err != nil {
		err = fmt.Errorf("error at commit cascade delete tx id = %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(ctx, gkill_log.Error, "error at commit cascade delete", "error", fmt.Sprintf("%q", err))
		discardCascadeDeleteTx(ctx, repositories, txID, userID, device)
		return nil, []*message.GkillError{{
			ErrorCode:    message.DeleteKyouCascadeError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
			Cause:        err,
		}}, nil
	}

	deleted := []CascadeDeletedRecord{{ID: targetID, DataType: entity}}
	for _, rekyou := range targets.rekyous {
		deleted = append(deleted, CascadeDeletedRecord{ID: rekyou.ID, DataType: "rekyou"})
	}
	for _, mirekyou := range targets.mirekyous {
		deleted = append(deleted, CascadeDeletedRecord{ID: mirekyou.ID, DataType: "mirekyou"})
	}
	for _, tag := range targets.tags {
		deleted = append(deleted, CascadeDeletedRecord{ID: tag.ID, DataType: "tag"})
	}
	for _, text := range targets.texts {
		deleted = append(deleted, CascadeDeletedRecord{ID: text.ID, DataType: "text"})
	}
	for _, notification := range targets.notifications {
		deleted = append(deleted, CascadeDeletedRecord{ID: notification.ID, DataType: "notification"})
	}
	return deleted, nil, nil
}

// discardCascadeDeleteTx は temp rep に積んだ行を捨てる。失敗はログだけ（本命のエラーが別にあり、これは後始末。
// temp はインメモリなので次の起動で消える）。
func discardCascadeDeleteTx(ctx context.Context, repositories *reps.GkillRepositories, txID, userID, device string) {
	if err := repositories.DiscardTx(ctx, txID, userID, device); err != nil {
		slog.Log(ctx, gkill_log.Error, "error at discard cascade delete tx", "error", fmt.Sprintf("%q", fmt.Errorf("tx id = %s: %w", txID, err)))
	}
}

// getCascadeDeleteRoot は起点の記録の最新版を型別に取る。無ければ (nil, …, nil)。
func getCascadeDeleteRoot(ctx context.Context, repositories *reps.GkillRepositories, entity string, id string) (any, time.Time, bool, error) {
	switch entity {
	case "kmemo":
		x, err := repositories.KmemoReps.GetKmemo(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "kc":
		x, err := repositories.KCReps.GetKC(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "urlog":
		x, err := repositories.URLogReps.GetURLog(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "nlog":
		x, err := repositories.NlogReps.GetNlog(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "timeis":
		x, err := repositories.TimeIsReps.GetTimeIs(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "mi":
		x, err := repositories.MiReps.GetMi(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "mirekyou":
		x, err := repositories.MiReKyouReps.GetMiReKyou(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "lantana":
		x, err := repositories.LantanaReps.GetLantana(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "idf":
		x, err := repositories.IDFKyouReps.GetIDFKyou(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	case "rekyou":
		x, err := repositories.ReKyouReps.GetReKyou(ctx, id, nil)
		if err != nil || x == nil {
			return nil, time.Time{}, false, err
		}
		return x, x.UpdateTime, x.IsDeleted, nil
	}
	return nil, time.Time{}, false, fmt.Errorf("unsupported entity %q", entity)
}

// discoverCascadeDeleteTargets は消すべきものを幅優先で集める。書き込みは1本もしない。
// 訪問済みの id の集合で循環参照（A→B→A や自己参照）を止める。
func (uc *UsecaseContext) discoverCascadeDeleteTargets(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, rootID string) (*cascadeDeleteTargets, []*message.GkillError) {
	targets := &cascadeDeleteTargets{}
	visited := map[string]struct{}{rootID: {}}
	frontier := []string{rootID}
	for depth := 0; len(frontier) != 0; depth++ {
		if depth > MaxCascadeDeleteDepth {
			err := fmt.Errorf("references are nested deeper than %d levels from %s", MaxCascadeDeleteDepth, rootID)
			slog.Log(ctx, gkill_log.Debug, "cascade delete depth exceeded", "error", fmt.Sprintf("%q", err))
			return nil, []*message.GkillError{{
				ErrorCode:    message.DeleteKyouCascadeDepthExceededError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "CASCADE_DELETE_DEPTH_EXCEEDED_MESSAGE"}),
				Cause:        err,
			}}
		}
		nextFrontier := []string{}
		for _, id := range frontier {
			tags, gkillErrors, err := uc.GetTagsByTargetID(ctx, repositories, userID, device, localeName, id)
			if gkillErrs := cascadeDiscoveryErrors(gkillErrors, err, localeName); len(gkillErrs) != 0 {
				return nil, gkillErrs
			}
			texts, gkillErrors, err := uc.GetTextsByTargetID(ctx, repositories, userID, device, localeName, id)
			if gkillErrs := cascadeDiscoveryErrors(gkillErrors, err, localeName); len(gkillErrs) != 0 {
				return nil, gkillErrs
			}
			notifications, gkillErrors, err := uc.GetNotificationsByTargetID(ctx, repositories, userID, device, localeName, id)
			if gkillErrs := cascadeDiscoveryErrors(gkillErrors, err, localeName); len(gkillErrs) != 0 {
				return nil, gkillErrs
			}
			rekyous, gkillErrors, err := uc.GetReKyousByTargetID(ctx, repositories, userID, device, localeName, id)
			if gkillErrs := cascadeDiscoveryErrors(gkillErrors, err, localeName); len(gkillErrs) != 0 {
				return nil, gkillErrs
			}
			mirekyous, gkillErrors, err := uc.GetMiReKyousByTargetID(ctx, repositories, userID, device, localeName, id)
			if gkillErrs := cascadeDiscoveryErrors(gkillErrors, err, localeName); len(gkillErrs) != 0 {
				return nil, gkillErrs
			}

			// 付随データの逆引きは削除済みを元から除いて返す
			targets.tags = append(targets.tags, tags...)
			targets.texts = append(targets.texts, texts...)
			targets.notifications = append(targets.notifications, notifications...)

			for _, rekyou := range latestReKyous(rekyous) {
				if _, seen := visited[rekyou.ID]; rekyou.IsDeleted || seen {
					continue
				}
				visited[rekyou.ID] = struct{}{}
				targets.rekyous = append(targets.rekyous, rekyou)
				nextFrontier = append(nextFrontier, rekyou.ID)
			}
			for _, mirekyou := range latestMiReKyous(mirekyous) {
				if _, seen := visited[mirekyou.ID]; mirekyou.IsDeleted || seen {
					continue
				}
				visited[mirekyou.ID] = struct{}{}
				targets.mirekyous = append(targets.mirekyous, mirekyou)
				nextFrontier = append(nextFrontier, mirekyou.ID)
			}
		}
		frontier = nextFrontier
	}
	return targets, nil
}

// cascadeDiscoveryErrors は逆引き1本の失敗を「探索の失敗（何も消さない）」へまとめる。
func cascadeDiscoveryErrors(gkillErrors []*message.GkillError, err error, localeName string) []*message.GkillError {
	if len(gkillErrors) != 0 {
		return gkillErrors
	}
	if err != nil {
		return []*message.GkillError{{
			ErrorCode:    message.DeleteKyouCascadeError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
			Cause:        err,
		}}
	}
	return nil
}

// latestReKyous は同じ id の履歴が複数返ったとき update_time が最大の1件だけを残す（追記型なので）。順序は最初に現れた順。
func latestReKyous(list []reps.ReKyou) []reps.ReKyou {
	index := map[string]int{}
	out := []reps.ReKyou{}
	for _, item := range list {
		if i, ok := index[item.ID]; ok {
			if item.UpdateTime.After(out[i].UpdateTime) {
				out[i] = item
			}
			continue
		}
		index[item.ID] = len(out)
		out = append(out, item)
	}
	return out
}

// latestMiReKyous は latestReKyous の MiReKyou 版。
func latestMiReKyous(list []reps.MiReKyou) []reps.MiReKyou {
	index := map[string]int{}
	out := []reps.MiReKyou{}
	for _, item := range list {
		if i, ok := index[item.ID]; ok {
			if item.UpdateTime.After(out[i].UpdateTime) {
				out[i] = item
			}
			continue
		}
		index[item.ID] = len(out)
		out = append(out, item)
	}
	return out
}

// cascadeDeleteUpdateTime は全件に刻む update_time。UPDATE_TIME は1秒解像度で保存されるので、
// どれかの現在の版と同じ秒に落ちると新しい版が最新と見なされず、黙って消えない。
// そのときは、いちばん新しい現在の版の次の秒へ進める（MCP の nextUpdateTime と同じ考え方）。
func cascadeDeleteUpdateTime(now time.Time, rootUpdateTime time.Time, targets *cascadeDeleteTargets) time.Time {
	latest := rootUpdateTime
	consider := func(t time.Time) {
		if t.After(latest) {
			latest = t
		}
	}
	for _, x := range targets.tags {
		consider(x.UpdateTime)
	}
	for _, x := range targets.texts {
		consider(x.UpdateTime)
	}
	for _, x := range targets.notifications {
		consider(x.UpdateTime)
	}
	for _, x := range targets.rekyous {
		consider(x.UpdateTime)
	}
	for _, x := range targets.mirekyous {
		consider(x.UpdateTime)
	}
	latestSecond := latest.Truncate(time.Second)
	if now.Truncate(time.Second).After(latestSecond) {
		return now
	}
	return latestSecond.Add(time.Second)
}

// stageCascadeDelete は集めたものに削除の印を付けて、1つの txID で temp rep へ積む。
// 付随データ → 参照元（深い方から）→ 起点の順（commit は1つのトランザクションなので順に意味は無いが、
// 「起点は最後」の前提を読み手に崩さない）。1件でも積めなければそこで止める。
func (uc *UsecaseContext) stageCascadeDelete(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, entity string, root any, targets *cascadeDeleteTargets, stamp cascadeDeleteStamp, txID string) []*message.GkillError {
	tx := &txID
	for _, x := range targets.tags {
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		if _, gkillErrors, err := uc.UpdateTag(ctx, repositories, userID, device, localeName, x, tx); len(gkillErrors) != 0 || err != nil {
			return cascadeStageErrors(gkillErrors, err, localeName)
		}
	}
	for _, x := range targets.texts {
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		if _, gkillErrors, err := uc.UpdateText(ctx, repositories, userID, device, localeName, x, tx); len(gkillErrors) != 0 || err != nil {
			return cascadeStageErrors(gkillErrors, err, localeName)
		}
	}
	for _, x := range targets.notifications {
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		if _, gkillErrors, err := uc.UpdateNotification(ctx, repositories, userID, device, localeName, x, tx); len(gkillErrors) != 0 || err != nil {
			return cascadeStageErrors(gkillErrors, err, localeName)
		}
	}
	for i := len(targets.rekyous) - 1; i >= 0; i-- {
		x := targets.rekyous[i]
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		if gkillErrors, err := uc.UpdateReKyou(ctx, repositories, userID, device, localeName, x, tx); len(gkillErrors) != 0 || err != nil {
			return cascadeStageErrors(gkillErrors, err, localeName)
		}
	}
	for i := len(targets.mirekyous) - 1; i >= 0; i-- {
		x := targets.mirekyous[i]
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		if gkillErrors, err := uc.UpdateMiReKyou(ctx, repositories, userID, device, localeName, x, tx); len(gkillErrors) != 0 || err != nil {
			return cascadeStageErrors(gkillErrors, err, localeName)
		}
	}
	gkillErrors, err := uc.stageCascadeDeleteRoot(ctx, repositories, userID, device, localeName, entity, root, stamp, tx)
	if len(gkillErrors) != 0 || err != nil {
		return cascadeStageErrors(gkillErrors, err, localeName)
	}
	return nil
}

// stageCascadeDeleteRoot は起点に削除の印を付けて積む。data_type ごとに更新のユースケースが違う。
func (uc *UsecaseContext) stageCascadeDeleteRoot(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, entity string, root any, stamp cascadeDeleteStamp, tx *string) ([]*message.GkillError, error) {
	switch x := root.(type) {
	case *reps.Kmemo:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateKmemo(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.KC:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateKC(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.URLog:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateURLog(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.Nlog:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateNlog(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.TimeIs:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateTimeIs(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.Mi:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateMi(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.MiReKyou:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateMiReKyou(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.Lantana:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateLantana(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.IDFKyou:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateIDFKyou(ctx, repositories, userID, device, localeName, *x, tx)
	case *reps.ReKyou:
		x.IsDeleted, x.UpdateTime, x.UpdateApp, x.UpdateUser, x.UpdateDevice = true, stamp.updateTime, stamp.updateApp, stamp.updateUser, stamp.device
		return uc.UpdateReKyou(ctx, repositories, userID, device, localeName, *x, tx)
	}
	return nil, errors.New("unsupported cascade delete root " + entity)
}

// cascadeStageErrors は積めなかった1件の失敗を返す。gkill のエラーがあればそれを、無ければ汎用の「何も消えていない」。
func cascadeStageErrors(gkillErrors []*message.GkillError, err error, localeName string) []*message.GkillError {
	if len(gkillErrors) != 0 {
		return gkillErrors
	}
	return []*message.GkillError{{
		ErrorCode:    message.DeleteKyouCascadeError,
		ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
		Cause:        err,
	}}
}
