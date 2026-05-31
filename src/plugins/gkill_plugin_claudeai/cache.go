package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// pluginCache はconversations.jsonをSQLite3にキャッシュする。
// {pluginDir}/cache.db に保存し、ソースファイルのmtimeで無効化する。
type pluginCache struct {
	mu sync.RWMutex
	db *sql.DB
}

var globalCache = &pluginCache{}

// cachedMessage はmsg_cacheテーブルの1行。
type cachedMessage struct {
	MsgID           string
	ConvID          string
	Sender          string
	Text            string
	RelatedTimeUnix int64
	CreateTimeUnix  int64
	UpdateTimeUnix  int64
}

// openDB はキャッシュDBを開く（初回は初期化する）。
func (c *pluginCache) openDB(pluginDir string) error {
	if c.db != nil {
		return nil
	}
	dbPath := filepath.Join(pluginDir, "cache.db")
	db, err := sqlite3impl.GetSQLiteDBConnection(context.Background(), dbPath)
	if err != nil {
		return fmt.Errorf("error at open cache db %s: %w", dbPath, err)
	}
	if err := initSchema(db); err != nil {
		db.Close()
		return err
	}
	c.db = db
	return nil
}

// initSchema はテーブルを作成する。
func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS cache_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS conv_cache (
  conv_id          TEXT PRIMARY KEY,
  title            TEXT NOT NULL,
  create_time_unix INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS msg_cache (
  msg_id            TEXT PRIMARY KEY,
  conv_id           TEXT NOT NULL,
  sender            TEXT NOT NULL,
  text              TEXT NOT NULL,
  related_time_unix INTEGER NOT NULL,
  create_time_unix  INTEGER NOT NULL,
  update_time_unix  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_msg_conv ON msg_cache(conv_id);
CREATE INDEX IF NOT EXISTS idx_msg_time ON msg_cache(related_time_unix);
`)
	if err != nil {
		return fmt.Errorf("error at init schema: %w", err)
	}
	return nil
}

// GetMsgByID はGetContentHTML用に、msgIDに対応するメッセージ1件と会話タイトルを返す。
func (c *pluginCache) GetMsgByID(pluginDir string, msgID string) (convTitle string, msg cachedMessage, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err = c.openDB(pluginDir); err != nil {
		return
	}

	srcPath := filepath.Join(pluginDir, conversationsFile)
	needRebuild, e := c.needsRebuild(srcPath)
	if e != nil {
		err = e
		return
	}
	if needRebuild {
		if e := c.rebuild(pluginDir, srcPath); e != nil {
			err = e
			return
		}
	}

	row := c.db.QueryRow(`
		SELECT m.msg_id, m.conv_id, m.sender, m.text,
		       m.related_time_unix, m.create_time_unix, m.update_time_unix,
		       COALESCE(c.title, '')
		FROM msg_cache m
		LEFT JOIN conv_cache c ON m.conv_id = c.conv_id
		WHERE m.msg_id = ?`, msgID)
	if e := row.Scan(&msg.MsgID, &msg.ConvID, &msg.Sender, &msg.Text,
		&msg.RelatedTimeUnix, &msg.CreateTimeUnix, &msg.UpdateTimeUnix,
		&convTitle); e != nil {
		err = fmt.Errorf("msg not found: %s", msgID)
	}
	return
}

// GetMessages はFindKyous用に全メッセージを返す。
// conversations.jsonのmtimeが変わっていればキャッシュを再構築する。
func (c *pluginCache) GetMessages(pluginDir string) ([]cachedMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return nil, err
	}

	srcPath := filepath.Join(pluginDir, conversationsFile)
	needRebuild, err := c.needsRebuild(srcPath)
	if err != nil {
		return nil, err
	}
	if needRebuild {
		if err := c.rebuild(pluginDir, srcPath); err != nil {
			return nil, err
		}
	}
	return c.readMessages()
}

// GetConvForMsg はGetContentHTML用に、msgIDが属する会話の全メッセージと会話タイトルを返す。
func (c *pluginCache) GetConvForMsg(pluginDir string, msgID string) (convTitle string, msgs []cachedMessage, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err = c.openDB(pluginDir); err != nil {
		return
	}

	srcPath := filepath.Join(pluginDir, conversationsFile)
	needRebuild, e := c.needsRebuild(srcPath)
	if e != nil {
		err = e
		return
	}
	if needRebuild {
		if e := c.rebuild(pluginDir, srcPath); e != nil {
			err = e
			return
		}
	}

	// convIDを特定
	var convID string
	row := c.db.QueryRow(`SELECT conv_id FROM msg_cache WHERE msg_id = ?`, msgID)
	if e := row.Scan(&convID); e != nil {
		err = fmt.Errorf("msg not found: %s", msgID)
		return
	}

	// 会話タイトル取得
	row = c.db.QueryRow(`SELECT title FROM conv_cache WHERE conv_id = ?`, convID)
	if e := row.Scan(&convTitle); e != nil {
		convTitle = ""
	}

	// 会話内の全メッセージを時系列順で取得
	rows, e := c.db.Query(`SELECT msg_id, conv_id, sender, text, related_time_unix, create_time_unix, update_time_unix FROM msg_cache WHERE conv_id = ? ORDER BY related_time_unix ASC`, convID)
	if e != nil {
		err = fmt.Errorf("error at query conv messages: %w", e)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var m cachedMessage
		if e := rows.Scan(&m.MsgID, &m.ConvID, &m.Sender, &m.Text, &m.RelatedTimeUnix, &m.CreateTimeUnix, &m.UpdateTimeUnix); e != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return
}

// needsRebuild はソースファイルのmtimeとキャッシュのmtimeを比較する。
func (c *pluginCache) needsRebuild(srcPath string) (bool, error) {
	info, err := os.Stat(srcPath)
	if err != nil {
		return false, fmt.Errorf("error at stat %s: %w", srcPath, err)
	}
	srcMtime := info.ModTime().Unix()

	var cached int64
	row := c.db.QueryRow(`SELECT value FROM cache_meta WHERE key = 'source_mtime_unix'`)
	if err := row.Scan(&cached); err != nil {
		// まだキャッシュなし
		return true, nil
	}
	return srcMtime != cached, nil
}

// rebuild はconversations.jsonを読み込んでキャッシュを再構築する。
func (c *pluginCache) rebuild(pluginDir, srcPath string) error {
	convs, err := loadConversations(pluginDir)
	if err != nil {
		return err
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	srcMtime := info.ModTime().Unix()

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM msg_cache`); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM conv_cache`); err != nil {
		return err
	}

	for _, conv := range convs {
		convID := conv.UUID
		title := conv.Name
		createUnix := conv.CreatedAt.Unix()
		if _, err = tx.Exec(`INSERT OR REPLACE INTO conv_cache(conv_id, title, create_time_unix) VALUES(?,?,?)`,
			convID, title, createUnix); err != nil {
			return fmt.Errorf("error at insert conv: %w", err)
		}
		for _, msg := range conv.ChatMessages {
			if msg.UUID == "" || msg.Text == "" {
				continue
			}
			if _, err = tx.Exec(`INSERT OR REPLACE INTO msg_cache(msg_id, conv_id, sender, text, related_time_unix, create_time_unix, update_time_unix) VALUES(?,?,?,?,?,?,?)`,
				msg.UUID, convID, msg.Sender, msg.Text,
				msg.CreatedAt.Unix(), msg.CreatedAt.Unix(), msg.UpdatedAt.Unix(),
			); err != nil {
				return fmt.Errorf("error at insert msg: %w", err)
			}
		}
	}

	if _, err = tx.Exec(`INSERT OR REPLACE INTO cache_meta(key,value) VALUES('source_mtime_unix', ?)`, fmt.Sprintf("%d", srcMtime)); err != nil {
		return err
	}
	return tx.Commit()
}

// readMessages は全メッセージをDBから読み込む。
func (c *pluginCache) readMessages() ([]cachedMessage, error) {
	rows, err := c.db.Query(`SELECT msg_id, conv_id, sender, text, related_time_unix, create_time_unix, update_time_unix FROM msg_cache ORDER BY related_time_unix DESC`)
	if err != nil {
		return nil, fmt.Errorf("error at query messages: %w", err)
	}
	defer rows.Close()
	var msgs []cachedMessage
	for rows.Next() {
		var m cachedMessage
		if err := rows.Scan(&m.MsgID, &m.ConvID, &m.Sender, &m.Text, &m.RelatedTimeUnix, &m.CreateTimeUnix, &m.UpdateTimeUnix); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// unixToTime はUnixタイムスタンプをtime.Timeに変換する。
func unixToTimeFromCache(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}
