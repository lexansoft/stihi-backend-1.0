package actions

import (
	"net/http"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"strings"
)

func GetExchangeRates(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "to", params["to"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "to", "string"), nil)
		return
	}
	to := strings.ToUpper(params["to"].(string))

	var result map[string]float64
	switch to {
	case "GBG":
		result = map[string]float64{
			"GBG": 1.0,
			"GOLOS": 1.0,
		}
	case "GOLOS":
		result = map[string]float64{
			"GBG": 1.0,
			"GOLOS": 1.0,
		}
	case "USD":
		result = map[string]float64{
			"GBG": 0.039,
			"GOLOS": 0.039,
		}
	case "EUR":
		result = map[string]float64{
			"GBG": 0.03,
			"GOLOS": 0.03,
		}
	default:
		result = map[string]float64{
			"GBG": 0.0,
			"GOLOS": 0.0,
		}
	}

	resp := make(map[string]interface{})
	resp["status"] = "ok"
	resp["rate"] = result

	DoJSONResponse(&w, &resp, nil)
}
