package reps

import (
	"errors"
	"slices"
)

// RepLoadFailure は構築時に読み込めなかったrep1本の記録。
//
// **RepName にパスを入れないこと。** この値は GkillMessage（警告）へ載る。
// GkillMessage は GkillError と違い MarshalJSON による伏せ処理が掛からないので、
// パスをそのまま入れるとホームディレクトリの利用者名が応答に出る。
type RepLoadFailure struct {
	// RepName は読み込めなかったrepの名前。
	// 利用者が「一覧からどれが消えたのか」を知り、保存済みの検索条件(列のReps)に
	// 残った名前を直せるようにするためのもの。query.reps へ渡せる名前と同じ規則。
	RepName string

	// RepType は user_config の rep 種別（"directory" / "kmemo" など）。
	RepType string

	// Err は原因。ログと診断用で、APIの応答へ生で載せない。
	Err error
}

// AppendLoadFailure は読み込めなかったrepを記録します。
func (g *GkillRepositories) AppendLoadFailure(failure RepLoadFailure) {
	syncState := g.syncState()
	syncState.loadFailuresMutex.Lock()
	defer syncState.loadFailuresMutex.Unlock()
	g.loadFailures = append(g.loadFailures, failure)
}

// LoadFailures は読み込めなかったrepの記録を返します。
func (g *GkillRepositories) LoadFailures() []RepLoadFailure {
	syncState := g.syncState()
	syncState.loadFailuresMutex.RLock()
	defer syncState.loadFailuresMutex.RUnlock()
	return slices.Clone(g.loadFailures)
}

// LoadFailureError は読み込めなかったrepの原因をまとめて返します。
// 1件も無ければnilを返します。ログ・診断用で、APIの応答へ生で載せないこと。
func (g *GkillRepositories) LoadFailureError() error {
	failures := g.LoadFailures()
	if len(failures) == 0 {
		return nil
	}
	errs := make([]error, 0, len(failures))
	for _, failure := range failures {
		errs = append(errs, failure.Err)
	}
	return errors.Join(errs...)
}
