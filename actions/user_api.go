package actions

import (
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	resp := map[string]interface{}{"status": "ok"}

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	idPresent := IsParamType(&w, "id", params["id"], "float64")
	namePresent := IsParamType(&w, "name", params["name"], "string")

	if !(idPresent || namePresent)  {
		err = errors_l10n.New(lang, "parameters.should_be", "id/name", "numeric/string")
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	var userInfo *cache_level2.UserInfo
	var id int64
	if idPresent {
		id = int64(params["id"].(float64))
	} else {
		name := params["name"].(string)
		id, err = DB.GetUserId(name)
		if err != nil {
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}
		if id == -1 {
			DB.SyncUsersByNames([]string{name})
			id, err = DB.GetUserId(name)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		}
	}
	userInfo, err = DB.GetUserInfo(id)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	if userInfo != nil {
		DB.SyncUsersByNames([]string{userInfo.Name})
	}

	resp["user"] = userInfo

	DoJSONResponse(&w, &resp, nil)
}

func UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	resp := map[string]interface{}{"status": "ok"}

	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	userId := int64((*claims)["sub"].(float64))

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	userInfo := cache_level2.UserInfo{}
	userInfo.Id = userId
	if IsParamType(&w, "nickname", params["nickname"], "string") {
		userInfo.NickName = params["nickname"].(string)
	}
	if IsParamType(&w, "biography", params["biography"], "string") {
		userInfo.Biography = params["biography"].(string)
	}
	if IsParamType(&w, "birthdate", params["birthdate"], "string") {
		bd, _ := time.Parse("2006-01-02", params["birthdate"].(string))
		nt := cache_level2.NullTime{ Time: bd }
		userInfo.BirthDate = nt.Format()
	}

	if IsParamType(&w, "sex", params["sex"], "string") {
		userInfo.Sex = params["sex"].(string)
	}
	if IsParamType(&w, "place", params["place"], "string") {
		userInfo.Place = params["place"].(string)
	}
	if IsParamType(&w, "web_site", params["web_site"], "string") {
		userInfo.WebSite = params["web_site"].(string)
	}
	if IsParamType(&w, "avatar", params["avatar"], "string") {
		userInfo.AvatarImage = params["avatar"].(string)
	}
	if IsParamType(&w, "background_image", params["background_image"], "string") {
		userInfo.BackgroundImage = params["background_image"].(string)
	}
	if IsParamType(&w, "pvt_posts_show_mode", params["pvt_posts_show_mode"], "string") {
		userInfo.PvtPostsShowMode = params["pvt_posts_show_mode"].(string)
	}

	err = DB.UpdateUserInfo(&userInfo)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	DoJSONResponse(&w, &resp, nil)
}

func GetUsersList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{"status": "ok"}

	lang := SetLang(r)

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "type", params["type"], "string") {
		err = errors_l10n.New(lang, "parameters.should_be", "type", "string")
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	filter := ""
	if IsParamType(&w, "filter", params["filter"], "string") {
		filter = params["filter"].(string)
	}

	if filter == "banned" {
		// Only for admins
		claims, _, err := jwt.Check(r)
		if err != nil {
			app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
			DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
			return
		}
		userRole := (*claims)["r"].(string)
		if !strings.Contains(userRole, "a") {
			err = errors_l10n.New(lang, "authorize.admin_required")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}
	}

	// Выдаем порцию следующих статей после указанной
	_, ok := params["after_user"]
	if ok {
		if !IsParamType(&w, "after_user", params["after_user"], "float64") {
			err = errors_l10n.New(lang, "parameters.should_be", "after_user", "number")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}
		if !IsParamType(&w, "count", params["count"], "float64") {
			err = errors_l10n.New(lang, "parameters.should_be", "count", "number")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}

		lastUserId := int64(params["after_user"].(float64))
		countF, ok := params["count"].(float64)
		if !ok {
			err = errors_l10n.New(lang, "parameters.wrong_set")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}
		count := int(countF)

		var list *[]*cache_level2.UserInfo

		switch params["type"].(string) {
		case "new", "":
			list, err = DB.GetNewUsersListAfter(lastUserId, count, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		case "name":
			list, err = DB.GetNameUsersListAfter(lastUserId, count, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		case "nickname":
			list, err = DB.GetNicknameUsersListAfter(lastUserId, count, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		}

		// Set user names
		for _, u := range *list {
			DB.GetUserNames(&u.User)
		}

		resp["list"] = list

		DoJSONResponse(&w, &resp, nil)
		return
	}


	// Выдаем все новые статьи перед указанной
	_, ok = params["before_user"]
	if ok {
		if !IsParamType(&w, "before_user", params["before_user"], "float64") {
			err = errors_l10n.New(lang, "parameters.should_be", "before_user", "number")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}

		firstUserId := int64(params["before_user"].(float64))

		var list *[]*cache_level2.UserInfo
		switch params["type"].(string) {
		case "new", "":
			list, err = DB.GetNewUsersListBefore(firstUserId, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		case "name":
			list, err = DB.GetNameUsersListBefore(firstUserId, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		case "nickname":
			list, err = DB.GetNicknameUsersListBefore(firstUserId, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		}

		// Set user names
		for _, u := range *list {
			DB.GetUserNames(&u.User)
		}

		resp["list"] = list

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Выдаем указанное количество последних статей
	_, ok = params["count"]
	if ok {
		if !IsParamType(&w, "count", params["count"], "float64") {
			err = errors_l10n.New(lang, "parameters.should_be", "count", "number")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}

		count := int(params["count"].(float64))

		var list *[]*cache_level2.UserInfo
		switch params["type"].(string) {
		case "new", "":
			list, err = DB.GetNewUsersListLast(count, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		case "name":
			list, err = DB.GetNameUsersListLast(count, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		case "nickname":
			list, err = DB.GetNicknameUsersListLast(count, filter)
			if err != nil {
				app.Error.Print(err)
				DoJSONError(&w, err, nil)
				return
			}
		}

		// Set user names
		for _, u := range *list {
			DB.GetUserNames(&u.User)
		}

		resp["list"] = list

		DoJSONResponse(&w, &resp, nil)
		return
	}

	// Если нет правильной комбинации параметров, выходим с ошибкой
	err = errors_l10n.New(lang, "parameters.wrong_set")
	app.Error.Print(err)
	DoJSONError(&w, err, nil)
	return
}

func GetUserTagsList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	resp := map[string]interface{}{"status": "ok"}

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "id", params["id"], "float64") {
		err = errors_l10n.New(lang, "parameters.should_be", "id", "numeric")
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}
	id := int64(params["id"].(float64))

	list, err := DB.GetTagsForUser(id)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}

// Вычисление батарейки для пользователя
func GetUserBattery(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{"status": "ok"}

	lang := SetLang(r)

	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)

	currentBat, err := DB.GetUserBatteryNodeos(login)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	resp["value"] = strconv.FormatFloat(currentBat, 'f', 2, 64)

	DoJSONResponse(&w, &resp, nil)
}

func GetUsersPeriodLeader(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{"status": "ok"}

	lang := SetLang(r)

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "days", params["days"], "float64") {
		err = errors_l10n.New(lang, "parameters.should_be", "days", "number")
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}
	days := int(params["days"].(float64))

	userId, err := DB.GetUserPeriodLeader(days)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	if userId > 0 {
		userInfo, err := DB.GetUserInfo(userId)
		if err != nil {
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}

		resp["leader"] = userInfo
	}

	DoJSONResponse(&w, &resp, nil)
}
