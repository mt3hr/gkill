package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type deviceStructDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewDeviceStructDAOSQLite3Impl(ctx context.Context, filename string) (DeviceStructDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "DEVICE_STRUCT" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  DEVICE_NAME NOT NULL,
  PARENT_FOLDER_ID,
  SEQ NOT NULL,
  CHECK_WHEN_INITED NOT NULL
);`
	log.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create DEVICE_STRUCT table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	log.Printf("sql: %s", sql)
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create DEVICE_STRUCT table to %s: %w", filename, err)
		return nil, err
	}

	return &deviceStructDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (d *deviceStructDAOSQLite3Impl) GetAllDeviceStructs(ctx context.Context) ([]*DeviceStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  DEVICE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
FROM DEVICE_STRUCT
`
	log.Printf("sql: %s", sql)
	stmt, err := d.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all device struct sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	log.Printf("sql: %s", sql)
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	deviceStructs := []*DeviceStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			deviceStruct := &DeviceStruct{}
			err = rows.Scan(
				&deviceStruct.ID,
				&deviceStruct.UserID,
				&deviceStruct.Device,
				&deviceStruct.DeviceName,
				&deviceStruct.ParentFolderID,
				&deviceStruct.Seq,
				&deviceStruct.CheckWhenInited,
			)
			deviceStructs = append(deviceStructs, deviceStruct)
		}
	}
	return deviceStructs, nil
}

func (d *deviceStructDAOSQLite3Impl) GetDeviceStructs(ctx context.Context, userID string, device string) ([]*DeviceStruct, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  DEVICE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
FROM DEVICE_STRUCT
WHERE USER_ID = ? AND DEVICE = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := d.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get device struct sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
		device,
	}
	log.Printf("sql: %s query: %#v", sql, queryArgs)
	rows, err := stmt.QueryContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	deviceStructs := []*DeviceStruct{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			deviceStruct := &DeviceStruct{}
			err = rows.Scan(
				&deviceStruct.ID,
				&deviceStruct.UserID,
				&deviceStruct.Device,
				&deviceStruct.DeviceName,
				&deviceStruct.ParentFolderID,
				&deviceStruct.Seq,
				&deviceStruct.CheckWhenInited,
			)
			deviceStructs = append(deviceStructs, deviceStruct)
		}
	}
	return deviceStructs, nil
}

func (d *deviceStructDAOSQLite3Impl) AddDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) (bool, error) {
	sql := `
INSERT INTO DEVICE_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  DEVICE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	log.Printf("sql: %s", sql)
	stmt, err := d.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add device struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		deviceStruct.ID,
		deviceStruct.UserID,
		deviceStruct.Device,
		deviceStruct.DeviceName,
		deviceStruct.ParentFolderID,
		deviceStruct.Seq,
		deviceStruct.CheckWhenInited,
	}
	log.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (d *deviceStructDAOSQLite3Impl) AddDeviceStructs(ctx context.Context, deviceStructs []*DeviceStruct) (bool, error) {
	tx, err := d.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, deviceStruct := range deviceStructs {
		sql := `
INSERT INTO DEVICE_STRUCT (
  ID,
  USER_ID,
  DEVICE,
  DEVICE_NAME,
  PARENT_FOLDER_ID,
  SEQ,
  CHECK_WHEN_INITED
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  ?
)
`
	log.Printf("sql: %s", sql)
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add device struct sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

		queryArgs := []interface{}{
			deviceStruct.ID,
			deviceStruct.UserID,
			deviceStruct.Device,
			deviceStruct.DeviceName,
			deviceStruct.ParentFolderID,
			deviceStruct.Seq,
			deviceStruct.CheckWhenInited,
		}
		log.Printf("sql: %s query: %#v", sql, queryArgs)
		_, err = stmt.ExecContext(ctx, queryArgs...)

		if err != nil {
			err = fmt.Errorf("error at query :%w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
	}
	err = tx.Commit()
	if err != nil {
		fmt.Errorf("error at commit: %w", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = fmt.Errorf("%w: %w", err, rollbackErr)
		}
		return false, err
	}

	return true, nil
}

func (d *deviceStructDAOSQLite3Impl) UpdateDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) (bool, error) {
	sql := `
UPDATE DEVICE_STRUCT SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  DEVICE_NAME = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?,
  CHECK_WHEN_INITED = ?
WHERE ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := d.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update device struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		deviceStruct.ID,
		deviceStruct.UserID,
		deviceStruct.Device,
		deviceStruct.DeviceName,
		deviceStruct.ParentFolderID,
		deviceStruct.Seq,
		deviceStruct.CheckWhenInited,
		deviceStruct.ID,
	}
	log.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (d *deviceStructDAOSQLite3Impl) DeleteDeviceStruct(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM DEVICE_STRUCT
WHERE ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := d.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete device struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		id,
	}
	log.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (d *deviceStructDAOSQLite3Impl) DeleteUsersDeviceStructs(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE FROM DEVICE_STRUCT
WHERE USER_ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := d.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete device struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	queryArgs := []interface{}{
		userID,
	}
	log.Printf("sql: %s query: %#v", sql, queryArgs)
	_, err = stmt.ExecContext(ctx, queryArgs...)

	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (d *deviceStructDAOSQLite3Impl) Close(ctx context.Context) error {
	return d.db.Close()
}
