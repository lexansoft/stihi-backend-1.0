package cache_level2

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/stihi/stihi-backend/blockchain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/mongodb"
	"gitlab.com/stihi/stihi-backend/cyber/cyberdb"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

// TODO: Сделать метод крона для периодического обновления данных о заработанных суммах для статей (для сортировки)

const (
	TimeJSONFormat = "2006-01-02 15:04:05 -0700 UTC"
)

/*
Константин К, [11.10.19 13:50]
[В ответ на Andy Vel]
_amount в типе ассета делится на 10^_decs и всё.
а на 4096 — это значения для ATMSP, они встречаются в батарейках, пулах, лайках
*/

var (
	StihiTag = "stihi-io"
)

type Comment struct {
	ParentAuthor    string 		`json:"parent_author,omitempty"`
	ParentPermlink  string 		`json:"parent_permlink,omitempty"`
	Author          string 		`json:"author,omitempty"`
	Permlink        string 		`json:"permlink,omitempty"`
	Title           string 		`json:"title,omitempty"`
	Body            string 		`json:"body,omitempty"`
	Editor			string		`json:"editor"`

	User			UserInfo	`json:"user"`

	Id 				int64		`json:"id,omitempty"`
	NodeosId		int64		`json:"nodeos_id,omitempty"`
	ParentId		int64		`json:"parent_id,omitempty"`
	Time			string 		`json:"time"`

	Level			int			`json:"level"`
	Ban				bool		`json:"ban"`

	VotesCount			int		`json:"votes_count"`
	VotesCountPositive	int		`json:"votes_count_positive"`
	VotesCountNegative	int		`json:"votes_count_negative"`
	VotesSumPositive	int64	`json:"votes_sum_positive"`
	VotesSumNegative	int64	`json:"votes_sum_negative"`
	ValCyber		float64		`json:"val_cyber"`
	ValGolos		float64		`json:"val_golos"`
	ValPower		float64		`json:"val_power"`

	Metadata		blockchain.ContentMetadata	`json:"metadata,flow"`

	Comments		[]*Comment 	`json:"comments,omitempty"`
	ParentArticle	*Article	`json:"parent_article,omitempty"`
}

type Article struct {
	Author          string 		`json:"author,omitempty"`
	Permlink        string 		`json:"permlink,omitempty"`
	Title           string 		`json:"title,omitempty"`
	Body            string 		`json:"body,omitempty"`
	Editor			string		`json:"editor"`

	User			UserInfo	`json:"user"`

	Id 				int64		`json:"id,omitempty"`
	NodeosId		int64		`json:"nodeos_id,omitempty"`
	Time			string		`json:"time"`
	Image			string		`json:"image"`
	LastCommentTime string		`json:"last_comment_time"`

	Ban				bool		`json:"ban"`

	CommentsCount 		int			`json:"comments_count"`
	TopCommentsCount 	int			`json:"top_comments_count"`

	VotesCount			int		`json:"votes_count"`
	VotesCountPositive	int		`json:"votes_count_positive"`
	VotesCountNegative	int		`json:"votes_count_negative"`
	VotesSumPositive	int64	`json:"votes_sum_positive"`
	VotesSumNegative	int64	`json:"votes_sum_negative"`
	ValCyber		float64		`json:"val_cyber"`
	ValGolos		float64		`json:"val_golos"`
	ValPower		float64		`json:"val_power"`

	Metadata		blockchain.ContentMetadata	`json:"metadata,flow"`

	Comments		[]*Comment `json:"comments,omitempty"`

	PrevArticleId		int64	`json:"prev_id"`
	NextArticleId		int64	`json:"next_id"`
	PrevBlogArticleId	*NavigationArticlePoint	`json:"prev_blog_id"`
	NextBlogArticleId	*NavigationArticlePoint	`json:"next_blog_id"`
}

type HistoryRecord struct {
	OpTime				string		`json:"time"`
	OpType				string		`json:"type"`

	ToUser				string		`json:"to_user"`
	FromUser			string		`json:"from_user,omitempty"`
	ToUserId			int64		`json:"to_user_id"`
	FromUserId			int64		`json:"from_user_id,omitempty"`
	CyberChange			float64		`json:"cyber_change"`
	GolosChange			float64		`json:"golos_change"`
	PowerChange			float64		`json:"power_change"`
	PowerChangeGolos	float64		`json:"power_change_golos"`
	ContentId			int64		`json:"content_id,omitempty"`
	ContentAuthor		string		`json:"content_author,omitempty"`
	ContentPermlink		string		`json:"content_permlink,omitempty"`
}

func (dbConn *CacheLevel2) DeleteContent(id int64) error {
	_, err := dbConn.Do(`
			DELETE FROM content WHERE id = $1	
		`,
		id,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}
	return nil
}

func (dbConn *CacheLevel2) GetContentId(author string, permlink string) (int64, error) {
	rows, err := dbConn.Query(
		"SELECT id " +
			"FROM content " +
			"WHERE " +
			"author = $1 AND permlink = $2",
		author, permlink,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if !rows.Next() {
		return -1, nil
	}

	var contentId int64
	err = rows.Scan(&contentId)
	if err != nil {
		app.Error.Println(err)
	}

	return contentId, err
}

func (dbConn *CacheLevel2) GetContentIdStrings(id int64) (string, string, error) {
	rows, err := dbConn.Query(
		"SELECT author, permlink " +
			"FROM content " +
			"WHERE " +
			"id = $1",
		id,
	)
	if err != nil {
		app.Error.Print(err)
		return "", "", err
	}
	defer rows.Close()

	if !rows.Next() {
		return "", "", nil
	}

	var author, permlink string
	rows.Scan(&author, &permlink)

	return author, permlink, nil
}

func (dbConn *CacheLevel2) GetContentLevel(author string, permlink string) (int, error) {
	rows, err := dbConn.Query(
		"SELECT level " +
			"FROM content " +
			"WHERE " +
			"author = $1 AND permlink = $2",
		author, permlink,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if !rows.Next() {
		return -1, nil
	}

	var level int
	rows.Scan(&level)

	return level, nil
}

func ArticleToCommentOperation(article *Article) (*operations.CreateMessageData) {
	meta, _ := json.Marshal(article.Metadata)
	op := operations.CreateMessageData{
		Id: &operations.MessageIdType{
			Author:   article.Author,
			Permlink: article.Permlink,
		},
		ParentId: &operations.MessageIdType{
			Author:   "",
			Permlink: "",
		},
		Header: article.Title,
		Body: article.Body,
		JsonMetadata: string(meta),
	}
	return &op
}

func CommentToCommentOperation(comment *Comment) (*operations.CreateMessageData) {
	meta, _ := json.Marshal(comment.Metadata)
	op := operations.CreateMessageData{
		Id: &operations.MessageIdType{
			Author:   comment.Author,
			Permlink: comment.Permlink,
		},
		ParentId: &operations.MessageIdType{
			Author:   comment.ParentAuthor,
			Permlink: comment.ParentPermlink,
		},
		Header: comment.Title,
		Body: comment.Body,
		JsonMetadata: string(meta),
	}
	return &op
}

func CommentOperationToArticle(op *operations.CreateMessageData) *Article {
	meta := ParseMetaToMap(op.JsonMetadata)

	article := Article{
		Author: op.Id.Author,
		Permlink: op.Id.Permlink,
		Title: op.Header,
		Body: op.Body,
		Metadata: meta,
	}

	return &article
}

func CommentOperationToComment(op *operations.CreateMessageData) (*Comment) {
	meta := ParseMetaToMap(op.JsonMetadata)

	comment := Comment{
		ParentAuthor: op.ParentId.Author,
		ParentPermlink: op.ParentId.Permlink,
		Author: op.Id.Author,
		Permlink: op.Id.Permlink,
		Title: op.Header,
		Body: op.Body,
		Metadata: meta,
	}

	return &comment
}

// Проверка что это наш контент на основании JSONMetadata контента
func IsStihiContent(metaData string) bool {
	meta := ParseMetaToMap(metaData)

	appName, ok := meta["app"].(string)
	if !ok || appName != app.StihiAppName {
		return false
	}

	tags, ok := meta["tags"].([]interface{})
	if ok && len(tags) > 0 {
		return tags[0].(string) == StihiTag
	}

	return false
}

func ParseMetaToMap(str string) map[string]interface{} {
	str = strings.TrimLeft(str, `"`)
	str = strings.TrimRight(str, `"`)
	str = strings.ReplaceAll(str, `\"`, `"`)

	meta := make(map[string]interface{})
	err := json.Unmarshal([]byte(str), &meta)
	if err != nil {
		app.Error.Printf("Error parse JsonMetadata: %s", err)
		return nil
	}

	return meta
}

// Проверка что это матерный контент на основании тэгов
func IsMatContent(metaData *MetaData) bool {
	for _, tag := range []string{"nsfw", "ru--mat"} {
		if metaData.IsTagPresent(tag) {
			return true
		}
	}

	return false
}

type NullTime struct {
	Time   time.Time
	Valid  bool // Valid is true if String is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullTime) Scan(value interface{}) error {
	if value == nil {
		ns.Valid = false
		return nil
	}
	ns.Valid = true

	switch value.(type) {
	case time.Time:
		ns.Time = value.(time.Time)
	case string:
		t, err := time.Parse(time.RFC3339Nano, value.(string))
		if err != nil {
			ns.Valid = false
			return err
		}
		ns.Time = t
	}

	return nil
}

// Value implements the driver Valuer interface.
func (ns NullTime) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.Time, nil
}

func (ns NullTime) Format() (string) {
	return ns.Time.Format(TimeJSONFormat)
}

func secondsInSQLInterval(count int64) string {
	if count < 86400 {
		return fmt.Sprintf("%d SECONDS", count)
	}
	days := count/86400
	return fmt.Sprintf("%d DAYS", days)
}

func (dbConn *CacheLevel2) GetContentNodeosIdById(id int64) (int64, error) {
	// Получаем permlink контента по id
	var permlink string
	rows, err := dbConn.Query(`SELECT permlink, nodeos_id FROM content WHERE id = $1`, id)
	if err != nil {
		return -1, errors.Wrap(err, "GetContentNodeosIdById")
	}
	defer rows.Close()
	if rows.Next() {
		var id sql.NullInt64
		err = rows.Scan(&permlink, &id)
		if err != nil {
			return -1, errors.Wrap(err, "GetContentNodeosIdById")
		}

		if id.Valid && id.Int64 > 0 {
			return id.Int64, nil
		}
	} else {
		return -1, nil
	}

	// Получаем message_id из коллекции permlink mongodb
	messageDBId, err := dbConn.GetContentNodeosIdByPermlink(permlink)
	if err != nil {
		return -1, errors.Wrap(err, "GetContentNodeosIdById")
	}

	return messageDBId, nil
}

func (dbConn *CacheLevel2) GetContentIdByNodeosId(nodeosId int64) (int64, error) {
	// Получаем id контента по nodeosId
	rows, err := dbConn.Query(`SELECT id FROM content WHERE nodeos_id = $1`, nodeosId)
	if err != nil {
		return -1, errors.Wrap(err, "GetContentIdByNodeosId")
	}
	defer rows.Close()
	if rows.Next() {
		var id sql.NullInt64
		err = rows.Scan(&id)
		if err != nil {
			return -1, errors.Wrap(err, "GetContentIdByNodeosId")
		}

		if id.Valid && id.Int64 > 0 {
			return id.Int64, nil
		}
	}

	// Получаем permlink из коллекции permlink mongodb
	permlink, err := dbConn.GetContentPermlinkByNodeosId(nodeosId)
	if err != nil {
		return -1, errors.Wrap(err, "GetContentIdByNodeosId")
	}

	// Получаем id контента по permlink
	id, err := dbConn.GetContentIdByPermlink(permlink)
	if err != nil {
		return -1, errors.Wrap(err, "GetContentIdByNodeosId")
	}

	return id, nil
}


func (dbConn *CacheLevel2) GetContentNodeosIdByPermlink(permlink string) (int64, error) {
	// Сначала ищем в нашей БД
	rows, err := dbConn.Query(`
			SELECT nodeos_id
			FROM content
			WHERE permlink = $1
		`,
		permlink,
	)
	if err != nil {
		return -1, errors.Wrap(err, "Error find of nodeos_id in our DB")
	}
	defer rows.Close()

	if rows.Next() {
		var id sql.NullInt64
		err = rows.Scan(&id)
		if err != nil {
			return -1, errors.Wrap(err, "Error scan on find of nodeos_id in our DB")
		}

		if id.Valid && id.Int64 > 0 {
			return id.Int64, nil
		}
	}

	// Если не нашли во внутренней БД - ищем в mongodb в коллекции Permlink
	dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("permlink")

	filter := bson.D{
		{"value", permlink},
	}

	permLinkDB := cyberdb.PermlinkType{}

	err = dbConn.mongo.Collection.FindOne(context.TODO(), filter).Decode(&permLinkDB)
	if err != nil && err != mongo.ErrNoDocuments {
		app.Error.Printf("Find DB permlink '%s' error: %s", permlink, err)
		return -1, err
	}

	id := int64(0)
	// Если исходное число 0 в любой степени, ставим 0
	idStr := permLinkDB.Id.String()
	if len(idStr) < 2 || idStr[0:2] != "0E" {
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			app.Error.Printf("Error convert Decimal128 to int64: %s", err)
		}
	}

	return id, nil
}

func (dbConn *CacheLevel2) GetContentPermlinkByNodeosId(nodeosId int64) (string, error) {
	// Сначала ищем в нашей БД
	rows, err := dbConn.Query(`
			SELECT permlink
			FROM content
			WHERE nodeos_id = $1
		`,
		nodeosId,
	)
	if err != nil {
		return "", errors.Wrap(err, "Error find of permlink by nodeos_id in our DB")
	}
	defer rows.Close()

	if rows.Next() {
		var permlink string
		err = rows.Scan(&permlink)
		if err != nil {
			return "", errors.Wrap(err, "Error scan on find of permlink by nodeos_id in our DB")
		}
		return permlink, nil
	}

	// Если не нашли во внутренней БД - ищем в mongodb в коллекции Permlink
	dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("permlink")

	filter := bson.D{
		{"message_id", nodeosId},
	}

	permLinkDB := cyberdb.PermlinkType{}

	err = dbConn.mongo.Collection.FindOne(context.TODO(), filter).Decode(&permLinkDB)
	if err != nil && err != mongo.ErrNoDocuments {
		app.Error.Printf("Find DB permlink '%s' error: %s", permLinkDB.Value, err)
		return "", err
	}

	return permLinkDB.Value, nil
}

func (dbConn *CacheLevel2) GetContentIdByPermlink(permlink string) (int64, error) {
	rows, err := dbConn.Query(`
			SELECT id
			FROM content
			WHERE permlink = $1
		`,
		permlink,
	)
	if err != nil {
		return -1, errors.Wrap(err, "Error find id by permlink in our DB")
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			return -1, errors.Wrap(err, "Error scan when find id by permlink in our DB")
		}

		return id, nil
	}

	return -1, nil
}

/*
Георгий Савчук, [06.10.19 15:47]
Берешь из gls.publish/messages pool_date и state.voteshares, по pool_date вытаскиваешь из gls.publish/rewardpools
соотвествующий пул c pool_date = created

Георгий Савчук, [06.10.19 15:49]
Ну а там посчитать пропорции. pool.state.funds * message.voteshares / pool.state.rsharesfn

Георгий Савчук, [06.10.19 15:49]
Как то так

Георгий Савчук, [06.10.19 15:49]
Возможно надо к voteshares приложить формулу кривой

Георгий Савчук, [06.10.19 15:51]
Я так пробовал сделать в своем боте, суммы сходились с теми, что показывал golos.io. Сейчас, после смены кривой,
не знаю, надо будет смотреть

Георгий Савчук, [06.10.19 15:52]
Вот так я в боте своем считал

discussion.payout = ((pool.state.funds._amount / 1000) * parseInt(message.state.voteshares) / parseInt(pool.state.rsharesfn._string) *
(message.rewardweight / 10000)).toFixed(3) + " GOLOS";

Георгий Савчук, [06.10.19 15:56]
Да, забыл упомянуть message.rewardweight. Это штраф на постинг. Можно 4 поста в сутки без штрафа постить, у последующих
постов rewardweight меньше 100%
*/

func (dbConn *CacheLevel2) MongoDBCheck() {
	dbConn.mongo.Check()
}

func (dbConn *CacheLevel2) GetContentRewardNodeos(nodeosId int64) (float64, error) {
	if dbConn.mongo == nil {
		var err error
		dbConn.mongo, err = mongodb.New()
		if err != nil {
			log.Panicf("Error connect mongodb: %s\n", err)
		}
	}

	dbConn.mongo.Check()

	dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("message")

	msg := cyberdb.MessageType{}

	c := dbConn.mongo.Collection

	// Сначала ищем message
	filter := bson.D{
		{"id", nodeosId},
	}

	err := c.FindOne(context.TODO(), filter).Decode(&msg)
	if err != nil {
		// Это штатная ситуация, поэтому ошибку не выводим
		// app.Error.Printf("Error read mongodb message for nodeosId: %d : %s", nodeosId, err)
		return 0, err
	}

	// Затем получаем rewardpools
	filter = bson.D{
		{"created", msg.PoolDate},
	}

	rp := dbConn.mongo.SetCollection("rewardpools").Collection
	rewardpool := cyberdb.RewardPoolType{}
	err = rp.FindOne(context.TODO(), filter).Decode(&rewardpool)
	if err != nil {
		return 0, err
	}

	// Запускаем байтокд-машину и вычисляем sharesfn
	rewardWaight := cyberdb.Dec128ToBigFloat(msg.RewardWeight, 1)
	rewardWaight.Quo(rewardWaight, big.NewFloat(10000))

	funds, _ := rewardpool.State.Funds.GetBigValue()

	sharesfn := rewardpool.Rules.MainFunc.Code.SetParams(msg.State.NetShares).Run()

	rsharesfn, _, _ := big.NewFloat(0).Parse(rewardpool.State.RSharesFN.String, 10)

	// reward := msg.RewardWeight * rewardpool.State.Funds * sharesfn / rewardpool.State.RSharesFN
	reward := new(big.Float).Mul(rewardWaight, funds)
	reward.Mul(reward, sharesfn)
	if rsharesfn.Cmp(big.NewFloat(0)) != 0 {
		reward.Quo(reward, rsharesfn)
	} else {
		reward = big.NewFloat(0)
	}

	mainRes, _ := reward.Float64()

	return mainRes, err
}

func (dbConn *CacheLevel2) GetUserBalanceNodeos(userName string) (golos *cyberdb.TokenAccountType,
																	cyber *cyberdb.TokenAccountType,
																	vesting *cyberdb.VestingAccountType,
																	err error) {
	dbConn.mongo.Check()

	// Получаем балансы в GOLOS и CYBER
	dbConn.mongo.SetDB(cyberdb.TokensDBName).SetCollection("accounts")

	c := dbConn.mongo.Collection

	filter := bson.D{
		{"_SERVICE_.scope", userName},
	}

	cur, err := c.Find(context.TODO(), filter)
	if err != nil {
		app.Error.Printf("Error read balance for user: %s : %s", userName, err)
		return nil, nil, nil, err
	}
	for cur.Next(context.TODO()) {
		acc := cyberdb.TokenAccountType{}
		err = cur.Decode(&acc)
		if err != nil {
			app.Error.Printf("Error decode balance for user: %s : %s", userName, err)
			return nil, nil, nil, err
		}

		if acc.Balance.Sym == "GOLOS" {
			golos = &acc
		} else if acc.Balance.Sym == "CYBER" {
			cyber = &acc
		}
	}
	_ = cur.Close(context.TODO())

	// Получаем силу голоса (vesting)
	dbConn.mongo.SetDB(cyberdb.VestingDBName).SetCollection("accounts")

	c = dbConn.mongo.Collection

	filter = bson.D{
		{"_SERVICE_.scope", userName},
	}

	acc := cyberdb.VestingAccountType{}

	err = c.FindOne(context.TODO(), filter).Decode(&acc)
	if err != nil {
		app.Error.Printf("Error read vesting for user: %s : %s", userName, err)
		return nil, nil, nil, err
	}

	vesting = &acc

	return golos, cyber, vesting, nil
}

func (dbConn *CacheLevel2) GetUserNamesNodeos(userName string) (map[string]string, error) {
	names := make(map[string]string)

	dbConn.mongo.Check()

	// Получаем балансы в GOLOS и CYBER
	dbConn.mongo.SetDB(cyberdb.CyberDBName).SetCollection("username")

	c := dbConn.mongo.Collection

	filter := bson.D{
		{"owner", userName },
	}

	cur, err := c.Find(context.TODO(), filter)
	if err != nil {
		app.Error.Printf("Error read usernames for user: %s : %s", userName, err)
		return nil, err
	}
	for cur.Next(context.TODO()) {
		name := cyberdb.UserNameType{}
		err = cur.Decode(&name)
		if err != nil {
			app.Error.Printf("Error decode usernames for user: %s : %s", userName, err)
			return nil, err
		}

		names[name.Scope] = name.Name
	}
	_ = cur.Close(context.TODO())

	return names, nil
}

func (dbConn *CacheLevel2) SyncContent(id, nodeosId int64) error {
	// Сумма заработанного статьей
	golos, err := dbConn.GetContentRewardNodeos(nodeosId)
	if err != nil {
		// Ситуация штатная, поэтому ошибку не выводим
		// app.Error.Printf("Error get reward values for nodeosId: %d (id: %d): %s", nodeosId, id, err)
		return nil
	}

	// Считать количество голосов из mongodb и выставлять счетчики в нашей БД
	var votesCount int
	var votesSumPositive int
	var votesSumNegative int
	var votesCountPositive int
	var votesCountNegative int
	votes, err := dbConn.GetVotesForContent(nodeosId)
	if err == nil && votes != nil {
		for _, v := range *votes {
			if v.Weight > 0 {
				votesCount++
				votesCountPositive++
				votesSumPositive += int(v.Weight)
			} else if v.Weight < 0 {
				votesCount++
				votesCountNegative++
				votesSumNegative += int(v.Weight)
			}
		}
	}
	if err != nil {
		app.Error.Printf("Error get votes: %s", err)
		return err
	}

	_, err = dbConn.Do(`
			UPDATE content
			SET
				votes_count = $1,
				votes_count_positive = $2,
				votes_count_negative = $3,
				votes_sum_positive = $4,
				votes_sum_negative = $5,
				last_sync_time = NOW()
			WHERE
				id = $6
			`,
		votesCount,
		votesCountPositive, votesCountNegative,
		votesSumPositive, votesSumNegative, id,
	)
	if err != nil {
		app.Error.Printf("Error update var_golos for nodeosId: %d (id: %d): %s", nodeosId, id, err)
		return err
	}
	val := int64( golos * FinanceSaveIndex )
	_, err = dbConn.Do(`
			UPDATE content
			SET
				val_golos_10x6 = $1,
				val_cyber_10x6 = 0,
				val_power_10x6 = 0
			WHERE
				id = $2 AND val_golos_10x6 < $3 
			`,
		val, id, val,
	)
	if err != nil {
		app.Error.Printf("Error update var_golos for nodeosId: %d (id: %d): %s", nodeosId, id, err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) UpdateContent(content *operations.UpdateMessageOp, id int64) error {
	imageUrl := ""
	editor := ""
	meta, err := ParseMeta(content.Data.JsonMetadata)
	if meta != nil && err == nil {
		switch (*meta)["image"].(type) {
		case string:
			imageUrl = (*meta)["image"].(string)
		case []interface{}:
			list := (*meta)["image"].([]interface{})
			if len(list) > 0 {
				imageUrl = list[0].(string)
			}
		}

		if (*meta)["editor"] != nil {
			editor = (*meta)["editor"].(string)
		}
	}

	mat := IsMatContent(meta)

	if imageUrl != "" || mat {
		// Article
		_, err = dbConn.Do(
			`UPDATE articles
			SET
				title = $1,
				body = $2,
				editor = $3,
				image = $4,
				mat = $5
			WHERE id = $6
			`,
			content.Data.Header, content.Data.Body, editor, imageUrl, mat, id,
		)

		if err != nil {
			return errors.Wrap(err, "SQL update articles")
		}
	} else {
		_, err = dbConn.Do(
			`UPDATE content
			SET
				title = $1,
				body = $2,
				editor = $3
			WHERE id = $4
		`,
			content.Data.Header, content.Data.Body, editor, id,
		)

		if err != nil {
			return errors.Wrap(err, "SQL update content")
		}
	}

	return nil
}
