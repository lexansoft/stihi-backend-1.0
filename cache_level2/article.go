package cache_level2

import (
	"database/sql"
	"fmt"
	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"github.com/pkg/errors"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/db"
	"gitlab.com/stihi/stihi-backend/app/filters"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

type BodyContentType int

type OrderField struct {
	Name	string
	Desc	bool
}

type JoinLink struct {
	Table		string
	MainField	string
	JoinField	string
}

type NavigationArticlePoint struct {
	Id		int64	`json:"id"`
	Mat		bool	`json:"mat"`
}

const (
	ArticleCharsInPreview = 100
	ArticlesListFieldSQL = ` a.id, a.nodeos_id, a.author, a.permlink, a.title, a.body, a.image, a.last_comment_time, a.comments_count, 
		a.votes_count, a.votes_count_positive, a.votes_count_negative, a.votes_sum_positive, a.votes_sum_negative,
		a.time,	a.ban, a.val_cyber_10x6, a.val_golos_10x6, a.val_power_10x6, a.editor, u.id, u.name,
		u.val_cyber_10x6, u.val_golos_10x6, u.val_power_10x6, u.val_reputation, u.ban, ui.nickname, ui.birthdate,
		ui.sex, ui.place, ui.web_site, ui.avatar_image, ui.background_image, ui.pvt_posts_show_mode, ui.email `

	BodyContentNone BodyContentType	= iota
	BodyContentFull
	BodyContentPreview
	BodyContentRaw
)

// Сохраняем пост на основе данных из блокчейна
func (dbConn *CacheLevel2) SaveArticleFromOperation(op *operations.CreateMessageData, ts time.Time) (int64, error) {
	// Все проверки на уровне cache_level1

	// Получаем NodeosId для статьи
	nodeosId, err := dbConn.GetContentNodeosIdByPermlink(op.Id.Permlink)
	if err != nil {
		return -1, errors.Wrap(err, "SaveArticleFromOperation - can not get NodeosId by permlink")
	}

	// Извлекаем из метаданных ссылку на картинку статьи
	imageUrl := ""
	editor := ""
	meta, err := ParseMeta(op.JsonMetadata)
	if meta != nil && err == nil {
		switch (*meta)["image"].(type) {
		case string:
			imageUrl = (*meta)["image"].(string)
		case []interface{}:
			list := (*meta)["image"].([]interface{})
			if len(list) > 0 {
				imageUrl = list[0].(string)
			}
		}

		if (*meta)["editor"] != nil {
			editor = (*meta)["editor"].(string)
		}
	}

	mat := IsMatContent(meta)

	id, err := dbConn.Insert(`
			INSERT INTO articles
				(author, permlink, title, body, image, time, editor, mat, nodeos_id)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (
				author, permlink
			)
			DO UPDATE SET
				title = EXCLUDED.title,
				body = EXCLUDED.body,
				editor = EXCLUDED.editor,
				mat = EXCLUDED.mat,
				image = EXCLUDED.image,
				nodeos_id = EXCLUDED.nodeos_id
		`,
		op.Id.Author, op.Id.Permlink, op.Header, op.Body, imageUrl, ts, editor, mat, nodeosId)
	if err != nil {
		app.EmailErrorf(err.Error())
		return -1, err
	}

	err = dbConn.SaveTagsFromOperation(op.JsonMetadata, id)
	if err != nil {
		app.EmailErrorf(err.Error())
		return -1, err
	}

	return id, nil
}

// Устанавливаем для записи поста время последнего комментария
func (dbConn *CacheLevel2) SetArticleLastCommentTime(id int64, ts time.Time) {
	_, err := dbConn.Do(`
		UPDATE articles 
			SET last_comment_time = $1 
		WHERE 
			id = $2 AND 
			(last_comment_time IS NULL OR last_comment_time < $3)`, ts, id, ts)
	if err != nil {
		app.EmailErrorf("Update last_comment_time error: %s", err)
	}
}

// Устанавливаем для поста общее количество коментариев
func (dbConn *CacheLevel2) IncArticleCommentsCount(id int64) {
	_, _ = dbConn.Do(`UPDATE articles SET comments_count = comments_count + 1 WHERE id = $1`, id)
}

// Получение поста по имени автора и permlink
func (dbConn *CacheLevel2) GetArticleByGID(author string, permlink string, adminMode bool) (*Article, error) {
	rows, err := dbConn.Query(
		`SELECT id FROM articles WHERE author = $1 AND permlink = $2 AND NOT ban`,
		author, permlink,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	if !rows.Next() {
		rows.Close()
		return nil, nil
	}

	var id int64
	_ = rows.Scan(
		&id,
	)
	rows.Close()

	if id > 0 {
		return dbConn.GetArticle(id, false, adminMode)
	}

	return nil, nil
}

// Получение поста по ID
func (dbConn *CacheLevel2) GetArticle(id int64, rawFormat, adminMode bool) (*Article, error) {
	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}
	rows, err := dbConn.Query(
		`
			SELECT `+ArticlesListFieldSQL+`
			FROM articles a,
				 users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE a.id = $1 AND a.author = u.name `+sqlBan,
		id,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	bodyContentType := BodyContentFull
	if rawFormat {
		bodyContentType = BodyContentRaw
	}

	content, _ := prepareArticle(rows, bodyContentType)

	list := []*Article{ content	}
	err = dbConn.SetArticleTopCommentsCount(&list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	/*
	err = dbConn.SetArticleListRewardsValues(list)
	if err != nil {
		app.Error.Println(err)
	}
	*/

	return content, nil
}

func (dbConn *CacheLevel2) GetArticlePreview(id int64, adminMode bool) (*Article, error) {
	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}
	rows, err := dbConn.Query(
		`
			SELECT `+ArticlesListFieldSQL+`
			FROM articles a,
				 users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE a.id = $1 AND a.author = u.name `+sqlBan,
		id,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	bodyContentType := BodyContentPreview

	content, _ := prepareArticle(rows, bodyContentType)

	list := []*Article{ content	}
	err = dbConn.SetArticleTopCommentsCount(&list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	/*
	err = dbConn.SetArticleListRewardsValues(list)
	if err != nil {
		app.Error.Println(err)
	}
	*/

	return content, nil
}

func (dbConn *CacheLevel2) GetArticlesListByIds(ids *[]int64) (*[]*Article, error) {
	if ids == nil || len(*ids) <= 0 {
		return nil, errors.New("Bad params for GetArticlesListByIds")
	}

	list, err := prepareArticlesList(dbConn.Query(`
			SELECT `+ArticlesListFieldSQL+`
			FROM 
				articles a, 
				users u 
					LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id IN (`+
					strings.Trim(strings.Replace(fmt.Sprint(*ids), " ", ",", -1), "[]")+
				`) AND a.author = u.name 
		`))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}


func (dbConn *CacheLevel2) GetArticlesCount() (int64, error) {
	return dbConn.GetTableCount("articles")
}

func (dbConn *CacheLevel2) GetArticlesLastTime() (*time.Time, error) {
	return dbConn.GetTableLastTime("articles")
}

/*
	ARTICLES AFTER
*/

func (dbConn *CacheLevel2) GetArticlesAfter(lastArticle int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool, filter string) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
		if filter == "banned" {
			sqlBan = " AND a.ban "
		}
	}

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id < $1 AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
			LIMIT $2
		`,
		lastArticle, count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetFollowArticlesAfter(userId, lastArticle int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a
				INNER JOIN follows f ON f.user_id = $1 AND f.subscribed_for = a.author`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id < $2 AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
			LIMIT $3
		`,
		userId, lastArticle, count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetActualArticlesAfter(lastArticle int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	// Сначала получаем last_comment_time данной статьи
	var lastCommentTime time.Time
	rows, err := dbConn.Query(
		`SELECT last_comment_time FROM articles WHERE id = $1`,
		lastArticle,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&lastCommentTime)
	} else {
		rows.Close()
		return nil, nil
	}
	rows.Close()

	// Получаем статьи с last_comment_time раньше либо равным данному
	list := make([]*Article, 0)
	rows, err = dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				(a.last_comment_time <= $1 OR a.last_comment_time IS NULL) AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.last_comment_time desc NULLS LAST, a.comments_count desc, a.id
			LIMIT $2
		`,
		lastCommentTime, count+10,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	// Запускаем добавление только когда в списке прошли указанны lastArticle
	// И добавляем только нужное количество строк (он озапрошено с запасом)
	isAfter := false
	countAdded := 0
	for rows.Next() {
		content, _ := prepareArticle(rows, BodyContentPreview)

		if isAfter && countAdded < count {
			list = append(list, content)
			countAdded++
		}

		if content.Id == lastArticle {
			isAfter = true
		}
	}

	err = dbConn.SetArticleTopCommentsCount(&list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return &list, nil
}

func (dbConn *CacheLevel2) GetPopularArticlesAfter(lastArticle int64, count int, period int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	// Сначала получаем votes_sum_positive данной статьи
	var lastVotesSum int
	rows, err := dbConn.Query(
		`SELECT votes_sum_positive FROM articles WHERE id = $1`,
		lastArticle,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&lastVotesSum)
	} else {
		rows.Close()
		return nil, nil
	}
	rows.Close()

	// Получаем статьи с votes_sum_positive меньше либо равным данному
	list := make([]*Article, 0)
	interval := secondsInSQLInterval(period)
	rows, err = dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.votes_sum_positive <= $1 AND
				a.time >= (NOW() - $2::interval)
				AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.val_golos_10x6 DESC, a.id DESC
			LIMIT $3
		`,
		lastVotesSum, interval, count+10,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	// Запускаем добавление только когда в списке прошли указанны lastArticle
	// И добавляем только нужное количество строк (он озапрошено с запасом)
	isAfter := false
	countAdded := 0
	for rows.Next() {
		content, _ := prepareArticle(rows, BodyContentPreview)

		if isAfter && countAdded < count {
			list = append(list, content)
			countAdded++
		}

		if content.Id == lastArticle {
			isAfter = true
		}
	}

	err = dbConn.SetArticleTopCommentsCount(&list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return &list, nil
}

func (dbConn *CacheLevel2) GetBlogArticlesAfter(lastArticle int64, count int, userId int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		return nil, err
	}
	cyberName := dbConn.GetNodeosName(userName)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id < $1 AND
				a.author = $2
				AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
			LIMIT $3
		`,
		lastArticle, cyberName, count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

/*
	ARTICLES BEFORE
*/

// Выдает повледние (самые свежие статьи) после указанной
func (dbConn *CacheLevel2) GetArticlesBefore(firstArticle int64, tags []string, rubrics []string, notMat bool, adminMode bool, filter string) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
		if filter == "banned" {
			sqlBan = " AND a.ban "
		}
	}

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id > $1 AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
		`,
		firstArticle,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetFollowArticlesBefore(userId, firstArticle int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a
				INNER JOIN follows f ON f.user_id = $1 AND f.subscribed_for = a.author`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id > $2 AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
		`,
		userId, firstArticle,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}


func (dbConn *CacheLevel2) GetActualArticlesBefore(lastArticle int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	// Сначала получаем last_comment_time данной статьи
	var lastCommentTime time.Time
	rows, err := dbConn.Query(
		`SELECT last_comment_time FROM articles WHERE id = $1`,
		lastArticle,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&lastCommentTime)
	} else {
		rows.Close()
		return nil, nil
	}
	rows.Close()

	// Получаем статьи с last_comment_time раньше либо равным данному
	list := make([]*Article, 0)
	rows, err = dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.last_comment_time >= $1 AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.last_comment_time desc NULLS LAST, a.comments_count desc, a.id
		`,
		lastCommentTime,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	// Запускаем добавление только когда в списке прошли указанны lastArticle
	// И добавляем только нужное количество строк (он озапрошено с запасом)
	for rows.Next() {
		content, _ := prepareArticle(rows, BodyContentPreview)

		if content.Id == lastArticle {
			break
		}

		list = append(list, content)
	}

	err = dbConn.SetArticleTopCommentsCount(&list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return &list, nil
}

func (dbConn *CacheLevel2) GetPopularArticlesBefore(lastArticle, period int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	// Сначала получаем last_comment_time данной статьи
	var lastVotesSum int
	rows, err := dbConn.Query(
		`SELECT votes_sum_positive FROM articles WHERE id = $1`,
		lastArticle,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&lastVotesSum)
	} else {
		rows.Close()
		return nil, errors.New("article not exists")
	}
	rows.Close()

	// Получаем статьи с votes_sum_positive больше либо равным данному
	list := make([]*Article, 0)
	interval := secondsInSQLInterval(period)
	rows, err = dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
				a.id, a.author, a.permlink, a.title, a.body, 
				a.image, a.last_comment_time, a.comments_count, 
				a.votes_count, a.votes_count_positive, a.votes_count_negative, a.votes_sum_positive, a.votes_sum_negative,
				a.time,
				ui.avatar_image, ui.nickname, u.id, a.ban, u.ban, a.val_cyber_10x6, u.val_reputation, a.val_golos_10x6, a.val_power_10x6
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.votes_sum_positive >= $1 AND
				a.time >= (NOW() - $2::interval)
				AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.val_golos_10x6 DESC, a.id DESC
		`,
		lastVotesSum, interval,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	// Запускаем добавление только когда в списке прошли указанны lastArticle
	// И добавляем только нужное количество строк (он озапрошено с запасом)
	for rows.Next() {
		content, _ := prepareArticle(rows, BodyContentPreview)

		if content.Id == lastArticle {
			break
		}

		list = append(list, content)

	}

	err = dbConn.SetArticleTopCommentsCount(&list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return &list, nil
}

func (dbConn *CacheLevel2) GetBlogArticlesBefore(firstArticle int64, userId int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	cyberName := dbConn.GetNodeosName(userName)

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.id > $1 AND
				a.author = $2
				AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
		`,
		firstArticle, cyberName,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}


/*
	LAST ARTICLES
*/

func (dbConn *CacheLevel2) GetLastArticles(count int, tags []string, rubrics []string, notMat bool, adminMode bool, filter string) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
		if filter == "banned" {
			sqlBan = " AND a.ban "
		}
	}

	sql := `
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
		ignoreTagsJoin+
		tagTable+
		rubricsTable+
		`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE 
				a.author = u.name `+sqlBan+
		tagWhere+
		rubricsWhere+
		ignoreTagsWhere+
		`ORDER BY a.id desc
			LIMIT $1
		`

	// app.Debug.Printf("GetLastArticles SQL:\n%s\n", sql)

	list, err := prepareArticlesList(dbConn.Query(
		sql,
		count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetFollowLastArticles(userId int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a
				INNER JOIN follows f ON f.user_id = $1 AND f.subscribed_for = a.author`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
			LIMIT $2
		`,
		userId, count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetActualLastArticles(count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.last_comment_time desc NULLS LAST, a.comments_count desc, a.id
			LIMIT $1
		`,
		count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetPopularLastArticles(count int, period int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	interval := secondsInSQLInterval(period)
	query := `
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
		ignoreTagsJoin+
		tagTable+
		rubricsTable+
		`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.time >= (NOW() - $1::interval) AND a.author = u.name `+sqlBan+
		tagWhere+
		rubricsWhere+
		ignoreTagsWhere+
		`ORDER BY a.val_golos_10x6 DESC, a.id DESC
			LIMIT $2
		`
	list, err := prepareArticlesList(dbConn.Query(
		query,
		interval, count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

func (dbConn *CacheLevel2) GetBlogLastArticles(count int, userId int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*Article, error) {
	tagTable, tagWhere, tagDistinct := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		return nil, err
	}
	cyberName := dbConn.GetNodeosName(userName)

	list, err := prepareArticlesList(dbConn.Query(
		`
			SELECT `+tagDistinct+` `+ArticlesListFieldSQL+`
			FROM articles a`+
			ignoreTagsJoin+
			tagTable+
			rubricsTable+
			`, users u LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE
				a.author = $1 AND a.author = u.name `+sqlBan+
			tagWhere+
			rubricsWhere+
			ignoreTagsWhere+
			`ORDER BY a.id desc
			LIMIT $2
		`,
		cyberName, count,
	))
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}

	err = dbConn.SetArticleTopCommentsCount(list)
	if err != nil {
		app.Error.Println(err)
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return list, nil
}

/*
Метод возвращает id предыдущей и следующей статьи относительно текущей
Параметры:
  articleId - id статьи относительно которой производится поиск
  orderField - имя поля, по которому производится сортировка
  orderDesc - если true, сортировка по убыванию, если false - по возрастанию
  filterList - условия фильтрации списка
  tags - вывод по определенным тэгам
  ignoreTags - список тэгов, контент с коротыми необходиммо игнорировать
  adminMode - если true - выводить в том числе и заблокированные статьи или статьи заблокированных пользователей
Возвращает:
  prevArticleId
  nextArticleId
  error
*/
// TODO: Сделать версию с передачей сортировки через параметр типа []OrderField
func (dbConn *CacheLevel2) GetArticlePrevNext(articleId int64, orderField string, orderDesc bool, join []JoinLink, filterList map[string]interface{}, tags []string, rubrics []string, notMat bool, adminMode bool) (*NavigationArticlePoint, *NavigationArticlePoint, error) {
	// Защищаем параметры от SQL-injection
	orderField = db.Escaped(orderField)

	// Получаем текущее значение поля сортировки для текущей статьи
	var orderFieldCurrent interface{}
	rows, err := dbConn.Query("SELECT "+orderField+" FROM articles a WHERE a.id = $1", articleId)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, nil, err
	}
	if rows.Next() {
		err = rows.Scan(&orderFieldCurrent)
		if err != nil {
			app.EmailErrorf(err.Error())
			rows.Close()
			return nil, nil, err
		}
	} else {
		rows.Close()
		return nil, nil, errors.New("can not find article")
	}
	rows.Close()

	// Формируем условие сортировки
	orderNext := ""
	orderPrev := " desc"
	compareOpNext := ">"
	compareOpPrev := "<"
	if orderDesc {
		orderNext = " desc"
		orderPrev = ""
		compareOpNext = "<"
		compareOpPrev = ">"
	}
	sortNext := orderField + orderNext + " NULLS LAST"
	sortPrev := orderField + orderPrev + " NULLS LAST"

	// Формируем условие WHERE и параметры для него
	tagTable, tagWhere, _ := prepareTagsSQL(tags, false)
	rubricsTable, rubricsWhere := prepareRubricsSQL(rubrics, false)
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)
	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	paramsIdx := int64(1)
	params := make([]interface{}, 0)
	where := " WHERE a.author = u.name "+tagWhere+rubricsWhere+ignoreTagsWhere+sqlBan
	for key, val := range filterList {
		where += " AND "+db.Escaped(key)+" = $"+strconv.FormatInt(paramsIdx, 10)
		params = append(params, val)
		paramsIdx++
	}

	// Формируем FROM
	from := "articles a"+ignoreTagsJoin+tagTable+rubricsTable+", users u LEFT JOIN users_info ui ON u.id = ui.user_id"

	// Формируем SQL для нахождения следующей и предыдущей статьи
	queryNext := `
		SELECT a.id, a.mat, `+orderField+`
		FROM `+from+
		where+` AND `+orderField+" "+compareOpNext+" "+"$"+strconv.FormatInt(paramsIdx, 10)+`
		ORDER BY `+sortNext+`
		LIMIT 1`
	queryPrev := `
		SELECT a.id, a.mat, `+orderField+`
		FROM `+from+
		where+` AND `+orderField+" "+compareOpPrev+" "+"$"+strconv.FormatInt(paramsIdx, 10)+`
		ORDER BY `+sortPrev+`
		LIMIT 1`
	params = append(params, orderFieldCurrent)
	paramsIdx++

	// Находим следующую статью
	nextPoint := &NavigationArticlePoint{}

	// fmt.Printf("NEXT SQL: %s\n%+v\n", queryNext, params)
	rows, err = dbConn.Query(queryNext, params...)
	if err != nil {
		app.EmailErrorf("Error: %s; Query: %s\n", err, queryNext)
		return nil, nil, err
	}
	if rows.Next() {
		var sortField interface{};
		err = rows.Scan(
			&nextPoint.Id,
			&nextPoint.Mat,
			&sortField,
		)
		if err != nil {
			app.Error.Println(err)
			rows.Close()
			return nil, nil, err
		}
	}
	rows.Close()

	// Находим предыдущую статью
	prevPoint := &NavigationArticlePoint{}

	// fmt.Printf("PREV SQL: %s\n%+v\n", queryPrev, params)
	rows, err = dbConn.Query(queryPrev, params...)
	if err != nil {
		app.EmailErrorf("Error: %s; Query: %s\n", err, queryPrev)
		return nil, nil, err
	}
	if rows.Next() {
		var sortField interface{};
		err = rows.Scan(
			&prevPoint.Id,
			&prevPoint.Mat,
			&sortField,
		)
		if err != nil {
			app.Error.Println(err)
			rows.Close()
			return nil, nil, err
		}
	}
	rows.Close()

	return prevPoint, nextPoint, nil
}

func (dbConn *CacheLevel2) SetArticleTopCommentsCount(list *[]*Article) error {
	if list == nil || len(*list) <= 0 {
		return nil
	}

	// Карта соответствия id статьи количеству top-комментов
	idsMap := make(map[int64]int)

	// Формируем список parentAuthor и parentPermlink
	ids := make([]int64, 0)
	for _, article := range *list {
		ids = append(ids, article.Id)
	}

	query := `
		SELECT 
			cp.id, count(*) as cnt
		FROM
			comments c,
			content cp
		WHERE
			cp.id IN (`+
				strings.Trim(strings.Replace(fmt.Sprint(ids), " ", ",", -1), "[]")+
			`) 
			AND cp.author = c.parent_author AND cp.permlink = c.parent_permlink
		GROUP BY
			cp.id
	`
	rows, err := dbConn.Query(query)
	if err != nil {
		app.EmailErrorf("Error: %s; Query: %s\n", err, query)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var artId int64
		var cnt int

		err = rows.Scan(
			&artId,
			&cnt,
		)
		if err != nil {
			app.EmailErrorf("Rows Scan Error: %s\n", err)
			return err
		}

		idsMap[artId] = cnt
	}

	// Заполняем данные в списке
	for _, article := range *list {
		article.TopCommentsCount = idsMap[article.Id]
	}

	// Устанавливаем для поста суммы заработанного ею из базы mongodb
	//err = dbConn.SetArticleListRewardsValues(*list)
	//if err != nil {
	//	app.Error.Println(err)
	//}

	return nil
}

func prepareArticlesList(rows *sql.Rows, err error) (*[]*Article, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Article, 0)
	for rows.Next() {
		content, _ := prepareArticle(rows, BodyContentPreview)
		list = append(list, content)
	}

	return &list, nil
}

func prepareArticle(rows *sql.Rows, bodyType BodyContentType) (*Article, error) {
	content := Article{}
	content.User = UserInfo{}
	var nodeosId sql.NullInt64
	var body string
	var valImage sql.NullString
	var valEditor sql.NullString
	var valTime NullTime
	var valLastCommentTime NullTime
	var valNickname sql.NullString
	var valBirthDate NullTime
	var valSex sql.NullString
	var valPlace sql.NullString
	var valWebSite sql.NullString
	var valAvatar sql.NullString
	var valBackgroundImage sql.NullString
	var valPvtPostsShowMode sql.NullString
	var valEmail sql.NullString

	var valCyber int64
	var valGolos int64
	var valPower int64

	var valUserCyber int64
	var valUserGolos int64
	var valUserPower int64

	err := rows.Scan(
		&content.Id,
		&nodeosId,
		&content.Author,
		&content.Permlink,
		&content.Title,
		&body,
		&valImage,
		&valLastCommentTime,
		&content.CommentsCount,
		&content.VotesCount,
		&content.VotesCountPositive,
		&content.VotesCountNegative,
		&content.VotesSumPositive,
		&content.VotesSumNegative,
		&valTime,
		&content.Ban,
		&valCyber,
		&valGolos,
		&valPower,
		&valEditor,
		&content.User.Id,
		&content.User.Name,
		&valUserCyber,
		&valUserGolos,
		&valUserPower,
		&content.User.ValReputation,
		&content.User.Ban,
		&valNickname,
		&valBirthDate,
		&valSex,
		&valPlace,
		&valWebSite,
		&valAvatar,
		&valBackgroundImage,
		&valPvtPostsShowMode,
		&valEmail,
	)
	if err != nil {
		app.EmailErrorf("Error when scan data row: %s", err)
		debug.PrintStack()
	}

	content.ValCyber = float64(valCyber) / FinanceSaveIndex
	content.ValGolos = float64(valGolos) / FinanceSaveIndex
	content.ValPower = float64(valPower) / FinanceSaveIndex
	content.User.ValCyber = float64(valUserCyber) / FinanceSaveIndex
	content.User.ValGolos = float64(valUserGolos) / FinanceSaveIndex
	content.User.ValPower = float64(valUserPower) / FinanceSaveIndex

	if nodeosId.Valid {
		content.NodeosId = nodeosId.Int64
	}

	if valImage.Valid {
		content.Image = valImage.String
	}

	if valEditor.Valid {
		content.Editor = valEditor.String
	}

	if valTime.Valid {
		content.Time = valTime.Format()
	}
	if valLastCommentTime.Valid {
		content.LastCommentTime = valLastCommentTime.Format()
	}

	if valNickname.Valid {
		content.User.NickName = valNickname.String
	}
	if valBirthDate.Valid {
		content.User.BirthDate = valBirthDate.Format()
	}
	if valSex.Valid {
		content.User.Sex = valSex.String
	}
	if valPlace.Valid {
		content.User.Place = valPlace.String
	}
	if valWebSite.Valid {
		content.User.WebSite = valWebSite.String
	}
	if valAvatar.Valid {
		content.User.AvatarImage = valAvatar.String
	}
	if valBackgroundImage.Valid {
		content.User.BackgroundImage = valBackgroundImage.String
	}
	if valPvtPostsShowMode.Valid {
		content.User.PvtPostsShowMode = valPvtPostsShowMode.String
	}
	if valEmail.Valid {
		content.User.Email = valEmail.String
	}

	switch bodyType {
	case BodyContentFull:
		content.Body = filters.HTMLBodyFilter(body)
	case BodyContentPreview:
		content.Body = filters.HTMLPreviewFilter(body, ArticleCharsInPreview)
	case BodyContentNone:
		content.Body = ""
	default:
		content.Body = body
	}

	return &content, nil
}

func prepareTagsSQL(tags []string, withWhere bool) (string, string, string) {
	if tags == nil || len(tags) == 0 {
		return " ", " ", " "
	}

	tagsList := ""
	sep := ""
	for _, tag := range tags {
		// Защита от SQL Injection
		tag = strings.Replace(tag, "'", "", -1)
		tag = translit.EncodeTag(tag)

		tagsList += sep+"'"+tag+"'"
		sep = ","
	}

	where := " AND "
	if withWhere {
		where = " WHERE "
	}

	return ", content_tags t ", where + "a.id = t.content_id AND t.tag IN ("+tagsList+") ", " DISTINCT "
}

func prepareIgnoreTagsSQL(notMat bool, withWhere bool) (string, string) {
	if !notMat {
		return " ", " "
	}

	where := " AND "
	if withWhere {
		where = " WHERE "
	}

	return " ", where + " NOT a.mat "
}

func prepareRubricsSQL(tags []string, withWhere bool) (string, string) {
	if tags == nil || len(tags) == 0 {
		return " ", " "
	}

	tagsList := ""
	sep := ""
	for _, tag := range tags {
		// Защита от SQL Injection
		tag = strings.Replace(tag, "'", "", -1)
		tag = translit.EncodeTag(tag)

		tagsList += sep+"'"+tag+"'"
		sep = ","
	}

	where := " AND "
	if withWhere {
		where = " WHERE "
	}

	return ", content_tags r ", where + "a.id = r.content_id AND r.tag IN ("+tagsList+") AND r.is_rubric"
}

// Из базы mongodb берем суммы заработанного статьей и устанавливаем их в записях нашей базы
/*
func (dbConn *CacheLevel2) SetArticleListRewardsValues(list []*Article) error {
	dbConn.mongo.Check()

	dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("message")

	for _, art := range list {
		if art.NodeosId > 0 {
			_ = dbConn.SetArticleRewardsValues(art)
		}
	}

	return nil
}

func (dbConn *CacheLevel2) SetArticleRewardsValues(art *Article) error {
	dbConn.mongo.Check()

	art.ValGolos, _ = dbConn.GetContentRewardNodeos(art.NodeosId)
	art.ValCyber = 0
	art.ValPower = 0

	return nil
}
*/

// Cron функция обновления заработанных сумм для статей в полях нашей БД
func (dbConn *CacheLevel2) SyncArticles() {
	app.Info.Println("SyncArticles start...")
	defer app.Info.Println("SyncArticles done")

	ids := make(map[int64]int64)

	// Выбираем статьи, у которых last_sync_time раньше чем нужно
	rows, err := dbConn.Query(
		`SELECT id, nodeos_id FROM articles WHERE last_sync_time iS NULL OR last_sync_time < (NOW() - $1::INTERVAL)`,
		ContentSyncPeriod,
	)
	if err != nil {
		app.Error.Printf("Error select articles for sync: %s", err)
		return
	}

	for rows.Next() {
		var id int64
		var nodeosIdVal sql.NullInt64
		err = rows.Scan(
			&id,
			&nodeosIdVal,
		)
		if err != nil {
			rows.Close()
			app.Error.Printf("Error scan articles for sync: %s", err)
			return
		}

		if nodeosIdVal.Valid {
			ids[nodeosIdVal.Int64] = id
		}
	}
	rows.Close()

	// Получаем в цикле и обновляем:
	// - суммы заработанного статьей из базы MongoDB
	// - количество и суммы голосов
	for nodeosId, id := range ids {
		_ = dbConn.SyncArticle(id, nodeosId)
	}
}

func (dbConn *CacheLevel2) SyncArticle(id, nodeosId int64) error {
	return dbConn.SyncContent(id, nodeosId)
}
