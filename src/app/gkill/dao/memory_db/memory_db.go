package memory_db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

var (
	MemoryDB *sql.DB
)

func init() {
	var err error
	MemoryDB, err = sql.Open("sqlite3", "file::memory:?_timeout=6000&_synchronous=1&mode=memory&_mutex=full")
	if err != nil {
		err = fmt.Errorf("error at open memory database: %w", err)
		gkill_log.Debug.Fatal(err)
	}

	MemoryDB.SetMaxOpenConns(1)
	MemoryDB.SetMaxIdleConns(1)
}
