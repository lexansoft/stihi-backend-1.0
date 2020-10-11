package cache_level1

import (
	"strconv"
	"time"

	"gitlab.com/stihi/stihi-backend/cache_level2"
)

// TODO: Реализовать правильное чтение голосов из mongodb

func (cache *CacheLevel1) GetVoteId(author string, permlink string, voter string) (int64, error) {
	key := KeyVoteIdByNames(author, permlink, voter)
	voteIdStr, err := cache.RedisConn.Get(key)
	if err == nil {
		voteId, err := strconv.ParseInt(voteIdStr, 10, 64)
		if err == nil {
			return voteId, nil
		}
	}

	voteId, err :=  cache.Level2.GetVoteId(author, permlink, voter)
	if err == nil && voteId > 0 {
		cache.RedisConn.Set(key, strconv.FormatInt(voteId, 10))
	}
	return voteId, err
}

// Возвращает id = 0, если нет записанного значения
func (cache *CacheLevel1) GetVoteIdFromCache(author string, permlink string, voter string) (int64, error) {
	key := KeyVoteIdByNames(author, permlink, voter)
	voteIdStr, err := cache.RedisConn.Get(key)
	if err == nil {
		voteId, err := strconv.ParseInt(voteIdStr, 10, 64)
		if err == nil {
			return voteId, nil
		}
	}

	return 0, err
}

func (cache *CacheLevel1) GetVotesForContent(nodeosId int64) (*[]*cache_level2.Vote, error) {
	return cache.Level2.GetVotesForContent(nodeosId)
}

func (cache *CacheLevel1) GetUserVotesForContentList(userId int64, list *[]int64) (*map[int64]int64, error) {
	return cache.Level2.GetUserVotesForContentList(userId, list)
}

func (cache *CacheLevel1) GetVotesForContentList(list *[]int64) (*map[int64][]*cache_level2.Vote, error) {
	return cache.Level2.GetVotesForContentList(list)
}

func (cache *CacheLevel1) GetUserVotesForDurationList(userId int64, period time.Duration) (*[]*cache_level2.Vote, error) {
	return cache.Level2.GetUserVotesForDurationList(userId, period)
}

func KeyVoteIdByNames(author, permlink, voter string) string {
	return "voteIdByNames:"+author+":"+permlink+":"+voter
}
