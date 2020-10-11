package actions

import (
	"net/http"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	)

func BanUser(w http.ResponseWriter, r *http.Request) {
	DoBanUnban("users", true, &w, r)
}

func UnbanUser(w http.ResponseWriter, r *http.Request) {
	DoBanUnban("users", false, &w, r)
}

func BanContent(w http.ResponseWriter, r *http.Request) {
	DoBanUnban("content", true, &w, r)
}


func UnbanContent(w http.ResponseWriter, r *http.Request) {
	DoBanUnban("content", false, &w, r)
}

func DoBanUnban(tableName string, ban bool, w *http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		app.Error.Printf(errors_l10n.New(lang, "authorize.required").Error())
		DoJSONError(w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	role := (*claims)["r"].(string)
	if role != "a" {
		DoJSONError(w, errors_l10n.New(lang, "authorize.admin_required"), nil)
		return
	}

	login := (*claims)["n"].(string)

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(w, err, nil)
		return
	}

	if !IsParamType(w, "id", params["id"], "float64") {
		DoJSONError(w, errors_l10n.New(lang, "parameters.should_be", "id", "number"), nil)
		return
	}
	id := int64(params["id"].(float64))

	description := ""
	if IsParamType(w, "description", params["description"], "string") {
		description = params["description"].(string)
	}

	err = DB.DoBanUnban(tableName, ban, id, login, description)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(w, err, nil)
		return
	}

	DoJSONResponse(w, &resp, nil)
}
