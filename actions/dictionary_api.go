package actions

import (
	"net/http"
	)

func GetRubricsList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	list, err := DB.GetRubrics()
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
	return
}
