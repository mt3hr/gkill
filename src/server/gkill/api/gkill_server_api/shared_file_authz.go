package gkill_server_api

// rep名一致で許さない理由と、判定手順を共有する理由:
// documents/adr/0042-shared-file-authz-by-query.md

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
)

// collectSharedIDFFilePaths は、共有IDの検索条件を再現して、共有ページに現れる
// IDFKyou の (rep名 → 相対パス集合) を返します。ファイル配信（/files/…）の認可に使います。
//
// 手順は handle_get_shared_kyous.go の Kyou→IDFKyou 解決と同一です（一覧を2箇所で
// 維持しないため）。共有クエリで最新版の Kyou 集合を引き、その ID で版を絞らずに
// IDFKyou を引いて、TargetFile を URL パスと同じ正規化にかけて許可集合を作ります。
// ViewType が "mi" の共有は IDFKyou を一切持たないので空集合を返します。
func (g *GkillServerAPI) collectSharedIDFFilePaths(ctx context.Context, sharedKyouInfo *share_kyou_info.ShareKyouInfo, repositories *reps.GkillRepositories) (map[string]map[string]struct{}, error) {
	allowed := map[string]map[string]struct{}{}

	// Mi ビューの共有は IDFKyou を返さない（handle_get_shared_kyous.go:190 のミラー）。
	if sharedKyouInfo.ViewType == "mi" {
		return allowed, nil
	}

	userID := sharedKyouInfo.UserID
	device := sharedKyouInfo.Device

	findQuery := &find.FindQuery{}
	err := json.Unmarshal([]byte(sharedKyouInfo.FindQueryJSON), findQuery)
	if err != nil {
		return nil, fmt.Errorf("error at parse shared find query json: %w", err)
	}
	findQuery.OnlyLatestData = true

	findFilter := &api.FindFilter{}
	kyous, _, err := findFilter.FindKyous(ctx, userID, device, g.GkillDAOManager, findQuery)
	if err != nil {
		return nil, fmt.Errorf("error at find shared kyous user id = %s device = %s: %w", userID, device, err)
	}

	instanceQueryValue := *findQuery
	instanceQuery := &instanceQueryValue
	if len(kyous) == 0 {
		// 検索結果なし → 実体も1件も返さない。ここは「明示的に0件指定」である
		// 非nilの空スライスでなければならない（nil は「IDで絞らない」の意味）。
		instanceQuery.IDs = []string{}
	} else {
		ids := make([]string, 0, len(kyous))
		for _, kyou := range kyous {
			ids = append(ids, kyou.ID)
		}
		instanceQuery.IDs = ids
	}
	instanceQuery.OnlyLatestData = false

	idfKyous, err := repositories.IDFKyouReps.FindIDFKyou(ctx, instanceQuery)
	if err != nil {
		return nil, fmt.Errorf("error at find shared idf kyous user id = %s device = %s: %w", userID, device, err)
	}

	for _, idfKyou := range idfKyous {
		rel := cleanSharedRelPath(idfKyou.TargetFile)
		if _, ok := allowed[idfKyou.RepName]; !ok {
			allowed[idfKyou.RepName] = map[string]struct{}{}
		}
		allowed[idfKyou.RepName][rel] = struct{}{}
	}
	return allowed, nil
}

// isSharedFileAllowed は、要求された (rep名, 相対パス) が許可集合に属するかを返します。
// 正規化は buildIDFFileURL 側（cleanRelativeURLPath）と同一にしてあり、
// サブディレクトリ・パス表記揺れ（. / .. / 連続スラッシュ）も同じに畳みます。
func isSharedFileAllowed(allowed map[string]map[string]struct{}, repName, requestedRel string) bool {
	paths, ok := allowed[repName]
	if !ok {
		return false
	}
	_, ok = paths[cleanSharedRelPath(requestedRel)]
	return ok
}

// cleanSharedRelPath は dao/reps/idf_file_url.go の cleanRelativeURLPath のミラー
// （あちらは非公開なので同ロジックを写している）。両側で同一に正規化するのが肝。
func cleanSharedRelPath(p string) string {
	cleaned := path.Clean("/" + filepath.ToSlash(p))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}
