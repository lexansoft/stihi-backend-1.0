package cache_level2

import (
	"gitlab.com/stihi/stihi-backend/app"
)

func (dbConn *CacheLevel2) SaveState(id string, value string) error {
	affected, err := dbConn.Do(`UPDATE states SET value = $1 WHERE id = $2`, value, id )
	if affected == 1 && err == nil {
		return nil
	}

	_, err = dbConn.Do(`INSERT INTO states (id, value) VALUES ($1, $2)`, id, value)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) GetState(id string) (string, error) {
	rows, err := dbConn.Query(`SELECT value FROM states WHERE id = $1`, id)
	if err != nil {
		app.Error.Print(err)
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		return "", nil
	}

	var val string
	rows.Scan(
		&val,
	)

	return val, nil
}
