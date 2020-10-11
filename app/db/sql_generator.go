package db

import (
	"strings"
)

func (conn *Connection) InsertFromMap(table string, data *RowData) (int64, error) {
	count := data.Len()

	columns 	:= make([]string, count)
	values 		:= make([]string, count)
	params 		:= make([]interface{}, count)

	i := 0
	for k, v := range *data.Raw() {
		columns[i] = k
		values[i] = "?"
		params[i] = v

		i++
	}

	query := "INSERT INTO "+table+" ("+strings.Join(columns, ", ")+") VALUES ("+strings.Join(values, ", ")+")"

	return conn.Insert(query, params...)
}

func (conn *Connection) UpdateFromMap(table string, id int64, data *RowData) (int64, error) {
	count := data.Len()

	columns 	:= make([]string, count)
	params 		:= make([]interface{}, count+1)

	i := 0
	for k, v := range *data.Raw() {
		columns[i] = k+" = ?"
		params[i] = v

		i++
	}
	params[i] = id

	query := "UPDATE "+table+" "+strings.Join(columns, ", ")+" WHERE id = ?"

	return conn.Do(query, params...)
}
