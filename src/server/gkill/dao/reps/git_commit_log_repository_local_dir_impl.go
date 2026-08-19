package reps

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// ErrNotGitRepository は、指定パスがgitリポジトリでないことを表します。
// git_commit_logのrep設定は `$HOME/Git/*` のようなglobで書かれ、
// 展開先にgitリポジトリでないディレクトリやファイルが混ざるのは異常ではないため、
// 呼び出し側はこのエラーを「そのrepだけスキップ」として扱ってください。
var ErrNotGitRepository = git.ErrRepositoryNotExists

type gitCommitLogRepositoryLocalImpl struct {
	gitrep                 *git.Repository
	filename               string
	m                      sync.RWMutex
	lastHeadHashes         map[string]string
	lastUpdateCacheChanged bool
}

// NewGitRep は、reppathのgitリポジトリを読むGitCommitLogRepositoryを生成します。
//
// reppathがgitリポジトリでない場合はErrNotGitRepositoryを含むエラーを返します。
func NewGitRep(reppath string) (GitCommitLogRepository, error) {
	gitrep, err := git.PlainOpen(reppath)
	if err != nil {
		// ErrNotGitRepositoryを判別できるよう%wで包む
		return nil, fmt.Errorf("error at open git repository %s: %w", reppath, err)
	}

	return &gitCommitLogRepositoryLocalImpl{
		gitrep:   gitrep,
		filename: reppath,
	}, nil
}
func (g *gitCommitLogRepositoryLocalImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {

	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = g.UpdateCache(ctx)
		if err != nil {
			repName, _ := g.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	g.m.RLock()
	defer g.m.RUnlock()

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	kyous := map[string][]Kyou{}

	var logs object.CommitIter
	if len(query.IDs) == 1 {
		logs, err = g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash((query.IDs)[0])})
		if err != nil {
			// IDがこのリポジトリに無いだけでもLogはエラーを返すため、
			// 全走査で切り分ける（該当なしならIDフィルタで自然に0件になる）
			logs, err = g.gitrep.Log(&git.LogOptions{All: true})
		}
	} else {
		logs, err = g.gitrep.Log(&git.LogOptions{All: true})
	}
	if err != nil {
		// 全走査も失敗する場合はgitリポジトリ自体の障害
		return nil, fmt.Errorf("error at git log %s: %w", repName, err)
	}
	defer func() { logs.Close() }()

	timeFilter := buildCommitTimeFilter(query)

loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// 判定
			match := true

			// gitコミットに削除の概念はないため、削除済み検索(IsDeleted=true)には該当しない
			if query.IsDeleted {
				continue
			}

			// ID検索
			if query.IDs != nil {
				// SQL側(GenerateFindSQLCommon)の「IDs非nil かつ IDs空 → 0件」と意味論を揃える。
				// 以前はIDs空だとループが回らずmatch=trueのままで、全コミットが返っていた
				match = false
				for _, id := range query.IDs {
					match = id == commit.Hash.String()
					if match {
						break
					}
				}
			}

			if query.HasWordFilter() {
				words := lowerFindWords(query.Words)
				notWords := lowerFindWords(query.NotWords)
				findWordText := findWordTextOfGitCommit(commit.Message, commit.ID().String())
				match = match && matchFindWords(findWordText, words, notWords, query.WordsAnd)
			}

			if !match {
				continue
			}

			// 日付範囲・時間帯・曜日の判定
			if !timeFilter.match(commit.Committer.When) {
				continue
			}

			kyou := Kyou{}
			kyou.IsDeleted = false
			kyou.ID = commit.Hash.String()
			kyou.RepName = repName
			kyou.RelatedTime = commit.Committer.When
			kyou.DataType = "git_commit_log"
			kyou.CreateTime = commit.Committer.When
			kyou.CreateApp = "git"
			kyou.CreateDevice = ""
			kyou.CreateUser = commit.Author.Name
			kyou.UpdateTime = commit.Committer.When
			kyou.UpdateApp = "git"
			kyou.UpdateDevice = ""
			kyou.UpdateUser = commit.Author.Name

			// 空スライスの事前確保はしない。存在しないキーへのappendはnilスライスに対して働くので
			// 結果は同じで、レコード1件につき1回の無駄な確保(実データで56万回)が消える。
			// 同じ整理は dao/reps/repositories.go の集約側では既に済んでいる。
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)

			if len(query.IDs) == 1 {
				break loop
			}
		}
	}
	return kyous, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	g.m.RLock()
	defer g.m.RUnlock()

	var err error

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	var matchKyou *Kyou
	logs, err := g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash(id)})
	if err != nil {
		// IDがこのリポジトリに無いだけでもLogはエラーを返すため、
		// 全走査で切り分ける（該当なしならIDフィルタで自然にnilになる）
		logs, err = g.gitrep.Log(&git.LogOptions{All: true})
	}
	if err != nil {
		// 全走査も失敗する場合はgitリポジトリ自体の障害
		return nil, fmt.Errorf("error at git log %s: %w", repName, err)
	}
	defer func() { logs.Close() }()
loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// 判定
			if id != commit.Hash.String() {
				continue
			}

			kyou := Kyou{}
			kyou.IsDeleted = false
			kyou.ID = commit.Hash.String()
			kyou.RepName = repName
			kyou.RelatedTime = commit.Committer.When
			kyou.DataType = "git_commit_log"
			kyou.CreateTime = commit.Committer.When
			kyou.CreateApp = "git"
			kyou.CreateDevice = ""
			kyou.CreateUser = fmt.Sprintf("%s", commit.Author)
			kyou.UpdateTime = commit.Committer.When
			kyou.UpdateApp = "git"
			kyou.UpdateDevice = ""
			kyou.UpdateUser = fmt.Sprintf("%s", commit.Author)

			matchKyou = &kyou
			break loop
		}
	}
	return matchKyou, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyou, err := g.GetKyou(ctx, id, nil)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories git commit log repositories %s: %w", id, err)
		return nil, err
	}
	if kyou == nil {
		return nil, err
	}
	return []Kyou{*kyou}, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(g.filename)
}

func (g *gitCommitLogRepositoryLocalImpl) UpdateCache(ctx context.Context) error {
	currentHeadHashes := map[string]string{}

	refs, err := g.gitrep.References()
	if err != nil {
		return fmt.Errorf("error at get references: %w", err)
	}
	defer refs.Close()

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			currentHeadHashes[ref.Name().String()] = ref.Hash().String()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error at iterate references: %w", err)
	}

	// 前回と比較
	changed := false
	if len(currentHeadHashes) != len(g.lastHeadHashes) {
		changed = true
	} else {
		for name, hash := range currentHeadHashes {
			if g.lastHeadHashes[name] != hash {
				changed = true
				break
			}
		}
	}

	g.lastUpdateCacheChanged = changed
	if changed {
		g.lastHeadHashes = currentHeadHashes
	}

	return nil
}

func (g *gitCommitLogRepositoryLocalImpl) LastUpdateCacheChanged() bool {
	return g.lastUpdateCacheChanged
}

func (g *gitCommitLogRepositoryLocalImpl) GetRepName(ctx context.Context) (string, error) {
	return filepath.Base(g.filename), nil
}

func (g *gitCommitLogRepositoryLocalImpl) Close(ctx context.Context) error {
	return nil
}

func (g *gitCommitLogRepositoryLocalImpl) FindGitCommitLog(ctx context.Context, query *find.FindQuery) ([]GitCommitLog, error) {

	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache {
		err = g.UpdateCache(ctx)
		if err != nil {
			repName, _ := g.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}
	timeFilter := buildCommitTimeFilter(query)

	g.m.RLock()
	defer g.m.RUnlock()

	// Phase 1: フィルタに一致するコミットを収集（StatsContextなし）
	var matchedCommits []*object.Commit
	var logs object.CommitIter
	if len(query.IDs) == 1 {
		logs, err = g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash((query.IDs)[0])})
		if err != nil {
			// IDがこのリポジトリに無いだけでもLogはエラーを返すため、
			// 全走査で切り分ける（該当なしならIDフィルタで自然に0件になる）
			logs, err = g.gitrep.Log(&git.LogOptions{All: true})
		}
	} else {
		logs, err = g.gitrep.Log(&git.LogOptions{All: true})
	}
	if err != nil {
		// 全走査も失敗する場合はgitリポジトリ自体の障害
		repName, _ := g.GetRepName(ctx)
		return nil, fmt.Errorf("error at git log %s: %w", repName, err)
	}
	defer func() { logs.Close() }()
loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// 判定
			match := true

			// id検索である場合のSQL追記
			if query.IDs != nil {
				ids := []string{}
				if query.IDs != nil {
					ids = query.IDs
				}
				for _, id := range ids {
					match = id == commit.Hash.String()
					if match {
						break
					}
				}
				if !match {
					continue
				}
			}

			if query.HasWordFilter() {
				words := lowerFindWords(query.Words)
				notWords := lowerFindWords(query.NotWords)
				findWordText := findWordTextOfGitCommit(commit.Message, commit.ID().String())
				match = match && matchFindWords(findWordText, words, notWords, query.WordsAnd)
			}

			if !match {
				continue
			}

			// 日付範囲・時間帯・曜日の判定。FindKyous と同じ判定を通す
			if !timeFilter.match(commit.Committer.When) {
				continue
			}

			matchedCommits = append(matchedCommits, commit)

			if len(query.IDs) == 1 {
				break loop
			}
		}
	}

	// Phase 2: StatsContextを並列実行してGitCommitLogを構築
	return g.buildGitCommitLogsParallel(ctx, matchedCommits)
}

// FindGitCommitLogByIDs 指定されたIDのコミットだけをStatsContext付きで取得する（差分更新用）
func (g *gitCommitLogRepositoryLocalImpl) FindGitCommitLogByIDs(ctx context.Context, ids []string) ([]GitCommitLog, error) {
	g.m.RLock()
	defer g.m.RUnlock()

	if len(ids) == 0 {
		return nil, nil
	}

	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	// 全コミットをイテレートし、指定IDに一致するものだけ収集
	var matchedCommits []*object.Commit
	logs, err := g.gitrep.Log(&git.LogOptions{All: true})
	if err != nil {
		// 全走査が失敗する場合はgitリポジトリ自体の障害
		repName, _ := g.GetRepName(ctx)
		return nil, fmt.Errorf("error at git log %s: %w", repName, err)
	}
	defer logs.Close()

	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if idSet[commit.Hash.String()] {
			matchedCommits = append(matchedCommits, commit)
			delete(idSet, commit.Hash.String())
			if len(idSet) == 0 {
				break
			}
		}
	}

	// StatsContextを並列実行
	return g.buildGitCommitLogsParallel(ctx, matchedCommits)
}

// buildGitCommitLogsParallel コミットのStatsContextをワーカープールで並列実行してGitCommitLogスライスを構築する
func (g *gitCommitLogRepositoryLocalImpl) buildGitCommitLogsParallel(ctx context.Context, commits []*object.Commit) ([]GitCommitLog, error) {
	if len(commits) == 0 {
		return nil, nil
	}

	repName, _ := g.GetRepName(ctx)

	results := make([]GitCommitLog, len(commits))
	var firstErr error
	var errOnce sync.Once
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	if numWorkers > len(commits) {
		numWorkers = len(commits)
	}
	sem := make(chan struct{}, numWorkers)

	for i, commit := range commits {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(idx int, c *object.Commit) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			addition, deletion := 0, 0
			stats, err := c.StatsContext(ctx)
			if err != nil {
				errOnce.Do(func() {
					firstErr = fmt.Errorf("error at get stat from commit: %w", err)
				})
				return
			}
			for _, stat := range stats {
				addition += stat.Addition
				deletion += stat.Deletion
			}

			results[idx] = GitCommitLog{
				IsDeleted:     false,
				ID:            c.Hash.String(),
				RepName:       repName,
				RelatedTime:   c.Committer.When,
				DataType:      "git_commit_log",
				CreateTime:    c.Committer.When,
				CreateApp:     "git",
				CreateDevice:  "",
				CreateUser:    c.Author.Name,
				UpdateTime:    c.Committer.When,
				UpdateApp:     "git",
				UpdateDevice:  "",
				UpdateUser:    c.Author.Name,
				CommitMessage: c.Message,
				Addition:      addition,
				Deletion:      deletion,
			}
		}(i, commit)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	g.m.RLock()
	defer g.m.RUnlock()

	var err error

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	var matchGitCommitLog *GitCommitLog
	logs, err := g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash(id)})
	if err != nil {
		// Log(From: hash) が失敗した場合（Android環境など）は Log(All: true) でフォールバック
		logs, err = g.gitrep.Log(&git.LogOptions{All: true})
		if err != nil {
			// 全走査も失敗する場合はgitリポジトリ自体の障害
			return nil, fmt.Errorf("error at git log %s: %w", repName, err)
		}
	}
	defer func() { logs.Close() }()

loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// 判定
			match := false
			if id == commit.Hash.String() {
				match = true
			}
			if updateTime != nil && updateTime.Format(sqlite3impl.TimeLayout) != commit.Committer.When.Format(sqlite3impl.TimeLayout) {
				match = false
			}
			if !match {
				continue
			}

			addition, deletion := 0, 0
			stats, err := commit.StatsContext(ctx)
			if err != nil {
				err = fmt.Errorf("error at get stat from commit: %w", err)
				return nil, err
			}

			for _, stat := range stats {
				addition += stat.Addition
				deletion += stat.Deletion
			}

			gitCommitLog := &GitCommitLog{}
			gitCommitLog.IsDeleted = false
			gitCommitLog.ID = commit.Hash.String()
			gitCommitLog.RepName = repName
			gitCommitLog.RelatedTime = commit.Committer.When
			gitCommitLog.DataType = "git_commit_log"
			gitCommitLog.CreateTime = commit.Committer.When
			gitCommitLog.CreateApp = "git"
			gitCommitLog.CreateDevice = ""
			gitCommitLog.CreateUser = commit.Author.Name
			gitCommitLog.UpdateTime = commit.Committer.When
			gitCommitLog.UpdateApp = "git"
			gitCommitLog.UpdateDevice = ""
			gitCommitLog.UpdateUser = commit.Author.Name
			gitCommitLog.CommitMessage = commit.Message
			gitCommitLog.Addition = addition
			gitCommitLog.Deletion = deletion

			matchGitCommitLog = gitCommitLog
			break loop
		}
	}
	return matchGitCommitLog, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	g.m.RLock()
	defer g.m.RUnlock()

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	latestDataRepositoryAddresses := []gkill_cache.LatestDataRepositoryAddress{}

	logs, err := g.gitrep.Log(&git.LogOptions{All: true})
	if err != nil {
		// 全走査が失敗する場合はgitリポジトリ自体の障害
		return nil, fmt.Errorf("error at git log %s: %w", repName, err)
	}
	defer logs.Close()

	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			addr := gkill_cache.LatestDataRepositoryAddress{
				IsDeleted:                false,
				TargetID:                 commit.Hash.String(),
				LatestDataRepositoryName: repName,
				DataUpdateTime:           commit.Committer.When,
			}
			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
		}
	}
	return latestDataRepositoryAddresses, nil
}

func (g *gitCommitLogRepositoryLocalImpl) UnWrapTyped() ([]GitCommitLogRepository, error) {
	return []GitCommitLogRepository{g}, nil
}

func (g *gitCommitLogRepositoryLocalImpl) UnWrap() ([]Repository, error) {
	return []Repository{g}, nil
}

func buildPeriodOfTimeSeconds(query *find.FindQuery) (use bool, stOK bool, stSec int, etOK bool, etSec int) {
	if !query.HasPeriodOfTimeFilter() {
		return false, false, 0, false, 0
	}
	use = true

	if query.PeriodOfTimeStartTimeSecond != nil {
		st := time.Unix(*query.PeriodOfTimeStartTimeSecond, 0).In(time.Local)
		stSec = st.Hour()*3600 + st.Minute()*60 + st.Second()
		stOK = true
	}
	if query.PeriodOfTimeEndTimeSecond != nil {
		et := time.Unix(*query.PeriodOfTimeEndTimeSecond, 0).In(time.Local)
		etSec = et.Hour()*3600 + et.Minute()*60 + et.Second()
		etOK = true
	}
	return
}

// commitTimeFilter は「日付範囲 + 時間帯 + 曜日」の判定に必要な値を1回だけ計算して持つ。
//
// FindKyous と FindGitCommitLog が同じ意味論で判定するために共有する。
// 以前は FindGitCommitLog がこの3つを一切適用しておらず、
// 同じインタフェースのキャッシュrep実装(GenerateFindSQLCommon を通す)と結果が食い違っていた。
type commitTimeFilter struct {
	useCalendar       bool
	calendarStartDate *time.Time
	calendarEndDate   *time.Time
	usePeriodOfTime   bool
	startOK           bool
	startSecond       int
	endOK             bool
	endSecond         int
	filterWeekdays    bool
	allowedWeekdays   [7]bool
}

// buildCommitTimeFilter はクエリから判定用の値を1回だけ組み立てる。
// コミット1件ごとに組み立て直すと全走査のたびに同じ計算を繰り返すことになる。
func buildCommitTimeFilter(query *find.FindQuery) commitTimeFilter {
	filter := commitTimeFilter{
		useCalendar:       query.HasCalendarFilter(),
		calendarStartDate: query.CalendarStartDate,
		calendarEndDate:   query.CalendarEndDate,
	}
	filter.usePeriodOfTime, filter.startOK, filter.startSecond, filter.endOK, filter.endSecond = buildPeriodOfTimeSeconds(query)
	filter.filterWeekdays, filter.allowedWeekdays = buildAllowedWeekdays(query)
	return filter
}

// match はコミット1件ぶんの判定。
func (f commitTimeFilter) match(commitTime time.Time) bool {
	// SQL側(unixepoch >= / <=)と同じく両端を含む(以前は排他で境界ちょうどのコミットが落ちていた)
	if f.useCalendar {
		if f.calendarStartDate != nil && commitTime.Before(*f.calendarStartDate) {
			return false
		}
		if f.calendarEndDate != nil && commitTime.After(*f.calendarEndDate) {
			return false
		}
	}
	if f.usePeriodOfTime {
		if !matchPeriodOfTime(commitTime, f.startOK, f.startSecond, f.endOK, f.endSecond) {
			return false
		}
		if !matchWeekOfDays(commitTime, f.filterWeekdays, f.allowedWeekdays) {
			return false
		}
	}
	return true
}

// buildAllowedWeekdays は曜日フィルタの状態を作る。
//
// nil=曜日制限なし / 非nilの空=0件 / 全7曜日=制限なし。
// nil を len==0 や len!=7 の分岐へ落とすと全件が消えるので、必ず nil を先に外すこと。
// 同じ約束を api/find_filter.go の sortAndTrimKyousMap と
// dao/sqlite3impl/sqlite3impl_util.go の generateFindSQLCommon も守っている。
func buildAllowedWeekdays(query *find.FindQuery) (filter bool, allowed [7]bool) {
	if query.PeriodOfTimeWeekOfDays == nil || len(query.PeriodOfTimeWeekOfDays) == 7 {
		return false, allowed
	}
	for _, weekday := range query.PeriodOfTimeWeekOfDays {
		if weekday >= find.SunDay && weekday <= find.SaturDay {
			allowed[weekday] = true
		}
	}
	return true, allowed
}

// matchWeekOfDays は t のローカル曜日が allowed に含まれるかを返す。
// filter が偽なら曜日制限なしなので常に真。
func matchWeekOfDays(t time.Time, filter bool, allowed [7]bool) bool {
	if !filter {
		return true
	}
	weekday := int(t.In(time.Local).Weekday())
	if weekday < 0 || weekday >= len(allowed) {
		return false
	}
	return allowed[weekday]
}

func matchPeriodOfTime(t time.Time, stOK bool, stSec int, etOK bool, etSec int) bool {
	if !stOK && !etOK {
		return true
	}
	lt := t.In(time.Local)
	sec := lt.Hour()*3600 + lt.Minute()*60 + lt.Second()

	if stOK && etOK {
		if stSec <= etSec {
			// 通常: start <= time <= end
			return sec >= stSec && sec <= etSec
		}
		// 夜またぎ: time >= start OR time <= end
		return sec >= stSec || sec <= etSec
	}
	if stOK {
		return sec >= stSec
	}
	return sec <= etSec
}
