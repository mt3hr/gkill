package user_config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type kftlTemplateDAOSQLite3Impl struct {
	filename string
	db       *sql.DB
	m        *sync.Mutex
}

func NewKFTLTemplateDAOSQLite3Impl(ctx context.Context, filename string) (KFTLTemplateDAO, error) {
	var err error
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		err = fmt.Errorf("error at open database %s: %w", filename, err)
		return nil, err
	}

	sql := `
CREATE TABLE IF NOT EXISTS "KFTL_TEMPLATE" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  TITLE NOT NULL,
  TEMPLATE NOT NULL,
  PARENT_FOLDER_ID,
  SEQ NOT NULL
);`
	log.Printf("sql: %s", sql)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at create KFTL_TEMPLATE table statement %s: %w", filename, err)
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at create KFTL_TEMPLATE table to %s: %w", filename, err)
		return nil, err
	}

	return &kftlTemplateDAOSQLite3Impl{
		filename: filename,
		db:       db,
		m:        &sync.Mutex{},
	}, nil
}
func (k *kftlTemplateDAOSQLite3Impl) GetAllKFTLTemplates(ctx context.Context) ([]*KFTLTemplate, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TITLE,
  TEMPLATE,
  PARENT_FOLDER_ID,
  SEQ
FROM KFTL_TEMPLATE
`
	log.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get all kftl templates sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	kftlTemplates := []*KFTLTemplate{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kftlTemplate := &KFTLTemplate{}
			err = rows.Scan(
				&kftlTemplate.ID,
				&kftlTemplate.UserID,
				&kftlTemplate.Device,
				&kftlTemplate.Title,
				&kftlTemplate.Template,
				&kftlTemplate.ParentFolderID,
				&kftlTemplate.Seq,
			)
			kftlTemplates = append(kftlTemplates, kftlTemplate)
		}
	}
	return kftlTemplates, nil
}

func (k *kftlTemplateDAOSQLite3Impl) GetKFTLTemplates(ctx context.Context, userID string, device string) ([]*KFTLTemplate, error) {
	sql := `
SELECT 
  ID,
  USER_ID,
  DEVICE,
  TITLE,
  TEMPLATE,
  PARENT_FOLDER_ID,
  SEQ
FROM KFTL_TEMPLATE
WHERE USER_ID = ? AND DEVICE = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at get get kftl templates sql: %w", err)
		return nil, err
	}
	defer stmt.Close()

	log.Printf("%s, %s", userID, device)
	rows, err := stmt.QueryContext(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return nil, err
	}
	defer rows.Close()

	kftlTemplates := []*KFTLTemplate{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			kftlTemplate := &KFTLTemplate{}
			err = rows.Scan(
				&kftlTemplate.ID,
				&kftlTemplate.UserID,
				&kftlTemplate.Device,
				&kftlTemplate.Title,
				&kftlTemplate.Template,
				&kftlTemplate.ParentFolderID,
				&kftlTemplate.Seq,
			)
			kftlTemplates = append(kftlTemplates, kftlTemplate)
		}
	}
	return kftlTemplates, nil
}

func (k *kftlTemplateDAOSQLite3Impl) AddKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) (bool, error) {
	sql := `
INSERT INTO KFTL_TEMPLATE (
  ID,
  USER_ID,
  DEVICE,
  TITLE,
  TEMPLATE,
  PARENT_FOLDER_ID,
  SEQ
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
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at add device struct sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	log.Printf(
		"%s, %s, %s, %s, %s, %s, %s",
		kftlTemplate.ID,
		kftlTemplate.UserID,
		kftlTemplate.Device,
		kftlTemplate.Title,
		kftlTemplate.Template,
		kftlTemplate.ParentFolderID,
		kftlTemplate.Seq,
	)
	_, err = stmt.ExecContext(ctx,
		kftlTemplate.ID,
		kftlTemplate.UserID,
		kftlTemplate.Device,
		kftlTemplate.Title,
		kftlTemplate.Template,
		kftlTemplate.ParentFolderID,
		kftlTemplate.Seq,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (k *kftlTemplateDAOSQLite3Impl) AddKFTLTemplates(ctx context.Context, kftlTemplates []*KFTLTemplate) (bool, error) {
	tx, err := k.db.Begin()
	if err != nil {
		fmt.Errorf("error at begin: %w", err)
		return false, err
	}
	for _, kftlTemplate := range kftlTemplates {
		sql := `
INSERT INTO KFTL_TEMPLATE (
  ID,
  USER_ID,
  DEVICE,
  TITLE,
  TEMPLATE,
  PARENT_FOLDER_ID,
  SEQ
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
		stmt, err := tx.PrepareContext(ctx, sql)
		if err != nil {
			err = fmt.Errorf("error at add kftl template sql: %w", err)
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("%w: %w", err, rollbackErr)
			}
			return false, err
		}
		defer stmt.Close()

	log.Printf(
		"%s, %s, %s, %s, %s, %s, %s",
		kftlTemplate.ID,
		kftlTemplate.UserID,
		kftlTemplate.Device,
		kftlTemplate.Title,
		kftlTemplate.Template,
		kftlTemplate.ParentFolderID,
		kftlTemplate.Seq,
	)
		_, err = stmt.ExecContext(
			ctx,
			kftlTemplate.ID,
			kftlTemplate.UserID,
			kftlTemplate.Device,
			kftlTemplate.Title,
			kftlTemplate.Template,
			kftlTemplate.ParentFolderID,
			kftlTemplate.Seq,
		)
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

func (k *kftlTemplateDAOSQLite3Impl) UpdateKFTLTemplate(ctx context.Context, kftlTemplate *KFTLTemplate) (bool, error) {
	sql := `
UPDATE KFTL_TEMPLATE SET
  ID = ?,
  USER_ID = ?,
  DEVICE = ?,
  TITLE = ?,
  TEMPLATE = ?,
  PARENT_FOLDER_ID = ?,
  SEQ = ?
WHERE ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at update kftl template sql: %w", err)
		return false, err
	}
	defer stmt.Close()

log.Printf(
		"%s, %s, %s, %s, %s, %s, %s, %s",
		kftlTemplate.ID,
		kftlTemplate.UserID,
		kftlTemplate.Device,
		kftlTemplate.Title,
		kftlTemplate.Template,
		kftlTemplate.ParentFolderID,
		kftlTemplate.Seq,
		kftlTemplate.ID,
	)
	_, err = stmt.ExecContext(ctx,
		kftlTemplate.ID,
		kftlTemplate.UserID,
		kftlTemplate.Device,
		kftlTemplate.Title,
		kftlTemplate.Template,
		kftlTemplate.ParentFolderID,
		kftlTemplate.Seq,
		kftlTemplate.ID,
	)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (k *kftlTemplateDAOSQLite3Impl) DeleteKFTLTemplate(ctx context.Context, id string) (bool, error) {
	sql := `
DELETE FROM KFTL_TEMPLATE
WHERE ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete kftl template sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	log.Printf("%s", id)
	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (k *kftlTemplateDAOSQLite3Impl) DeleteUsersKFTLTemplates(ctx context.Context, userID string) (bool, error) {
	sql := `
DELETE FROM KFTL_TEMPLATE
WHERE USER_ID = ?
`
	log.Printf("sql: %s", sql)
	stmt, err := k.db.PrepareContext(ctx, sql)
	if err != nil {
		err = fmt.Errorf("error at delete kftl template sql: %w", err)
		return false, err
	}
	defer stmt.Close()

	log.Printf("%s", userID)
	_, err = stmt.ExecContext(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error at query :%w", err)
		return false, err
	}
	return true, nil
}

func (k *kftlTemplateDAOSQLite3Impl) Close(ctx context.Context) error {
	return k.db.Close()
}
