package db

import (
	"database/sql"
	"github.com/pkg/errors"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"gitlab.com/stihi/stihi-backend/app"
)

const (
	EnvDbFileConfig = "DB_CONFIG"
)

var (
	dbSettings *Settings
)

type QueryProcessor interface {
	CheckConnect()
	Query(query string, params ...interface{}) (*sql.Rows, error)
	Insert(query string, params ...interface{}) (int64, error)
	Do(query string, params ...interface{}) (int64, error)
}

type Connection struct {
	Db          *sql.DB
}

type Transaction struct {
	Connection
	Transaction *sql.Tx
}

func InitSettings(cfg *Settings) {
	dbSettings = cfg
}

func New() *Connection {
	if dbSettings == nil {
		app.Error.Fatalf("DB config not initialized!!")
	}

	dbConn := &Connection{}
	err := dbConn.Connect()
	if err != nil {
		dbConn = nil
		app.Error.Printf("Error connect to DB: %s", err)
	}

	return dbConn
}

func (conn *Connection) Connect() error {
	app.Info.Println("DB: CONNECT")
	dbConnection := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbSettings.User,
		dbSettings.Password,
		dbSettings.Host,
		dbSettings.Port,
		dbSettings.DBName,
	)

	db, err := sql.Open("postgres", dbConnection)
	if err != nil {
		return err
	}

	conn.Db = db
	return nil
}

func (conn *Connection) Close() {
	if conn.Db == nil {
		return
	}

	conn.Db.Close()

	conn.Db = nil
}

func (conn *Connection) StartTransaction() (*Transaction, error) {
	conn.CheckConnect()

	trx, err := conn.Db.Begin()
	if err != nil {
		return nil, err
	}

	trans := Transaction{
		Connection: *conn,
		Transaction: trx,
	}

	return &trans, err
}

func (conn *Transaction) CommitTransaction() error {
	if conn.Transaction == nil {
		return errors.New("COMMIT TRANSACTION: Transaction not started.")
	}

	err := conn.Transaction.Commit()
	if err != nil {
		conn.Transaction = nil
		conn.CheckConnect()
		return err
	}

	conn.Transaction = nil
	return nil
}

func (conn *Transaction) RollbackTransaction() error {
	if conn.Transaction == nil {
		return errors.New("COMMIT TRANSACTION: Transaction not started.")
	}

	err := conn.Transaction.Rollback()
	if err != nil {
		conn.Transaction = nil
		conn.CheckConnect()
		return err
	}

	conn.Transaction = nil
	return nil
}

func (conn *Connection) CheckConnect() {
	oldDb := conn.Db

	if conn.Db == nil {
		conn.Connect()
	} else {
		err := conn.Db.Ping()

		if err != nil {
			app.Error.Println("DB ping error: ", err)
			conn.Db.Close()
			conn.Connect()
		}
	}

	if oldDb != nil && oldDb != conn.Db {
		oldDb.Close()
	}
}

func (conn *Connection) Query(query string, params ...interface{}) (*sql.Rows, error) {
	return conn.Db.Query(query, params...)
}


func (conn *Transaction) Query(query string, params ...interface{}) (*sql.Rows, error) {
	return conn.Transaction.Query(query, params...)
}

func (conn *Connection) Insert(query string, params ...interface{}) (int64, error) {
	if !strings.Contains(query, "RETURNING") {
		query = strings.TrimRight(query, ";")
		query = query + " RETURNING id"
	}

	rows, err := conn.Query(query, params...)
	if err != nil {
		conn.CheckConnect()
		return -1, err
	}
	defer rows.Close()
	if rows.Next() {
		var lastInsertId int64
		err = rows.Scan(&lastInsertId)
		if err != nil {
			return -1, err
		}

		return lastInsertId, nil
	}

	return -1, nil
}

func (conn *Transaction) Insert(query string, params ...interface{}) (int64, error) {
	if !strings.Contains(query, "RETURNING") {
		query = strings.TrimRight(query, ";")
		query = query + " RETURNING id"
	}

	rows, err := conn.Query(query, params...)
	if err != nil {
		conn.CheckConnect()
		return -1, err
	}
	defer rows.Close()
	if rows.Next() {
		var lastInsertId int64
		err = rows.Scan(&lastInsertId)
		if err != nil {
			return -1, err
		}

		return lastInsertId, nil
	}

	return -1, nil
}

func (conn *Connection) Do(query string, params ...interface{}) (int64, error) {
	res, err := conn.Db.Exec(query, params...)

	if err != nil {
		app.Error.Println(err)
		conn.CheckConnect()
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		app.Error.Println(err)
		conn.CheckConnect()
		return 0, err
	}

	return affected, nil
}

func (conn *Transaction) Do(query string, params ...interface{}) (int64, error) {
	res, err := conn.Transaction.Exec(query, params...)

	if err != nil {
		app.Error.Printf("Transaction exec error: %s\n%s\n--- Params:\n%+v", err, query, params)
		conn.CheckConnect()
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		app.Error.Println(err)
		conn.CheckConnect()
		return 0, err
	}

	return affected, nil
}

func ScanToMap(rows *sql.Rows) (*RowData, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columns := make([]interface{}, len(cols))
	columnPointers := make([]interface{}, len(cols))
	for i, _ := range columns {
		columnPointers[i] = &columns[i]
	}

	err = rows.Scan(columnPointers...)
	if err != nil {
		return nil, err
	}

	m := make(RowData)
	for i, colName := range cols {
		val := columnPointers[i].(*interface{})
		m[colName] = *val
	}

	return &m, nil
}

func GetOneRow(rows *sql.Rows) (*RowData, error) {
	var err error
	var cols *RowData

	if rows.Next() {
		cols, err = ScanToMap(rows)
		if err != nil {
			return nil, err
		}
	} else {
		err = errors.New("user data not found")
		return nil, err
	}

	return cols, nil
}
