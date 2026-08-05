package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// MiReKyouTempRepository はTX確定前のMiReKyou（既存のKyouをタスク化したもの）を置く
// 一時リポジトリが満たす契約です。
//
// 実体は利用者ごとの一時DB（--cache_in_memory=true ならインメモリ、falseなら
// caches/temp_cache/{userID}_temp_.db）のMIREKYOU表で、通常の列に加えて
// USER_ID / DEVICE / TX_ID を持ちます。保持するのはタスク化対象への参照（TARGET_ID）と
// Miのスケジュール項目だけで、タイトルは持ちません。AddMiReKyouInfo で貯め、
// commit_tx が GetMiReKyousByTXID で取り出して本リポジトリへ書き、
// discard_tx が DeleteByTXID で捨てます。
//
// 他のデータ型の一時リポジトリと違い、この実装は渡された一時DBハンドルをそのまま使うため、
// 委譲系のメソッドも一時DBの中身を返します。ただしどれも TX_ID では絞り込みません。
// TX単位で扱いたいときは (txID, userID, device) を取るメソッドを使ってください。
type MiReKyouTempRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	// MiReKyouは1レコードから作成・チェック・期限・開始・終了の5射影のKyouへ展開されます
	// （射影の元になる列がNULLの行はその射影を出しません）。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath は一時リポジトリでは常にエラーを返します。
	// DBハンドルを外から渡される作りで、自分のファイルを持たないためです。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache は一時リポジトリでは何もせず nil を返します。
	// 契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress は一時DBのMIREKYOU表から最新版の所在を返します。
	// 他のデータ型の一時リポジトリがエラーを返すのに対し、ここだけは実際に値を返します。
	// 契約は Repository.GetLatestDataRepositoryAddress を参照。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName は固定名 "mirekyou_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close は何もせず nil を返します。
	// 一時DBは全一時リポジトリで共有しているため、ここで閉じてはいけないからです。
	Close(ctx context.Context) error

	// FindMiReKyou は検索条件に一致するMiReKyouを返します。
	// FindKyous と違い、射影は query の IncludeCreateMi / IncludeCheckMi / IncludeLimitMi /
	// IncludeStartMi / IncludeEndMi で絞られるため、どれも立っていなければ0件になります。
	// ターゲットIDを解決できないものも除外しますが、一時リポジトリは解決に使うリポジトリ群を
	// 持たないため、この除外は行われず素通しになります。順序は保証しません。
	FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error)

	// GetMiReKyou は id に対応するMiReKyouを1件返します。
	// updateTime が nil なら最新バージョンです。見つからない場合は (nil, nil) を返します。
	GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error)

	// GetMiReKyouHistories は id の全バージョンを返します。
	GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error)

	// AddMiReKyouInfo は mirekyou を (txID, userID, device) 付きで一時DBに追記します。
	// mirekyou.TargetID（タスク化する対象のKyou ID）の存在確認はここでは行いません。
	// 追記専用なので、同一IDの更新は新しい UPDATE_TIME の版を足すことで表します。
	AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou, txID string, userID string, device string) error

	// GetMiReKyousAllLatest はターゲット解決を行わない生のMiReKyou（ID毎の最新）を返します。
	// ターゲットの存在確認やワード検索は FindMiReKyou / FindKyous 側で行います。
	GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error)

	// GetBoardNames は一時DBのMIREKYOU表にあるボード名を重複排除して返します。
	// 版や削除フラグでは絞らないため、旧版や削除済みのボード名も含みます。順序は保証しません。
	GetBoardNames(ctx context.Context) ([]string, error)

	// GetKyousByTXID は GetMiReKyousByTXID の結果をKyouへ写して返します。
	// RelatedTime には CreateTime が入り、DataType は "mirekyou_create" 固定です
	// （FindKyous のような5射影への展開はしません）。
	// 該当0件はエラーではなく空スライスです。
	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	// GetMiReKyousByTXID は (txID, userID, device) が一致する一時データをすべて返します。
	// バージョンの絞り込みはしないので、同一IDに複数版があればすべて含みます。
	// DataType は作成射影の "mirekyou_create" 固定です。
	// 該当0件はエラーではなく空スライスです。
	GetMiReKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]MiReKyou, error)

	// DeleteByTXID は (txID, userID, device) が一致する一時データを行ごと削除します。
	// 本リポジトリの論理削除と違って物理削除で、該当0件でもエラーになりません。
	// タスク化対象のKyouには触れません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は MiReKyouTempRepository 型のままリーフ実装へ平坦化します。
	// 契約は Repository.UnWrap を参照。
	UnWrapTyped() ([]MiReKyouTempRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
