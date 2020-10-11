package actions

import (
	"bytes"
	"encoding/base64"
	"gitlab.com/stihi/stihi-backend/app"
	"io/ioutil"
	"net/http"
	"net/url"
)

func SharepayGetShareCount(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{"status": "ok"}

	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	u := params["url"].(string)

	form := url.Values{
		"u": 		{u},
		"spid":  	{Config.Sharpay.SPID},
		"scm":  	{"page"},
	}

	app.Info.Printf("Sharepay request: %+v\n", form)

	body := bytes.NewBufferString(form.Encode())
	rsp, err := http.Post(Config.Sharpay.URL, "application/x-www-form-urlencoded", body)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}
	defer rsp.Body.Close()

	respBody, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	respVal, err := base64.StdEncoding.DecodeString(string(respBody))
	if err != nil {
		app.Error.Println(err)
		resp["result"] = string(respBody)
	} else {
		resp["result"] = string(respVal)
	}

	DoJSONResponse(&w, &resp, nil)
}
