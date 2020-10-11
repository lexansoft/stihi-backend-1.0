package cache_level1

import "gitlab.com/stihi/stihi-backend/cache_level2"

func (cache *CacheLevel1) GetRubrics() (*[]*cache_level2.Rubric, error) {
	return cache.Level2.GetRubrics()
}
