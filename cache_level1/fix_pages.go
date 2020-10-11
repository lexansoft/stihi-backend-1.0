package cache_level1

import "gitlab.com/stihi/stihi-backend/cache_level2"

func (cache *CacheLevel1) UpdateFixPage(code string, html string, title string, adminName string) error {
	return cache.Level2.UpdateFixPage(code, html, title, adminName)
}

func (cache *CacheLevel1) GetFixPage(code string) (*cache_level2.FixPage, error) {
	return cache.Level2.GetFixPage(code)
}

func (cache *CacheLevel1) GetFixPagesList() ([]*cache_level2.FixPage, error) {
	return cache.Level2.GetFixPagesList()
}
