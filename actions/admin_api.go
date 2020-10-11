package actions

import (
	"net/http"
		)

// Выдает в JSON информацию по заполненности БД и номеру последнего отсканиравонного блока в blockchain
func GetContentInfo(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{ "status": "ok" }
	info := map[string]interface{}{}

	var err error
	info["last_block"], err = DB.GetState("last_scan_block")
	if err != nil {
		info["last_block"] = "-"
	}

	articlesInfo := map[string]interface{}{}
	articlesInfo["count"], err = DB.Level2.GetArticlesCount()
	if err != nil {
		articlesInfo["count"] = "-"
	}
	articlesInfo["last_time"], err = DB.Level2.GetArticlesLastTime()
	if err != nil {
		articlesInfo["last_time"] = "-"
	}
	info["articles"] = articlesInfo

	commentsInfo := map[string]interface{}{}
	commentsInfo["count"], err = DB.Level2.GetCommentsCount()
	if err != nil {
		commentsInfo["count"] = "-"
	}
	commentsInfo["last_time"], err = DB.Level2.GetCommentsLastTime()
	if err != nil {
		commentsInfo["last_time"] = "-"
	}
	info["comments"] = commentsInfo

	tagsInfo := map[string]interface{}{}
	tagsInfo["count"], err = DB.Level2.GetTagsCount()
	if err != nil {
		tagsInfo["count"] = "-"
	}
	info["tags"] = tagsInfo

	votesInfo := map[string]interface{}{}
	votesInfo["count"], err = DB.Level2.GetVotesCount()
	if err != nil {
		votesInfo["count"] = "-"
	}
	votesInfo["last_time"], err = DB.Level2.GetVotesLastTime()
	if err != nil {
		votesInfo["last_time"] = "-"
	}
	info["votes"] = votesInfo

	usersInfo := map[string]interface{}{}
	usersInfo["count"], err = DB.Level2.GetUsersCount()
	if err != nil {
		usersInfo["count"] = "-"
	}
	info["users"] = usersInfo

	followsInfo := map[string]interface{}{}
	followsInfo["count"], err = DB.Level2.GetFollowsCount()
	if err != nil {
		followsInfo["count"] = "-"
	}
	info["follows"] = followsInfo
/*
	users, err := DB.GetNewUsersListLast(10)
	if err != nil {
		app.Error.Printf("Error when get last user: %s", err)
		resp["error"] = err.Error()
	}

	listNames := make([]string, 0)
	for _, user := range *users {
		listNames = append(listNames, user.Name)
	}

	if len(*users) > 0 {
		blockchain.SyncUser(&Config.RPC, listNames)
	}
*/
	resp["info"] = info
	DoJSONResponse(&w, &resp, nil)
	return
}