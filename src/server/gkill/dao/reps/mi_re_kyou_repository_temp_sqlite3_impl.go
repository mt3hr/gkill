package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	_ "modernc.org/sqlite"
)

// miReKyouTempRepositorySQLite3Impl はTX中のMiReKyouを保持する一時リポジトリです。
// テーブル定義に USER_ID / DEVICE / TX_ID を足した以外は実体と同じです。
type miReKyouTempRepositorySQLite3Impl miReKyouRepositorySQLite3Impl

func NewMiReKyouTempRepositorySQLite3Impl(ctx context.Context, db *sqllib.DB, m *sync.RWMutex) (MiReKyouTempRepository, error) {
	filename := "mirekyou_temp"

	sql := `CREATE TABLE IF NOT EXISTS "` + miReKyouTableName + `" (` + miReKyouColumns + `,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TX_ID NOT NULL
);`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU table statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU table to %s: %w", filename, err)
		return nil, err
	}

	indexSQL := `CREATE INDEX IF NOT EXISTS INDEX_MIREKYOU ON ` + miReKyouTableName + ` (ID, UPDATE_TIME);`
	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	indexStmt, err := db.PrepareContext(ctx, indexSQL)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU index statement %s: %w", filename, err)
		return nil, err
	}
	defer func() {
		err := indexStmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	slog.Log(ctx, gkill_log.TraceSQL, "index sql", "sql", fmt.Sprintf("%q", indexSQL))
	_, err = indexStmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create MIREKYOU index to %s: %w", filename, err)
		return nil, err
	}

	// 一時表への問い合わせは WHERE TX_ID = ? AND USER_ID = ? AND DEVICE = ? の形なので、
	// ID先頭の既存索引では効かない。
	if err := sqlite3impl.EnsureTxIDIndex(ctx, db, miReKyouTableName); err != nil {
		return nil, err
	}

	return &miReKyouTempRepositorySQLite3Impl{
		filename: filename,
		db:       db,
		m:        m,
		// 渡されたインメモリDBをそのまま使うためfullConnectを立てる
		fullConnect: true,
	}, nil
}

// impl は実体リポジトリとして扱うための変換です。
func (m *miReKyouTempRepositorySQLite3Impl) impl() miReKyouRepositorySQLite3Impl {
	return miReKyouRepositorySQLite3Impl(*m)
}

func (m *miReKyouTempRepositorySQLite3Impl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	impl := m.impl()
	return impl.FindKyous(ctx, query)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	impl := m.impl()
	return impl.GetKyou(ctx, id, updateTime)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	impl := m.impl()
	return impl.GetKyouHistories(ctx, id)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	return "", fmt.Errorf("miReKyouTempRepositorySQLite3Impl does not support GetPath")
}

func (m *miReKyouTempRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	return nil
}

func (m *miReKyouTempRepositorySQLite3Impl) LastUpdateCacheChanged() bool {
	return true
}

func (m *miReKyouTempRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	return "mirekyou_temp", nil
}

func (m *miReKyouTempRepositorySQLite3Impl) Close(ctx context.Context) error {
	// インメモリDBは共有されているためここでは閉じない
	return nil
}

func (m *miReKyouTempRepositorySQLite3Impl) FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error) {
	impl := m.impl()
	return impl.FindMiReKyou(ctx, query)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	impl := m.impl()
	return impl.GetMiReKyou(ctx, id, updateTime)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	impl := m.impl()
	return impl.GetMiReKyouHistories(ctx, id)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error) {
	impl := m.impl()
	return impl.GetMiReKyousAllLatest(ctx)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetBoardNames(ctx context.Context) ([]string, error) {
	impl := m.impl()
	return impl.GetBoardNames(ctx)
}

func (m *miReKyouTempRepositorySQLite3Impl) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	impl := m.impl()
	return impl.GetLatestDataRepositoryAddress(ctx, updateCache)
}

func (m *miReKyouTempRepositorySQLite3Impl) AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou, txID string, userID string, device string) error {
	m.m.Lock()
	defer m.m.Unlock()

	sql := `INSERT INTO ` + miReKyouTableName + ` (` + miReKyouInsertColumnNames + `,
  USER_ID,
  DEVICE,
  TX_ID
) VALUES (` + miReKyouInsertPlaceHolders + `, ?, ?, ?)`
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
	stmt, err := m.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add mirekyou sql %s: %w", mirekyou.ID, err)
		return err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	queryArgs := append(miReKyouInsertArgs(mirekyou), userID, device, txID)
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	_, err = stmt.ExecContext(ctx, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at insert in to mirekyou %s: %w", mirekyou.ID, err)
		return err
	}
	return nil
}

func (m *miReKyouTempRepositorySQLite3Impl) GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error) {
	mirekyous, err := m.GetMiReKyousByTXID(ctx, txID, userID, device)
	if err != nil {
		return nil, err
	}

	kyous := []Kyou{}
	for _, mirekyou := range mirekyous {
		kyou := Kyou{}
		kyou.IsDeleted = mirekyou.IsDeleted
		kyou.ID = mirekyou.ID
		kyou.RepName = mirekyou.RepName
		kyou.RelatedTime = mirekyou.CreateTime
		kyou.DataType = mirekyou.DataType
		kyou.CreateTime = mirekyou.CreateTime
		kyou.CreateApp = mirekyou.CreateApp
		kyou.CreateDevice = mirekyou.CreateDevice
		kyou.CreateUser = mirekyou.CreateUser
		kyou.UpdateTime = mirekyou.UpdateTime
		kyou.UpdateApp = mirekyou.UpdateApp
		kyou.UpdateDevice = mirekyou.UpdateDevice
		kyou.UpdateUser = mirekyou.UpdateUser
		kyous = append(kyous, kyou)
	}
	return kyous, nil
}

func (m *miReKyouTempRepositorySQLite3Impl) GetMiReKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]MiReKyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	repName, err := m.GetRepName(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rep name at mirekyou temp: %w", err)
		return nil, err
	}

	sql := "SELECT " + miReKyouSelectColumns + `  ? AS DATA_TYPE
FROM ` + miReKyouTableName + `
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	queryArgs := []any{
		repName,
		miReKyouProjections[0].dataType,
		txID,
		userID,
		device,
	}

	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	rows, err := m.db.QueryContext(ctx, sql, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at select from MIREKYOU: %w", err)
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	return scanMiReKyous(ctx, rows, repName)
}

func (m *miReKyouTempRepositorySQLite3Impl) DeleteByTXID(ctx context.Context, txID string, userID string, device string) error {
	m.m.Lock()
	defer m.m.Unlock()

	sql := `DELETE FROM ` + miReKyouTableName + `
WHERE TX_ID = ?
AND USER_ID = ?
AND DEVICE = ?
`
	queryArgs := []any{
		txID,
		userID,
		device,
	}
	slog.Log(ctx, gkill_log.TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
	_, err := m.db.ExecContext(ctx, sql, queryArgs...)
	if err != nil {
		err = fmt.Errorf("error at delete from MIREKYOU by tx id %s: %w", txID, err)
		return err
	}
	return nil
}

func (m *miReKyouTempRepositorySQLite3Impl) UnWrapTyped() ([]MiReKyouTempRepository, error) {
	return []MiReKyouTempRepository{m}, nil
}

func (m *miReKyouTempRepositorySQLite3Impl) UnWrap() ([]Repository, error) {
	return []Repository{m}, nil
}
