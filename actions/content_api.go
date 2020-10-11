package actions

import (
	"github.com/pkg/errors"
	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"net/http"
	"strings"
)

const (
	DefaultPeriodForPopular = float64(3600.0 * 24 * 30)
	RecalcArticlesValsLimit = 10000000
)

type SourceList struct {
	List      	string
	SortField 	string
	DescOrder	bool
	UserId   	int64
	Tags		[]string
	Rubrics		[]string
}

func GetArticlesList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация если есть
	var userId int64
	var userInfo *cache_level2.UserInfo
	var notMat bool
	authorized := false
	adminMode := false
	claims, _, err := jwt.Check(r)
	if err == nil {
		authorized = true
		userId = int64((*claims)["sub"].(float64))
		role := (*claims)["r"].(string)
		adminMode = role == "a"
		userInfo, err = DB.GetUserInfo(userId)
		if err != nil {
			app.Error.Print(errors.Wrap(err, "GetUserInfo"))
		}
		if userInfo != nil && strings.Trim(userInfo.PvtPostsShowMode, " ") == "H" {
			notMat = true
		}
	}
	// Это НЕ аналогично natMat = !authorized - НЕ МЕНЯТЬ!!!
	if !authorized {
		notMat = true
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, errors.Wrap(err, "decode request"), nil)
		return
	}
	if !IsParamType(&w, "type", params["type"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "type", "string"), nil)
		return
	}
	var tags []string
	tagsF, ok := params["tags"]
	if ok {
		switch tagsF.(type) {
		case []string:
			tags = tagsF.([]string)
		case []interface{}:
			tags = make([]string, 0)
			for _, tag := range tagsF.([]interface{}) {
				tags = append(tags, tag.(string))
			}
		default:
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "tags", "array of strings"), nil)
			return
		}
	} else {
		tags = nil
	}
	// Если тэги есть - перекодируем
	if tags != nil && len(tags) > 0 {
		tags = translit.EncodeTags(tags)
	}

	var rubrics []string
	rubricsF, ok := params["rubrics"]
	if ok {
		switch rubricsF.(type) {
		case []string:
			rubrics = rubricsF.([]string)
		case []interface{}:
			rubrics = make([]string, 0)
			for _, rubric := range rubricsF.([]interface{}) {
				rubrics = append(rubrics, rubric.(string))
			}
		default:
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "rubrics", "array of strings"), nil)
			return
		}
	} else {
		rubrics = nil
	}
	// Если рубрики есть - перекодируем
	if rubrics != nil && len(rubrics) > 0 {
		rubrics = translit.EncodeTags(rubrics)
	}

	filter := ""
	if IsParamType(&w, "filter", params["filter"], "string") {
		filter = params["filter"].(string)
	}

	// Выдаем порцию следующих статей после указанной
	_, ok = params["after_article"]
	if ok {
		if !IsParamType(&w, "after_article", params["after_article"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "after_article", "number"), nil)
			return
		}
		if !IsParamType(&w, "count", params["count"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
			return
		}

		lastArticleId := int64(params["after_article"].(float64))
		countF, ok := params["count"].(float64)
		if !ok {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
			return
		}
		count := int(countF)

		var list *[]*cache_level2.Article
		switch params["type"].(string) {
		case "new", "":
			list, err = DB.GetArticlesAfter(lastArticleId, count, tags, rubrics, notMat, adminMode, filter)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetArticlesAfter"), nil)
				return
			}
		case "follow":
			if !authorized {
				DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
				return
			}

			list, err = DB.GetFollowArticlesAfter(userId, lastArticleId, count, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetFollowArticlesAfter"), nil)
				return
			}
		case "actual":
			list, err = DB.GetActualArticlesAfter(lastArticleId, count, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetActualArticlesAfter"), nil)
				return
			}
		case "popular":
			periodF, ok := params["period"].(float64)
			if !ok || periodF <= 0.0 {
				periodF = DefaultPeriodForPopular
			}
			period := int64(periodF)

			list, err = DB.GetPopularArticlesAfter(lastArticleId, count, period, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetPopularArticlesAfter"), nil)
				return
			}
		case "blog":
			userIdF, ok := params["user_id"].(float64)
			if !ok {
				DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
				return
			}
			blogUserId := int64(userIdF)

			if authorized && userId == blogUserId {
				notMat = false
			}

			list, err = DB.GetBlogArticlesAfter(lastArticleId, count, blogUserId, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetBlogArticlesAfter"), nil)
				return
			}
		}

		resp["list"] = list

		lenList := 0
		if list != nil {
			lenList = len(*list)
		}

		contentIds := make([]int64, lenList)
		if lenList > 0 {
			for idx, content := range *list {
				contentIds[idx] = content.NodeosId
			}
		}

		// All votes
		votes, err := DB.GetVotesForContentList(&contentIds)
		if err == nil {
			resp["votes"] = votes
		} else {
			app.Error.Printf("Error get votes: %s", err)
		}

		if authorized {
			// Votes for authorized user

			userVotes, err := DB.GetUserVotesForContentList(userId, &contentIds)

			if err == nil && userVotes != nil {
				resp["current_user_votes"] = userVotes
			}
			if err != nil {
				app.Error.Printf("GetUserVotesForContentList error: %s", err)
			}
		}

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Выдаем все новые статьи перед указанной
	_, ok = params["before_article"]
	if ok {
		if !IsParamType(&w, "before_article", params["before_article"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "before_article", "number"), nil)
			return
		}

		firstArticleId := int64(params["before_article"].(float64))

		var list *[]*cache_level2.Article
		switch params["type"].(string) {
		case "new", "":
			list, err = DB.GetArticlesBefore(firstArticleId, tags, rubrics, notMat, adminMode, filter)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetArticlesBefore"), nil)
				return
			}
		case "follow":
			if !authorized {
				DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
				return
			}

			list, err = DB.GetFollowArticlesBefore(userId, firstArticleId, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetFollowArticlesBefore"), nil)
				return
			}
		case "actual":
			list, err = DB.GetActualArticlesBefore(firstArticleId, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetActualArticlesBefore"), nil)
				return
			}
		case "popular":
			periodF, ok := params["period"].(float64)
			if !ok || periodF <= 0.0 {
				periodF = DefaultPeriodForPopular
			}
			period := int64(periodF)

			list, err = DB.GetPopularArticlesBefore(firstArticleId, period, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetPopularArticlesBefore"), nil)
				return
			}
		case "blog":
			userIdF, ok := params["user_id"].(float64)
			if !ok {
				DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
				return
			}
			userId := int64(userIdF)

			list, err = DB.GetBlogArticlesBefore(firstArticleId, userId, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetBlogArticlesBefore"), nil)
				return
			}
		}

		resp["list"] = list

		lenList := 0
		if list != nil {
			lenList = len(*list)
		}

		contentIds := make([]int64, lenList)
		if lenList > 0 {
			for idx, content := range *list {
				contentIds[idx] = content.NodeosId
			}
		}

		// All votes
		votes, err := DB.GetVotesForContentList(&contentIds)
		if err == nil {
			resp["votes"] = votes
		} else {
			app.Error.Printf("Error get votes: %s", err)
		}

		if authorized {
			// Votes for authorized user

			userVotes, err := DB.GetUserVotesForContentList(userId, &contentIds)

			if err == nil && userVotes != nil {
				resp["current_user_votes"] = userVotes
			}
			if err != nil {
				app.Error.Printf("GetUserVotesForContentList error: %s", err)
			}
		}

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Выдаем указанное количество последних статей
	_, ok = params["count"]
	if ok {
		if !IsParamType(&w, "count", params["count"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
			return
		}

		count := int(params["count"].(float64))

		var list *[]*cache_level2.Article
		switch params["type"].(string) {
		case "new", "":
			list, err = DB.GetLastArticles(count, tags, rubrics, notMat, adminMode, filter)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetLastArticles"), nil)
				return
			}
		case "follow":
			if !authorized {
				DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
				return
			}

			list, err = DB.GetFollowLastArticles(userId, count, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetFollowLastArticles"), nil)
				return
			}
		case "actual":
			list, err = DB.GetActualLastArticles(count, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetActualLastArticles"), nil)
				return
			}
		case "popular":
			periodF, ok := params["period"].(float64)
			if !ok || periodF <= 0.0 {
				periodF = DefaultPeriodForPopular
			}
			period := int64(periodF)

			list, err = DB.GetPopularLastArticles(count, period, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetPopularLastArticles"), nil)
				return
			}
		case "blog":
			userIdF, ok := params["user_id"].(float64)
			if !ok {
				DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
				return
			}
			userId := int64(userIdF)

			list, err = DB.GetBlogLastArticles(count, userId, tags, rubrics, notMat, adminMode)
			if err != nil {
				DoJSONError(&w, errors.Wrap(err, "GetBlogLastArticles"), nil)
				return
			}
		}

		// Обновляем стоимость для глючных статей
		// Обновляем данные по суммам для статьи
		ids := make([][]string, 0)
		for _, art := range *list {
			// Делаем расчет только если сумма больше 1 тысячи GBG
			if art.ValCyber < RecalcArticlesValsLimit {
				continue
			}

			rec := make([]string, 2)
			rec[0] = art.Author
			rec[1] = art.Permlink
			ids = append(ids, rec)
		}

		//if len(ids) > 0 {
		//	blockchain.SyncContentList(&Config.RPC, ids)
		//}

		resp["list"] = list

		lenList := 0
		if list != nil {
			lenList = len(*list)
		}

		contentIds := make([]int64, lenList)
		if lenList > 0 {
			for idx, content := range *list {
				contentIds[idx] = content.NodeosId
			}
		}

		// All votes
		votes, err := DB.GetVotesForContentList(&contentIds)
		if err == nil {
			resp["votes"] = votes
		} else {
			app.Error.Printf("Error get votes: %s", err)
		}

		if authorized {
			// Votes for authorized user

			userVotes, err := DB.GetUserVotesForContentList(userId, &contentIds)

			if err == nil && userVotes != nil {
				resp["current_user_votes"] = userVotes
			}
			if err != nil {
				app.Error.Printf("GetUserVotesForContentList error: %s", err)
			}
		}

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Если нет правильной комбинации параметров, выходим с ошибкой
	DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
	return
}

func GetArticle(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация если есть
	var userId int64
	authorized := false
	adminMode := false
	claims, _, err := jwt.Check(r)
	if err == nil {
		userId = int64((*claims)["sub"].(float64))
		role := (*claims)["r"].(string)
		adminMode = role == "a"
		authorized = true
		_, err = DB.GetUserInfo(userId)
		if err != nil {
			app.Error.Print(err)
		}
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	rawFormat := false
	rawFormatVal, ok := params["raw"]
	if ok {
		rawFormat = rawFormatVal.(bool)
	}

	// Параметр source_list
	//   type - тип списка (list|follow|blog)
	//   sort_field - поле для сортировки (id, time, last_comment_time, val_cyber)
	//   user_id - id пользователя для типов списка follow и blog (для list может отсутствовать)
	var sourceList *SourceList
	if IsParamType(&w, "source_list", params["source_list"], "map[string]interface {}") {
		sourceList = &SourceList{}
		sourceListVal, ok := params["source_list"].(map[string]interface{})
		if ok {
			listVal, ok := sourceListVal["list"]
			if ok {
				sourceList.List = listVal.(string)
			}

			sortFieldVal, ok := sourceListVal["sort_field"]
			if ok {
				sourceList.SortField = sortFieldVal.(string)
			}

			descOrderVal, ok := sourceListVal["desc_order"]
			if ok {
				sourceList.DescOrder = descOrderVal.(bool)
			}

			userIdVal, ok := sourceListVal["user_id"]
			if ok {
				sourceList.UserId = int64(userIdVal.(float64))
			}

			var tags []string
			tagsF, ok := sourceListVal["tags"]
			if ok {
				switch tagsF.(type) {
				case []string:
					tags = tagsF.([]string)
				case []interface{}:
					tags = make([]string, 0)
					for _, tag := range tagsF.([]interface{}) {
						tags = append(tags, tag.(string))
					}
				default:
					app.Error.Println(errors_l10n.New(lang, "parameters.should_be", "tags", "array of strings"))
				}
			} else {
				tags = nil
			}
			// Если тэги есть - перекодируем
			if tags != nil && len(tags) > 0 {
				tags = translit.EncodeTags(tags)
				sourceList.Tags = tags
			}

			var rubrics []string
			rubricsF, ok := sourceListVal["rubrics"]
			if ok {
				switch rubricsF.(type) {
				case []string:
					rubrics = rubricsF.([]string)
				case []interface{}:
					rubrics = make([]string, 0)
					for _, rubric := range rubricsF.([]interface{}) {
						rubrics = append(rubrics, rubric.(string))
					}
				default:
					DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "rubrics", "array of strings"), nil)
					return
				}
			} else {
				rubrics = nil
			}
			// Если рубрики есть - перекодируем
			if rubrics != nil && len(rubrics) > 0 {
				rubrics = translit.EncodeTags(rubrics)
				sourceList.Rubrics = rubrics
			}
		}

		if sourceList.List == "" {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "source_list.list", "string"), nil)
			return
		}

		if sourceList.SortField == "" {
			sourceList.SortField = "time"
		}

		if (sourceList.List == "follow" || sourceList.List == "blog") && sourceList.UserId <= 0 {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "source_list.user_id", "number"), nil)
			return
		}
	}

	_, ok = params["id"]
	if ok {
		if !IsParamType(&w, "id", params["id"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "id", "number"), nil)
			return
		}

		id := int64(params["id"].(float64))

		article, err := DB.GetArticle(id, rawFormat, adminMode)
		if err != nil {
			DoJSONError(&w, err, nil)
			return
		}
		if article == nil {
			app.Error.Printf("GetArticle return nil article: %d", id)
			DoJSONError(&w, errors_l10n.New(lang, "content.article_not_found"), nil)
			return
		}

		resp["content"] = article

		votes, err := DB.GetVotesForContentList(&[]int64{article.NodeosId})
		if err == nil {
			resp["votes"] = votes
		} else {
			app.Error.Printf("Error get votes: %s", err)
		}

		if authorized {
			// Votes for authorized user

			userVotes, err := DB.GetUserVotesForContentList(userId, &[]int64{article.NodeosId})
			if err == nil && userVotes != nil {
				resp["current_user_votes"] = userVotes
			}
			if err != nil {
				app.Error.Printf("GetUserVotesForContent error: %s", err)
			}
		}


		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Если нет правильной комбинации параметров, выходим с ошибкой
	DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
	return
}

func GetCommentsList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация если есть
	var userId int64
	authorized := false
	adminMode := false
	claims, _, err := jwt.Check(r)
	if err == nil {
		authorized = true
		userId = int64((*claims)["sub"].(float64))
		role := (*claims)["r"].(string)
		adminMode = role == "a"
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	_, ok := params["parent_id"]
	if ok {
		if !IsParamType(&w, "parent_id", params["parent_id"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "parent_id", "number"), nil)
			return
		}
		id := int64(params["parent_id"].(float64))

		full := true
		fullI, ok := params["full"]
		if ok {
			if !IsParamType(&w, "full", params["full"], "bool") {
				DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "full", "boolean"), nil)
				return
			}
		}
		full = fullI.(bool)

		var ids []int64

		var list *[]*cache_level2.Comment
		if full {
			list, ids, err = DB.GetCommentsForContentFull(id, adminMode)

			if err != nil {
				DoJSONError(&w, err, nil)
				return
			}
		} else {
			list, ids, err = DB.GetCommentsForContent(id, adminMode)
			if err != nil {
				DoJSONError(&w, err, nil)
				return
			}
		}

		resp["list"] = list

		// All votes
		votes, err := DB.GetVotesForContentList(&ids)
		if err == nil {
			resp["votes"] = votes
		} else {
			app.Error.Printf("Error get votes: %s", err)
		}

		if authorized {
			// Votes for authorized user

			userVotes, err := DB.GetUserVotesForContentList(userId, &ids)

			if err == nil && userVotes != nil {
				resp["current_user_votes"] = userVotes
			}
			if err != nil {
				app.Error.Printf("GetUserVotesForContentList error: %s", err)
			}
		}

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Если нет правильной комбинации параметров, выходим с ошибкой
	DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
	return
}

func GetUserCommentsList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация если есть
	var userId int64
	authorized := false
	adminMode := false
	claims, _, err := jwt.Check(r)
	if err == nil {
		authorized = true
		userId = int64((*claims)["sub"].(float64))
		role := (*claims)["r"].(string)
		adminMode = role == "a"
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	pagination := cache_level2.PaginationParams{}
	if !IsParamType(&w, "user_id", params["user_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
		return
	}
	if !IsParamType(&w, "type", params["type"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "type", "string"), nil)
		return
	}
	if !IsParamType(&w, "full", params["full"], "bool") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "full", "boolean"), nil)
		return
	}

	if !IsParamType(&w, "mode", params["mode"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "mode", "string"), nil)
		return
	}
	mode := params["mode"].(string)
	switch mode {
	case "first":
		pagination.Mode = cache_level2.PaginationModeFirst
	case "after":
		pagination.Mode = cache_level2.PaginationModeAfter
	case "before":
		pagination.Mode = cache_level2.PaginationModeBefore
	}

	if !IsParamType(&w, "id", params["id"], "float64") {
		if pagination.Mode != cache_level2.PaginationModeFirst {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "id", "number"), nil)
			return
		}
	} else {
		pagination.Id = int64(params["id"].(float64))
	}

	if !IsParamType(&w, "count", params["count"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
		return
	}
	forUserId := int64(params["user_id"].(float64))
	listType := params["type"].(string)
	full := params["full"].(bool)

	pagination.Count = int(params["count"].(float64))

	var list *[]*cache_level2.Comment
	var ids []int64
	var articlesIds []int64

	if full {
		switch listType {
		case "owner":
			list, ids, articlesIds, err = DB.GetUserCommentsFull(forUserId, pagination, adminMode)
		case "reply":
			list, ids, articlesIds, err = DB.GetUserContentCommentsFull(forUserId, pagination, adminMode)
		}

		if err != nil {
			DoJSONError(&w, err, nil)
			return
		}
	} else {
		switch listType {
		case "owner":
			list, ids, articlesIds, err = DB.GetUserComments(forUserId, pagination, adminMode)
		case "reply":
			list, ids, articlesIds, err = DB.GetUserContentComments(forUserId, pagination, adminMode)
		}

		if err != nil {
			DoJSONError(&w, err, nil)
			return
		}
	}

	resp["list"] = list

	// All votes
	votes, err := DB.GetVotesForContentList(&ids)
	if err == nil {
		resp["votes"] = votes
	} else {
		app.Error.Printf("Error get votes: %s", err)
	}

	// All articles
	articles, err := DB.GetArticlesListByIds(&articlesIds)
	articlesHash := make(map[int64]*cache_level2.Article)
	if err == nil {
		for _, art := range *articles {
			articlesHash[art.Id] = art
		}
		resp["articles"] = articlesHash
	}

	if authorized {
		// Votes for authorized user

		userVotes, err := DB.GetUserVotesForContentList(userId, &ids)

		if err == nil && userVotes != nil {
			resp["current_user_votes"] = userVotes
		}
		if err != nil {
			app.Error.Printf("GetUserVotesForContentList error: %s", err)
		}
	}

	// Количество комментариев
	countOwner, err := DB.GetUserCommentsCount(forUserId, adminMode)
	if err != nil {
		app.Error.Println(err)
		countOwner = 0
	}
	countReply, err := DB.GetUserContentCommentsCount(forUserId, adminMode)
	if err != nil {
		app.Error.Println(err)
		countReply = 0
	}
	resp["owner_count"] = countOwner
	resp["reply_count"] = countReply

	DoJSONResponse(&w, &resp, nil)
	return
}

func GetComment(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация если есть
	adminMode := false
	claims, _, err := jwt.Check(r)
	if err == nil {
		role := (*claims)["r"].(string)
		adminMode = role == "a"
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	rawFormat := false
	rawFormatVal, ok := params["raw"]
	if ok {
		rawFormat = rawFormatVal.(bool)
	}

	_, ok = params["id"]
	if ok {
		if !IsParamType(&w, "id", params["id"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "id", "number"), nil)
			return
		}

		id := int64(params["id"].(float64))

		comment, err := DB.GetComment(id, rawFormat, adminMode)
		if err != nil {
			DoJSONError(&w, err, nil)
			return
		}

		resp["content"] = comment

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Если нет правильной комбинации параметров, выходим с ошибкой
	DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
	return
}


func GetVotesList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "id", params["id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "id", "number"), nil)
		return
	}
	id := int64(params["id"].(float64))

	var list *[]*cache_level2.Vote
	nodeosId, err := DB.GetContentNodeosIdById(id)
	if err == nil {
		list, err = DB.GetVotesForContent(nodeosId)
		if err != nil {
			DoJSONError(&w, err, nil)
			return
		}
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}

// Выдача коментариев для станицы "Комментарии"
// Все комментарии первого уровня с сортировкой от новых к старым + дерево ответов
func GetAllCommentsList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация если есть
	var userId int64
	authorized := false
	adminMode := false
	claims, _, err := jwt.Check(r)
	if err == nil {
		authorized = true
		userId = int64((*claims)["sub"].(float64))
		role := (*claims)["r"].(string)
		adminMode = role == "a"
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	var list *[]*cache_level2.Comment
	var ids []int64
	var articlesIds []int64

	worked := false
	_, okAfter := params["after_comment"]
	if okAfter {
		// Выводим указанное количество комментариев после указанного (более старые)
		if !IsParamType(&w, "after_comment", params["after_comment"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "after_comment", "number"), nil)
			return
		}
		if !IsParamType(&w, "count", params["count"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
			return
		}
		after := int64(params["after_comment"].(float64))
		count := int(params["count"].(float64))

		list, ids, articlesIds, err = DB.GetAllCommentsAfter(after, count, adminMode)
		worked = true
	}

	_, okBefore := params["before_comment"]
	if okBefore && !okAfter {
		// Выводим все комментарии перед указанным (более новые)
		if !IsParamType(&w, "before_comment", params["before_comment"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "before_comment", "number"), nil)
			return
		}
		before := int64(params["before_comment"].(float64))

		list, ids, articlesIds, err = DB.GetAllCommentsBefore(before, adminMode)
		worked = true
	}

	_, okCount := params["count"]
	if okCount && !okAfter && !okBefore {
		// Выводим указанное количество последних комментариев
		if !IsParamType(&w, "count", params["count"], "float64") {
			DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
			return
		}
		count := int(params["count"].(float64))

		list, ids, articlesIds, err = DB.GetAllCommentsLast(count, adminMode)
		worked = true
	}

	if worked {
		if err != nil {
			DoJSONError(&w, err, nil)
			return
		}
	} else {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
		return
	}

	resp["list"] = list

	// All votes
	votes, err := DB.GetVotesForContentList(&ids)
	if err == nil {
		resp["votes"] = votes
	} else {
		app.Error.Printf("Error get votes: %s", err)
	}

	// All articles
	articles, err := DB.GetArticlesListByIds(&articlesIds)
	articlesHash := make(map[int64]*cache_level2.Article)
	if err == nil {
		for _, art := range *articles {
			articlesHash[art.Id] = art
		}
		resp["articles"] = articlesHash
	}

	if authorized {
		// Votes for authorized user

		userVotes, err := DB.GetUserVotesForContentList(userId, &ids)

		if err == nil && userVotes != nil {
			resp["current_user_votes"] = userVotes
		}
		if err != nil {
			app.Error.Printf("GetUserVotesForContentList error: %s", err)
		}
	}

	DoJSONResponse(&w, &resp, nil)
	return
}
