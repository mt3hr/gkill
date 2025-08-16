package memory_db

import (
	"database/sql"
	"fmt"
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
	MemoryDB, err = sql.Open("sqlite3", "file::memory:?_journal_mode=MEMORY&_busy_timeout=6000&_synchronous=NORMAL&_cache_size=-200000")
	if err != nil {
		err = fmt.Errorf("error at open memory database: %w", err)
		gkill_log.Debug.Fatal(err)
	}

	MemoryDB.SetMaxOpenConns(1)
	MemoryDB.SetMaxIdleConns(1)
}
