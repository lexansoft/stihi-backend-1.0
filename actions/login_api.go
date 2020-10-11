package actions

import (
	cyberway "github.com/UncleAndy/cyberway-go"
	"github.com/UncleAndy/cyberway-go/domain"
	"github.com/UncleAndy/cyberway-go/ecc"
	"github.com/UncleAndy/cyberway-go/system"
	"github.com/UncleAndy/cyberway-go/token"
	"github.com/dchest/captcha"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/blockchain"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/cyber"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	NewUsersPrefix = "sth1"
)

/*
Действия для раздачи ключу newuser всех прав, нужных для регистрации пользователей

{
  "expiration": "2019-12-27T00:00:00.000",
  "ref_block_num": 0,
  "ref_block_prefix": 0,
  "max_net_usage_words": 0,
  "max_cpu_usage_ms": 0,
  "max_ram_kbytes": 0,
  "max_storage_kbytes": 0,
  "delay_sec": 0,
  "context_free_actions": [],
  "actions":[
	{
      "data": {
        "code": "cyber",
        "type": "newaccount",
        "account": "evdazo52xujg",
        "requirement": "newuser"
      },
      "authorization": [
			{
				"actor":"evdazo52xujg",
				"permission":"active"
			}
	  ],
      "account":"cyber",
	  "name":"linkauth"
    },
	{
      "data": {
        "code": "cyber.token",
        "type": "open",
        "account": "evdazo52xujg",
        "requirement": "newuser"
      },
      "authorization": [
			{
				"actor":"evdazo52xujg",
				"permission":"active"
			}
	  ],
      "account":"cyber",
	  "name":"linkauth"
    },
	{
      "data": {
        "code": "gls.vesting",
        "type": "open",
        "account": "evdazo52xujg",
        "requirement": "newuser"
      },
      "authorization": [
			{
				"actor":"evdazo52xujg",
				"permission":"active"
			}
	  ],
      "account":"cyber",
	  "name":"linkauth"
    }
  ],
  "transaction_extensions": []
}

{
  "expiration": "2019-12-27T00:00:00.000",
  "ref_block_num": 0,
  "ref_block_prefix": 0,
  "max_net_usage_words": 0,
  "max_cpu_usage_ms": 0,
  "max_ram_kbytes": 0,
  "max_storage_kbytes": 0,
  "delay_sec": 0,
  "context_free_actions": [],
  "actions":[
	{
      "data": {
        "code": "cyber",
        "type": "providebw",
        "account": "evdazo52xujg",
        "requirement": "newuser"
      },
      "authorization": [
			{
				"actor":"evdazo52xujg",
				"permission":"active"
			}
	  ],
      "account":"cyber",
	  "name":"linkauth"
    }
  ],
  "transaction_extensions": []
}

{
  "expiration": "2019-12-27T00:00:00.000",
  "ref_block_num": 0,
  "ref_block_prefix": 0,
  "max_net_usage_words": 0,
  "max_cpu_usage_ms": 0,
  "max_ram_kbytes": 0,
  "max_storage_kbytes": 0,
  "delay_sec": 0,
  "context_free_actions": [],
  "actions":[
	{
      "data": {
        "code": "gls.vesting",
        "type": "delegate",
        "account": "evdazo52xujg",
        "requirement": "newuser"
      },
      "authorization": [
			{
				"actor":"evdazo52xujg",
				"permission":"active"
			}
	  ],
      "account":"cyber",
	  "name":"linkauth"
    },
	{
      "data": {
        "code": "gls.vesting",
        "type": "undelegate",
        "account": "evdazo52xujg",
        "requirement": "newuser"
      },
      "authorization": [
			{
				"actor":"evdazo52xujg",
				"permission":"active"
			}
	  ],
      "account":"cyber",
	  "name":"linkauth"
    }
  ],
  "transaction_extensions": []
}


Newuser PUB: GLS6je7BQg5WDFWD1u2MM7doEqxML5Wsc14dagjRbQ1ny2D29mQ5Q
*/


/*
	Логин производим, запрашивая у пользователя его пароль или приватный постинг-ключ и сравнивая сгенерированные публичные ключи с сохраненными.
	Затем приватный постинг-ключ запоминаем в зашифрованном виде в JWT B используем при необходимости на бэкэнде, предварительно расшифровав.
*/
func Login(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "name", params["name"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "name", "string"), nil)
		return
	}
	if !IsParamType(&w, "password", params["password"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "password", "string"), nil)
		return
	}

	login, okL := params["name"].(string)
	password, okP := params["password"].(string)
	if !okL || !okP || password == "" || login == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
		return
	}
	login = strings.Trim(login, " \n\r\t")
	password = strings.Trim(password, " \n\r\t")

	if len(password) < 5 {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.password_should_be_gt", 5), nil)
		return
	}

	type key struct {
		keyType string
		pvt string
		pub string
	}
	var curKey key
	keyTypes := []string{"posting", "active", "owner"}
	privKeys := make([]key, 0, 3)
	if strings.HasPrefix(password, "P") {
		golosLogin := DB.GetGolosLogin(login)
		// Если пароль - полный пароль - получаем приватные ключи из пароля
		for _, kt := range keyTypes {
			pk := blockchain.GetPrivateKey(golosLogin, kt, password)
			privKeys = append(privKeys, key{
				keyType: kt,
				pvt:     pk,
				pub: 	 blockchain.GetPublicKey("GLS", pk),
			})
		}
	} else {
		curKey.pvt = password
	}

	// Получаем из БД пользователя с публичными ключами
	var user, errGetUser = DB.GetUserByName(login)
	if errGetUser != nil {
		app.Error.Printf("Auth error: %s", errGetUser)
		DoJSONError(&w, errGetUser, nil)
		return
	}
	if user == nil {
		app.Error.Printf("Auth error: no user")
		DoJSONError(&w, errors_l10n.New(lang, "authorize.wrong"), nil)
		return
	}

	if len(privKeys) == 0 {
		// Вариант ввода одного приватного ключа
		// по публичному ключу ищем его у пользователя

		curKey.pub = blockchain.GetPublicKey("GLS", curKey.pvt)

		// Проверяем какого типа ключ используется и есть-ли он у пользователя
		for kt, k := range user.Keys {
			if k == curKey.pub {
				curKey.keyType = kt
				break
			}
		}

		// Проверяем что есть такой ключ
		if curKey.keyType == "" {
			app.Error.Printf("Auth error. Public key absent for %s: %s", login, curKey.pub)
			DoJSONError(&w, errors_l10n.New(lang, "authorize.wrong"), nil)
			return
		}
	} else {
		// Определяем у пользователя подходящий публичный ключ в порядке keyTypes
		for i, kt := range keyTypes {
			upk, ok := user.Keys[kt]
			if ok && upk == privKeys[i].pub {
				curKey = privKeys[i]
				break
			}
		}
	}

	if curKey.keyType == "" {
		app.Error.Printf("Auth error. Public key absent for %s: %s", login, curKey.pub)
		DoJSONError(&w, errors_l10n.New(lang, "authorize.wrong"), nil)
		return
	}

	// Формируем JWT и выдаем его в ответе
	role := "u"
	if LoginIsAdmin(login) {
		role = "a"
	}

	encKey, err := jwt.EncryptPrivKey(curKey.pvt)
	app.Debug.Printf("User for JWT: %+v\n")
	token, jwsToken, err := jwt.New(user.Id, user.Name, role, encKey, curKey.keyType)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	_ = DB.SetStihiUser(user.Id)

	resp["token"] = string(token)

	DoJSONResponse(&w, &resp, jwsToken)
}

func Signup(w http.ResponseWriter, r *http.Request) {
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

	if !IsParamType(&w, "name", params["name"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "name", "string"), nil)
		return
	}
	if !IsParamType(&w, "password", params["password"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "password", "string"), nil)
		return
	}

	nickname := ""
	if IsParamType(&w, "nickname", params["nickname"], "string") {
		nickname = params["nickname"].(string)
	}
	email := ""
	if IsParamType(&w, "email", params["email"], "string") {
		email = params["email"].(string)
	}

	if !IsParamType(&w, "captcha_id", params["captcha_id"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "captcha_id", "string"), nil)
		return
	}

	if !IsParamType(&w, "captcha_resolve", params["captcha_resolve"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "captcha_resolve", "string"), nil)
		return
	}

	if !captcha.VerifyString(params["captcha_id"].(string), params["captcha_resolve"].(string)) {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.wrong_captcha"), nil)
		return
	}

	login, ok := params["name"].(string)
	if !ok {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.wrong_set"), nil)
		return
	}

	// Проверяем логин на допустимые символы
	correct, _ := regexp.MatchString(`^[a-zA-Z0-9\-]+$`, login)
	if !correct {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.login_wrong_symbols"), nil)
		return
	}

	// 1. Проверяем что пользователя с данным именем еще не существует
	id, err := DB.GetUserId(login)
	if err != nil && err.Error() != "l10n:info.data_absent" {
		DoJSONError(&w, err, nil)
		return
	}
	err = nil
	if id > 0 {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.already_exists"), nil)
		return
	}

	// 2. Генерируем общий пароль и все ключи
	// 2.1. Проверяем публичные ключи по данному паролю на существование их у других пользователей.
	//      Если существуют - генерируем другой пароль

	password := params["password"].(string)
	if password[0] != 'P' {
		password = "P" + password
	}

	// Создание пользователя

	// Синтетический cyber id пользователя
	var cyberName string
	var isExists bool
	for cyberName == "" || isExists {
		cyberName = cyber.GenCyberUserId(NewUsersPrefix)

		idExists, err := DB.GetUserId(cyberName)
		if idExists == -1 && err == nil {
			isExists = false
		}
	}

	app.Debug.Printf("NewCyberName = %s", cyberName)

	api := cyberway.New(Config.RPC.BaseURL("http"))

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	ownerPubKeyStr := blockchain.GetPublicKey("GLS", blockchain.GetPrivateKey(login, "owner", password))
	ownerPubKey, err := ecc.NewPublicKey(ownerPubKeyStr)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	activePvtKeyStr := blockchain.GetPrivateKey(login, "active", password)
	activePubKeyStr := blockchain.GetPublicKey("GLS", activePvtKeyStr)
	activePubKey, err := ecc.NewPublicKey(activePubKeyStr)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	postingPubKeyStr := blockchain.GetPublicKey("GLS", blockchain.GetPrivateKey(login, "posting", password))
	postingPubKey, err := ecc.NewPublicKey(postingPubKeyStr)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(Config.Golos.CreatorKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	api.SetSigner(keyBag)

	tx := cyberway.NewTransaction([]*cyberway.Action{
		system.NewNewAccount(
			cyberway.AN(Config.Golos.CreatorName),
			cyberway.AN(cyberName),
			ownerPubKey, activePubKey,
			Config.Golos.CreatorPermission ),
		domain.NewNewUserName(
			"gls",
			"gls",
			cyberName,
			login,
			"createuser" ),
		token.NewTokenOpen(
			Config.Golos.CreatorName,
			cyberName,
			cyberway.Symbol{
				Precision: 3,
				Symbol:    "GOLOS",
			},
			Config.Golos.CreatorPermission),
		token.NewVestingOpen(
			Config.Golos.CreatorName,
			cyberName,
			cyberway.Symbol{
				Precision: 6,
				Symbol:    "GOLOS",
			},
			Config.Golos.CreatorPermission),
		},
		txOpts,
	)

	// TODO: Добавить делегирование вестинга 30.000 GOLOS

	/*
	app.Debug.Printf("DBG:\n%+v\n", *tx)
	for _, act := range tx.Actions {
		app.Debug.Printf("DBG:\n%+v\n", *act)
	}
	 */

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

	// Добавляем posting ключ

	keyBag = &cyberway.KeyBag{}
	err = keyBag.Add(activePvtKeyStr)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	err = keyBag.Add(Config.Golos.CreatorKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	err = keyBag.Add(Config.Golos.Delegation.Key)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	api.SetSigner(keyBag)

	txPosting := cyberway.NewTransaction([]*cyberway.Action{
		system.NewProvideBW(
			cyberway.AccountName(cyberName),
			cyberway.AccountName(Config.Golos.CreatorName),
			cyberway.AccountName(Config.Golos.CreatorName),
			cyberway.PermissionName(Config.Golos.CreatorPermission)),
		system.NewUpdateAuth(
			cyberway.AccountName(cyberName),
			"posting",
			"active",
			cyberway.Authority{
				Threshold: 1,
				Keys:      []cyberway.KeyWeight{
					{ PublicKey: postingPubKey,	Weight: 1 },
				},
				Accounts:  []cyberway.PermissionLevelWeight{},
				Waits:     []cyberway.WaitWeight{},
			},
			cyberway.AccountName(cyberName),
			"active"),
		system.NewLinkAuth(
			cyberway.AccountName(cyberName),
			"gls.publish",
			"",
			"posting",
			cyberway.AccountName(cyberName),
			"active"),
		system.NewLinkAuth(
			cyberway.AccountName(cyberName),
			"gls.social",
			"",
			"posting",
			cyberway.AccountName(cyberName),
			"active"),
		token.NewDelegateVesting(
			Config.Golos.Delegation.From,
			cyberName,
			cyberway.Asset{
				Amount: cyberway.Int64(Config.Golos.Delegation.Value * 1000000),
				Symbol: cyberway.Symbol{
					Precision: 6,
					Symbol:    "GOLOS",
				},
			},
			0,
			Config.Golos.Delegation.Permission,
			"active",
			),
		},
		txOpts,
	)

	txPosting.Expiration.Time = time.Now().UTC().Add(1800 * time.Second)

	app.Debug.Printf("DBG:\n%+v\n", *tx)
	for _, act := range txPosting.Actions {
		app.Debug.Printf("DBG:\n%+v\n", *act)
	}

	_, packedTx, err = api.SignTransaction(txPosting, txOpts.ChainID, cyberway.CompressionNone)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	response, err = api.PushTransaction(packedTx)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	// 4. Сохраняем данные о пользователе в БД
	// err = DB.CreateUser(cyberName, login, ownerPubKeyStr, activePubKeyStr, postingPubKeyStr, time.Now().UTC())
	err = DB.CreateUser(cyberName, login, ownerPubKeyStr, activePubKeyStr, "", time.Now().UTC())
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	_ = DB.SetStihiUserByLogin(login)

	// Сохраняем nickname и/или email
	if nickname != "" || email != "" {
		userId, err := DB.GetUserId(login)
		if err != nil {
			app.Error.Print(err)
		} else {
			userInfo := cache_level2.UserInfo{
				User: cache_level2.User{
					Id: userId,
				},
				NickName: nickname,
				Email: email,
			}
			err = DB.UpdateUserInfo(&userInfo)
			if err != nil {
				app.Error.Print(err)
			}
		}
	}

	// 4. Выдаем пользователю сгенерированный пароль и предупреждаем о необходимости его сохранить, т.к. он не восстанавливаемый
	resp["password"] = password

	DoJSONResponse(&w, &resp, nil)
}

func NewCaptcha(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	resp["captcha_id"] = captcha.New()

	DoJSONResponse(&w, &resp, nil)
}

func LoginIsAdmin(login string) bool {
	return login == "stihi-io"
}

func GeneratePassword(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	resp["password"] = blockchain.GenPassword()

	DoJSONResponse(&w, &resp, nil)
}