package actions

import (
	"gitlab.com/stihi/stihi-backend/blockchain"
	"net/http"
	"strings"
	"time"

	"github.com/UncleAndy/cyberway-go"
	"github.com/UncleAndy/cyberway-go/token"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
)

const (
	multValueForConvertGOLOS = float64(1000000.0)
)

/*
Параметры:
login - логин отправителя
password - пароль или активный ключ отправителя
target - имя получателя (логин)
value - сумма перевода
unit - единица перевода
*/
func WalletSendTokens(w http.ResponseWriter, r *http.Request) {
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

	// Проверяем параметры запроса
	if !IsParamType(&w, "login", params["login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "login", "string"), nil)
		return
	}
	if !IsParamType(&w, "password", params["password"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "password", "string"), nil)
		return
	}
	if !IsParamType(&w, "target", params["target"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "target", "string"), nil)
		return
	}
	if !IsParamType(&w, "value", params["value"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "value", "numeric"), nil)
		return
	}
	if !IsParamType(&w, "unit", params["unit"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "unit", "string"), nil)
		return
	}
	login := params["login"].(string)
	activeKey := params["password"].(string)

	// Проверяем не является-ли activeKey паролем
	if activeKey[0] == 'P' {
		// Если пароль - полный пароль - получаем приватный ключ из пароля
		// Получаем вместо login cyber имя из golos
		golosLogin := DB.GetGolosLogin(login)
		activeKey = blockchain.GetPrivateKey(golosLogin, "active", activeKey)
	}

	// Провеяем величину отправляемых GOLOS
	value := params["value"].(float64)
	if value <= 0 {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "value", "positive numeric"), nil)
		return
	}

	target := params["target"].(string)
	unit := params["unit"].(string)

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

	var balance float64
	switch unit {
	case "GOLOS":
		balance = userInfo.ValGolos
	case "CYBER":
		balance = userInfo.ValCyber
	}
	if balance < value {
		err = errors_l10n.New(lang, "transfer.not_enough_money")
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
			target,
			cyberway.Asset{
				Amount: cyberway.Int64(value * 1000),
				Symbol: cyberway.Symbol{
					Precision: 3,
					Symbol:    unit,
				},
			},
			"Transfer",
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

	resp["op_resp"] = response

	// blockchain.SyncUserApi(api, trans, []string{login})
	DoJSONResponse(&w, &resp, nil)
}

/*
Параметры:
login - логин пользователя
password - полный пароль
*/
func WalletShowPrivateKeys(w http.ResponseWriter, r *http.Request) {
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

	// Проверяем параметры запроса
	if !IsParamType(&w, "login", params["login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "login", "string"), nil)
		return
	}
	if !IsParamType(&w, "password", params["password"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "password", "string"), nil)
		return
	}
	login := strings.Trim(params["login"].(string), " \n\r\t")
	password := strings.Trim(params["password"].(string), " \n\r\t")

	if login == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
		return
	}
	if len(password) < 5 {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
		return
	}

	// Получаем вместо login cyber имя из golos
	login = DB.GetGolosLogin(login)

	resp["posting"] = blockchain.GetPrivateKey(login, "posting", password)
	resp["active"] = blockchain.GetPrivateKey(login, "active", password)
	resp["owner"] = blockchain.GetPrivateKey(login, "owner", password)
	resp["memo"] = blockchain.GetPrivateKey(login, "memo_key", password)

	DoJSONResponse(&w, &resp, nil)
}

/*
Параметры:
login - логин отправителя
password - пароль или активный ключ отправителя
target - имя получателя (логин) (если пустое - target = login)
value - сумма перевода
*/
func WalletConvertGolosToPower(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	// На время разработки
	DoJSONError(&w, errors_l10n.New("ru", "info.under_contruction"), nil)
	return
/*
	lang := SetLang(r)

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	var target string
	if !IsParamType(&w, "login", params["login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "login", "string"), nil)
		return
	}
	login := params["login"].(string)

	if !IsParamType(&w, "password", params["password"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "password", "string"), nil)
		return
	}
	if IsParamType(&w, "target", params["target"], "string") {
		target = params["target"].(string)
	} else {
		target = login
	}
	if !IsParamType(&w, "value", params["value"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "value", "numeric"), nil)
		return
	}
	activeKey := params["password"].(string)

	// Проверяем не является-ли activeKey паролем
	if activeKey[0] == 'P' {
		// Если пароль - полный пароль - получаем приватный ключ из пароля
		activeKey = client.GetPrivateKey(login, "active", activeKey)
	}

	// Провеяем величину отправляемых GOLOS
	value := params["value"].(float64)
	if value <= 0 {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "value", "positive numeric"), nil)
		return
	}

	// Проверяем баланс пользователя
	blockchain.SyncUser(&Config.RPC, []string{ login })
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

	balance := userInfo.ValGolos
	if balance < value {
		err = errors_l10n.New(lang, "transfer.not_enough_money")
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}

	// Выполняем оплату
	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	api.SetKeys(&client.Keys{
		AKey: []string{ activeKey },
	})

	amount := types.Asset{
		Amount: value,
		Symbol: "GOLOS",
	}
	resp["op_resp"], err = api.TransferToVesting(login, target, &amount)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// cache_level1.DB.ChangeUserBalances(login, int64( -value * cache_level2.FinanceSaveIndex ), 0, 0)
	trans, err := cache_level1.DB.StartTransaction()
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	blockchain.SyncUserApi(api, trans, []string{login})
	err = trans.CommitTransaction()
	if err != nil {
		app.Error.Println(err)
	}

	DoJSONResponse(&w, &resp, nil)
*/
}

/*
Параметры:
login - логин отправителя
password - пароль или активный ключ отправителя
target - имя получателя (логин) (если пустое - target = login)
value - сумма перевода (GOLOS)
*/
func WalletConvertPowerToGolos(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	// На время разработки
	DoJSONError(&w, errors_l10n.New("ru", "info.under_contruction"), nil)
	return
/*
	lang := SetLang(r)

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	var target string
	if !IsParamType(&w, "login", params["login"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "login", "string"), nil)
		return
	}
	login := params["login"].(string)

	if !IsParamType(&w, "password", params["password"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "password", "string"), nil)
		return
	}
	if IsParamType(&w, "target", params["target"], "string") {
		target = params["target"].(string)
	} else {
		target = login
	}
	if !IsParamType(&w, "value", params["value"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "value", "numeric"), nil)
		return
	}
	activeKey := params["password"].(string)

	// Проверяем не является-ли activeKey паролем
	if activeKey[0] == 'P' {
		// Если пароль - полный пароль - получаем приватный ключ из пароля
		activeKey = client.GetPrivateKey(login, "active", activeKey)
	}

	// Провеяем величину отправляемых GOLOS (power)
	value := params["value"].(float64)
	if value < 0 {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "value", "positive or zero numeric"), nil)
		return
	}

	// Подключаемся к ноде
	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	// Значение перевода в GESTS
	var valueGESTS float64
	if value == 0.0 {
		valueGESTS = 0.0
	} else {
		golosPerVests, err := blockchain.GolosPerVestsMult(api, multValueForConvertGOLOS)
		if err != nil {
			app.Error.Printf(err.Error())
			DoJSONError(&w, err, nil)
			return
		}
		valueGESTS = value * golosPerVests / multValueForConvertGOLOS

		app.Debug.Printf("GOLOS2GESTS_API: %f GOLOS -> %f GESTS", value, valueGESTS)

		// Проверяем баланс пользователя
		blockchain.SyncUser(&Config.RPC, []string{ login })
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

		balance := float64(userInfo.ValPower)
		if balance < valueGESTS {
			err = errors_l10n.New(lang, "transfer.not_enough_money")
			app.Error.Print(err)
			DoJSONError(&w, err, nil)
			return
		}
	}

	api.SetKeys(&client.Keys{
		AKey: []string{ activeKey },
	})

	// Режим перевода
	resp["op_resp"], err = api.SetWithdrawVestingRoute(login, target, 100, true)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Перевод
	amount := types.Asset{
		Amount: valueGESTS,
		Symbol: "GESTS",
	}
	resp["op_resp"], err = api.WithdrawVesting(login, &amount)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// cache_level1.DB.ChangeUserBalances(login, 0, 0, int64( -valueGESTS * cache_level2.FinanceSaveIndex ))
	trans, err := cache_level1.DB.StartTransaction()
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	blockchain.SyncUserApi(api, trans, []string{login})
	err = trans.CommitTransaction()
	if err != nil {
		app.Error.Println(err)
	}

	DoJSONResponse(&w, &resp, nil)
*/
}

// Возвращает информацию о текущем понижении СГ
/*
ПРЕДПОЛОЖЕНИЕ:
to_withdraw - общее количество текущих выводимых GESTS (*1000000)
vesting_withdraw_rate - сколько выводится GESTS за один вывод (либо курс GESTS/GOLOS на момент начала снижения СГ)
withdrawn - сколько уже выведено GESTS в текущем выводе
next_vesting_withdrawal - время следующего вывода
*/
func WalletGetWithdrawInfo(w http.ResponseWriter, r *http.Request) {
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

	account, err := cache_level1.DB.GetUserAccountByName(login)
	if account == nil {
		err = errors_l10n.New(lang, "info.data_absent")
		DoJSONError(&w, err, nil)
		return
	}

	// Подключаемся к ноде
	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	golosPerVests, err := blockchain.GolosPerVestsMult(api, multValueForConvertGOLOS)
	if err != nil {	app.Error.Println(err) }

	totalWithdraw := float64( *account.ToWithdraw / 1000000.0 ) * golosPerVests / multValueForConvertGOLOS
	currentWithdraw := float64( *account.Withdrawn / 1000000.0 ) * golosPerVests / multValueForConvertGOLOS

	nextWithdraw := account.NextVestingWithdrawal.Format(cache_level2.TimeJSONFormat)

	resp := map[string]interface{}{ "status": "ok" }
	resp["info"] = map[string]interface{}{
		"total_withdraw_golos": int64(totalWithdraw * cache_level2.FinanceSaveIndex),
		"current_withdraw_golos": int64(currentWithdraw * cache_level2.FinanceSaveIndex),
		"next_withdraw_time": nextWithdraw,
	}

	DoJSONResponse(&w, &resp, nil)
*/
}

// Возвращает историю операций с финансами
func WalletGetHistory(w http.ResponseWriter, r *http.Request) {
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

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	if !IsParamType(&w, "offset", params["offset"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "offset", "numeric"), nil)
		return
	}
	if !IsParamType(&w, "count", params["count"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "count", "numeric"), nil)
		return
	}
	offset := int64(params["offset"].(float64))
	count := int64(params["count"].(float64))

	history, err := DB.GetUserHistory(login, offset, count)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Подключаемся к ноде
	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	golosPerVests, err := blockchain.GolosPerVestsMult(api, multValueForConvertGOLOS)
	if err != nil {
		app.Error.Printf(err.Error())
	} else {
		// Конвертируем GESTS в GOLOS
		for i := 0; i < len(*history); i++ {
			record := (*history)[i]
			record.PowerChangeGolos = float64(record.PowerChange) * golosPerVests / multValueForConvertGOLOS
		}
	}

	resp["history"] = history

	DoJSONResponse(&w, &resp, nil)
*/
}

func WalletRefreshBalance(w http.ResponseWriter, r *http.Request) {
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
	login := (*claims)["n"].(string)

	resp := map[string]interface{}{"status": "ok"}

	DB.SyncUsersByNames([]string{ login })

	DoJSONResponse(&w, &resp, nil)
}