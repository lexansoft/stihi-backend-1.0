package actions

import (
	"encoding/json"
	"github.com/UncleAndy/cyberway-go"
	"github.com/UncleAndy/cyberway-go/token"
	"gitlab.com/stihi/stihi-backend/blockchain"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"net/http"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"time"
)

const (
	InviteCheckExistsCount = 3
	InvitePriceValue = 20
	InvitePriceUnit = "GOLOS"
)

func CreateInvite(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "author_login", params["author_login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "author_login", "string"), nil)
		return
	}
	authorLogin := params["author_login"].(string)

	if !IsParamType(&w, "login", params["login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "login", "string"), nil)
		return
	}
	login := params["login"].(string)

	if !IsParamType(&w, "active_key", params["active_key"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "active_key", "string"), nil)
		return
	}
	activeKey := params["active_key"].(string)

	count := InviteCheckExistsCount
	if IsParamType(&w, "count", params["count"], "float64") {
		count = int(params["count"].(float64))
	}

	// Проверяем не является-ли activeKey паролем
	if activeKey[0] == 'P' {
		// Если пароль - полный пароль - получаем приватный ключ из пароля
		golosLogin := DB.GetGolosLogin(login)
		activeKey = blockchain.GetPrivateKey(golosLogin, "active", activeKey)
	}

	user, err := DB.GetUserInfoByName(authorLogin)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем есть-ли уже в показе данной странице анонса анонс данного произведения. Если есть - выдаем ошибку.
	list, err := DB.GetInvitesList( count )
	if err == nil && list != nil && len(*list) > 0 {
		for _, invite := range *list {
			if invite.Id == user.Id {
				DoJSONError(&w, errors_l10n.New(lang, "invite.already_exists"), nil)
				return
			}
		}
	}

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

	login = userInfo.Name

	balance := float64(0)
	switch InvitePriceUnit {
	case "GOLOS":
		balance = userInfo.ValGolos
	case "CYBER":
		balance = userInfo.ValCyber
	}
	if balance < InvitePriceValue {
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
				Amount: cyberway.Int64(InvitePriceValue * 1000),
				Symbol: cyberway.Symbol{
					Precision: 3,
					Symbol:    InvitePriceUnit,
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

	payData, _ := json.Marshal(response)

	trans, err := cache_level1.DB.StartTransaction()
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	err = trans.CreateInvite( authorLogin, string(payData) )
	if err != nil {
		app.Error.Printf(err.Error())
	}

	balanceChange := cache_level2.Balance{}
	switch InvitePriceUnit {
	case "GOLOS":
		balanceChange.Golos = -InvitePriceValue * cache_level2.FinanceSaveIndex
	case "CYBER":
		balanceChange.Cyber = -InvitePriceValue * cache_level2.FinanceSaveIndex
	}
	err = trans.ChangeUserBalances(login, balanceChange)
	if err != nil {
		app.Error.Println(err)
	}

	err = trans.CommitTransaction()
	if err != nil {
		app.Error.Println(err)
	}
	DoJSONResponse(&w, &resp, nil)
}

func GetInvitesList(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "count", params["count"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "number"), nil)
		return
	}
	count := int(params["count"].(float64))

	list, err := DB.GetInvitesList( count )
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	resp["list"] = list

	DoJSONResponse(&w, &resp, nil)
}
