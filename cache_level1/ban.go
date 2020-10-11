package cache_level1

func (cache *CacheLevel1) BanUser(userId int64, adminName string, description string) error {
	return cache.Level2.BanUser(userId, adminName, description)
}

func (cache *CacheLevel1) UnbanUser(userId int64, adminName string, description string) error {
	return cache.Level2.UnbanUser(userId, adminName, description)
}

func (cache *CacheLevel1) BanContent(contentId int64, adminName string, description string) error {
	return cache.Level2.BanContent(contentId, adminName, description)
}

func (cache *CacheLevel1) UnbanContent(contentId int64, adminName string, description string) error {
	return cache.Level2.UnbanContent(contentId, adminName, description)
}

func (cache *CacheLevel1) DoBanUnban(tableName string, ban bool, id int64, adminName string, description string) error {
	return cache.Level2.DoBanUnban(tableName, ban, id, adminName, description)
}
