package cache_level2

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cyber/cyberdb"
)

var (
//	TPSaveVote	map[string]int64
)

func init() {
//	TPSaveVote = make(map[string]int64)
}

type Vote struct {
	Id 				int64		`json:"id,omitempty"`
	ContentId 		int64		`json:"content_id,omitempty"`
	VoterId 		int64		`json:"voter_id,omitempty"`
	VoterNickname	string		`json:"nickname,omitempty"`
	Time 			string		`json:"time,omitempty"`

	Voter    		string 		`json:"voter"`
	Author   		string 		`json:"author"`
	Permlink 		string 		`json:"permlink"`
	Weight   		int64  		`json:"weight"`
}

func (dbConn *CacheLevel2) GetVoteId(author string, permlink string, voter string) (int64, error) {
	// Находим Permlink
	messageDBId, err := dbConn.GetContentNodeosIdByPermlink(permlink)
	if err != nil {
		app.Error.Printf("Find DB permlink error: %s", err)
		return -1, err
	}

	filter := bson.D{
		{ "message_id", messageDBId},
		{ "voter", voter },
	}

	voteDB := cyberdb.VoteType{}
	err = dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("vote").Collection.FindOne(context.TODO(), filter).Decode(&voteDB)

	return cyberdb.Dec128ToInt64L(voteDB.Id), nil
}

func (dbConn *CacheLevel2) GetVotesForContent(nodeosId int64) (*[]*Vote, error) {
	// Запрашиваем все голоса для данного контента
	res, err := dbConn.GetVotesForContentList(&[]int64{nodeosId})
	if err != nil {
		app.Error.Printf("GetVotesForContent error: %s", err)
	}
	list := (*res)[nodeosId]

	return &list, nil
}

func (dbConn *CacheLevel2) GetVotesCount() (int64, error) {
	return dbConn.GetTableCount("content_votes")
}

func (dbConn *CacheLevel2) GetVotesLastTime() (*time.Time, error) {
	return dbConn.GetTableLastTime("content_votes")
}

func (dbConn *CacheLevel2) RecalcVotesForContent(id int64) error {
	var posCount, negCount int
	var posSum, negSum int64

	// Получаем ID контента в mongo из ID контента нашей БД
	mongoMsgId, err := dbConn.GetContentNodeosIdById(id)
	if err != nil {
		return errors.Wrap(err, "RecalcVotesForContent")
	}

	// Обновление общих счетчиков голосов для контента
	query := bson.A{
		bson.M{
			"$match": bson.M{
				"$and": bson.M{
					"weight": bson.M{"$gt": 0},
					"message_id": bson.M{"$eq": mongoMsgId},
				},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id":        "",
				"sum_weight": bson.M{"$sum": "$weight"},
				"count":      bson.M{"$sum": 1},
			},
		},
	}

	res, err := dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("vote").Collection.Aggregate(context.TODO(), query)
	if err != nil {
		app.Error.Printf("Error select pos votes for content: %s", err)
		return err
	}
	if res.Next(context.TODO()) {
		resMap := make(map[string]interface{})
		err = res.Decode(&resMap)
		if err != nil {
			app.Error.Printf("Error decode pos votes for content: %s", err)
			return err
		}

		posCount = resMap["count"].(int)
		posSum = resMap["sum_weight"].(int64)
	}
	_ = res.Close(context.TODO())

	// Для отрицательных голосов
	query = bson.A{
		bson.M{
			"$match": bson.M{
				"$and": bson.M{
					"weight": bson.M{"$lt": 0},
					"message_id": bson.M{"$eq": mongoMsgId},
				},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id":        "",
				"sum_weight": bson.M{"$sum": "$weight"},
				"count":      bson.M{"$sum": 1},
			},
		},
	}

	res, err = dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("vote").Collection.Aggregate(context.TODO(), query)
	if err != nil {
		app.Error.Printf("Error select neg votes for content: %s", err)
		return err
	}
	if res.Next(context.TODO()) {
		resMap := make(map[string]interface{})
		err = res.Decode(&resMap)
		if err != nil {
			app.Error.Printf("Error decode neg votes for content: %s", err)
			return err
		}

		negCount = resMap["count"].(int)
		negSum = resMap["sum_weight"].(int64)
	}
	_ = res.Close(context.TODO())

	// Обновляем данные в нашей БД
	_, err = dbConn.Do(`
		UPDATE content 
		SET 
			votes_count = $1, 
			votes_sum_positive = $2, 
			votes_sum_negative = $3,
			votes_count_positive = $4,
			votes_count_negative = $5
		WHERE
			id = $6`,
		posCount + negCount, posSum, negSum, posCount, negCount, id,
	)
	if err != nil {
		app.Error.Printf("Error update votes for content: %s", err)
		return err
	}
	return nil
}

func (dbConn *CacheLevel2) AddVoteForContent(id int64, weight int) error {
	if weight >= 0 {
		_, err := dbConn.Do(`
			UPDATE content 
			SET 
				votes_count = votes_count + 1, 
				votes_sum_positive = votes_sum_positive + $1, 
				votes_count_positive = votes_count_positive + 1
			WHERE
				id = $2`,
			weight, id,
		)
		if err != nil {
			app.Error.Printf("Error pos update content for vote: %s", err)
			return err
		}
	} else {
		_, err := dbConn.Do(`
			UPDATE content 
			SET 
				votes_count = votes_count + 1, 
				votes_sum_negative = votes_sum_negative + $1, 
				votes_count_negative = votes_count_negative + 1
			WHERE
				id = $2`,
			weight, id,
		)
		if err != nil {
			app.Error.Printf("Error neg update content for vote: %s", err)
			return err
		}
	}
	return nil
}

// Получаем голоса от определенного пользователя за определенный контент списком
func (dbConn *CacheLevel2) GetUserVotesForContentList(userId int64, list *[]int64) (*map[int64]int64, error) {
	if list == nil || len(*list) <= 0 {
		return nil, nil
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		return nil, errors.Wrap( err, "GetUserVotesForContentList")
	}

	filter := bson.D{
		{ "voter", userName },
		{ "message_id", bson.D{
			{ "$in", cyberdb.ListInt64ToBsonA(list) },
		}},
	}

	cur, err := dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("vote").Collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "GetUserVotesForContentList")
	}
	res := make(map[int64]int64)
	for cur.Next(context.TODO()) {
		voteDB := cyberdb.VoteType{}
		err = cur.Decode(&voteDB)
		if err != nil {
			app.Error.Printf("GetUserVotesForContentList: Error scan data: %s", err)
		}

		res[cyberdb.Dec128ToInt64L(voteDB.MessageId)] = voteDB.Weight
	}

	return &res, nil
}

func (dbConn *CacheLevel2) GetVotesForContentList(list *[]int64) (*map[int64][]*Vote, error) {
	if list == nil || len(*list) <= 0 {
		return nil, nil
	}

	filter := bson.D{
		{"message_id", bson.D{
			{ "$in", cyberdb.ListInt64ToBsonA(list)},
		}},
	}

	cur, err := dbConn.mongo.SetDB(cyberdb.PublishDBName).SetCollection("vote").Collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "GetVotesForContentList")
	}

	res := make(map[int64][]*Vote)
	for cur.Next(context.TODO()) {
		voteDB := cyberdb.VoteType{}
		err = cur.Decode(&voteDB)
		if err != nil {
			app.Error.Printf("GetVotesForContentList: Error scan data: %s", err)
		}

		msgId := cyberdb.Dec128ToInt64L(voteDB.MessageId)
		if msgId > 0 {
			list, ok := res[msgId]
			if !ok || list == nil {
				list = make([]*Vote, 0)
			}

			vote := Vote{}
			vote.Id = cyberdb.Dec128ToInt64L(voteDB.Id)
			vote.Voter = voteDB.Voter
			vote.ContentId = msgId
			vote.Weight = voteDB.Weight

			list = append(list, &vote)

			res[msgId] = list
		}
	}

	return &res, nil
}

// Выдача голосований пользователя за период с сортировкой от старых к новым
func (dbConn *CacheLevel2) GetUserVotesForDurationList(userId int64, period time.Duration) (*[]*Vote, error) {
	if period <= 0 {
		return nil, nil
	}

	userName, err := dbConn.GetUserNameById(userId)
	if err != nil || userName == "" {
		app.Error.Print(err)
		return nil, err
	}

	rows, err := dbConn.Query(
		`
			SELECT v.id, v.voter, v.weight, v.time, v.content_id, u.id, ui.nickname  
			FROM 
				content_votes v, 
				users u
					LEFT JOIN users_info ui ON ui.user_id = u.id
			WHERE voter = $1 AND v.weight != 0 AND time >= $2::TIMESTAMP AND v.voter = u.name
		`,
		userName, time.Now().UTC().Add(-period),
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	list := make([]*Vote, 0)
	for rows.Next() {
		var voteTime NullTime
		var nickname sql.NullString
		vote := Vote{}
		err = rows.Scan(
			&vote.Id,
			&vote.Voter,
			&vote.Weight,
			&voteTime,
			&vote.ContentId,
			&vote.VoterId,
			&nickname,
		)
		if err != nil {
			app.Error.Print(err)
		}

		if nickname.Valid {
			vote.VoterNickname = nickname.String
		}

		vote.Time = voteTime.Format()

		list = append(list, &vote)
	}

	return &list, nil
}
