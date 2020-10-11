package cache_level2

import (
	"github.com/pkg/errors"
	"reflect"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
)

type MetaData map[string]interface{}

func (dbConn *CacheLevel2) SaveTagsFromOperation(jsonMetadata string, contentId int64) error {
	meta, err := ParseMeta(jsonMetadata)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	// Сначала удаляем все имеющиеся в БД тэги для данного контента
	_, err = dbConn.Do(`DELETE FROM content_tags WHERE content_id = $1`, contentId)

	for idx, tag := range meta.Tags() {
		isRubric := idx == 1
		_, err = dbConn.Do(
			`
				INSERT INTO content_tags
					(content_id, tag, is_rubric)
				VALUES
					($1, $2, $3)
				ON CONFLICT
					(content_id, tag)
				DO NOTHING
			`,
			contentId, tag, isRubric,
		)
		if err != nil {
			app.Error.Print(err)
			return err
		}
	}

	return nil
}

func (dbConn *CacheLevel2) GetTagsForContent(id int64) (*[]string, error) {
	tags := make([]string, 0)

	rows, err := dbConn.Query(
		"SELECT tag " +
			"FROM content_tags " +
			"WHERE " +
			"content_id = $1 " +
			"ORDER BY id",
		id,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		rows.Scan(&tag)

		tags = append(tags, tag)
	}

	return &tags, nil
}

func (dbConn *CacheLevel2) GetTagsForUser(userId int64) ([]string, error) {
	tags := make([]string, 0)

	userName, err := dbConn.GetUserNameById(userId)
	if err == nil && userName == "" {
		err = errors.New("l10n:info.data_absent")
	}
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	rows, err := dbConn.Query(
		"SELECT DISTINCT t.tag " +
			"FROM content_tags t, articles a " +
			"WHERE " +
			"a.author = $1 AND a.id = t.content_id",
		userName,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		rows.Scan(&tag)

		tags = append(tags, tag)
	}

	return tags, nil
}

func (dbConn *CacheLevel2) GetTagsCount() (int64, error) {
	return dbConn.GetTableCount("content_tags")
}

func (dbConn *CacheLevel2) GetTagsLastTime() (*time.Time, error) {
	return dbConn.GetTableLastTime("content_tags")
}

func ParseMeta(op string) (*MetaData, error) {
	meta := MetaData(ParseMetaToMap(op))

	return &meta, nil
}

func (meta *MetaData) Tags() []string {
	list := make([]string, 0)
	if meta == nil || (*meta)["tags"] == nil {
		return list
	}

	tags, ok := (*meta)["tags"]
	if !ok || tags == nil {
		return list
	}

	switch tags.(type) {
	case []string:
		ok = true
		for _, tag := range tags.([]string) {
			list = append(list, tag)
		}
	case []interface{}:
		ok = true
		for _, tag := range tags.([]interface{}) {
			switch tag.(type) {
			case string:
				list = append(list, tag.(string))
			default:
				app.Error.Printf("Wrong type of tag: %s\n%+v", reflect.TypeOf(tag), tag)
			}
		}
	default:
		app.Error.Printf("Tags error. Type is: %s", reflect.TypeOf(tags))
	}

	return list
}

func (meta *MetaData) IsTagPresent(tag string) bool {
	for _, mTag := range meta.Tags() {
		if tag == mTag {
			return true
		}
	}

	return false
}
