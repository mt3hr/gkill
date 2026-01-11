package memory_db

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

var (
	CacheMemoryDBMutex = &sync.Mutex{}
	CacheMemoryDB      *sql.DB
	TempMemoryDBMutex  = &sync.Mutex{}
	TempMemoryDB       *sql.DB
)

func init() {
	InitMemoryDB()
}

func InitMemoryDB() {
	var err error
	CacheMemoryDB, err = sql.Open("sqlite3", "file:gkill_memory_db?mode=memory&cache=shared&_busy_timeout=6000&_txlock=immediate&_journal_mode=MEMORY&_synchronous=OFF")
	if err != nil {
		err = fmt.Errorf("error at open memory database: %w", err)
		gkill_log.Debug.Fatal(err)
	}
	CacheMemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
	CacheMemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
	CacheMemoryDB.SetConnMaxLifetime(0)             // 無限
	CacheMemoryDB.SetConnMaxIdleTime(0)             // 無限

	TempMemoryDB, err = sql.Open("sqlite3", "file:gkill_temp_db?mode=memory&cache=shared&_busy_timeout=6000&_txlock=immediate&_journal_mode=MEMORY&_synchronous=OFF")
	if err != nil {
		err = fmt.Errorf("error at open memory database: %w", err)
		gkill_log.Debug.Fatal(err)
	}
	TempMemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
	TempMemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
	TempMemoryDB.SetConnMaxLifetime(0)             // 無限
	TempMemoryDB.SetConnMaxIdleTime(0)             // 無限
}
