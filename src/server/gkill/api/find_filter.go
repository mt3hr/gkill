package api

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

const (
	NoTags = "no tags"
)

// containsNoTags はタグ条件に「タグ無し」仮想タグ(NoTags)が含まれるかを返します。
//
// 照合は完全一致。NoTags の判定は filterTagsKyous / filterTagsTimeIs でも
// `tag == NoTags` の完全一致で行っているので、ここだけ大小無視にしてはいけません
// (片方だけ当たると「タグなし集合を作らないのに NoTags 分岐へ入る」= 常に0件になります)。
func containsNoTags(tags []string) bool {
	return slices.Contains(tags, NoTags)
}

type FindFilter struct {
}

func (f *FindFilter) FindKyous(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, findQuery *find.FindQuery) ([]reps.Kyou, []*message.GkillError, error) {
	// ReKyou/MiReKyouのターゲット解決を1検索の中で使い回すためのメモ。
	// 委譲が入れ子になっていて、同じqueryでの解決が何度も走る。
	// メモが無くても各repは今までどおり自前で解決するので、
	// ここを通らない経路(単体テスト・repの直叩き)も動く。
	ctx = reps.WithTargetResolutionMemo(ctx)

	findKyouContext := &FindKyouContext{}

	// QueryをContextに入れる
	findKyouContext.UserID = userID
	findKyouContext.Device = device
	findKyouContext.GkillDAOManager = gkillDAOManager
	findKyouContext.ParsedFindQuery = findQuery
	findKyouContext.MatchReps = map[string]reps.Repository{}
	findKyouContext.AllHideTagsWhenUnchecked = map[string]reps.Tag{}
	findKyouContext.MatchHideTagsWhenUncheckedKyou = map[string]reps.Tag{}
	findKyouContext.MatchHideTagsWhenUncheckedTimeIs = map[string]reps.Tag{}
	findKyouContext.RelatedTagIDs = map[string]struct{}{}
	findKyouContext.MatchTags = map[string]reps.Tag{}
	findKyouContext.MatchTexts = map[string]reps.Text{}
	findKyouContext.MatchTimeIssAtFindTimeIs = map[string]reps.TimeIs{}
	findKyouContext.MatchTimeIssAtFilterTags = map[string]reps.TimeIs{}
	findKyouContext.MatchTimeIsTags = map[string]reps.Tag{}
	findKyouContext.MatchTimeIsTexts = map[string]reps.Text{}
	findKyouContext.MatchKyousCurrent = map[string][]reps.Kyou{}
	// メモリキャッシュ有効の場合、型につき1つのDBに全部のデータがある。
	// だから、LatestDataRepositoryAddressを知る必要がない
	findKyouContext.DisableLatestDataRepositoryCache = gkill_options.IsCacheInMemory

	// ユーザのRep取得
	gkillErr, err := f.getRepositories(ctx, userID, device, gkillDAOManager, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish getRepositories")

	// フィルタ
	gkillErr, err = f.selectMatchRepsFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at select match reps: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish selectMatchRepsFromQuery")
	if findKyouContext.ParsedFindQuery.UpdateCache {
		gkillErr, err = f.updateCache(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at update cache: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish updateCache")

	}

	// 「件数を見て分岐 → 読み込み → 書き戻し」を1つのロックで囲む必要があるので、
	// GkillRepositories側のメソッドにまとめてある
	if err := findKyouContext.Repositories.RefreshLatestDataRepositoryAddresses(ctx); err != nil {
		return nil, nil, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish update latest data repository address")

	wg := &sync.WaitGroup{}
	// 容量は送信箇所数(現在6: タグ2+タグ検索1+テキスト1+TimeIs2)以上であればよい(全送信がブロックしないためのバッファ)
	errch := make(chan error, 6)
	gkillErrch := make(chan []*message.GkillError, 6)
	defer close(errch)
	defer close(gkillErrch)

	catchErrFunc := func() ([]*message.GkillError, error) {
		return drainFindErrors(wg, errch, gkillErrch)
	}

	// タグの取得は**1回のスキャンで済ませる**。
	//
	// ここで作るのは2つ。
	//   - MatchTags        … クエリのタグ名に一致するタグ(Kyouタグ絞り込み用)
	//   - RelatedTagIDs    … タグが1つでも付いているIDの集合(「タグ無し」仮想タグ用)
	//
	// **タグ名の絞り込みをSQLへ降ろしてはいけない。** tag rep のワード検索は
	// `LOWER(TAG) = LOWER(?)` を出すので、列に関数がかかって索引が効かず、
	// **全行に LOWER() を適用**したうえでクエリのタグ名の数だけ繰り返す。
	// 実データのプロファイル(2026-08-19)では、条件なしの全タグ取得が3.3秒だったのに対し
	// 名前で絞る findTags が40.4秒 ―― **絞り込むほうが12倍高かった**
	// (うち sqlite の _lowerFunc が17.1秒)。
	// 全部取ってGo側で `strings.EqualFold` で照合すれば同じ結果がはるかに安く出る。
	// 照合の意味論は filterTagsKyous のAND分岐と同じ(完全一致・大小無視)。
	needsMatchTags := findQuery.Tags != nil
	needsRelatedTagIDs := containsNoTags(findQuery.Tags) || (findQuery.HasTimeIsFilter() && containsNoTags(findQuery.TimeIsTags))
	if needsMatchTags || needsRelatedTagIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ge, e := f.collectTagsForFilter(ctx, findKyouContext, needsMatchTags, needsRelatedTagIDs)
			if e != nil {
				e = fmt.Errorf("error at collect tags for filter: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish collectTagsForFilter", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()
	}

	// 非表示タグ(チェックが外れているタグ)の集合。
	// こちらは NoTags と無関係に、タグ絞り込みの結果から対象を消すために使うので、
	// 「タグ絞り込みを使っているか」で起動する
	if findQuery.Tags != nil || (findQuery.HasTimeIsFilter() && findQuery.TimeIsTags != nil) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ge, e := f.getAllHideTagsWhenUnChecked(ctx, findKyouContext, userID, device)
			if e != nil {
				e = fmt.Errorf("error at get hide tags when unchecked tags: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish getAllHideTagsWhenUnChecked", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()
	}

	// テキスト取得
	if findQuery.HasWordFilter() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ge, e := f.findTexts(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at find texts: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish findTexts", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()
	}

	if findQuery.HasTimeIsFilter() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ge, e := f.findTimeIsTexts(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at find timeis texts: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish findTimeIsTexts", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			ge, e := f.findTimeIsTags(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at find timeis tags: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish findTimeIsTags", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()
	}

	// タグなどの取得待ち（drainFindErrorsの中で待ち合わせてから回収する）
	gkillErr, err = catchErrFunc()
	if err != nil {
		return nil, gkillErr, err
	}
	// 並列取得で回収したGkillErrorは後続ステップのgkillErr代入で上書きされるため、
	// 別変数に持ち回って成功時にも返す(以前は最終returnで常にnilに潰されていた)
	parallelGkillErrs := gkillErr

	// TimeIs取得
	if findQuery.HasTimeIsFilter() {
		gkillErr, err = f.findTimeIs(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at find timeis: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish findTimeIs", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

		// 非表示タグ集合はfilterTagsTimeIsが適用する。
		// 以前は条件が「TimeIsタグを使わないとき」と逆になっており、
		// 適用側と噛み合わずTimeIsの非表示タグが一度も機能していなかった
		if findQuery.TimeIsTags != nil {
			gkillErr, err = f.getMatchHideTagsWhenUnckedTimeIs(ctx, findKyouContext)
			if err != nil {
				err = fmt.Errorf("error at get match hide tags when unchecked timeis: %w", err)
				return nil, gkillErr, err
			}
			slog.Log(ctx, gkill_log.Trace, "finish getMatchHideTagsWhenUnckedTimeIs", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
		}

		gkillErr, err = f.filterTagsTimeIs(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at filter tags timeis: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish filterTagsTimeIs", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	}

	gkillErr, err = f.findKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish findKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.sortAndTrimKyousMap(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort and trim kyousMap: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish sortAndTrimKyousMap", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterMiForMi(ctx, findKyouContext) //miの場合のみ
	if err != nil {
		err = fmt.Errorf("error at filter mi for mi: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterMiForMi", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	if findQuery.Tags != nil {
		gkillErr, err = f.getMatchHideTagsWhenUnckedKyou(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at get match hide tags when unchecked kyou: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish getMatchHideTagsWhenUnckedKyou", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.Tags != nil {
		gkillErr, err = f.filterTagsKyous(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at filter tags kyous: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish filterTagsKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	}
	gkillErr, err = f.filterPlaingTimeIsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter plaing time is kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterPlaingTimeIsKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterLocationKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter location kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterLocationKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterImageKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter image kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterImageKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.replaceLatestKyouInfos(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at replace latest kyou infos: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish replaceLatestKyouInfos", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.overrideKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at override kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish overrideKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	// 先に総数を数えてから確保する。事前確保しないと56万件で約20回の再確保が起き、
	// そのたびに確保済みぶん(最終的に130MB級)をコピーし直すことになる。
	// マップをもう1周するほうが遥かに安い。
	totalResultKyouCount := 0
	for _, kyous := range findKyouContext.MatchKyousCurrent {
		totalResultKyouCount += len(kyous)
	}
	findKyouContext.ResultKyous = slices.Grow(findKyouContext.ResultKyous, totalResultKyouCount)
	for _, kyous := range findKyouContext.MatchKyousCurrent {
		findKyouContext.ResultKyous = append(findKyouContext.ResultKyous, kyous...)
	}

	gkillErr, err = f.sortResultKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort result kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish sortResultKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	return findKyouContext.ResultKyous, parallelGkillErrs, nil
}

func (f *FindFilter) getRepositories(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	repositories, err := gkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		return nil, err
	}
	findCtx.Repositories = repositories

	return nil, nil
}

func (f *FindFilter) selectMatchRepsFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	repositories := findCtx.Repositories

	// Step1: タイプ系フィルタ（ForMi / IsImageOnly / PlaingTime指定 / RepTypes指定）で候補repを構築する
	// rep名指定（Reps）の有無に関わらず先に評価することで、rep種別指定がrep名指定に依存していたバグを修正する
	// 複数指定された場合は和集合にする（以前はif/else ifだったため、
	// ForMiとrep種別指定を併用するとRepTypesが無視されていた）
	typeMatchReps := []reps.Repository{}
	hasTypeFilter := findCtx.ParsedFindQuery.ForMi ||
		findCtx.ParsedFindQuery.IsImageOnly ||
		findCtx.ParsedFindQuery.PlaingTime != nil ||
		findCtx.ParsedFindQuery.RepTypes != nil

	if findCtx.ParsedFindQuery.ForMi {
		// ForMiだったらMi/MiReKyou以外は無視する
		for _, rep := range repositories.MiReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
		for _, rep := range repositories.MiReKyouReps.MiReKyouRepositories {
			typeMatchReps = append(typeMatchReps, rep)
		}
	}
	if findCtx.ParsedFindQuery.IsImageOnly {
		// ImageOnlyだったらIDFRep以外は無視する
		for _, rep := range repositories.IDFKyouReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	}
	if findCtx.ParsedFindQuery.PlaingTime != nil {
		// PlaingだったらTimeIsRep以外は無視する
		for _, rep := range repositories.TimeIsReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	}
	// RepTypes は nil=未指定 / 非nil空=タイプ候補0件（hasTypeFilterは真のまま）
	if findCtx.ParsedFindQuery.RepTypes != nil {
		// RepType指定の場合、指定以外は除外する
		for _, repType := range findCtx.ParsedFindQuery.RepTypes {
			switch repType {
			case "kmemo":
				for _, rep := range repositories.KmemoReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "kc":
				for _, rep := range repositories.KCReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "urlog":
				for _, rep := range repositories.URLogReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "timeis":
				for _, rep := range repositories.TimeIsReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "mi":
				for _, rep := range repositories.MiReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "nlog":
				for _, rep := range repositories.NlogReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "lantana":
				for _, rep := range repositories.LantanaReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "rekyou":
				for _, rep := range repositories.ReKyouReps.ReKyouRepositories {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "mirekyou":
				for _, rep := range repositories.MiReKyouReps.MiReKyouRepositories {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "directory":
				for _, rep := range repositories.IDFKyouReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "git_commit_log":
				for _, rep := range repositories.GitCommitLogReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			}
		}
	}

	// Step2: タイプフィルタもrep名指定も無し → 全repをそのまま追加して終了
	// （Reps は nil=未指定 / 非nil空=rep名候補0件=0件）
	if !hasTypeFilter && findCtx.ParsedFindQuery.Reps == nil {
		for _, rep := range repositories.Reps {
			rep := rep
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				return nil, err
			}
			if _, exist := findCtx.MatchReps[repName]; !exist {
				findCtx.MatchReps[repName] = rep
			}
		}
		return nil, nil
	}

	// タイプフィルタなし（rep名のみ指定）→ 全repを候補にする
	if !hasTypeFilter {
		typeMatchReps = append(typeMatchReps, repositories.Reps...)
	}

	// Step3: rep名指定なし → typeMatchReps を全てMatchRepsへ追加して終了
	//
	// ここでUnWrap()してはいけない。
	// typeMatchRepsの要素は repositories.MiReps などから採っており、
	// --cache_in_memory（既定true）ではインメモリキャッシュのrepが入っている。
	// UnWrap()するとその中の生のディスクrepに戻ってしまい、
	// mi板・画像のみ・Plaing・rep種別指定の検索だけがキャッシュを丸ごとバイパスして
	// 重複rep（同一ファイルの端末別登録）ぶんディスクを舐めることになる。
	// rep名での絞り込みが要るのはStep4だけなので、ここは名前解決も不要。
	if findCtx.ParsedFindQuery.Reps == nil {
		for _, matchRep := range typeMatchReps {
			repName, err := matchRep.GetRepName(ctx)
			if err != nil {
				return nil, err
			}
			if _, exist := findCtx.MatchReps[repName]; !exist {
				findCtx.MatchReps[repName] = matchRep
			}
		}
		return nil, nil
	}

	// Step4: rep名指定あり → typeMatchRepsをrep名でさらにフィルタ
	targetRepNames := findCtx.ParsedFindQuery.Reps

	for _, matchRep := range typeMatchReps {
		repImpls, err := matchRep.UnWrap()
		if err != nil {
			return nil, err
		}
		for _, repImpl := range repImpls {
			repName, err := repImpl.GetRepName(ctx)
			if err != nil {
				return nil, err
			}

			for _, targetRepName := range targetRepNames {
				if targetRepName == repName {
					if _, exist := findCtx.MatchReps[repName]; !exist {
						findCtx.MatchReps[repName] = repImpl
					}
					break
				}
			}
		}
	}
	return nil, nil
}

func (f *FindFilter) updateCache(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	err := findCtx.Repositories.UpdateCache(ctx)
	if err != nil {
		err = fmt.Errorf("error at update repositories cache: %w", err)
		return nil, err
	}
	findCtx.ParsedFindQuery.UpdateCache = false
	return nil, nil
}

// maxTagNamesForSQLFilter はタグ名の絞り込みをSQLへ降ろす上限です。
//
// tag rep のワード検索は `LOWER(TAG) = LOWER(?) OR LOWER(ID) = LOWER(?)` を出す。
// 列に関数がかかるので索引が効かず、**全行に LOWER() を適用**したうえで、
// それをクエリのタグ名の数だけ繰り返す ―― つまり O(行数 × 名前の数)。
// 一方「全部取ってGoで照合」は名前の数によらず O(行数) だが、
// reps.Tag(240バイト + 文字列10本)を行数ぶん実体化する確保を毎回払う。
//
// どちらが安いかは名前の個数で逆転する(dao/reps/tag_find_bench_test.go の実測)。
// 2万タグで交差点はおよそ30。SQL側もGo側も行数に比例するので、
// 交差する「名前の個数」は行数によらずほぼ一定。
// 確保の少ないSQL側に寄せたいので、閾値は交差点よりやや上に置く。
const maxTagNamesForSQLFilter = 32

// collectTagsForFilter はタグ絞り込みに要る2つを作ります。
//
//   - MatchTags     … クエリのタグ名に一致するタグ(needMatchTags のとき)
//   - RelatedTagIDs … タグが1つでも付いているIDの集合(needRelatedTagIDs のとき)
//
// **「タグ無し」仮想タグを使う検索では、名前の照合もGo側でやる。**
// RelatedTagIDs のために結局は全タグを取るので、そこから名前を拾うぶんはタダになる。
// 以前は「全タグの取得」と「名前で絞る検索」を別々に投げていて、
// 本番のプロファイル(2026-08-19)ではタグ名の絞り込みだけで実質CPUの44%(40.4秒)を使っていた。
//
// 照合は `strings.EqualFold` の完全一致・大小無視。filterTagsKyous のAND分岐と同じ意味論で、
// SQLが出していた `LOWER(TAG) = LOWER(?)` と等価。SQL は TAG 列だけでなく
// ID 列とも突き合わせていたので、そこも写してある。
func (f *FindFilter) collectTagsForFilter(ctx context.Context, findCtx *FindKyouContext, needMatchTags bool, needRelatedTagIDs bool) ([]*message.GkillError, error) {
	var queryTagNames []string
	if needMatchTags {
		// 同じ名前を何度も比べない
		queryTagNames = uniqueStrings(findCtx.ParsedFindQuery.Tags)
	}

	// 全タグを取るなら、名前の照合もそこからやったほうが安い。
	// 全タグを取らないなら、名前が少ないうちはSQLに絞らせたほうが安い
	if !needRelatedTagIDs && len(queryTagNames) <= maxTagNamesForSQLFilter {
		return f.findTagsByNameInSQL(ctx, findCtx)
	}

	// 全タグ取得用検索クエリ。IDごとの最新版のみを対象にする
	findTagsQuery := &find.FindQuery{IsDeleted: false, OnlyLatestData: true}

	allTagsList, err := collectFromRepos([]reps.TagRepository(findCtx.Repositories.TagReps), func(tagRep reps.TagRepository) ([]reps.Tag, error) {
		return tagRep.FindTags(ctx, findTagsQuery)
	})
	if err != nil {
		return nil, fmt.Errorf("error at get all tags: %w", err)
	}

	matchesQueryTagName := func(tag reps.Tag) bool {
		for _, queryTagName := range queryTagNames {
			if strings.EqualFold(queryTagName, tag.Tag) || strings.EqualFold(queryTagName, tag.ID) {
				return true
			}
		}
		return false
	}

	// rep跨ぎでIDごとの最新版を決める。保持するのは判定に要る3つだけ。
	// ここで reps.Tag(240バイト)をそのまま持つと実データで約180MBになる
	type latestTagRef struct {
		updateTime time.Time
		targetID   string
		isDeleted  bool
	}
	latestTags := make(map[string]latestTagRef, len(allTagsList))
	// 名前が一致したタグだけは実体で持つ(件数が少ないので安い)
	matchTagCandidates := map[string]reps.Tag{}
	for _, tag := range allTagsList {
		if existing, exist := latestTags[tag.ID]; exist && !tag.UpdateTime.After(existing.updateTime) {
			continue
		}
		latestTags[tag.ID] = latestTagRef{updateTime: tag.UpdateTime, targetID: tag.TargetID, isDeleted: tag.IsDeleted}
		if !needMatchTags {
			continue
		}
		if matchesQueryTagName(tag) {
			matchTagCandidates[tag.ID] = tag
			continue
		}
		// 新しい版で名前が変わって一致しなくなったなら、古い版の分を取り消す
		delete(matchTagCandidates, tag.ID)
	}

	if needRelatedTagIDs {
		// タグの対象をリスト
		for id, tag := range latestTags {
			if !findCtx.isLatestData(id, tag.updateTime) {
				continue
			}
			if tag.isDeleted {
				continue
			}
			findCtx.RelatedTagIDs[tag.targetID] = struct{}{}
		}
	}

	for id, tag := range matchTagCandidates {
		if !findCtx.isLatestData(id, tag.UpdateTime) {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		findCtx.MatchTags[id] = tag
	}

	return nil, nil
}

// findTagsByNameInSQL はタグ名の絞り込みをSQLへ降ろします。
// 名前が少ないうちは全タグの実体化を避けられるぶんこちらが安い(上の閾値を参照)。
func (f *FindFilter) findTagsByNameInSQL(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	query := &find.FindQuery{
		// IsDeleted: false, // TagReps.FindTags内に考慮があるため削除
		Words:    findCtx.ParsedFindQuery.Tags,
		WordsAnd: false,
		// 編集前のタグ名でヒットしないよう、IDごとの最新版のみを対象にする
		OnlyLatestData: true,
	}
	matchTags, err := findCtx.Repositories.TagReps.FindTags(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error at get tags by name %#v: %w", findCtx.ParsedFindQuery.Tags, err)
	}
	for _, tag := range matchTags {
		if !findCtx.isLatestData(tag.ID, tag.UpdateTime) {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		findCtx.MatchTags[tag.ID] = tag
	}
	return nil, nil
}

func (f *FindFilter) getAllHideTagsWhenUnChecked(ctx context.Context, findCtx *FindKyouContext, userID string, device string) ([]*message.GkillError, error) {
	hideTagNames := []string{}
	if findCtx.ParsedFindQuery.HideTags != nil {
		hideTagNames = append(hideTagNames, findCtx.ParsedFindQuery.HideTags...)
	}

	for _, hideTagName := range hideTagNames {
		hideTagsInReps, err := findCtx.Repositories.TagReps.GetTagsByTagName(ctx, hideTagName)
		if err != nil {
			err = fmt.Errorf("error at get tags by tagname tagname=%s: %w", hideTagName, err)
			return nil, err
		}
		for _, hideTag := range hideTagsInReps {
			if !findCtx.isLatestData(hideTag.ID, hideTag.UpdateTime) {
				continue
			}
			if hideTag.IsDeleted {
				continue
			}
			findCtx.AllHideTagsWhenUnchecked[hideTag.ID] = hideTag
		}
	}
	return nil, nil
}

func (f *FindFilter) getMatchHideTagsWhenUnchecked(
	findCtx *FindKyouContext,
	checkedTagNames []string,
	output map[string]reps.Tag,
) {
	for _, hideTag := range findCtx.AllHideTagsWhenUnchecked {
		if !containsString(checkedTagNames, hideTag.Tag) {
			output[hideTag.ID] = hideTag
		}
	}
}

func (f *FindFilter) getMatchHideTagsWhenUnckedKyou(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.Tags == nil {
		return nil, nil
	}
	f.getMatchHideTagsWhenUnchecked(findCtx, findCtx.ParsedFindQuery.Tags, findCtx.MatchHideTagsWhenUncheckedKyou)
	return nil, nil
}

func (f *FindFilter) getMatchHideTagsWhenUnckedTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	f.getMatchHideTagsWhenUnchecked(findCtx, findCtx.ParsedFindQuery.TimeIsTags, findCtx.MatchHideTagsWhenUncheckedTimeIs)
	return nil, nil
}

func (f *FindFilter) findTimeIsTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// タグを使わない場合はnil
	if findCtx.ParsedFindQuery.TimeIsTags == nil {
		return nil, nil
	}

	for _, tagName := range findCtx.ParsedFindQuery.TimeIsTags {
		matchTags, err := findCtx.Repositories.TagReps.GetTagsByTagName(ctx, tagName)
		if err != nil {
			err = fmt.Errorf("error at get tags by name %s: %w", tagName, err)
			return nil, err
		}
		for _, tag := range matchTags {
			if !findCtx.isLatestData(tag.ID, tag.UpdateTime) {
				continue
			}
			if tag.IsDeleted {
				continue
			}
			findCtx.MatchTimeIsTags[tag.ID] = tag
		}
	}
	return nil, nil
}

func (f *FindFilter) findKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error

	// text検索用クエリ
	targetIDs := []string{}
	for _, text := range findCtx.MatchTexts {
		targetIDs = append(targetIDs, text.TargetID)
	}

	matchTextFindByIDQuery := &find.FindQuery{
		IDs:            targetIDs,
		OnlyLatestData: true,
	}

	matchReps := reps.Repositories{}
	for _, rep := range findCtx.MatchReps {
		matchReps = append(matchReps, rep)
	}

	// repで検索
	kyousMap, err := matchReps.FindKyous(ctx, findCtx.ParsedFindQuery)
	if err != nil {
		return nil, err
	}
	// textでマッチしたものをID検索
	textMatchKyousMap := map[string][]reps.Kyou{}
	if len(targetIDs) != 0 {
		textMatchKyousMap, err = matchReps.FindKyous(ctx, matchTextFindByIDQuery)
		if err != nil {
			return nil, err
		}
	}
	for id, textMatchKyous := range textMatchKyousMap {
		if _, exist := kyousMap[id]; !exist {
			kyousMap[id] = []reps.Kyou{}
		}
		kyousMap[id] = append(kyousMap[id], textMatchKyous...)
	}

	// 削除隅のものは消す
	deleteTargetIDs := []string{}
	for id, kyous := range kyousMap {
		var latestKyou reps.Kyou
		for _, kyou := range kyous {
			if kyou.UpdateTime.After(latestKyou.UpdateTime) {
				latestKyou = kyou
			}
		}
		if latestKyou.IsDeleted {
			deleteTargetIDs = append(deleteTargetIDs, id)
		}
	}
	for _, deleteTargetID := range deleteTargetIDs {
		delete(kyousMap, deleteTargetID)
	}
	findCtx.MatchKyousCurrent = kyousMap
	return nil, nil
}

func (f *FindFilter) sortAndTrimKyousMap(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	query := findCtx.ParsedFindQuery
	// 56万件規模ではマップの伸長ごとの再ハッシュが効くので、入力と同じ容量で確保しておく。
	resultKyous := make(map[string][]reps.Kyou, len(findCtx.MatchKyousCurrent))

	// 時間帯フィルタが有効かどうかの判定はゲートヘルパに任せる。
	// ここから下はその内側の高速化で、検索中に変わらない値をKyouごとではなく
	// 最初に一度だけ計算しておく。時刻への変換と曜日スライスの走査は、
	// 検索結果が多いほど無視できない負荷になる。
	filterPeriodOfTime := query.HasPeriodOfTimeFilter()
	var filterWeekdays bool
	var allowedWeekdays [7]bool
	var hasPeriodStart, hasPeriodEnd bool
	var periodStartSecond, periodEndSecond int64
	if filterPeriodOfTime {
		// 曜日フィルタ（nil=曜日制限なし / 非nil空=0件 / 全7曜日=制限なし）
		// nil を len==0 や len!=7 の分岐へ落とすと全件が消えるので、必ず nil を先に外すこと
		filterWeekdays = query.PeriodOfTimeWeekOfDays != nil && len(query.PeriodOfTimeWeekOfDays) != 7
		if filterWeekdays {
			for _, weekday := range query.PeriodOfTimeWeekOfDays {
				if weekday >= find.SunDay && weekday <= find.SaturDay {
					allowedWeekdays[weekday] = true
				}
			}
		}

		hasPeriodStart = query.PeriodOfTimeStartTimeSecond != nil
		if hasPeriodStart {
			start := time.Unix(*query.PeriodOfTimeStartTimeSecond, 0).In(time.Local)
			periodStartSecond = int64(start.Hour()*3600 + start.Minute()*60 + start.Second())
		}
		hasPeriodEnd = query.PeriodOfTimeEndTimeSecond != nil
		if hasPeriodEnd {
			end := time.Unix(*query.PeriodOfTimeEndTimeSecond, 0).In(time.Local)
			periodEndSecond = int64(end.Hour()*3600 + end.Minute()*60 + end.Second())
		}
	}

	// Kyou1件ぶんの期間判定。検索中に変わらない値は上で1回だけ計算済みなので、ここは比較だけ。
	// 単一entryの高速路と従来の重複排除路の両方から同じ判定を使うために切り出してある。
	passesPeriodFilter := func(kyou reps.Kyou) bool {
		if (query.CalendarStartDate != nil && kyou.RelatedTime.Before(*query.CalendarStartDate)) ||
			(query.CalendarEndDate != nil && kyou.RelatedTime.After(*query.CalendarEndDate)) {
			return false
		}
		if !filterPeriodOfTime {
			return true
		}
		localTime := kyou.RelatedTime.In(time.Local)

		// 曜日フィルタ
		if filterWeekdays && !allowedWeekdays[localTime.Weekday()] {
			return false
		}

		// 時間帯フィルタ
		timeSec := int64(localTime.Hour()*3600 + localTime.Minute()*60 + localTime.Second())
		if hasPeriodStart && hasPeriodEnd {
			if periodStartSecond > periodEndSecond {
				// 夜跨ぎ: timeSec >= start OR timeSec <= end
				if timeSec < periodStartSecond && timeSec > periodEndSecond {
					return false
				}
			} else {
				// 通常: timeSec >= start AND timeSec <= end
				if timeSec < periodStartSecond || timeSec > periodEndSecond {
					return false
				}
			}
		} else if hasPeriodStart {
			if timeSec < periodStartSecond {
				return false
			}
		} else if hasPeriodEnd {
			if timeSec > periodEndSecond {
				return false
			}
		}
		return true
	}

	for id, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			continue
		}

		// 単一entryの高速路。
		//
		// usecase.GetKyous が OnlyLatestData=true を固定し、--cache_in_memory(既定true)では
		// 型ごとにrepが1つへ畳まれるので、実データではIDの大半がここを通る。
		// 1件しかないバケツに重複排除も並び替えも要らないのに、従来はIDごとに一時マップを作り、
		// slices.Collectでスライスへ集め直してからno-opのソートをかけていた。
		// 集め直したスライスはID 1件につき1確保で、実データ(56万件)では無視できない。
		//
		// 入力スライスをそのまま持ち回るが、後段でこれを書き換えるのは overrideKyous だけで、
		// そこは ForMi のときに他が保持していないスライスに対して行われる
		// (Repositories.FindKyous も各リポジトリ実装も、呼び出しごとに新しいスライスを作る)。
		if len(kyous) == 1 {
			if !passesPeriodFilter(kyous[0]) {
				continue
			}
			resultKyous[id] = kyous
			continue
		}

		// 重複排除キーは「版(UpdateTime) × 射影(DataType) × 表示時刻(RelatedTime)」。
		// 同一版の同一射影がrep間で重複したものだけを1件に潰す。
		// 以前はRelatedTime.Unix()だけをキーにしていたため、同一IDの新旧版が同秒に衝突して
		// スライス順(チャネル回収順=非決定的)で片方だけが残り、旧版が残った検索では
		// replaceLatestKyouInfosがレコードごと除外して「出たり消えたり」していた。
		trimedKyousMap := map[kyouEntryKey]reps.Kyou{}
		for _, kyou := range kyous {
			if !passesPeriodFilter(kyou) {
				continue
			}
			trimedKyousMap[kyouEntryKey{
				updateTimeUnix:  kyou.UpdateTime.Unix(),
				dataType:        kyou.DataType,
				relatedTimeUnix: kyou.RelatedTime.Unix(),
			}] = kyou
		}

		sortedKyous := slices.Collect(maps.Values(trimedKyousMap))
		if len(sortedKyous) == 0 {
			continue
		}
		// RelatedTime降順。同時刻はUpdateTime降順→DataTypeで決定化し、
		// 後段のkyous[0]参照が常に最新版を見るようにする
		slices.SortFunc(sortedKyous, func(a, b reps.Kyou) int {
			if c := b.RelatedTime.Compare(a.RelatedTime); c != 0 {
				return c
			}
			if c := b.UpdateTime.Compare(a.UpdateTime); c != 0 {
				return c
			}
			return strings.Compare(a.DataType, b.DataType)
		})

		resultKyous[id] = sortedKyous
	}

	if query.PlaingTime != nil || query.ForMi {
		for id, kyousInID := range resultKyous {
			// 最新版の代表射影1件に決定的に絞る(以前は不安定ソートで
			// 開始/終了射影のどちらが残るかが実行毎に変わり、plaingの表示時刻が揺れていた)。
			// 既に1件なら選び直す余地が無いので、スライスを作り直さない。
			if len(kyousInID) == 1 {
				continue
			}
			resultKyous[id] = []reps.Kyou{newestKyouEntry(kyousInID)}
		}
	}

	findCtx.MatchKyousCurrent = resultKyous
	return nil, nil
}

func (f *FindFilter) filterMiForMi(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !(findCtx.ParsedFindQuery.ForMi) {
		return nil, nil
	}

	// Miを取得位する
	// 作成日時以外の条件でmiを取得する。その後、作成日時で取得して追加する。
	allMis := map[string]reps.Mi{}
	withoutCreatedMiFindQuery := *findCtx.ParsedFindQuery
	withoutCreatedMiFindQuery.IncludeCreateMi = false
	withoutCreatedMis, err := findCtx.Repositories.MiReps.FindMi(ctx, &withoutCreatedMiFindQuery)
	if err != nil {
		err = fmt.Errorf("error at get without created mis: %w", err)
		return nil, err
	}
	for _, mi := range withoutCreatedMis {
		if existMi, exist := allMis[mi.ID]; exist {
			if mi.UpdateTime.After(existMi.UpdateTime) {
				allMis[mi.ID] = mi
			}
		} else {
			allMis[mi.ID] = mi
		}
	}

	// IncludeCreateMi が既に false なら withoutCreatedMiFindQuery と ParsedFindQuery は
	// 同じ内容の構造体なので、この2回目は1回目とまったく同じ行を返す。
	// 受け側のループは `if !exist` しか見ないので何も足さない ―― 丸ごと無駄な集約検索。
	// mi板の追加検索が4回から2回になる。
	if findCtx.ParsedFindQuery.IncludeCreateMi {
		withCreatedMis, err := findCtx.Repositories.MiReps.FindMi(ctx, findCtx.ParsedFindQuery)
		if err != nil {
			err = fmt.Errorf("error at get all mis: %w", err)
			return nil, err
		}
		for _, mi := range withCreatedMis {
			if _, exist := allMis[mi.ID]; !exist {
				allMis[mi.ID] = mi
			}
		}
	}

	// MiReKyouもMiと同じ扱いで板に並べる。Miへ変換してから同じパイプラインに乗せる
	mireKyouIDs := map[string]struct{}{}
	withoutCreatedMiReKyous, err := findCtx.Repositories.MiReKyouReps.FindMiReKyou(ctx, &withoutCreatedMiFindQuery)
	if err != nil {
		err = fmt.Errorf("error at get without created mirekyous: %w", err)
		return nil, err
	}
	for _, mirekyou := range withoutCreatedMiReKyous {
		mi := mirekyou.ToMi()
		mireKyouIDs[mi.ID] = struct{}{}
		if existMi, exist := allMis[mi.ID]; exist {
			if mi.UpdateTime.After(existMi.UpdateTime) {
				allMis[mi.ID] = mi
			}
		} else {
			allMis[mi.ID] = mi
		}
	}

	// IncludeCreateMi が既に false なら withoutCreatedMiFindQuery と ParsedFindQuery は
	// 同じ内容の構造体なので、この2回目は1回目とまったく同じ行を返す。
	// 受け側のループは `if !exist` しか見ないので何も足さない ―― 丸ごと無駄な集約検索。
	// mi板の追加検索が4回から2回になる。
	if findCtx.ParsedFindQuery.IncludeCreateMi {
		withCreatedMiReKyous, err := findCtx.Repositories.MiReKyouReps.FindMiReKyou(ctx, findCtx.ParsedFindQuery)
		if err != nil {
			err = fmt.Errorf("error at get all mirekyous: %w", err)
			return nil, err
		}
		for _, mirekyou := range withCreatedMiReKyous {
			mi := mirekyou.ToMi()
			mireKyouIDs[mi.ID] = struct{}{}
			if _, exist := allMis[mi.ID]; !exist {
				allMis[mi.ID] = mi
			}
		}
	}

	// チェック状態から対象Miを抽出する
	targetMis := []reps.Mi{}
	for _, mi := range allMis {
		switch string(findCtx.ParsedFindQuery.MiCheckState) {
		case string(find.Checked):
			if mi.IsChecked {
				targetMis = append(targetMis, mi)
			}
		case string(find.UncCheck):
			if !mi.IsChecked {
				targetMis = append(targetMis, mi)
			}
		default:
			// find.All、および未指定(空文字)・未知の値は全件対象にする。
			// 以前はdefault節が無く、mi_check_stateを送らないクライアント(MCP等)で
			// mi検索が無条件0件になっていた
			targetMis = append(targetMis, mi)
		}
	}

	// 対象MiのKyouのみを抽出する
	// 主経路(get_kyous)はOnlyLatestData=true固定でMatchKyousCurrentのキーはkyou.IDそのものだが、
	// OnlyLatestData=falseの場合はID+UpdateTime.Unix()形式になりmi.IDで直接引けない。
	// どちらでも動くよう、値の kyous[0].ID と照合してキーをmi.IDに正規化して格納する。
	//
	// Mi 1件ごとに MatchKyousCurrent を舐めると O(Mi数 × Kyou数) になる。
	// mi板は Mi数 ≒ Kyou数 なので実質 O(K^2)。
	// kyous[0].ID を引ける索引を1回だけ作って引く。
	kyousByKyouID := make(map[string][]reps.Kyou, len(findCtx.MatchKyousCurrent))
	for _, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			continue
		}
		if _, exist := kyousByKyouID[kyous[0].ID]; !exist {
			kyousByKyouID[kyous[0].ID] = kyous
		}
	}
	filteredKyous := map[string][]reps.Kyou{}
	for _, mi := range targetMis {
		if kyous, exist := kyousByKyouID[mi.ID]; exist {
			filteredKyous[mi.ID] = kyous
		}
	}

	findCtx.MatchMisAtFilterMi = allMis
	findCtx.MiReKyouIDs = mireKyouIDs
	findCtx.MatchKyousCurrent = filteredKyous
	return nil, nil
}

func (f *FindFilter) filterTagsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// タグで絞る指定なのにタグが1つもチェックされていない場合は0件にする。
	// 以前はORが「何かタグの付いた全Kyou」・ANDが0件と挙動が割れていた
	if len(findCtx.ParsedFindQuery.Tags) == 0 {
		findCtx.MatchKyousCurrent = map[string][]reps.Kyou{}
		return nil, nil
	}

	if !(findCtx.ParsedFindQuery.TagsAnd) {
		// ORの場合のフィルタリング処理。
		//
		// 以前は「タグ一致」「タグ無し」「合成」の3つのマップを結果件数ぶん作っていた。
		// 求める集合は同じなので、MatchKyousCurrentを1周して残さないキーをdeleteする形に寄せてある
		// (Goはrange中の自キーのdeleteを許す)。56万件ではマップ3つぶんの確保がまるごと消える。
		// MatchKyousCurrentはsortAndTrimKyousMapが作ったばかりで他に持ち主がいないので、
		// その場で削って問題ない。

		// クエリのタグが付いているKyouのID集合
		matchOrTagTargetIDs := make(map[string]struct{}, len(findCtx.MatchTags))
		for _, tag := range findCtx.MatchTags {
			matchOrTagTargetIDs[tag.TargetID] = struct{}{}
		}

		// タグ無し込であればそれもいれる
		existNoTags := false
		for _, tag := range findCtx.ParsedFindQuery.Tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		for id := range findCtx.MatchKyousCurrent {
			if _, hasQueryTag := matchOrTagTargetIDs[id]; hasQueryTag {
				continue
			}
			if existNoTags {
				if _, relatedTagKyou := findCtx.RelatedTagIDs[id]; !relatedTagKyou {
					continue
				}
			}
			delete(findCtx.MatchKyousCurrent, id)
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedKyou {
			delete(findCtx.MatchKyousCurrent, hideTag.TargetID)
		}
	} else {
		// ANDの場合のフィルタリング処理
		// クエリのタグ名を基準に交差する。
		//  - クエリ中のタグ名が1件もヒットしなければ結果は空(ANDの意味論。
		//    以前は実在タグだけを回っていたため、存在しないタグ名が黙って無視されANDが緩んでいた)
		//  - タグ名の照合は大文字小文字を無視(OR分岐が頼るSQLの LOWER()= と同じ意味論。
		//    以前はGoの==で大小を区別しており、OR/ANDで結果が非対称だった)
		//  - NoTags("no tags")は「タグが1つも付いていない」という仮想タグとして交差に参加する
		queryTagNames := uniqueStrings(findCtx.ParsedFindQuery.Tags)

		// クエリタグ名ごとに、そのタグを持つKyouをkyou.IDで引けるmapを作る。
		// 突き合わせはmap参照で行うこと。
		// 以前は交差の内側もループして文字列比較しており、
		// タグAND5個 × Kyou 5,000件で約1.25億回の比較になっていた。
		kyousByQueryTagName := make(map[string]map[string][]reps.Kyou, len(queryTagNames))
		for _, tag := range findCtx.MatchTags {
			kyous, exist := findCtx.MatchKyousCurrent[tag.TargetID]
			if !exist {
				continue
			}
			for _, queryTagName := range queryTagNames {
				if !strings.EqualFold(queryTagName, tag.Tag) {
					continue
				}
				if _, exist := kyousByQueryTagName[queryTagName]; !exist {
					kyousByQueryTagName[queryTagName] = map[string][]reps.Kyou{}
				}
				kyousByQueryTagName[queryTagName][tag.TargetID] = kyous
			}
		}

		// タグ無し(NoTags)の対象Kyouを構築する
		for _, queryTagName := range queryTagNames {
			if queryTagName != NoTags {
				continue
			}
			noTagKyous := make(map[string][]reps.Kyou, len(findCtx.MatchKyousCurrent))
			for id, kyous := range findCtx.MatchKyousCurrent {
				if _, relatedTagKyou := findCtx.RelatedTagIDs[id]; !relatedTagKyou {
					noTagKyous[id] = kyous
				}
			}
			kyousByQueryTagName[queryTagName] = noTagKyous
		}

		// クエリの全タグ名に存在するKyouだけを抽出
		filteredByTags := map[string][]reps.Kyou{}
		for index, queryTagName := range queryTagNames {
			currentMatchKyous := kyousByQueryTagName[queryTagName] // ヒット無しならnil=空集合
			if index == 0 {
				for kyouID, kyous := range currentMatchKyous {
					filteredByTags[kyouID] = kyous
				}
				continue
			}
			matchThisLoopKyousMap := make(map[string][]reps.Kyou, len(filteredByTags))
			for kyouID, kyous := range filteredByTags {
				if _, exist := currentMatchKyous[kyouID]; exist {
					matchThisLoopKyousMap[kyouID] = kyous
				}
			}
			filteredByTags = matchThisLoopKyousMap
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedKyou {
			delete(filteredByTags, hideTag.TargetID)
		}

		findCtx.MatchKyousCurrent = filteredByTags
	}

	return nil, nil
}

func (f *FindFilter) filterTagsTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// TimeIsタグ絞り込みを使わない場合(またはタグ列が未指定=nil)は、
	// 削除済みを除いた全TimeIsを通し、非表示タグだけを適用する。
	// 以前はTimeIsTags==nilだとOR/ANDどちらの分岐にも入らず、
	// MatchTimeIssAtFilterTagsが空のまま=検索全体が0件になっていた
	if findCtx.ParsedFindQuery.TimeIsTags == nil {
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			if timeis.IsDeleted {
				continue
			}
			findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
		}
		// 非表示タグの対象を消す(以前はこの分岐にdeleteが無く、
		// 計算済みの非表示タグ集合が適用されなかった)
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedTimeIs {
			delete(findCtx.MatchTimeIssAtFilterTags, hideTag.TargetID)
		}
		return nil, nil
	}

	// タグで絞る指定なのにタグが1つもチェックされていない場合は0件にする(Kyou側と同じ)
	if len(findCtx.ParsedFindQuery.TimeIsTags) == 0 {
		return nil, nil
	}

	if !(findCtx.ParsedFindQuery.TimeIsTagsAnd) {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagTimeIss := map[string]reps.TimeIs{}
		for _, tag := range findCtx.MatchTimeIsTags {
			matchTimeis, exist := findCtx.MatchTimeIssAtFindTimeIs[tag.TargetID]
			if !exist {
				continue
			}
			matchOrTagTimeIss[matchTimeis.ID] = matchTimeis
		}

		// タグ無しKyouリスト
		noTagTimeIss := map[string]reps.TimeIs{}
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			_, relatedTagTimeIs := findCtx.RelatedTagIDs[timeis.ID]
			if !relatedTagTimeIs {
				noTagTimeIss[timeis.ID] = timeis
			}
		}

		// タグ無し込であればそれもいれる
		existNoTags := false
		for _, tag := range findCtx.ParsedFindQuery.TimeIsTags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		// タグフィルタしたものをCtxに収める
		for _, timeis := range matchOrTagTimeIss {
			findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
		}
		if existNoTags {
			for _, timeis := range noTagTimeIss {
				findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
			}
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedTimeIs {
			delete(findCtx.MatchTimeIssAtFilterTags, hideTag.TargetID)
		}

	} else {
		// ANDの場合のフィルタリング処理
		// Kyou側(filterTagsKyous)と同じく、クエリのタグ名を基準に交差する。
		//  - クエリ中のタグ名が1件もヒットしなければ結果は空(ANDの意味論)
		//  - タグ名の照合は大文字小文字を無視(SQLの LOWER()= と同じ意味論)
		//  - NoTags("no tags")は仮想タグとして交差に参加する。
		//    以前はNoTags用の内側mapを初期化しないままnil mapへ代入しており、
		//    タグ無しTimeIsが1件でもあるとpanicしていた
		queryTagNames := uniqueStrings(findCtx.ParsedFindQuery.TimeIsTags)

		timeIssByQueryTagName := make(map[string]map[string]reps.TimeIs, len(queryTagNames))
		for _, tag := range findCtx.MatchTimeIsTags {
			timeis, exist := findCtx.MatchTimeIssAtFindTimeIs[tag.TargetID]
			if !exist {
				continue
			}
			for _, queryTagName := range queryTagNames {
				if !strings.EqualFold(queryTagName, tag.Tag) {
					continue
				}
				if _, exist := timeIssByQueryTagName[queryTagName]; !exist {
					timeIssByQueryTagName[queryTagName] = map[string]reps.TimeIs{}
				}
				timeIssByQueryTagName[queryTagName][timeis.ID] = timeis
			}
		}

		// タグ無し(NoTags)の対象TimeIsを構築する
		for _, queryTagName := range queryTagNames {
			if queryTagName != NoTags {
				continue
			}
			noTagTimeIss := map[string]reps.TimeIs{}
			for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
				if _, relatedTagTimeIs := findCtx.RelatedTagIDs[timeis.ID]; !relatedTagTimeIs {
					noTagTimeIss[timeis.ID] = timeis
				}
			}
			timeIssByQueryTagName[queryTagName] = noTagTimeIss
		}

		// クエリの全タグ名に存在するTimeIsだけを抽出
		hasAllMatchTagsTimeIssMap := map[string]reps.TimeIs{}
		for index, queryTagName := range queryTagNames {
			currentMatchTimeIss := timeIssByQueryTagName[queryTagName] // ヒット無しならnil=空集合
			if index == 0 {
				for id, timeis := range currentMatchTimeIss {
					hasAllMatchTagsTimeIssMap[id] = timeis
				}
				continue
			}
			matchThisLoopTimeIssMap := make(map[string]reps.TimeIs, len(hasAllMatchTagsTimeIssMap))
			for id, timeis := range hasAllMatchTagsTimeIssMap {
				if _, exist := currentMatchTimeIss[id]; exist {
					matchThisLoopTimeIssMap[id] = timeis
				}
			}
			hasAllMatchTagsTimeIssMap = matchThisLoopTimeIssMap
		}

		findCtx.MatchTimeIssAtFilterTags = hasAllMatchTagsTimeIssMap

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedTimeIs {
			delete(findCtx.MatchTimeIssAtFilterTags, hideTag.TargetID)
		}
	}
	return nil, nil
}

func (f *FindFilter) filterPlaingTimeIsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.HasTimeIsFilter() {
		return nil, nil
	}

	intervals := make([]inclusiveTimeInterval, 0, len(findCtx.MatchTimeIssAtFilterTags))
	for _, timeis := range findCtx.MatchTimeIssAtFilterTags {
		intervals = append(intervals, inclusiveTimeInterval{start: timeis.StartTime, end: timeis.EndTime})
	}
	intervalIndex := newInclusiveTimeIntervalIndex(intervals)

	filteredByTimeIs := make(map[string][]reps.Kyou, len(findCtx.MatchKyousCurrent))
	for id, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			continue
		}
		if intervalIndex.contains(kyous[0].RelatedTime) {
			filteredByTimeIs[id] = kyous
		}
	}
	findCtx.MatchKyousCurrent = filteredByTimeIs
	return nil, nil
}

func (f *FindFilter) findTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.HasTimeIsFilter() {
		return nil, nil
	}

	// 対象TimeIs取得用検索クエリ
	// （外側ゲートによりTimeIsWords/TimeIsNotWordsの少なくとも一方は非nilなので、
	// 転記先のワードフィルタも必ず有効になる）
	timeisFindKyouQuery := &find.FindQuery{
		Words:             findCtx.ParsedFindQuery.TimeIsWords,
		NotWords:          findCtx.ParsedFindQuery.TimeIsNotWords,
		WordsAnd:          findCtx.ParsedFindQuery.TimeIsWordsAnd,
		CalendarStartDate: findCtx.ParsedFindQuery.CalendarStartDate,
		CalendarEndDate:   findCtx.ParsedFindQuery.CalendarEndDate,
		IncludeEndTimeIs:  true,
		// 編集前のタイトルでヒットしたり、旧版の期間で絞り込んだりしないよう、
		// IDごとの最新版のみを対象にする(findTags/findTextsGenericと同じ考慮)
		OnlyLatestData: true,
	}

	// text検索用クエリ
	targetIDs := []string{}
	for _, text := range findCtx.MatchTimeIsTexts {
		targetIDs = append(targetIDs, text.TargetID)
	}
	matchTextFindByIDQuery := &find.FindQuery{
		IDs:            targetIDs,
		OnlyLatestData: true,
	}

	allTimeIss, err := collectFromRepos([]reps.TimeIsRepository(findCtx.Repositories.TimeIsReps), func(rep reps.TimeIsRepository) ([]reps.TimeIs, error) {
		timeiss, err := rep.FindTimeIs(ctx, timeisFindKyouQuery)
		if err != nil {
			return nil, err
		}
		if len(targetIDs) != 0 {
			textMatchTimeiss, err := rep.FindTimeIs(ctx, matchTextFindByIDQuery)
			if err != nil {
				return nil, err
			}
			timeiss = append(timeiss, textMatchTimeiss...)
		}
		return timeiss, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error at find timeiss: %w", err)
	}

	// TimeIs集約
	for _, timeis := range allTimeIss {
		if !findCtx.isLatestData(timeis.ID, timeis.UpdateTime) {
			continue
		}
		upsertIfNewer(findCtx.MatchTimeIssAtFindTimeIs, timeis.ID, timeis, func(t reps.TimeIs) time.Time { return t.UpdateTime })
	}

	deletedIDs := []string{}
	for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
		if timeis.IsDeleted {
			deletedIDs = append(deletedIDs, timeis.ID)
		}
	}
	for _, deletedID := range deletedIDs {
		delete(findCtx.MatchTimeIssAtFindTimeIs, deletedID)
	}

	return nil, nil
}

func (f *FindFilter) filterLocationKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.HasMapFilter() {
		return nil, nil
	}

	matchKyous := map[string][]reps.Kyou{}

	// 開始日を取得
	startTime := findCtx.ParsedFindQuery.CalendarStartDate
	endTime := findCtx.ParsedFindQuery.CalendarEndDate

	// radius, latitude, longitudeを取得
	// 半径が未指定(0以下)の場合は絞り込まずに素通しする。
	// 以前はradius=0のまま距離判定に使われ、全Kyouが黙って消えていた
	if *findCtx.ParsedFindQuery.MapRadius <= 0 {
		return nil, nil
	}
	radius := *findCtx.ParsedFindQuery.MapRadius / 1000
	latitude := *findCtx.ParsedFindQuery.MapLatitude
	longitude := *findCtx.ParsedFindQuery.MapLongitude

	// 日付のnil解決 もしくは全部の日付
	isAllDays := false
	if startTime != nil && endTime == nil {
		s := time.Time(*startTime)
		e := time.Time(*startTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else if startTime != nil && endTime != nil {
		s := time.Time(*startTime)
		e := time.Time(*endTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else {
		isAllDays = true
	}
	// GPSLogを取得する
	matchGPSLogs, err := collectFromRepos([]reps.GPSLogRepository(findCtx.Repositories.GPSLogReps), func(rep reps.GPSLogRepository) ([]reps.GPSLog, error) {
		if isAllDays {
			return rep.GetAllGPSLogs(ctx)
		}
		return rep.GetGPSLogs(ctx, startTime, endTime)
	})
	if err != nil {
		return nil, fmt.Errorf("error at filter gpslogs: %w", err)
	}

	// 並び替え
	slices.SortFunc(matchGPSLogs, func(a, b reps.GPSLog) int { return a.RelatedTime.Compare(b.RelatedTime) })

	// 該当する時間を出す
	matchGPSLogSetList := [][]reps.GPSLog{}

	// 圏内の点の前後両方の区間を滞在区間とする。
	// 以前は「圏内点→次点」しか追加されず、圏外→圏内の入りの区間が落ち、
	// 最後の点だけが圏内の場合は区間が1つも作られず全Kyouが消えていた
	preTrue := false // 一つ前の点が圏内だった
	for i := range matchGPSLogs {
		inRadius := calcDistanceKm(latitude, longitude, matchGPSLogs[i].Latitude, matchGPSLogs[i].Longitude) <= radius
		if i > 0 && (preTrue || inRadius) {
			matchGPSLogSetList = append(matchGPSLogSetList, []reps.GPSLog{
				matchGPSLogs[i-1],
				matchGPSLogs[i],
			})
		}
		preTrue = inRadius
	}

	intervals := make([]inclusiveTimeInterval, 0, len(matchGPSLogSetList))
	for _, gpsLogSet := range matchGPSLogSetList {
		end := gpsLogSet[1].RelatedTime
		intervals = append(intervals, inclusiveTimeInterval{start: gpsLogSet[0].RelatedTime, end: &end})
	}
	intervalIndex := newInclusiveTimeIntervalIndex(intervals)

	// KyouがLocation内か判定。他の時刻フィルタと同じく両端を含む。
	for id, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			continue
		}
		if intervalIndex.contains(kyous[0].RelatedTime) {
			matchKyous[id] = kyous
		}
	}

	findCtx.MatchKyousCurrent = matchKyous
	return nil, nil
}
func (f *FindFilter) overrideKyous(_ context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.ForMi {
		// kyou検索の場合は何もしない
		return nil, nil
	}
	// miの場合は
	// 表示したとき、指定日時か作成日時かわかるようにDataTypeを上書きする
	for _, mi := range findCtx.MatchMisAtFilterMi {
		kyous, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			// MiReKyou由来のものはクライアントがtyped_mirekyouを引けるよう接頭辞を変える
			dataTypePrefix := "mi"
			if _, isMiReKyou := findCtx.MiReKyouIDs[mi.ID]; isMiReKyou {
				dataTypePrefix = "mirekyou"
			}

			if string(findCtx.ParsedFindQuery.MiSortType) == string(find.CreateTime) {
				kyous[0].DataType = dataTypePrefix + "_create"
				kyous[0].RelatedTime = mi.CreateTime
			} else if string(findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateStartTime) && mi.EstimateStartTime != nil {
				kyous[0].DataType = dataTypePrefix + "_start"
				kyous[0].RelatedTime = *mi.EstimateStartTime
			} else if string(findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateEndTime) && mi.EstimateEndTime != nil {
				kyous[0].DataType = dataTypePrefix + "_end"
				kyous[0].RelatedTime = *mi.EstimateEndTime
			} else if string(findCtx.ParsedFindQuery.MiSortType) == string(find.LimitTime) && mi.LimitTime != nil {
				kyous[0].DataType = dataTypePrefix + "_limit"
				kyous[0].RelatedTime = *mi.LimitTime
			} else {
				// ソート基準の時刻(見積開始等)が未設定のMiは作成日時にフォールバックする。
				// 以前は_createを名乗りながらUpdateTimeを入れており、表示時刻が作成日時とずれていた
				kyous[0].DataType = dataTypePrefix + "_create"
				kyous[0].RelatedTime = mi.CreateTime
			}

		}
	}
	return nil, nil
}

func (f *FindFilter) sortResultKyous(_ context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.ForMi {
		// kyouとしてソート。並び順は RelatedTime(秒)降順、同着はID昇順。
		//
		// **本体スライスを直接 SortFunc しない。** reps.Kyou は232バイトあり、
		// pdqsort はその実体を動かすので、56万件では比較のたびの Unix() 計算
		// (n log n 回で2600万回級)と合わせて memmove がGB級になる。
		// 32バイトのキーだけ並べ替えてから、本体は巡回置換で1要素あたり1回だけ動かす。
		// 追加で確保するのはキー配列1本(1件32バイト)だけで、本体のコピーは作らない。
		sortResultKyousByKey(findCtx.ResultKyous)
		return nil, nil
	}

	// miとしてソート。指定日時でソートする。指定日時がないものは、末尾に作成日時でくっつける
	sortType := findCtx.ParsedFindQuery.MiSortType
	slices.SortFunc(findCtx.ResultKyous, func(a, b reps.Kyou) int {
		aMi := findCtx.MatchMisAtFilterMi[a.ID]
		bMi := findCtx.MatchMisAtFilterMi[b.ID]

		compareTimes := func(aT, bT *time.Time) int {
			if aT != nil && bT != nil {
				return aT.Compare(*bT)
			}
			if aT == nil && bT != nil {
				return 1
			}
			if aT != nil && bT == nil {
				return -1
			}
			return aMi.CreateTime.Compare(bMi.CreateTime)
		}

		result := 0
		switch string(sortType) {
		case string(find.CreateTime):
			result = compareTimes(&aMi.CreateTime, &bMi.CreateTime)
		case string(find.EstimateStartTime):
			result = compareTimes(aMi.EstimateStartTime, bMi.EstimateStartTime)
		case string(find.EstimateEndTime):
			result = compareTimes(aMi.EstimateEndTime, bMi.EstimateEndTime)
		case string(find.LimitTime):
			result = compareTimes(aMi.LimitTime, bMi.LimitTime)
		}
		if result != 0 {
			return result
		}
		// 同着はIDで決定化する。slices.SortFuncは安定ソートではないため、
		// タイブレークが無いと実行毎に並び順と直後のID重複除去の生き残りが変わる
		if a.ID < b.ID {
			return -1
		} else if a.ID > b.ID {
			return 1
		}
		return 0
	})

	// IDが重複するKyouを除去する（ソート済みのため先頭がソート条件に最もマッチする）
	seen := map[string]struct{}{}
	deduped := make([]reps.Kyou, 0, len(findCtx.ResultKyous))
	for _, kyou := range findCtx.ResultKyous {
		if _, exist := seen[kyou.ID]; !exist {
			seen[kyou.ID] = struct{}{}
			deduped = append(deduped, kyou)
		}
	}
	findCtx.ResultKyous = deduped

	return nil, nil
}

// kyouSortKey は sortResultKyous の並べ替え用キー。
// index は「並べ替え後のその位置に来るべき、元の位置」を持ちます。
type kyouSortKey struct {
	relatedTimeUnix int64
	id              string
	index           int32
}

// sortResultKyousByKey は kyous を RelatedTime(秒)降順・同着ID昇順に並べ替えます。
// 比較と並べ替えはキー配列で行い、本体は巡回置換でその場で入れ替えます。
// 結果は本体を直接 slices.SortFunc したときと同一です。
func sortResultKyousByKey(kyous []reps.Kyou) {
	if len(kyous) < 2 {
		return
	}
	keys := make([]kyouSortKey, len(kyous))
	for i := range kyous {
		keys[i] = kyouSortKey{
			relatedTimeUnix: kyous[i].RelatedTime.Unix(),
			id:              kyous[i].ID,
			index:           int32(i),
		}
	}
	slices.SortFunc(keys, func(a, b kyouSortKey) int {
		if a.relatedTimeUnix != b.relatedTimeUnix {
			if a.relatedTimeUnix > b.relatedTimeUnix {
				return -1
			}
			return 1
		}
		return strings.Compare(a.id, b.id)
	})

	// 巡回置換で本体を並べ替える。keys[i].index は「位置 i に来るべき元の位置」。
	//
	// **単純な swap ループにしてはいけない** ―― `swap(kyous[i], kyous[keys[i].index])` を
	// 回す書き方は逆順列を適用してしまい、並びが静かに変わる
	// (sort_result_kyous_test.go の参照実装との突き合わせで落ちる)。
	// 巡回をたどって「次に来るべき要素」を引き寄せる形にすると、
	// 各要素はちょうど1回だけ動き、一時領域も1件ぶんで済む。
	for i := range keys {
		if keys[i].index == int32(i) {
			continue
		}
		current := int32(i)
		carried := kyous[i]
		for {
			next := keys[current].index
			keys[current].index = current
			if next == int32(i) {
				kyous[current] = carried
				break
			}
			kyous[current] = kyous[next]
			current = next
		}
	}
}

func (f *FindFilter) findTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	return f.findTextsGeneric(ctx, findCtx,
		findCtx.ParsedFindQuery.Words, findCtx.ParsedFindQuery.NotWords, findCtx.ParsedFindQuery.WordsAnd,
		findCtx.MatchTexts)
}

func (f *FindFilter) findTextsGeneric(
	ctx context.Context, findCtx *FindKyouContext,
	words, notWords []string, wordsAnd bool,
	targetMap map[string]reps.Text,
) ([]*message.GkillError, error) {
	// words, notWordsをパースする
	w := []string{}
	nw := []string{}
	if words != nil {
		w = words
	}
	if notWords != nil {
		nw = notWords
	}

	findTextsQuery := &find.FindQuery{
		Words:    w,
		NotWords: nw,
		WordsAnd: wordsAnd,
		// 編集前の本文でヒットしないよう、IDごとの最新版のみを対象にする
		OnlyLatestData: true,
	}

	repos := make([]reps.TextRepository, 0, len(findCtx.Repositories.TextReps))
	for _, rep := range findCtx.Repositories.TextReps {
		repos = append(repos, rep)
	}

	allTexts, err := collectFromRepos(repos, func(textRep reps.TextRepository) ([]reps.Text, error) {
		return textRep.FindTexts(ctx, findTextsQuery)
	})
	if err != nil {
		return nil, fmt.Errorf("error at find texts: %w", err)
	}

	// Text集約
	for _, text := range allTexts {
		if !findCtx.isLatestData(text.ID, text.UpdateTime) {
			continue
		}
		upsertIfNewer(targetMap, text.ID, text, func(t reps.Text) time.Time { return t.UpdateTime })
	}

	return nil, nil
}

func (f *FindFilter) filterImageKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !(findCtx.ParsedFindQuery.IsImageOnly) {
		return nil, nil
	}

	filterdImageKyous := map[string][]reps.Kyou{}
	for id, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			continue
		}
		if kyous[0].IsImage || kyous[0].IsVideo {
			filterdImageKyous[id] = kyous
		}
	}
	findCtx.MatchKyousCurrent = filterdImageKyous
	return nil, nil
}

func (f *FindFilter) findTimeIsTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	return f.findTextsGeneric(ctx, findCtx,
		findCtx.ParsedFindQuery.TimeIsWords, findCtx.ParsedFindQuery.TimeIsNotWords, findCtx.ParsedFindQuery.TimeIsWordsAnd,
		findCtx.MatchTimeIsTexts)
}
func (f *FindFilter) replaceLatestKyouInfos(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	latestKyousMap := make(map[string][]reps.Kyou, len(findCtx.MatchKyousCurrent))

	// 最新版アドレスは結果1件ごとに引く。GetLatestDataRepositoryAddressを直接呼ぶと
	// 1件ごとに遅延初期化のプロセス共有mutexと読み取りロックを取り直すので、
	// 56万件では最内ループの頻度で全リクエストが互いを待つことになる。ロックは1回だけ取る。
	// このループの中からアドレス表を触る他のメソッドを呼んではいけない(再帰RLock)。
	latestDataAddressReader, releaseLatestDataAddressRead := findCtx.Repositories.BeginLatestDataRepositoryAddressRead()
	defer releaseLatestDataAddressRead()

	// キャッシュ設定(DisableLatestDataRepositoryCache)によらず同じ規則で判定する。
	// 以前は2ブランチに分かれており、Plaing判定の粒度(Equal=ナノ秒 vs Unix=秒)、
	// アドレス未登録時の扱い(素通し vs 全除外)、保持件数(1件 vs 同UpdateTime全件)が
	// 食い違っていて、キャッシュ設定で検索結果が変わっていた。
	for id, currentKyou := range findCtx.MatchKyousCurrent {
		if len(currentKyou) == 0 {
			continue
		}

		// マッチした中の最新UpdateTime
		var newestUpdateTime time.Time
		for _, kyou := range currentKyou {
			if kyou.UpdateTime.After(newestUpdateTime) {
				newestUpdateTime = kyou.UpdateTime
			}
		}

		// マッチした中の最新がグローバル最新(LatestDataRepositoryAddress)でなければ、
		// 最新版は別rep(非表示/未同期rep)にあり検索には古い版しか載っていない。
		// 検索条件に合う最新版が存在しないのでレコードごと除外する。
		// アドレス表に載らないrep種(plugin/git/gpsなど)は hasLatestData=false のまま素通しする
		// (以前はキャッシュ無効時に無条件で除外され、プラグイン由来のKyouが全滅していた)。
		// 判定粒度はアドレス表の格納精度(Unix秒)に合わせる。
		latestData, hasLatestData := latestDataAddressReader.Get(id)
		if hasLatestData {
			if findCtx.ParsedFindQuery.PlaingTime != nil {
				if newestUpdateTime.Unix() != latestData.DataUpdateTime.Unix() {
					continue
				}
			} else if newestUpdateTime.Unix() < latestData.DataUpdateTime.Unix() {
				continue
			}
		}

		// 最新版(newestUpdateTime)のentryのみ残す。TimeIsのstart/end・ForMi時のMiの各射影は
		// 同一UpdateTimeなので全部残り、他repの古い版entryだけが落ちる。
		isMiData := strings.HasPrefix(currentKyou[0].DataType, "mi") && findCtx.ParsedFindQuery.ForMi
		isTimeIsData := strings.HasPrefix(currentKyou[0].DataType, "timeis")
		if isTimeIsData || isMiData {
			// 全部が最新版(古い版entryが1件も無い)なら、選び直す必要が無いのでそのまま使う。
			// 実データではこちらが普通で、作り直すとID 1件につき1確保になる。
			allLatest := true
			for _, kyou := range currentKyou {
				if !kyou.UpdateTime.Equal(newestUpdateTime) {
					allLatest = false
					break
				}
			}
			if allLatest {
				latestKyousMap[id] = currentKyou
				continue
			}
			latestVersionKyous := make([]reps.Kyou, 0, len(currentKyou))
			for _, kyou := range currentKyou {
				if kyou.UpdateTime.Equal(newestUpdateTime) {
					latestVersionKyous = append(latestVersionKyous, kyou)
				}
			}
			latestKyousMap[id] = latestVersionKyous
		} else if len(currentKyou) == 1 {
			// 候補が1件なら newestKyouEntry の戻り値はその1件そのものなので、
			// 1要素スライスを作り直さずに入力をそのまま持ち回る。
			// 実データではIDの大半がここを通る。
			latestKyousMap[id] = currentKyou
		} else {
			// 射影を持たない型は最新版の代表1件に決定的に絞る
			latestKyousMap[id] = []reps.Kyou{newestKyouEntry(currentKyou)}
		}
	}

	// miの場合は最新以外消す
	if findCtx.ParsedFindQuery.ForMi {
		for id, kyous := range latestKyousMap {
			// 0件は触らない。1件なら選び直す余地が無いのでスライスを作り直さない。
			if len(kyous) <= 1 {
				continue
			}
			latestKyousMap[id] = []reps.Kyou{newestKyouEntry(kyous)}
		}
	}

	findCtx.MatchKyousCurrent = latestKyousMap
	return nil, nil
}
