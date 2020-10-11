package actions

import (
	"encoding/hex"
	"encoding/json"
	"github.com/UncleAndy/cyberway-go"
	"github.com/UncleAndy/cyberway-go/token"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/blockchain"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"net/http"
	"strings"
	"time"
)

const (
	AnnounceCheckExistsCount = 3
)

func GetAnnouncePagesList(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	list, err := DB.GetAnnouncesPages()
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}
	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}

func CreateAnnounce(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "login", params["login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "login", "string"), nil)
		return
	}
	login := params["login"].(string)

	if !IsParamType(&w, "article_id", params["article_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "article_id", "numeric"), nil)
		return
	}
	articleId := int64(params["article_id"].(float64))

	if !IsParamType(&w, "page_code", params["page_code"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "page_code", "string"), nil)
		return
	}
	pageCode := params["page_code"].(string)

	if !IsParamType(&w, "active_key", params["active_key"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "active_key", "string"), nil)
		return
	}
	activeKey := params["active_key"].(string)

	count := AnnounceCheckExistsCount
	if IsParamType(&w, "count", params["count"], "float64") {
		count = int(params["count"].(float64))
	}

	// Проверяем не является-ли activeKey паролем
	if activeKey[0] == 'P' {
		// Если пароль - полный пароль - получаем приватный ключ из пароля
		golosLogin := DB.GetGolosLogin(login)
		activeKey = blockchain.GetPrivateKey(golosLogin, "active", activeKey)
	}

	// Проверяем есть-ли уже на данной странице анонса анонс данного произведения. Если есть - выдаем ошибку.
	list, err := DB.GetAnnouncesList(pageCode, count, false, false)
	if err == nil && list != nil && len(*list) > 0 {
		for _, announce := range *list {
			if announce.Id == articleId {
				DoJSONError(&w, errors_l10n.New(lang, "announce.already_exists"), nil)
				return
			}
		}
	}

	// Получаем прайс для данной страницы
	price, err := DB.GetAnnouncePage(pageCode)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	app.Debug.Printf("DBG: announce prices: %+v\n", price)

	// Проверяем баланс пользователя
	DB.SyncUsersByNames([]string{ login })
	userId, err := DB.GetUserId( login )
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}
	userInfo, err := DB.GetUserInfo(userId)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}
	if userInfo == nil {
		err = errors_l10n.New(lang, "users.absent_id")
		DoJSONError(&w, err, nil)
		return
	}

	// Используем cyberId
	login = userInfo.Name

	balance := float64(0)
	switch price.Unit {
	case "GOLOS":
		balance = userInfo.ValGolos
	case "CYBER":
		balance = userInfo.ValCyber
	}
	if balance < float64(price.Price) {
		err = errors_l10n.New(lang, "announce.not_enough_money")
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	// Выполняем оплату
	api := cyberway.New(Config.RPC.BaseURL("http"))

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(activeKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	api.SetSigner(keyBag)

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	tx := cyberway.NewTransaction([]*cyberway.Action{
		token.NewTransfer(
			login,
			Config.Golos.PaymentsTo,
			cyberway.Asset{
				Amount: cyberway.Int64(price.Price * 1000),
				Symbol: cyberway.Symbol{
					Precision: 3,
					Symbol:    price.Unit,
				},
			},
			"Announce payment",
		)},
		txOpts,
	)

	tx.Expiration.Time = time.Now().UTC().Add(1800 * time.Second)

	_, packedTx, err := api.SignTransaction(tx, txOpts.ChainID, cyberway.CompressionNone)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	response, err := api.PushTransaction(packedTx)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	app.Info.Printf("Transaction [%s] submitted to the network succesfully.\n", hex.EncodeToString(response.Processed.ID))

	payData, _ := json.Marshal(response)

	trans, err := cache_level1.DB.StartTransaction()
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	err = trans.CreateAnnounce( pageCode, articleId, login, string(payData) )
	if err != nil {
		app.Error.Printf(err.Error())
	}

	err = trans.CommitTransaction()
	if err != nil {
		app.Error.Println(err)
	}

	DoJSONResponse(&w, &resp, nil)
}

func GetAnnouncesList(w http.ResponseWriter, r *http.Request) {
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
			app.Error.Print(err)
		}
		if userInfo != nil && strings.Trim(userInfo.PvtPostsShowMode, " ") != "" {
			notMat = true
		}
	}
	if !authorized {
		notMat = true
	}

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "page_code", params["page_code"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "page_code", "string"), nil)
		return
	}
	pageCode := params["page_code"].(string)

	if !IsParamType(&w, "count", params["count"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
		return
	}
	count := int(params["count"].(float64))

	list, err := DB.GetAnnouncesList(pageCode, count, notMat, adminMode)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	lenList := 0
	if list != nil {
		lenList = len(*list)
	}

	contentIds := make([]int64, lenList)
	if lenList > 0 {
		for idx, content := range *list {
			contentIds[idx] = content.Id
		}
	}

	// All votes
	votes, err := DB.GetVotesForContentList(&contentIds)
	if err == nil {
		resp["votes"] = votes
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
}
