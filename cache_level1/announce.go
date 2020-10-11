package cache_level1

import "gitlab.com/stihi/stihi-backend/cache_level2"

func (cache *CacheLevel1) GetAnnouncesPages() (*[]*cache_level2.AnnouncePage, error) {
	return cache.Level2.GetAnnouncesPages()
}

func (cache *CacheLevel1) GetAnnouncePage(code string) (*cache_level2.AnnouncePage, error) {
	return cache.Level2.GetAnnouncePage(code)
}

func (cache *CacheLevel1) CreateAnnounce(pageCode string, contentId int64, payer string, payData string) error {
	return cache.Level2.CreateAnnounce(pageCode, contentId, payer, payData)
}

func (cache *CacheLevel1) GetAnnouncesList(code string, count int, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetAnnouncesList(code, count, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}
