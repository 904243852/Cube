package module

import (
	"context"
	"database/sql"
)

func init() {
	register("db", func(worker Worker, db Db) interface{} {
		return &DatabaseClient{worker, db}
	})
}

type DatabaseTransaction struct {
	Transaction *sql.Tx
}

func ExportDatabaseRows(rows *sql.Rows) ([]interface{}, error) {
	defer rows.Close()

	columns, _ := rows.Columns()
	buf := make([]interface{}, len(columns))
	for index := range columns {
		var a interface{}
		buf[index] = &a
	}

	var records []interface{}

	for rows.Next() {
		_ = rows.Scan(buf...)

		record := make(map[string]interface{})
		for index, data := range buf {
			record[columns[index]] = *data.(*interface{})
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (d *DatabaseTransaction) Query(stmt string, params ...interface{}) ([]interface{}, error) {
	rows, err := d.Transaction.Query(stmt, params...)
	if err != nil {
		return nil, err
	}
	return ExportDatabaseRows(rows)
}

func (d *DatabaseTransaction) Exec(stmt string, params ...interface{}) (int64, error) {
	res, err := d.Transaction.Exec(stmt, params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DatabaseTransaction) Commit() error {
	return d.Transaction.Commit()
}

func (d *DatabaseTransaction) Rollback() error {
	return d.Transaction.Rollback()
}

type DatabaseClient struct {
	worker Worker
	db     Db
}

func (d *DatabaseClient) BeginTx() (*DatabaseTransaction, error) {
	if tx, err := d.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted}); err != nil { // 开启一个新事务，隔离级别为读已提交
		return nil, err
	} else {
		d.worker.AddHandle(tx)
		return &DatabaseTransaction{
			Transaction: tx,
		}, nil
	}
}

func (d *DatabaseClient) Query(stmt string, params ...interface{}) ([]interface{}, error) {
	rows, err := d.db.Query(stmt, params...)
	if err != nil {
		return nil, err
	}
	return ExportDatabaseRows(rows)
}

func (d *DatabaseClient) Exec(stmt string, params ...interface{}) (int64, error) {
	res, err := d.db.Exec(stmt, params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
