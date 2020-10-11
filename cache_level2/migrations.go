package cache_level2

import (
	"sort"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/migrations"
)

func (dbConn *CacheLevel2) RunMigrations(silent bool) {
	// Получаем список миграций
	mFiles := migrations.List.FileNames()

	// Сортируем их по порядку
	sort.Slice(*mFiles, func(i, j int) bool {
		return (*mFiles)[i] < (*mFiles)[j]
	})

	// Проверяем каждую миграцию на исполнение и если не исполнена - запускаем
	list, err := dbConn.ListMigrations()
	if err != nil {
		app.Error.Printf("Get processed migrations error: %s", err)
		dbConn.CreateMigrationsTable()
	}

	for _, mFile := range *mFiles {
		_, present := (*list)[mFile]
		if !present {
			if !silent {
				app.Info.Printf("Migration PROCESS: %s", mFile)
			}
			sql, ok := migrations.List.String("/migrations/"+mFile)
			if ok {
				err = dbConn.DoMigration(mFile, sql)
				if err != nil {
					app.Error.Fatalf("Migration '%s' error: %s", mFile, err)
				}
			}
		} else {
			if !silent {
				app.Info.Printf("Migration SKIP: %s", mFile)
			}
		}
	}
}


func (dbConn *CacheLevel2) CreateMigrationsTable() {
	dbConn.Do(`
		CREATE SCHEMA migrations;
		CREATE TABLE migrations.processed (id BIGSERIAL NOT NULL PRIMARY KEY, name VARCHAR(255), time TIMESTAMP DEFAULT NOW());
	`)
}

func (dbConn *CacheLevel2) DoMigration(name string, sql string) error {
	_, err := dbConn.Do(sql)
	if err != nil {
		return err
	}

	dbConn.Insert(`INSERT INTO migrations.processed (name) VALUES ($1)`, name)
	return nil
}

func (dbConn *CacheLevel2) ListMigrations() (*map[string]bool, error) {
	list := make(map[string]bool)
	rows, err := dbConn.Query(`SELECT name FROM migrations.processed;`)
	if err != nil {
		return &list, err
	}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		list[name] = true
	}
	rows.Close()

	return &list, nil
}
