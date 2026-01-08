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
	Mutex    = &sync.Mutex{}
	MemoryDB *sql.DB
)

func init() {
	InitMemoryDB()
}

func InitMemoryDB() {
	var err error
	MemoryDB, err = sql.Open("sqlite3", "file::memory:?mode=memory&cache=shared&_busy_timeout=6000&_txlock=immediate&_journal_mode=MEMORY&_synchronous=OFF")
	if err != nil {
		err = fmt.Errorf("error at open memory database: %w", err)
		gkill_log.Debug.Fatal(err)
	}

	MemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
	MemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
	MemoryDB.SetConnMaxLifetime(0)             // 無限
	MemoryDB.SetConnMaxIdleTime(0)             // 無限
}
