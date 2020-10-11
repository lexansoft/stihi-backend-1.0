package cache_level2

import (
	"database/sql"

	"gitlab.com/stihi/stihi-backend/app"
)

type Rubric struct {
	Level int		`json:"level"`
	Name string		`json:"name"`
	Tag string		`json:"tag"`
}

func (dbConn *CacheLevel2) GetRubrics() (*[]*Rubric, error) {
	list := make([]*Rubric, 0)
	rows, err := dbConn.Query(
		`
			SELECT
				parent_id, name, tag_name
			FROM rubrics
			ORDER BY id
		`,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	// Запускаем добавление только когда в списке прошли указанны lastArticle
	// И добавляем только нужное количество строк (он озапрошено с запасом)
	for rows.Next() {
		content := Rubric{}
		var parentId int64
		var tagName sql.NullString
		rows.Scan(
			&parentId,
			&content.Name,
			&tagName,
		)

		if tagName.Valid {
			content.Tag = tagName.String
		}

		if parentId == 0 {
			content.Level = 0
		} else {
			content.Level = 1
		}

		list = append(list, &content)

	}

	return &list, nil
}
