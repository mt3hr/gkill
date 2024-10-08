package account_state

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type fileUploadHistoryDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewFileUploadHistoryDAOSQLite3Impl(ctx context.Context, filename string) (FileUploadHistoryDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "FILE_UPLOAD_HISTORY" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  FILE_NAME NOT NULL,
  FILE_SIZE_BYTE NOT NULL,
  SUCCESSED NOT NULL,
  SOURCE_ADDRESS NOT NULL,
  UPLOAD_TIME NOT NULL
);`
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create FILE_UPLOAD_HISTORY table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create FILE_UPLOAD_HISTORY table to %s: %w", filename, err)
		return nil, err
	}

	return &fileUploadHistoryDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (f *fileUploadHistoryDAOSQLite3Impl) GetAllFileUploadHistories(ctx context.Context) ([]*FileUploadHistory, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  FILE_NAME,
  FILE_SIZE_BYTE,
  SUCCESSED,
  SOURCE_ADDRESS,
  UPLOAD_TIME
FROM FILE_UPLOAD_HISTORY
`
	stmt, err := f.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all file upload histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	fileUploadHistories := []*FileUploadHistory{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			fileUploadHistory := &FileUploadHistory{}
			uploadTimeStr := ""
			err = rows.Scan(
				&fileUploadHistory.ID,
				&fileUploadHistory.UserID,
				&fileUploadHistory.Device,
				&fileUploadHistory.FileName,
				&fileUploadHistory.FileSizeByte,
				&fileUploadHistory.Successed,
				&fileUploadHistory.SourceAddress,
				&uploadTimeStr,
			)

			fileUploadHistory.UploadTime, err = time.Parse(sqlite3impl.TimeLayout, uploadTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in FILE_UPLOAD_HISTORY: %w", uploadTimeStr, fileUploadHistory.ID, err)
				return nil, err
			}

			fileUploadHistories = append(fileUploadHistories, fileUploadHistory)
		}
	}
	return fileUploadHistories, nil
}

func (f *fileUploadHistoryDAOSQLite3Impl) GetFileUploadHistories(ctx context.Context, userID string, device string) ([]*FileUploadHistory, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  FILE_NAME,
  FILE_SIZE_BYTE,
  SUCCESSED,
  SOURCE_ADDRESS,
  UPLOAD_TIME
FROM FILE_UPLOAD_HISTORY
WHERE USER_ID = ?
`
	stmt, err := f.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get file upload histories sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	fileUploadHistories := []*FileUploadHistory{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			fileUploadHistory := &FileUploadHistory{}
			uploadTimeStr := ""
			err = rows.Scan(
				&fileUploadHistory.ID,
				&fileUploadHistory.UserID,
				&fileUploadHistory.Device,
				&fileUploadHistory.FileName,
				&fileUploadHistory.FileSizeByte,
				&fileUploadHistory.Successed,
				&fileUploadHistory.SourceAddress,
				&uploadTimeStr,
			)

			fileUploadHistory.UploadTime, err = time.Parse(sqlite3impl.TimeLayout, uploadTimeStr)
			if err != nil {
				err = fmt.Errorf("error at parse file upload time %s at %s in FILE_UPLOAD_HISTORY: %w", uploadTimeStr, fileUploadHistory.ID, err)
				return nil, err
			}

			fileUploadHistories = append(fileUploadHistories, fileUploadHistory)
		}
	}
	return fileUploadHistories, nil
}

func (f *fileUploadHistoryDAOSQLite3Impl) AddFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) (bool, error) {
	sql := `
INSERT INTO FILE_UPLOAD_HISTORY (
  ID,
  USER_ID,
  DEVICE,
  FILE_NAME,
  FILE_SIZE_BYTE,
  SUCCESSED,
  SOURCE_ADDRESS,
  UPLOAD_TIME
)
VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	stmt, err := f.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add file upload histories sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		fileUploadHistory.ID,
		fileUploadHistory.UserID,
		fileUploadHistory.Device,
		fileUploadHistory.FileName,
		fileUploadHistory.FileSizeByte,
		fileUploadHistory.Successed,
		fileUploadHistory.UploadTime.Format(sqlite3impl.TimeLayout),
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (f *fileUploadHistoryDAOSQLite3Impl) UpdateFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) (bool, error) {
	sql := `
UPDATE FILE_UPLOAD_HISTORY SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  FILE_NAME = ?,
  FILE_SIZE_BYTE = ?,
  SUCCESSED = ?,
  SOURCE_ADDRESS = ?,
  UPLOAD_TIME = ?
WHERE ID = ?
`
	stmt, err := f.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add file upload histories sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		fileUploadHistory.ID,
		fileUploadHistory.UserID,
		fileUploadHistory.Device,
		fileUploadHistory.FileName,
		fileUploadHistory.FileSizeByte,
		fileUploadHistory.Successed,
		fileUploadHistory.UploadTime.Format(sqlite3impl.TimeLayout),
		fileUploadHistory.ID,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (f *fileUploadHistoryDAOSQLite3Impl) DeleteFileUploadHistory(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FILE_UPLOAD_HISTORY
WHERE ID = ?
`
	stmt, err := f.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete file upload history sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (f *fileUploadHistoryDAOSQLite3Impl) Close(ctx context.Context) error {
	return f.db.Close()
}
