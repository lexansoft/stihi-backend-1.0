package actions

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/UncleAndy/cyberway-go/system"
	"net/http"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/UncleAndy/cyberway-go"
	"github.com/UncleAndy/cyberway-go/forum"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/filters"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/blockchain"
	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

/*
Георгий Савчук, [12.10.19 16:31]
[В ответ на Andy Vel]
{ "actions":
	[
		{
			"account": "gls.publish",
			"name": "createmssg",
			"authorization": [
				{
					"actor": "jyrnxqhjzjmb",
					"permission": "posting"
				}
			],
			"data": {
				"message_id": {
					"author": "jyrnxqhjzjmb",
					"permlink": "abrakadabrasimsalabim1"
				},
				"parent_id": {
					"author": "",
					"permlink": "test"
				},
				"beneficiaries": [],
				"tokenprop": 5000,
				"vestpayment": false,
				"headermssg": "Привет",
				"bodymssg": "Привет",
				"languagemssg": "",
				"tags": [],
				"jsonmetadata": "{\"app\":\"stihi.io\",\"format\":\"markdown\",\"tags\":[\"test\"]}",
				"curators_prcnt": 5000,
				"max_payout": null
			}
		}
	]
}

Георгий Савчук, [12.10.19 16:34]
import { JsonRpc, Api } from 'cyberwayjs';
import { TextEncoder, TextDecoder } from 'text-encoding';
import JsSignatureProvider from 'cyberwayjs/dist/eosjs-jssig';

const HOST = 'http://localhost:8888';

export async function sendTransaction(keys, trx, host) {
  const rpc = new JsonRpc(HOST);
  const signatureProvider = new JsSignatureProvider(keys);

  const api = new Api({
    rpc,
    signatureProvider,
    textDecoder: new TextDecoder(),
    textEncoder: new TextEncoder(),
  });

  const results = await api.transact(trx, {
    blocksBehind: 5,
    expireSeconds: 3600,
  });

  return results;
}

Георгий Савчук, [12.10.19 16:35]
json выше, скормить функции ниже, sendTransaction
*/

func CreateArticle(w http.ResponseWriter, r *http.Request) {
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
	encPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPvtKey)
	keyType, ok := (*claims)["kpt"]
	if !ok || keyType == nil {
		app.Error.Printf("Key type absent in JWT for '%s'", login)
		DoJSONError(&w, errors.New("Key type absent in JWT"), nil)
		return
	}
	postingPvtKeyType := jwt.DecodeKeyType(keyType.(string))
	if err != nil {
		app.Error.Printf("Posting key decryption error: "+err.Error())
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	// Проверка пользователя на бан. Если бан - возвращаем ошибку.
	user, err := DB.GetUserInfoByName(login)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	if user.Ban {
		app.Error.Printf(errors_l10n.New(lang, "authorize.banned").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.banned"), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	contentParam, ok := params["content"]
	if !ok {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}
	if !IsParamType(&w, "content", params["content"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}

	content := contentParam.(map[string]interface{})

	if !IsParamType(&w, "content/title", content["title"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.title", "string"), nil)
		return
	}
	if !IsParamType(&w, "content/body", content["body"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}
	if !IsParamType(&w, "content/image", content["image"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.image", "string"), nil)
		return
	}
	if !IsParamType(&w, "content/metadata", content["metadata"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content/metadata", "struct"), nil)
		return
	}

	if !IsParamType(&w, "reward_type", params["reward_type"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "reward_type", "numeric:100/50/0"), nil)
		return
	}
	if !IsParamType(&w, "self_vote", params["self_vote"], "bool") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "self_vote", "boolean"), nil)
		return
	}
	/*
	rewardType := uint16(params["reward_type"].(float64))
	selfVote := params["self_vote"].(bool)
	 */
	image := content["image"].(string)

	if image != "" && !filters.IsImageURL(image) {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be_image"), nil)
		return
	}

	tags := getTags(content["metadata"])
	// Проверяем список тэгов и добавляем/переставляем тэг "stihi-io" в начало списка
	if tags == nil || len(tags) == 0 {
		tags = []string{app.StihiMainTag}
	} else {
		tags = sliceExcludeString(tags, app.StihiMainTag)
		tags = append([]string{app.StihiMainTag}, tags...)
	}
	tags = translit.EncodeTags(tags)


	editor := getMetaString(content["metadata"], "editor")

	meta := blockchain.ContentMetadata{
		"tags": tags,
		"image": content["image"].(string),
		"app": app.StihiAppName,
		"editor": editor,
	}
	metaStr, _ := meta.MarshalJSON()

	body := PreProcessStr(content["body"].(string))
	if body == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}

	title := PreProcessStr(content["title"].(string))
	if title == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.title", "string"), nil)
		return
	}

	times := strconv.FormatInt(time.Now().UnixNano(), 16)
	permlink := blockchain.GenPermlink("", "-"+times, "", title)

	// Check utf8
	if !utf8.ValidString(body) || !utf8.ValidString(title) {
		err = errors_l10n.New(lang, "parameters.wrong_utf8")
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	api := cyberway.New(Config.RPC.BaseURL("http"))
	app.Debug.Printf("CyberAPIConnect... %s", time.Now().Format(cache_level2.TimeJSONFormat))

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(postingPvtKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	cyberName := DB.GetNodeosName(login)

	provideBWRequired := ProvideBWRequired(cyberName)

	tx := cyberway.NewTransaction([]*cyberway.Action{},
		txOpts,
	)

	if provideBWRequired {
		tx.Actions = append(tx.Actions,
			system.NewProvideBW(
				cyberway.AccountName(cyberName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.PermissionName(Config.Golos.CreatorPermission)))

		err = keyBag.Add(Config.Golos.CreatorKey)
		if err != nil {
			app.Error.Println(err)
			DoJSONError(&w, err, nil)
			return
		}
	}


	tx.Actions = append(tx.Actions,
		forum.CreateMessage(postingPvtKeyType, &forum.CreateMessageData{
			Id: cyberway.MssgId{
				Author:   cyberway.AccountName(cyberName),
				Permlink: permlink,
			},
			ParentId:      cyberway.MssgId{},
			Language:      "ru",
			Header:        title,
			Body:          body,
			Tags:          []string{app.StihiMainTag},
			JsonMetadata:  string(metaStr),
			TokenProp:     5000,
			Beneficiaries: []cyberway.Beneficiary{},
			CuratorsPrcnt: 0,
			VestPayment:   false,
			MaxPayout: "1000000000.00 GOLOS",
		}))

	api.SetSigner(keyBag)

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

	// Если нет ошибки - сохраняем в локальной БД и возвращаем внутренний id статьи
	article := operations.CreateMessageData{
		Id: &operations.MessageIdType{
			Author:   cyberName,
			Permlink: permlink,
		},
		Language:      "ru",
		Header:        title,
		Body:          body,
		Tags:          []string{app.StihiAppName},
		JsonMetadata:  string(metaStr),
	}

	// Создание новой статьи или апдэйт если такая есть
	// app.Debug.Printf("DBG: SaveArticleFromOperation... %s", time.Now().Format(cache_level2.TimeJSONFormat))
	id, err := DB.SaveArticleFromOperation(&article, time.Now().UTC())
	if err != nil {
		app.Error.Printf("Error when save article to db: %s", err)
	}
	if id > 0 {
		resp["id"] = id
	}

	// Reset user battery cache
	DB.ResetCacheUserBatteryNodeos(login)


	// app.Debug.Printf("DBG: SaveArticleFromOperation done %s", time.Now().Format(cache_level2.TimeJSONFormat))

	// Если есть голосание за свой пост - создаем голос
	/*
	if selfVote {
		_, err := blockchain.Vote(&Config.RPC, postingPvtKey, login, login, postResp.PermLink, 10000)
		if err != nil {
			app.Error.Printf(err.Error())
		} else {
			_, err = blockchain.Vote(&Config.RPC, postingPvtKey, login, login, postResp.PermLink, 10000)
			if err != nil {
				app.Error.Print(err)
			}
		}
	}
	*/

	DoJSONResponse(&w, &resp, nil)
}

func UpdateArticle(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	postingPvtKeyType := jwt.DecodeKeyType((*claims)["kpt"].(string))
	if err != nil {
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	// Проверка пользователя на бан. Если бан - возвращаем ошибку.
	user, err := DB.GetUserInfoByName(login)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	if user.Ban {
		app.Error.Printf(errors_l10n.New(lang, "authorize.banned").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.banned"), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	if !IsParamType(&w, "content_id", params["content_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content_id", "number"), nil)
		return
	}
	contentId := int64(params["content_id"].(float64))

	contentParam, ok := params["content"]
	if !ok {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}
	if !IsParamType(&w, "content", params["content"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}

	content := contentParam.(map[string]interface{})

	if !IsParamType(&w, "content/title", content["title"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.title", "string"), nil)
		return
	}
	if !IsParamType(&w, "content/body", content["body"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}
	if !IsParamType(&w, "content/image", content["image"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.image", "string"), nil)
		return
	}
	if !IsParamType(&w, "content/metadata", content["metadata"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content/metadata", "struct"), nil)
		return
	}

	authorStr, permlink, err := DB.GetContentIdStrings(contentId)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}
	if authorStr != login {
		DoJSONError(&w, errors_l10n.New(lang, "content.alian"), nil)
		return
	}

	tags := getTags(content["metadata"])
	editor := getMetaString(content["metadata"], "editor")

	if tags == nil {
		tags = []string{app.StihiMainTag}
	}
	tags = translit.EncodeTags(tags)

	meta := blockchain.ContentMetadata{
		"tags": tags,
		"image": content["image"].(string),
		"app": app.StihiAppName,
		"editor": editor,
	}
	metaStr, _ := meta.MarshalJSON()

	body := PreProcessStr(content["body"].(string))
	if body == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}

	title := PreProcessStr(content["title"].(string))
	if title == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.title", "string"), nil)
		return
	}

	// Check utf8
	if !utf8.ValidString(body) || !utf8.ValidString(title) {
		err = errors_l10n.New(lang, "parameters.wrong_utf8")
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	// app.Debug.Printf("DBG: UpdateArticle... %s", time.Now().Format(cache_level2.TimeJSONFormat))

	api := cyberway.New(Config.RPC.BaseURL("http"))

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(postingPvtKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	cyberName := DB.GetNodeosName(login)

	provideBWRequired := ProvideBWRequired(cyberName)

	tx := cyberway.NewTransaction([]*cyberway.Action{},
		txOpts,
	)

	if provideBWRequired {
		tx.Actions = append(tx.Actions,
			system.NewProvideBW(
				cyberway.AccountName(cyberName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.PermissionName(Config.Golos.CreatorPermission)))

		err = keyBag.Add(Config.Golos.CreatorKey)
		if err != nil {
			app.Error.Println(err)
			DoJSONError(&w, err, nil)
			return
		}
	}

	tx.Actions = append(tx.Actions,
		forum.UpdateMessage(postingPvtKeyType, &forum.UpdateMessageData{
			Id: cyberway.MssgId{
				Author:   cyberway.AccountName(cyberName),
				Permlink: permlink,
			},
			Language:      "ru",
			Header:        title,
			Body:          body,
			Tags:          []string{app.StihiMainTag},
			JsonMetadata:  string(metaStr),
		}))

	tx.Expiration.Time = time.Now().UTC().Add(1800 * time.Second)

	api.SetSigner(keyBag)

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

	// Если нет ошибки - сохраняем в локальной БД и возвращаем внутренний id статьи
	article := operations.CreateMessageData{
		Id: &operations.MessageIdType{
			Author:   cyberName,
			Permlink: permlink,
		},
		Language:      "ru",
		Header:        title,
		Body:          body,
		Tags:          []string{app.StihiAppName},
		JsonMetadata:  string(metaStr),
	}

	// Создание новой статьи или апдэйт если такая есть
	// app.Debug.Printf("DBG: SaveArticleFromOperation... %s", time.Now().Format(cache_level2.TimeJSONFormat))
	id, err := DB.SaveArticleFromOperation(&article, time.Now().UTC())
	if err != nil {
		app.Error.Printf("Error when save article to db: %s", err)
	}
	if id > 0 {
		resp["id"] = id
	}
	// app.Debug.Printf("DBG: SaveArticleFromOperation done %s", time.Now().Format(cache_level2.TimeJSONFormat))

	// Reset user battery cache
	DB.ResetCacheUserBatteryNodeos(login)

	DoJSONResponse(&w, &resp, nil)
}

func CreateComment(w http.ResponseWriter, r *http.Request) {
	// TODO: ОТЛАДИТЬ ДОБАВЛЕНИЕ КОММЕНТАРИЯ В ПЛАНЕ ОТМЕТКИ comments_count и last_comment_time в статье

	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	postingPvtKeyType := jwt.DecodeKeyType((*claims)["kpt"].(string))
	if err != nil {
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}
	publicPostingKey := blockchain.GetPublicKey("GLS", postingPvtKey)
	app.Debug.Printf("Auth info: %s - %s\n", login, publicPostingKey)

	// Проверка пользователя на бан. Если бан - возвращаем ошибку.
	user, err := DB.GetUserInfoByName(login)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	if user.Ban {
		app.Error.Printf(errors_l10n.New(lang, "authorize.banned").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.banned"), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	contentParam, ok := params["content"]
	if !ok {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}
	if !IsParamType(&w, "content", params["content"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}

	content := contentParam.(map[string]interface{})

	if !IsParamType(&w, "content/parent_id", content["parent_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.parent_id", "number"), nil)
		return
	}
	if !IsParamType(&w, "content/body", content["body"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}
	if content["metadata"] != nil && !IsParamType(&w, "content/metadata", content["metadata"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.metadata", "struct"), nil)
		return
	}

	// Получаем author и permlink для контента - владельца
	parentId := content["parent_id"].(float64)
	parentAuthor, parentPermlink, err := DB.GetContentIdStrings(int64(parentId))
	if err != nil || parentAuthor == "" || parentPermlink == "" {
		DoJSONError(&w, errors.New("parameter 'parent_id' should link with real content"), nil)
	}
	tags := getTags(content["metadata"])
	if tags == nil {
		tags = []string{app.StihiMainTag}
	}
	tags = translit.EncodeTags(tags)

	app.Debug.Printf("Post comment...")

	body := PreProcessStr(content["body"].(string))
	var title string
	titleI, ok := content["title"]
	if ok {
		title = PreProcessStr(titleI.(string))
	}
	var image string
	imageI, ok := content["image"]
	if ok {
		image = imageI.(string)
	}

	editor := getMetaString(content["metadata"], "editor")

	times := strconv.FormatInt(time.Now().UnixNano(), 16)
	permlink := blockchain.GenPermlink("re-", "-"+times+"stihiio", login, parentPermlink)

	// Check utf8
	if !utf8.ValidString(body) || !utf8.ValidString(title) {
		err = errors_l10n.New(lang, "parameters.wrong_utf8")
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	meta := blockchain.ContentMetadata{
		"tags": tags,
		"image": image,
		"app": app.StihiAppName,
		"editor": editor,
	}
	metaStr, _ := meta.MarshalJSON()

	api := cyberway.New(Config.RPC.BaseURL("http"))

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(postingPvtKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	if body == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}

	// Get cyber name from gls
	cyberName := DB.GetNodeosName(login)

	provideBWRequired := ProvideBWRequired(cyberName)

	tx := cyberway.NewTransaction([]*cyberway.Action{},
		txOpts,
	)

	if provideBWRequired {
		tx.Actions = append(tx.Actions,
			system.NewProvideBW(
				cyberway.AccountName(cyberName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.PermissionName(Config.Golos.CreatorPermission)))

		err = keyBag.Add(Config.Golos.CreatorKey)
		if err != nil {
			app.Error.Println(err)
			DoJSONError(&w, err, nil)
			return
		}
	}

	tx.Actions = append(tx.Actions,
		forum.CreateMessage(postingPvtKeyType, &forum.CreateMessageData{
			Id: cyberway.MssgId{
				Author:   cyberway.AccountName(cyberName),
				Permlink: permlink,
			},
			ParentId:      cyberway.MssgId{
				Author: cyberway.AccountName(parentAuthor),
				Permlink: parentPermlink,
			},
			Language:      "ru",
			Header:        title,
			Body:          body,
			Tags:          []string{app.StihiMainTag},
			JsonMetadata:  string(metaStr),
			TokenProp:     5000,
			Beneficiaries: []cyberway.Beneficiary{},
			CuratorsPrcnt: 0,
			VestPayment:   false,
			MaxPayout: "1000000000.00 GOLOS",
		}))

	tx.Expiration.Time = time.Now().UTC().Add(1800 * time.Second)

	api.SetSigner(keyBag)

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

	// Если нет ошибки - сохраняем в локальной БД и возвращаем внутренний id статьи
	comment := operations.CreateMessageData{
		Id: &operations.MessageIdType{
			Author:   cyberName,
			Permlink: permlink,
		},
		ParentId: &operations.MessageIdType{
			Author: parentAuthor,
			Permlink: parentPermlink,
		},
		Language:      "ru",
		Header:        title,
		Body:          body,
		Tags:          []string{app.StihiAppName},
		JsonMetadata:  string(metaStr),
	}

	id, err := DB.SaveCommentFromOperation(&comment, time.Now().UTC())
	if err != nil {
		app.Error.Printf("Error when save comment to db: %s", err)
	} else {
		resp["id"] = id
	}

	// Reset user battery cache
	DB.ResetCacheUserBatteryNodeos(login)

	DoJSONResponse(&w, &resp, nil)
}

func UpdateComment(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	postingPvtKeyType := jwt.DecodeKeyType((*claims)["kpt"].(string))
	if err != nil {
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}
	publicPostingKey := blockchain.GetPublicKey("GLS", postingPvtKey)
	app.Debug.Printf("Auth info: %s - %s\n", login, publicPostingKey)

	// Проверка пользователя на бан. Если бан - возвращаем ошибку.
	user, err := DB.GetUserInfoByName(login)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	if user.Ban {
		app.Error.Printf(errors_l10n.New(lang, "authorize.banned").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.banned"), nil)
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	if !IsParamType(&w, "content_id", params["content_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content_id", "number"), nil)
		return
	}
	contentId := int64(params["content_id"].(float64))

	contentParam, ok := params["content"]
	if !ok {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}
	if !IsParamType(&w, "content", params["content"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content", "struct"), nil)
		return
	}

	content := contentParam.(map[string]interface{})

	if !IsParamType(&w, "content/parent_id", content["parent_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.parent_id", "number"), nil)
		return
	}
	if !IsParamType(&w, "content/body", content["body"], "string") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}
	if content["metadata"] != nil && !IsParamType(&w, "content/metadata", content["metadata"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.metadata", "struct"), nil)
		return
	}

	author, permlink, err := DB.GetContentIdStrings(contentId)
	if err != nil {
		app.Error.Print(err)
		DoJSONError(&w, err, nil)
		return
	}
	if author != login {
		DoJSONError(&w, errors_l10n.New(lang, "content.alian"), nil)
		return
	}

	// Получаем author и permlink для контента - владельца
	parentId := content["parent_id"].(float64)
	parentAuthor, parentPermlink, err := DB.GetContentIdStrings(int64(parentId))
	if err != nil || parentAuthor == "" || parentPermlink == "" {
		DoJSONError(&w, errors.New("parameter 'parent_id' should link with real content"), nil)
	}

	editor := getMetaString(content["metadata"], "editor")

	tags := getTags(content["metadata"])
	if tags == nil {
		tags = []string{app.StihiMainTag}
	}
	tags = translit.EncodeTags(tags)
	meta := blockchain.ContentMetadata{
		"tags": tags,
		"app": app.StihiAppName,
		"editor": editor,
	}
	metaStr, _ := meta.MarshalJSON()

	body := PreProcessStr(content["body"].(string))
	title := ""
	titleI, ok := content["title"]
	if ok {
		PreProcessStr(titleI.(string))
	}


	// Check utf8
	if !utf8.ValidString(body) || !utf8.ValidString(title) {
		err = errors_l10n.New(lang, "parameters.wrong_utf8")
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	if body == "" {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "content.body", "string"), nil)
		return
	}

	//app.Debug.Printf("DBG: UpdateArticle... %s", time.Now().Format(cache_level2.TimeJSONFormat))

	api := cyberway.New(Config.RPC.BaseURL("http"))

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(postingPvtKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	cyberName := DB.GetNodeosName(login)

	provideBWRequired := ProvideBWRequired(cyberName)

	tx := cyberway.NewTransaction([]*cyberway.Action{},
		txOpts,
	)

	if provideBWRequired {
		tx.Actions = append(tx.Actions,
			system.NewProvideBW(
				cyberway.AccountName(cyberName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.PermissionName(Config.Golos.CreatorPermission)))

		err = keyBag.Add(Config.Golos.CreatorKey)
		if err != nil {
			app.Error.Println(err)
			DoJSONError(&w, err, nil)
			return
		}
	}

	tx.Actions = append(tx.Actions,
		forum.UpdateMessage(postingPvtKeyType, &forum.UpdateMessageData{
			Id: cyberway.MssgId{
				Author:   cyberway.AccountName(cyberName),
				Permlink: permlink,
			},
			Language:      "ru",
			Header:        title,
			Body:          body,
			Tags:          []string{app.StihiMainTag},
			JsonMetadata:  string(metaStr),
		}))

	tx.Expiration.Time = time.Now().UTC().Add(1800 * time.Second)

	api.SetSigner(keyBag)

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

	// Если нет ошибки - сохраняем в локальной БД и возвращаем внутренний id статьи
	comment := operations.CreateMessageData{
		Id: &operations.MessageIdType{
			Author:   cyberName,
			Permlink: permlink,
		},
		ParentId: &operations.MessageIdType{
			Author:   parentAuthor,
			Permlink: parentPermlink,
		},
		Language:      "ru",
		Header:        title,
		Body:          body,
		Tags:          []string{app.StihiAppName},
		JsonMetadata:  string(metaStr),
	}

	id, err := DB.SaveCommentFromOperation(&comment, time.Now().UTC())
	if err != nil {
		app.Error.Printf("Error when save comment to db: %s", err)
	} else {
		resp["id"] = id
	}

	// Reset user battery cache
	DB.ResetCacheUserBatteryNodeos(login)

	DoJSONResponse(&w, &resp, nil)
}

func CreateVote(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

	resp := map[string]interface{}{ "status": "ok" }

	lang := SetLang(r)

	// Авторизация
	claims, _, err := jwt.Check(r)
	if err != nil {
		DoJSONError(&w, errors_l10n.New(lang, "authorize.required"), nil)
		return
	}
	login := (*claims)["n"].(string)
	encPostingPvtKey := (*claims)["kp"].(string)
	postingPvtKey, err := jwt.DecryptPrivKey(encPostingPvtKey)
	postingPvtKeyType := jwt.DecodeKeyType((*claims)["kpt"].(string))
	if err != nil {
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	// Проверка пользователя на бан. Если бан - возвращаем ошибку.
	user, err := DB.GetUserInfoByName(login)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	if user.Ban {
		app.Error.Printf(errors_l10n.New(lang, "authorize.banned").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.banned"), nil)
		return
	}


	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	voteParam, ok := params["vote"]
	if !ok {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "vote", "struct"), nil)
		return
	}
	if !IsParamType(&w, "vote", params["vote"], "map[string]interface {}") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "vote", "struct"), nil)
		return
	}

	vote := voteParam.(map[string]interface{})

	if !IsParamType(&w, "vote/content_id", vote["content_id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "vote.content_id", "number"), nil)
		return
	}
	if !IsParamType(&w, "vote/weight", vote["weight"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "vote.weight", "number"), nil)
		return
	}

	contentId := int64(vote["content_id"].(float64))
	weight := int(vote["weight"].(float64))

	// Получаем author и permlink для контента
	author, permlink, err := DB.GetContentIdStrings(contentId)
	if err != nil || author == "" || permlink == "" {
		DoJSONError(&w, errors.New("parameter 'content_id' should be linked with real content"), nil)
		return
	}

	// Отправляем голос в блокчейн
	api := cyberway.New(Config.RPC.BaseURL("http"))

	keyBag := &cyberway.KeyBag{}
	err = keyBag.Add(postingPvtKey)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	txOpts := &cyberway.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}

	cyberName := DB.GetNodeosName(login)

	provideBWRequired := ProvideBWRequired(cyberName)

	tx := cyberway.NewTransaction([]*cyberway.Action{},
		txOpts,
	)

	if provideBWRequired {
		tx.Actions = append(tx.Actions,
			system.NewProvideBW(
				cyberway.AccountName(cyberName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.AccountName(Config.Golos.CreatorName),
				cyberway.PermissionName(Config.Golos.CreatorPermission)))

		err = keyBag.Add(Config.Golos.CreatorKey)
		if err != nil {
			app.Error.Println(err)
			DoJSONError(&w, err, nil)
			return
		}
	}

	tx.Actions = append(tx.Actions,
		forum.NewVote(
			postingPvtKeyType,
			cyberName,
			author,
			permlink,
			weight,
			))

	tx.Expiration.Time = time.Now().UTC().Add(1800 * time.Second)

	api.SetSigner(keyBag)

	app.Debug.Printf("DBG:\n%+v\n", *tx)
	for _, act := range tx.Actions {
		app.Debug.Printf("DBG:\n%+v\n", *act)
	}

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

	// TODO: Актуализировать счетчик голосов для контента
	nodeosId, err := DB.GetContentNodeosIdById(contentId)
	if err != nil {
		app.Error.Println(err)
	} else {
		_ = DB.Level2.SyncContent(contentId, nodeosId)
	}

	// Reset user battery cache
	DB.ResetCacheUserBatteryNodeos(login)

	DoJSONResponse(&w, &resp, nil)
}

func DeleteContent(w http.ResponseWriter, r *http.Request) {
	if CORSOptionsProcess(&w, r) {
		return
	}

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
		app.Error.Printf("Posting key decryption error: " + err.Error())
		DoJSONError(&w, errors.New("Posting key decryption error: "+err.Error()), nil)
		return
	}

	// Проверка пользователя на бан. Если бан - возвращаем ошибку.
	user, err := DB.GetUserInfoByName(login)
	if err != nil {
		app.Error.Println(err)
		DoJSONError(&w, err, nil)
		return
	}
	if user.Ban {
		app.Error.Printf(errors_l10n.New(lang, "authorize.banned").Error())
		DoJSONError(&w, errors_l10n.New(lang, "authorize.banned"), nil)
		return
	}

	resp := map[string]interface{}{"status": "ok"}

	// Декодируем параметры запроса
	params, err := DecodeRequest(r)
	if err != nil {
		app.Error.Printf(err.Error())
		DoJSONError(&w, err, nil)
		return
	}

	// Проверяем параметры запроса
	if !IsParamType(&w, "id", params["id"], "float64") {
		DoJSONError(&w, errors_l10n.New(lang, "parameters.should_be", "id", "number"), nil)
		return
	}
	contentId := int64(params["id"].(float64))

	// Получаем author и permlink для контента
	author, permlink, err := DB.GetContentIdStrings(int64(contentId))
	if err != nil || author == "" || permlink == "" {
		DoJSONError(&w, errors_l10n.New(lang, "l10n:info.data_absent"), nil)
		return
	}

	// Проверяем что авторизованный юзер является владельцем контента
	if author != login {
		DoJSONError(&w, errors_l10n.New(lang, "l10n:info.different_users"), nil)
		return
	}

	api, err := blockchain.GolosAPIConnect(&Config.RPC)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}
	defer api.Close()

	api.SetKeys(&client.Keys{
		PKey: []string{ postingPvtKey },
	})

	_, err = api.DeleteComment(author, permlink)
	if err != nil {
		DoJSONError(&w, err, nil)
		return
	}

	// Если нет ошибки - удаляем контент из БД
	err = DB.DeleteContent(contentId)
	if err != nil {
		app.Error.Print(err)
	}

	// Reset user battery cache
	DB.ResetCacheUserBatteryNodeos(login)
	DoJSONResponse(&w, &resp, nil)
	*/
}

func PreProcessStr(str string) (string) {
	// Сначала - конвертация в UTF8
	strUTF, err := encodeStr(str)
	if err != nil {
		app.Error.Print(err)
		strUTF = str
	}

	/*
	if Config.RPC.BlockchanName == "test" {
		reg, _ := regexp.Compile(`\<[^\<\>]+\>`)
		strUTF = reg.ReplaceAllString(strUTF, "")

		return strUTF
	}
	*/

	return strUTF
}

func dbgSaveFile(filename string, str string) {
	file, err := os.Create(filename)
	if err != nil{
		fmt.Println("Unable to create file:", err)
		os.Exit(1)
	}
	defer file.Close()
	file.Write([]byte(str))
}
