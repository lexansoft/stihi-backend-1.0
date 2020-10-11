package actions

import (
	"net/http"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	)

func UpdateFixPage(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	role := (*claims)["r"].(string)
	if role != "a" {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.admin_required"), nil)
		return
	}
	login := (*claims)["n"].(string)

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "code", params["code"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "code", "string"), nil)
		return
	}
	if !IsParamType(&w, "html", params["html"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "html", "text"), nil)
		return
	}
	if !IsParamType(&w, "title", params["title"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "title", "string"), nil)
		return
	}
	code := params["code"].(string)
	html := params["html"].(string)
	title := params["title"].(string)

	err = DB.UpdateFixPage(code, html, title, login)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	DoJSONResponse(&w, &resp, nil)
}

func GetFixPage(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "code", params["code"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "code", "string"), nil)
		return
	}
	code := params["code"].(string)

	page, err := DB.GetFixPage(code)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	resp["page"] = page

	DoJSONResponse(&w, &resp, nil)
}

func GetFixPagesList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	role := (*claims)["r"].(string)
	if role != "a" {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.admin_required"), nil)
		return
	}

	resp := map[string]interface{}{"status": "ok"}

	list, err := DB.GetFixPagesList()
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}
