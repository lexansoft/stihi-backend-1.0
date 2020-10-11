package cache_level1

import "gitlab.com/stihi/stihi-backend/cache_level2"

/*
func (cache *CacheLevel1) SaveFollowFromOperation(op *types.FollowOperation, ts time.Time) (int64, error) {
	return cache.Level2.SaveFollowFromOperation(op, ts)
}

func (cache *CacheLevel1) IgnoreRemove(userId int64, name string) error {
	return cache.Level2.IgnoreRemove(userId, name)
}

func (cache *CacheLevel1) FollowRemove(userId int64, name string) error {
	return cache.Level2.FollowRemove(userId, name)
}

func (cache *CacheLevel1) IgnoreAdd(userId int64, name string) (error) {
	return cache.Level2.IgnoreAdd(userId, name)
}

func (cache *CacheLevel1) FollowAdd(userId int64, name string) (error) {
	return cache.Level2.FollowAdd(userId, name)
}
*/

func (cache *CacheLevel1) GetUserFollowersList(userId int64) ([]*cache_level2.UserInfo, error) {
	return cache.Level2.GetUserFollowersList(userId)
}

func (cache *CacheLevel1) GetUserFollowsList(userId int64) ([]*cache_level2.UserInfo, error) {
	return cache.Level2.GetUserFollowsList(userId)
}
