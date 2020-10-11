package cache_level2

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/db"
	"gitlab.com/stihi/stihi-backend/app/mongodb"
)

const (
	PaginationModeFirst = 0 + iota
	PaginationModeAfter
	PaginationModeBefore

	FinanceSaveIndex 		= 1000000
	ContentSyncPeriod		= "10 minutes"
)

type PaginationMode int

type PaginationParams struct {
	Mode		PaginationMode
	Count 		int
	Id		 	int64
}

type Balance struct {
	Cyber		int64
	Golos		int64
	Power		int64
	Delegated	int64
	Received	int64
}

type CacheLevel2 struct {
	db.QueryProcessor
	mongo *mongodb.Connection
}

func (dbConn *CacheLevel2) StartTransaction() (*CacheLevel2, error) {
	switch dbConn.QueryProcessor.(type) {
	case *db.Connection:
		conn := dbConn.QueryProcessor.(*db.Connection)
		trans, err := conn.StartTransaction()
		if err != nil {
			return nil, err
		}

		newConn := CacheLevel2{
			QueryProcessor: trans,
		}

		return &newConn, nil
	}

	return nil, errors.New("bad type of dbConn (should by *db.Connection)")
}

func (dbConn *CacheLevel2) CommitTransaction() error {
	switch dbConn.QueryProcessor.(type) {
	case *db.Transaction:
		conn := dbConn.QueryProcessor.(*db.Transaction)
		return conn.CommitTransaction()
	}

	return errors.New("bad type of dbConn (should by *db.Transaction)")
}

func (dbConn *CacheLevel2) RollbackTransaction() error {
	switch dbConn.QueryProcessor.(type) {
	case *db.Transaction:
		conn := dbConn.QueryProcessor.(*db.Transaction)
		return conn.RollbackTransaction()
	}

	return errors.New("bad type of dbConn (should by *db.Transaction)")
}

func New(dbConfigFileName string, mongoConfigFileName string) (*CacheLevel2, error) {
	dbConn, mongoConn := Init(dbConfigFileName, mongoConfigFileName)
	err := dbConn.Connect()
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	if mongoConfigFileName != "" && mongoConn != nil {
		err = mongoConn.Connect()
		if err != nil {
			app.Error.Print(err)
			return nil, err
		}
	}

	cacheL2 := CacheLevel2{
		QueryProcessor: dbConn,
		mongo:          mongoConn,
	}

	return &cacheL2, nil
}

func Init(dbConfigFileName string, mongoConfigFileName string) (*db.Connection, *mongodb.Connection) {
	db.InitFromFile(dbConfigFileName)
	dbConn := db.New()

	var mongoConn *mongodb.Connection
	var err error
	if mongoConfigFileName != "" {
		mongodb.InitFromFile(mongoConfigFileName)
		mongoConn, err = mongodb.New()
		if err != nil {
			app.Error.Printf("Mongo error: %s", err)
		}
		err = mongoConn.Connect()
		if err != nil {
			app.Error.Printf("Mongo error: %s", err)
		}
		mongoConn.Client.Database(mongodb.Settings.DBName)
	}

	return dbConn, mongoConn
}

func (dbConn *CacheLevel2) GetTableCount(table string) (int64, error) {
	rows, err := dbConn.Query("SELECT COUNT(*) FROM "+table)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if !rows.Next() {
		return -1, errors.New("l10n:info.data_absent")
	}

	var count int64
	rows.Scan(
		&count,
	)

	return count, nil
}

func (dbConn *CacheLevel2) GetTableLastTime(table string) (*time.Time, error) {
	rows, err := dbConn.Query("SELECT MAX(time) FROM "+table)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("l10n:info.data_absent")
	}

	var ts time.Time
	rows.Scan(
		&ts,
	)

	return &ts, nil
}

func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid: true,
	}
}

func DetectPostgresError(err error) *pq.Error {
	pgError, ok := err.(*pq.Error)
	if ok {
		app.Error.Printf("POSTGRES ERROR: %+v", pgError)
		return pgError
	}
	return nil
}

func TPAddTime(val *map[string]int64, key string, delta time.Duration) {
	cur, ok := (*val)[key]
	if ok {
		(*val)[key] = cur + delta.Nanoseconds()
	} else {
		(*val)[key] = delta.Nanoseconds()
	}
}
