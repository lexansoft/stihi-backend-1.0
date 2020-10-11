package cache_level1

import (
	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"strconv"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

const (
	ExpirationTagsForUser = 10 * time.Minute
)

// TODO: Сделать кэширующий вариант

func (cache *CacheLevel1) SaveTagsFromOperation(op *operations.CreateMessageData, content_id int64) error {
	return cache.Level2.SaveTagsFromOperation(op.JsonMetadata, content_id)
}

func (cache *CacheLevel1) GetTagsForContent(id int64) (*[]string, error) {
	return cache.Level2.GetTagsForContent(id)
}

func (cache *CacheLevel1) GetTagsForUser(userId int64) ([]string, error) {
	key := "tags_for_user:"+strconv.FormatInt(userId, 10)

	var cached []string
	err := cache.GetObject(key, &cached)
	if err == nil && cached != nil {
		return cached, nil
	}

	cache.Lock(key)
	defer cache.Unlock(key)

	// После лока сначала проверяем кэш, а затем, если в кэше данных нет, загружаем из БД
	err = cache.GetObject(key, &cached)
	if err == nil && cached != nil {
		return cached, nil
	}

	list, err := cache.Level2.GetTagsForUser(userId)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	list = translit.DecodeTags(list)

	err = cache.SaveEx(key, list, ExpirationTagsForUser)
	if err != nil {
		app.Error.Print(err)
	}

	return list, nil
}
