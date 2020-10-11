package actions

import (
	"net/http"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
)

func GetUserSubscriptionsList(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "user_id", params["user_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
		return
	}
	userId := int64(params["user_id"].(float64))

	list, err := DB.GetUserFollowsList(userId)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}

func GetUserSubscribersList(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "user_id", params["user_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
		return
	}
	userId := int64(params["user_id"].(float64))

	list, err := DB.GetUserFollowersList(userId)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}

func GetUserBlocked(w http.ResponseWriter, r *http.Request) {
}

func UserSubscribe(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	// На время разработки
	DoJSONError(&w, errors_l10n.New("ru", "info.under_contruction"), nil)
	return
	/*

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	if err != nil {
		app.Error.Printf("Posting key decryption error: "+err.Error())
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "user_id", params["user_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
		return
	}
	userId := int64(params["user_id"].(float64))
	userName, err := DB.GetUserNameById(userId)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	api.SetKeys(&client.Keys{
		PKey: []string{ postingPvtKey },
	})

	oldAsync := api.AsyncProtocol
	api.AsyncProtocol = true
	followResp, err := api.Follows(login, userName, "blog")
	if err != nil {
		app.Error.Printf("Add follow error: %s\n%+v", err, followResp)
		DoJSONError(&w, err, nil)
		return
	}
	api.AsyncProtocol = oldAsync

	op := types.FollowOperation{
		Follower: 	login,
		Following: 	userName,
		What:		[]string{"blog"},
	}
	_, err = DB.SaveFollowFromOperation(&op, time.Now().UTC())
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	DoJSONResponse(&w, &resp, nil)
	*/
}

func UserUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	// На время разработки
	DoJSONError(&w, errors_l10n.New("ru", "info.under_contruction"), nil)
	return

	/*

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	if err != nil {
		app.Error.Printf("Posting key decryption error: "+err.Error())
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "user_id", params["user_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
		return
	}
	userId := int64(params["user_id"].(float64))
	userName, err := DB.GetUserNameById(userId)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	api.SetKeys(&client.Keys{
		PKey: []string{ postingPvtKey },
	})

	oldAsync := api.AsyncProtocol
	api.AsyncProtocol = true
	followResp, err := api.Follows(login, userName, "")
	if err != nil {
		app.Error.Printf("Add follow error: %s\n%+v", err, followResp)
		DoJSONError(&w, err, nil)
		return
	}
	api.AsyncProtocol = oldAsync

	op := types.FollowOperation{
		Follower: 	login,
		Following: 	userName,
		What:		[]string{""},
	}
	_, err = DB.SaveFollowFromOperation(&op, time.Now().UTC())
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	DoJSONResponse(&w, &resp, nil)
	*/
}

func UserIgnore(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	// На время разработки
	DoJSONError(&w, errors_l10n.New("ru", "info.under_contruction"), nil)
	return
/*
	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	if err != nil {
		app.Error.Printf("Posting key decryption error: "+err.Error())
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "user_id", params["user_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "user_id", "number"), nil)
		return
	}
	userId := int64(params["user_id"].(float64))
	userName, err := DB.GetUserNameById(userId)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	api.SetKeys(&client.Keys{
		PKey: []string{ postingPvtKey },
	})

	oldAsync := api.AsyncProtocol
	api.AsyncProtocol = true
	followResp, err := api.Follows(login, userName, "ignore")
	if err != nil {
		app.Error.Printf("Add follow error: %s\n%+v", err, followResp)
		DoJSONError(&w, err, nil)
		return
	}
	api.AsyncProtocol = oldAsync

	op := types.FollowOperation{
		Follower: 	login,
		Following: 	userName,
		What:		[]string{"ignore"},
	}
	_, err = DB.SaveFollowFromOperation(&op, time.Now().UTC())
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}
	DoJSONResponse(&w, &resp, nil)
*/
}

func UserUnignore(w http.ResponseWriter, r *http.Request) {
	UserUnsubscribe(w, r)
}
