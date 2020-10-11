package cache_level2

import (
	"database/sql"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/filters"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

const (
	CommentsListFieldSQL = ` c.id, c.nodeos_id, cp.id, c.parent_author, c.parent_permlink, c.author, c.permlink, c.title, c.body,
		c.votes_count, c.votes_count_positive, c.votes_count_negative, c.votes_sum_positive, c.votes_sum_negative,
		c.time, c.ban, c.level, c.val_cyber_10x6, c.val_golos_10x6, c.val_power_10x6, c.editor,
		u.id, u.name,
		u.val_cyber_10x6, u.val_golos_10x6, u.val_power_10x6, u.val_reputation, u.ban, ui.nickname, ui.birthdate,
		ui.sex, ui.place, ui.web_site, ui.avatar_image, ui.background_image, ui.pvt_posts_show_mode, ui.email `
	CommentsListFieldSQLRecursiveFields = ` id, nodeos_id, parent_id, parent_author, parent_permlink, author, permlink, title, body,
		votes_count, votes_count_positive, votes_count_negative, votes_sum_positive, votes_sum_negative,
		time, ban, level, val_cyber_10x6, val_golos_10x6, val_power_10x6, editor,
		u_id, u_name,
		u_val_cyber_10x6, u_val_golos_10x6, u_val_power_10x6, u_val_reputation, u_ban, ui_nickname, ui_birthdate,
		ui_sex, ui_place, ui_web_site, ui_avatar_image, ui_background_image, ui_pvt_posts_show_mode, 
		ui_email `
	CommentsListFieldSQLRecursiveFields1 = ` c1.id, c1.nodeos_id, a1.id, c1.parent_author, c1.parent_permlink, c1.author, c1.permlink, c1.title, c1.body, 
		c1.votes_count, c1.votes_count_positive, c1.votes_count_negative, c1.votes_sum_positive, c1.votes_sum_negative,
		c1.time, c1.ban, c1.level, c1.val_cyber_10x6, c1.val_golos_10x6, c1.val_power_10x6, c1.editor,
		u1.id, u1.name,
		u1.val_cyber_10x6, u1.val_golos_10x6, u1.val_power_10x6, u1.val_reputation, u1.ban, ui1.nickname, ui1.birthdate,
		ui1.sex, ui1.place, ui1.web_site, ui1.avatar_image, ui1.background_image, ui1.pvt_posts_show_mode, ui1.email `
	CommentsListFieldSQLRecursiveFields2 = ` c2.id, c2.nodeos_id, c3.id, c2.parent_author, c2.parent_permlink, c2.author, c2.permlink, c2.title, c2.body, 
		c2.votes_count, c2.votes_count_positive, c2.votes_count_negative, c2.votes_sum_positive, c2.votes_sum_negative,
		c2.time, c2.ban, c2.level, c2.val_cyber_10x6, c2.val_golos_10x6, c2.val_power_10x6, c2.editor,
		u2.id, u2.name, 
		u2.val_cyber_10x6, u2.val_golos_10x6, u2.val_power_10x6, u2.val_reputation, u2.ban, ui2.nickname, ui2.birthdate,
		ui2.sex, ui2.place, ui2.web_site, ui2.avatar_image, ui2.background_image, ui2.pvt_posts_show_mode, ui2.email `
	AllCommentsListFieldSQL = ` c1.id, c1.nodeos_id, a1.id, c1.parent_author, c1.parent_permlink, c1.author, c1.permlink, c1.title, c1.body, 
		c1.votes_count, c1.votes_count_positive, c1.votes_count_negative, c1.votes_sum_positive, c1.votes_sum_negative,
		c1.time, c1.ban, c1.level, c1.val_cyber_10x6, c1.val_golos_10x6, c1.val_power_10x6, c1.editor,
		u1.id, u1.name, 
		u1.val_cyber_10x6, u1.val_golos_10x6, u1.val_power_10x6, u1.val_reputation, u1.ban, ui1.nickname, ui1.birthdate,
		ui1.sex, ui1.place, ui1.web_site, ui1.avatar_image, ui1.background_image, ui1.pvt_posts_show_mode, ui1.email `
)

func (dbConn *CacheLevel2) SaveCommentFromOperation(op *operations.CreateMessageData, ts time.Time, isInsert bool) (int64, error) {
	// Все проверки на уровне cache_level1

	// Получаем NodeosId для комментария
	nodeosId, err := dbConn.GetContentNodeosIdByPermlink(op.Id.Permlink)
	if err != nil {
		return -1, errors.Wrap(err, "SaveCommentFromOperation - can not get NodeosId by permlink")
	}

	// Получаем level parent контента
	parentLevel, _ := dbConn.GetContentLevel(op.ParentId.Author, op.ParentId.Permlink)
	if parentLevel < 0 {
		return -1, errors.New("parent absent")
	}

	editor := ""
	meta, err := ParseMeta(op.JsonMetadata)
	if err == nil && (*meta)["editor"] != nil {
		editor = (*meta)["editor"].(string)
	}

	id, err := dbConn.Insert(`
		INSERT INTO comments
			(parent_author, parent_permlink, author, permlink, title, body, time, level, editor, nodeos_id)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (
			author, permlink
		)
		DO UPDATE SET
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			editor = EXCLUDED.editor,
			nodeos_id = EXCLUDED.nodeos_id
		`,
		op.ParentId.Author, op.ParentId.Permlink, op.Id.Author, op.Id.Permlink, op.Header, op.Body, ts, parentLevel + 1,
		editor, nodeosId)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	// Собираем вспомагательную информацию по комментариям для статьи
	artId, err := dbConn.GetArticleForComment(id, true)
	app.Debug.Printf("DBG: Article for comment: %d\n", artId)
	if err == nil {
		app.Debug.Printf("DBG: SetArticleLastCommentTime for article %d\n", artId)
		dbConn.SetArticleLastCommentTime(artId, ts)
		if isInsert {
			app.Debug.Printf("DBG: IncArticleCommentsCount for article %d\n", artId)
			dbConn.IncArticleCommentsCount(artId)
		}
	} else {
		app.Error.Printf("GetArticleForComment error: %s", err)
	}

	return id, nil
}

func (dbConn *CacheLevel2) GetComment(id int64, rawFormat, adminMode bool) (*Comment, error) {
	sqlBan := " AND NOT c.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	rows, err := dbConn.Query(`
			SELECT `+CommentsListFieldSQL+`
			FROM 
				comments c, 
				content cp, 
				users u
					LEFT JOIN users_info ui ON ui.user_id = u.id
			WHERE
				c.id = $1 AND c.author = u.name AND cp.author = c.parent_author AND cp.permlink = c.parent_permlink `+sqlBan,
		id,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	content, _ := prepareComment(rows, rawFormat)

	// Устанавливаем для коммента суммы заработанного из базы mongodb
	err = dbConn.SetCommentRewardsValues(content)
	if err != nil {
		app.Error.Println(err)
	}

	return content, nil
}

func (dbConn *CacheLevel2) GetCommentsCount() (int64, error) {
	return dbConn.GetTableCount("comments")
}

func (dbConn *CacheLevel2) GetCommentsLastTime() (*time.Time, error) {
	return dbConn.GetTableLastTime("comments")
}

func (dbConn *CacheLevel2) GetCommentsForContentFull(contentId int64, adminMode bool) (*[]*Comment, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	listById := make(map[int64]*Comment)

	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	sqlBan2 := " AND NOT c2.ban AND NOT u2.ban "
	if adminMode {
		sqlBan1 = " "
		sqlBan2 = " "
	}

	// Рекурсивный SQL запрос
	rows, err := dbConn.Query(`
			WITH RECURSIVE all_comments (`+CommentsListFieldSQLRecursiveFields+`) 
			AS (
				SELECT 	`+CommentsListFieldSQLRecursiveFields1+`
					FROM 
						comments c1
							LEFT JOIN content a1 ON a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink,
						users u1
							LEFT JOIN users_info ui1 ON ui1.user_id = u1.id
					WHERE a1.id = $1 AND c1.author = u1.name `+sqlBan1+`
				UNION
				SELECT 	`+CommentsListFieldSQLRecursiveFields2+`
                	FROM 
						comments c2
							LEFT JOIN comments c3 ON c3.author = c2.parent_author AND c3.permlink = c2.parent_permlink
							INNER JOIN all_comments ON c2.parent_author = all_comments.author AND c2.parent_permlink = all_comments.permlink,
						users u2
							LEFT JOIN users_info ui2 ON ui2.user_id = u2.id
					WHERE c2.author = u2.name `+sqlBan2+`
			)
			
			SELECT * FROM all_comments ORDER BY level, time DESC
		`,
		contentId,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		// Формируем список id
		ids = append(ids, content.NodeosId)

		// Сразу строим дерево объектов
		listById[content.Id] = content

		if content.ParentId == contentId {
			// Если это верхний уровень - просто добавляем в список
			list = append(list, content)
		} else {
			// Если ниже - находим запомненный parent объект и добавляем в его список Comments
			parent := listById[content.ParentId]
			if parent != nil {
				if parent.Comments == nil {
					parent.Comments = make([]*Comment, 0)
				}
				parent.Comments = append(parent.Comments, content)
			}
		}
	}

	return &list, ids, nil
}

func (dbConn *CacheLevel2) GetCommentsForContent(contentId int64, adminMode bool) (*[]*Comment, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)

	sqlBan := " AND NOT c.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	parentAuthor, parentPermlink, err := dbConn.GetContentIdStrings(contentId)

	rows, err := dbConn.Query(`
			SELECT `+CommentsListFieldSQL+`
			FROM 
				comments c, 
				content cp, 
				users u
					LEFT JOIN users_info ui ON ui.user_id = u.id
			WHERE c.parent_author = $1 AND c.parent_permlink = $2 
				AND c.author = u.name
				AND cp.author = c.parent_author AND cp.permlink = c.parent_permlink `+sqlBan+
			` ORDER BY c.time DESC`,
		parentAuthor, parentPermlink,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)

		list = append(list, content)
	}

	return &list, ids, nil
}

// Ищет id статьи для данного коментария
func (dbConn *CacheLevel2) GetArticleForComment(commentId int64, adminMode bool) (int64, error) {
	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	sqlBan2 := " AND NOT c2.ban AND NOT u2.ban "
	if adminMode {
		sqlBan1 = " "
		sqlBan2 = " "
	}

	rows, err := dbConn.Query(
		`
			WITH RECURSIVE all_contents ( id, parent_author, parent_permlink ) AS (
				SELECT c1.id, c1.parent_author, c1.parent_permlink
					FROM content c1, users u1
					WHERE c1.id = $1 AND c1.author = u1.name `+sqlBan1+`
				UNION
				SELECT c2.id, c2.parent_author, c2.parent_permlink
                	FROM content c2
					INNER JOIN all_contents ON c2.author = all_contents.parent_author AND c2.permlink = all_contents.parent_permlink,
					users u2
					WHERE c2.author = u2.name `+sqlBan2+`
			)
			
			SELECT id FROM all_contents WHERE (parent_author IS NULL OR parent_author = '') 
		`,
		commentId,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if !rows.Next() {
		return -1, nil
	}

	var id int64
	err = rows.Scan(
		&id,
	)
	if err != nil {
		return -1, err
	}

	return id, nil
}

// Комментарии первого уровня, написанные данным пользователем и все его дерево ответов
func (dbConn *CacheLevel2) GetUserCommentsFull(userId int64, pagination PaginationParams, paginationComment *Comment, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	listById := make(map[int64]*Comment)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	sqlBan2 := " AND NOT c2.ban AND NOT u2.ban "
	if adminMode {
		sqlBan1 = " "
		sqlBan2 = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	limitPaginationSQL := ""
	wherePaginationSQL := ""
	paginationParams := []interface{}{
		userName,
	}
	switch pagination.Mode {
	case PaginationModeFirst:
		limitPaginationSQL = " LIMIT $2 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeAfter:
		wherePaginationSQL = " AND c1.time < $2 AND c1.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
		limitPaginationSQL = " LIMIT $4 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeBefore:
		wherePaginationSQL = " AND c1.time > $2 AND c1.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
	}

	topList := make([]string, 0)
	rows, err := dbConn.Query(
		`
			SELECT c1.id
				FROM 
					comments c1
						LEFT JOIN content a1 ON a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink, 
					users u1
						LEFT JOIN users_info ui1 ON ui1.user_id = u1.id
				WHERE c1.level = 1 AND c1.author = $1 AND c1.author = u1.name `+sqlBan1+wherePaginationSQL+
				`
				ORDER BY c1.level, c1.time DESC
				`+
				limitPaginationSQL,
		paginationParams...,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			rows.Close()
			app.Error.Print(err)
			return nil, nil, nil, err
		}
		str := strconv.FormatInt(id, 10)
		topList = append(topList, str)
	}
	rows.Close()

	if len(topList) < 1 {
		return &list, ids, nil, nil
	}

	app.Debug.Printf("DBG: Count top list = %d", len(topList))

	// Рекурсивный SQL запрос
	rows, err = dbConn.Query(
		`
			WITH RECURSIVE all_comments (`+CommentsListFieldSQLRecursiveFields+`) 
			AS (
				SELECT 	`+CommentsListFieldSQLRecursiveFields1+`
					FROM 
						comments c1
							LEFT JOIN content a1 ON a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink, 
						users u1
							LEFT JOIN users_info ui1 ON ui1.user_id = u1.id
					WHERE c1.id IN (`+strings.Join(topList, `, `)+`) AND c1.author = u1.name
				UNION
				SELECT 	`+CommentsListFieldSQLRecursiveFields2+`
                	FROM 
						comments c2
							LEFT JOIN content c3 ON c3.author = c2.parent_author AND c3.permlink = c2.parent_permlink
							INNER JOIN all_comments ON c2.parent_author = all_comments.author AND c2.parent_permlink = all_comments.permlink,
						users u2
							LEFT JOIN users_info ui2 ON ui2.user_id = u2.id
					WHERE c2.author = u2.name `+sqlBan2+`
			)
			
			SELECT * FROM all_comments ORDER BY level, time DESC
		`,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)

		// Сразу строим дерево объектов
		listById[content.Id] = content

		if content.Level == 1 {
			// Если это верхний уровень - просто добавляем в список
			list = append(list, content)
			articlesIdsMap[content.ParentId] = true
		} else {
			// Если ниже - находим запомненный parent объект и добавляем в его список Comments
			parent := listById[content.ParentId]
			if parent != nil {
				if parent.Comments == nil {
					parent.Comments = make([]*Comment, 0)
				}
				parent.Comments = append(parent.Comments, content)
			}
		}
	}

	app.Debug.Printf("DBG: Count list = %d", len(list))

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	return &list, ids, articlesIds, nil
}

func (dbConn *CacheLevel2) GetUserComments(userId int64, pagination PaginationParams, paginationComment *Comment, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	listById := make(map[int64]*Comment)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c.ban AND NOT u.ban "
	if adminMode {
		sqlBan1 = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	limitPaginationSQL := ""
	wherePaginationSQL := ""
	paginationParams := []interface{}{
		userName,
	}
	switch pagination.Mode {
	case PaginationModeFirst:
		limitPaginationSQL = " LIMIT $2 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeAfter:
		wherePaginationSQL = " AND c.time < $2 AND c.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
		limitPaginationSQL = " LIMIT $4 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeBefore:
		wherePaginationSQL = " AND c.time > $2 AND c.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
	}

	rows, err := dbConn.Query(`
				SELECT `+CommentsListFieldSQL+`
				FROM 
					comments c
						LEFT JOIN content cp ON cp.author = c.parent_author AND cp.permlink = c.parent_permlink, 
					users u
						LEFT JOIN users_info ui ON ui.user_id = u.id
				WHERE c.level = 1 AND c.author = $1 AND c.author = u.name `+sqlBan1+wherePaginationSQL+`
				ORDER BY c.time DESC`+
				limitPaginationSQL,
		paginationParams...,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)

		// Сразу строим дерево объектов
		listById[content.Id] = content

		if content.Level == 1 {
			// Если это верхний уровень - просто добавляем в список
			list = append(list, content)
			articlesIdsMap[content.ParentId] = true
		} else {
			// Если ниже - находим запомненный parent объект и добавляем в его список Comments
			parent := listById[content.ParentId]
			if parent != nil {
				if parent.Comments == nil {
					parent.Comments = make([]*Comment, 0)
				}
				parent.Comments = append(parent.Comments, content)
			}
		}
	}

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	return &list, ids, articlesIds, nil
}

func (dbConn *CacheLevel2) GetUserCommentsCount(userId int64, adminMode bool) (int64, error) {
	sqlBan1 := " AND NOT c.ban AND NOT u.ban "
	if adminMode {
		sqlBan1 = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	rows, err := dbConn.Query(`
				SELECT COUNT(*)
				FROM 
					comments c
						LEFT JOIN content cp ON cp.author = c.parent_author AND cp.permlink = c.parent_permlink, 
					users u
						LEFT JOIN users_info ui ON ui.user_id = u.id
				WHERE c.level = 1 AND c.author = $1 AND c.author = u.name `+sqlBan1,
		userName,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if rows.Next() {
		var count int64
		err = rows.Scan(&count)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}
		return count, nil
	}

	return 0, nil
}

// Комментарии, написанные в ответ на статьи данного пользователя
func (dbConn *CacheLevel2) GetUserContentCommentsFull(userId int64, pagination PaginationParams, paginationComment *Comment, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	listById := make(map[int64]*Comment)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	sqlBan2 := " AND NOT c2.ban AND NOT u2.ban "
	if adminMode {
		sqlBan1 = " "
		sqlBan2 = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	limitPaginationSQL := ""
	wherePaginationSQL := ""
	paginationParams := []interface{}{
		userName,
	}
	switch pagination.Mode {
	case PaginationModeFirst:
		limitPaginationSQL = " LIMIT $2 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeAfter:
		wherePaginationSQL = " AND c1.time < $2 AND c1.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
		limitPaginationSQL = " LIMIT $4 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeBefore:
		wherePaginationSQL = " AND c1.time > $2 AND c1.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
	}

	topList := make([]string, 0)
	rows, err := dbConn.Query(
		`
			SELECT c1.id
				FROM 
					comments c1
						LEFT JOIN content a1 ON a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink,
					users u1
						LEFT JOIN users_info ui1 ON ui1.user_id = u1.id 
				WHERE c1.level = 1 AND a1.author = $1 AND c1.author = u1.name `+sqlBan1+wherePaginationSQL+
				`
				ORDER BY c1.level, c1.time DESC
				`+
				limitPaginationSQL,
		paginationParams...,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			rows.Close()
			app.Error.Print(err)
			return nil, nil, nil, err
		}
		str := strconv.FormatInt(id, 10)
		topList = append(topList, str)
	}
	rows.Close()

	if len(topList) < 1 {
		return &list, ids, nil, nil
	}

	app.Debug.Printf("DBG: Count top list = %d", len(topList))

	// Рекурсивный SQL запрос
	rows, err = dbConn.Query(
		`
			WITH RECURSIVE all_comments (`+CommentsListFieldSQLRecursiveFields+`) 
			AS (
				SELECT 	`+CommentsListFieldSQLRecursiveFields1+`
					FROM 
						comments c1
							LEFT JOIN content a1 ON a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink,
						users u1
							LEFT JOIN users_info ui1 ON ui1.user_id = u1.id 
					WHERE c1.id IN (`+strings.Join(topList, `, `)+`) AND c1.author = u1.name
				UNION
				SELECT 	`+CommentsListFieldSQLRecursiveFields2+`
                	FROM 
						comments c2
							LEFT JOIN content c3 ON c3.author = c2.parent_author AND c3.permlink = c2.parent_permlink
							INNER JOIN all_comments ON c2.parent_author = all_comments.author AND c2.parent_permlink = all_comments.permlink,
						users u2
							LEFT JOIN users_info ui2 ON ui2.user_id = u2.id
					WHERE c2.author = u2.name `+sqlBan2+`
			)
			
			SELECT * FROM all_comments ORDER BY level, time DESC
		`,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)

		// Сразу строим дерево объектов
		listById[content.Id] = content

		if content.Level == 1 {
			// Если это верхний уровень - просто добавляем в список
			list = append(list, content)
			articlesIdsMap[content.ParentId] = true
		} else {
			// Если ниже - находим запомненный parent объект и добавляем в его список Comments
			parent := listById[content.ParentId]
			if parent != nil {
				if parent.Comments == nil {
					parent.Comments = make([]*Comment, 0)
				}
				parent.Comments = append(parent.Comments, content)
			}
		}
	}

	app.Debug.Printf("DBG: Count list = %d", len(list))

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	return &list, ids, articlesIds, nil
}

func (dbConn *CacheLevel2) GetUserContentComments(userId int64, pagination PaginationParams, paginationComment *Comment, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	listById := make(map[int64]*Comment)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c.ban AND NOT u.ban "
	if adminMode {
		sqlBan1 = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	limitPaginationSQL := ""
	wherePaginationSQL := ""
	paginationParams := []interface{}{
		userName,
	}
	switch pagination.Mode {
	case PaginationModeFirst:
		limitPaginationSQL = " LIMIT $2 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeAfter:
		wherePaginationSQL = " AND c.time < $2 AND c.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
		limitPaginationSQL = " LIMIT $4 "
		paginationParams = append(paginationParams, pagination.Count)
	case PaginationModeBefore:
		wherePaginationSQL = " AND c.time > $2 AND c.id <> $3"
		paginationParams = append(paginationParams, paginationComment.Time)
		paginationParams = append(paginationParams, paginationComment.Id)
	}

	rows, err := dbConn.Query(`
				SELECT `+CommentsListFieldSQL+`
				FROM 
					comments c
						LEFT JOIN content cp ON cp.author = c.parent_author AND cp.permlink = c.parent_permlink,
					users u
						LEFT JOIN users_info ui ON ui.user_id = u.id 					
				WHERE c.level = 1 AND cp.author = $1 AND c.author = u.name `+sqlBan1+wherePaginationSQL+`
				ORDER BY c.time DESC`+
				limitPaginationSQL,
		paginationParams...,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)

		// Сразу строим дерево объектов
		listById[content.Id] = content

		if content.Level == 1 {
			// Если это верхний уровень - просто добавляем в список
			list = append(list, content)
			articlesIdsMap[content.ParentId] = true
		} else {
			// Если ниже - находим запомненный parent объект и добавляем в его список Comments
			parent := listById[content.ParentId]
			if parent != nil {
				if parent.Comments == nil {
					parent.Comments = make([]*Comment, 0)
				}
				parent.Comments = append(parent.Comments, content)
			}
		}
	}

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	return &list, ids, articlesIds, nil
}

func (dbConn *CacheLevel2) GetUserContentCommentsCount(userId int64, adminMode bool) (int64, error) {
	sqlBan1 := " AND NOT c.ban AND NOT u.ban "
	if adminMode {
		sqlBan1 = " "
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	rows, err := dbConn.Query(`
				SELECT COUNT(*)
				FROM 
					comments c
						LEFT JOIN content cp ON cp.author = c.parent_author AND cp.permlink = c.parent_permlink,
					users u
						LEFT JOIN users_info ui ON ui.user_id = u.id 					
				WHERE c.level = 1 AND cp.author = $1 AND c.author = u.name `+sqlBan1,
		userName,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if rows.Next() {
		var count int64
		err = rows.Scan(&count)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}
		return count, nil
	}

	return 0, nil
}

// Возвращает определенное количество комментариев первого уровня с ответами
func (dbConn *CacheLevel2) GetAllCommentsLast(count int, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	if adminMode {
		sqlBan1 = " "
	}

	if count <= 0 {
		return &list, ids, []int64{}, nil
	}

	rows, err := dbConn.Query(`
				SELECT 	`+AllCommentsListFieldSQL+`
					FROM 
						comments c1,
						articles a1,
						users u1
							LEFT JOIN users_info ui1 ON ui1.user_id = u1.id
					WHERE 
						c1.level = 1 AND c1.author = u1.name AND 
						a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink `+sqlBan1+`
					ORDER BY c1.time DESC
					LIMIT $1`,
		count,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)
		articlesIdsMap[content.ParentId] = true

		list = append(list, content)
	}
	rows.Close()

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	// В цикле заполняем ответы для комментариев первого уровня
	for i := 0; i < len(list); i++ {
		comments, commentsIds, err := dbConn.GetCommentsForContentFull(list[i].Id, adminMode)
		if err != nil {
			app.Error.Print(err)
		}

		if comments != nil {
			list[i].Comments = *comments
			ids = append(ids, commentsIds...)
		}
	}

	return &list, ids, articlesIds, nil
}

// Возвращает все комментарии первого уровня с ответами перед (более новые) указанным комментарием
func (dbConn *CacheLevel2) GetAllCommentsBefore(before int64, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	if adminMode {
		sqlBan1 = " "
	}

	rows, err := dbConn.Query(`
				SELECT 	`+AllCommentsListFieldSQL+`
					FROM 
						comments c1,
						articles a1,
						users u1
							LEFT JOIN users_info ui1 ON ui1.user_id = u1.id 					
					WHERE 
						c1.id > $1 AND c1.level = 1 AND c1.author = u1.name AND
						a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink `+sqlBan1+`
					ORDER BY c1.time DESC`,
		before,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)
		articlesIdsMap[content.ParentId] = true

		list = append(list, content)
	}
	rows.Close()

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	// В цикле заполняем ответы для комментариев первого уровня
	for i := 0; i < len(list); i++ {
		comments, commentsIds, err := dbConn.GetCommentsForContentFull(list[i].Id, adminMode)
		if err != nil {
			app.Error.Print(err)
		}

		if comments != nil {
			list[i].Comments = *comments
			ids = append(ids, commentsIds...)
		}
	}

	return &list, ids, articlesIds, nil
}

// Возвращает определенное количество комментариев первого уровня с ответами после (более старые) указанного комментария
func (dbConn *CacheLevel2) GetAllCommentsAfter(after int64, count int, adminMode bool) (*[]*Comment, []int64, []int64, error) {
	list := make([]*Comment, 0)
	ids := make([]int64, 0)
	articlesIdsMap := make(map[int64]bool)

	sqlBan1 := " AND NOT c1.ban AND NOT u1.ban "
	if adminMode {
		sqlBan1 = " "
	}

	if count <= 0 {
		return &list, ids, []int64{}, nil
	}

	rows, err := dbConn.Query(
		`
				SELECT `+AllCommentsListFieldSQL+`
					FROM 
						comments c1,
						articles a1,
						users u1
							LEFT JOIN users_info ui1 ON ui1.user_id = u1.id 					
					WHERE 
						c1.id < $1 AND c1.level = 1 AND c1.author = u1.name AND
						a1.author = c1.parent_author AND a1.permlink = c1.parent_permlink `+sqlBan1+`
					ORDER BY c1.time DESC
					LIMIT $2`,
		after, count,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, nil, nil, err
	}

	for rows.Next() {
		content, _ := prepareComment(rows, false)

		// Устанавливаем для коммента суммы заработанного из базы mongodb
		err = dbConn.SetCommentRewardsValues(content)
		if err != nil {
			app.Error.Println(err)
		}

		ids = append(ids, content.NodeosId)
		articlesIdsMap[content.ParentId] = true

		list = append(list, content)
	}
	rows.Close()

	// Заполняем список id статей
	articlesIds := make([]int64, 0, len(articlesIdsMap))
	for id := range articlesIdsMap {
		articlesIds = append(articlesIds, id)
	}

	// В цикле заполняем ответы для комментариев первого уровня
	for i := 0; i < len(list); i++ {
		comments, commentsIds, err := dbConn.GetCommentsForContentFull(list[i].Id, adminMode)
		if err != nil {
			app.Error.Print(err)
		}

		if comments != nil {
			list[i].Comments = *comments
			ids = append(ids, commentsIds...)
		}
	}

	return &list, ids, articlesIds, nil
}

func prepareComment(rows *sql.Rows, rawFormat bool) (*Comment, error) {
	content := Comment{}
	content.User = UserInfo{}
	var nodeosId sql.NullInt64
	var body string
	var valTime NullTime
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
		&content.ParentId,
		&content.ParentAuthor,
		&content.ParentPermlink,
		&content.Author,
		&content.Permlink,
		&content.Title,
		&body,
		&content.VotesCount,
		&content.VotesCountPositive,
		&content.VotesCountNegative,
		&content.VotesSumPositive,
		&content.VotesSumNegative,
		&valTime,
		&content.Ban,
		&content.Level,
		&valCyber,
		&valGolos,
		&valPower,
		&content.Editor,
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
		app.Error.Printf("Error when scan data row: %s", err)
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

	if valTime.Valid {
		content.Time = valTime.Format()
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

	if rawFormat {
		content.Body = body
	} else {
		content.Body = filters.HTMLBodyFilter(body)
	}

	return &content, nil
}

// Из базы mongodb берем суммы заработанного статьей и устанавливаем их в записях нашей базы
func (dbConn *CacheLevel2) SetCommentListRewardsValues(list []*Comment) error {
	for _, cmnt := range list {
		if cmnt.NodeosId > 0 {
			_ = dbConn.SetCommentRewardsValues(cmnt)
		}
	}

	return nil
}

func (dbConn *CacheLevel2) SetCommentRewardsValues(cmnt *Comment) error {
	cmnt.ValGolos, _ = dbConn.GetContentRewardNodeos(cmnt.NodeosId)
	cmnt.ValCyber = 0
	cmnt.ValPower = 0

	return nil
}
