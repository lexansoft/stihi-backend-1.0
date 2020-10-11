package cache_level1

import (
	"gitlab.com/stihi/stihi-backend/cyber/operations"
	"strconv"
	"strings"

	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"github.com/pkg/errors"
	"gopkg.in/redis.v4"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cache_level2"
)

func (cache *CacheLevel1) loadTagsForArticles(list *[]*cache_level2.Article) {
	for _, content := range *list {
		if content == nil {
			continue
		}

		tags, err := cache.GetTagsForContent(content.Id)
		if err != nil {
			continue
		}

		tagsP := translit.DecodeTags(*tags)
		tags = &tagsP

		content.Metadata = map[string]interface{}{
			"tags": tags,
		}
	}
}

func (cache *CacheLevel1) loadTagsForComments(list *[]*cache_level2.Comment) {
	for _, content := range *list {
		if content == nil {
			continue
		}

		tags, err := cache.GetTagsForContent(content.Id)
		if err != nil {
			continue
		}

		content.Metadata = map[string]interface{}{
			"tags": tags,
		}
	}
}

func (cache *CacheLevel1) loadUserNamesForArticles(list *[]*cache_level2.Article) {
	for _, content := range *list {
		if content == nil {
			continue
		}

		_ = cache.GetUserNames(&content.User.User)
	}
}

func (cache *CacheLevel1) loadUserNamesForComments(list *[]*cache_level2.Comment) {
	for _, content := range *list {
		if content == nil {
			continue
		}

		_ = cache.GetUserNames(&content.User.User)

		if content.Comments != nil && len(content.Comments) > 0 {
			cache.loadUserNamesForComments(&content.Comments)
		}
	}
}


func keyContentId(author string, permlink string) string {
	return ContentIdPrefix + ":" + author + ":" + permlink
}

func keyContentById(id int64) string {
	return ContentByIdPrefix + ":" + strconv.FormatInt(id, 10)
}

func (cache *CacheLevel1) saveContentIdLink(id int64, author string, permlink string) error {
	idKey := keyContentId( author, permlink )
	byIdKey := keyContentById( id )

	err := cache.RedisConn.Set(idKey, strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	if id > 0 {
		err = cache.RedisConn.Set(byIdKey, author+":"+permlink)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cache *CacheLevel1) GetContentId(author string, permlink string) (int64, error) {
	id, err := cache.GetContentIdFromCache(author, permlink)
	if err == nil && id > 0 {
		return id, nil
	}

	id, err = cache.Level2.GetContentId(author, permlink)
	if err != nil {
		app.Error.Println(err)
		return -1, err
	}
	cache.SaveContentIdToCache(author, permlink, id)

	return id, nil
}

func (cache *CacheLevel1) GetContentIdFromCache(author string, permlink string) (int64, error) {
	idKey := keyContentId( author, permlink )

	var id int64
	idStr, err := cache.RedisConn.Get(idKey)
	if err == nil && err != redis.Nil && idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			id = 0
		}
		return id, err
	}

	return 0, err
}

func (cache *CacheLevel1) SaveContentIdToCache(author string, permlink string, id int64) {
	idKey := keyContentId( author, permlink )

	cache.RedisConn.Set(idKey, strconv.FormatInt(id, 10))
}

func (cache *CacheLevel1) GetContentIdStrings(id int64) (string, string, error) {
	idKey := keyContentById( id )

	var author, permlink string
	idStr, err := cache.RedisConn.Get(idKey)
	if err == nil && err != redis.Nil && idStr != "" {
		data := strings.Split(idStr, ":")
		if len(data) > 1 {
			author = data[0]
			permlink = data[1]
			return author, permlink, nil
		} else {
			return "", "", errors.New("wrong string for athor:permlink: "+idStr)
		}
	}

	author, permlink, err = cache.Level2.GetContentIdStrings(id)
	if idStr != "" {
		cache.RedisConn.Set(idKey, author+":"+permlink)
	}

	return author, permlink, nil
}

func (cache *CacheLevel1) IsContentPresent(author string, permlink string) bool {
	id, err := cache.GetContentId(author, permlink)
	if err != nil && err != redis.Nil {
		return false
	}
	return id > 0
}

func (cache *CacheLevel1) DeleteContent(id int64) error {
	return cache.Level2.DeleteContent(id)
}

func keyContentNodeosId(permlink string) string {
	return ContentNodeosIdPrefix + ":" + permlink
}

func keyContentIdByNodeosId(id int64) string {
	return ContentIdByNodeosIdPrefix + ":" + strconv.FormatInt(id, 10)
}

func keyContentNodeosIdById(id int64) string {
	return ContentNodeosIdByIdPrefix + ":" + strconv.FormatInt(id, 10)
}

func (cache *CacheLevel1) GetContentNodeosIdFromCache(permlink string) (int64, error) {
	idKey := keyContentNodeosId( permlink )

	var id int64
	idStr, err := cache.RedisConn.Get(idKey)
	if err == nil && err != redis.Nil && idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			id = 0
		}
		return id, err
	}

	return 0, err
}

func (cache *CacheLevel1) SaveContentNodeosIdToCache( permlink string, id int64 ) {
	idKey := keyContentNodeosId( permlink )

	_ = cache.RedisConn.Set(idKey, strconv.FormatInt(id, 10))
}


func (cache *CacheLevel1) GetContentIdByNodeosIdFromCache(nodeosId int64) (int64, error) {
	idKey := keyContentIdByNodeosId( nodeosId )

	var id int64
	idStr, err := cache.RedisConn.Get(idKey)
	if err == nil && err != redis.Nil && idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			id = 0
		}
		return id, err
	}

	return 0, err
}

func (cache *CacheLevel1) SaveContentNodeosIdByIdToCache( nodeosId, id int64 ) {
	idKey := keyContentNodeosIdById( id )

	_ = cache.RedisConn.Set(idKey, strconv.FormatInt(nodeosId, 10))
}

func (cache *CacheLevel1) GetContentNodeosIdByIdFromCache(id int64) (int64, error) {
	idKey := keyContentNodeosIdById( id )

	var nodeosId int64
	idStr, err := cache.RedisConn.Get(idKey)
	if err == nil && err != redis.Nil && idStr != "" {
		nodeosId, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			nodeosId = 0
		}
		return nodeosId, err
	}

	return 0, err
}

func (cache *CacheLevel1) SaveContentIdByNodeosIdToCache( nodeosId, id int64 ) {
	idKey := keyContentIdByNodeosId( nodeosId )

	_ = cache.RedisConn.Set(idKey, strconv.FormatInt(id, 10))
}

func (cache *CacheLevel1) GetContentNodeosIdById(id int64) (int64, error) {
	nodeosId, err := cache.GetContentNodeosIdByIdFromCache(id)
	if err == nil && nodeosId > 0 {
		return id, nil
	}

	nodeosId, err = cache.Level2.GetContentNodeosIdById(id)
	if err != nil {
		app.Error.Println(err)
		return -1, err
	}
	cache.SaveContentNodeosIdByIdToCache(nodeosId, id)

	return nodeosId, nil
}

func (cache *CacheLevel1) GetContentIdByNodeosId(nodeosId int64) (int64, error) {
	id, err := cache.GetContentIdByNodeosIdFromCache( nodeosId )
	if err == nil && id > 0 {
		return id, nil
	}

	id, err = cache.Level2.GetContentIdByNodeosId(id)
	if err != nil {
		app.Error.Println(err)
		return -1, err
	}
	cache.SaveContentIdByNodeosIdToCache(nodeosId, id)

	return nodeosId, nil
}

func (cache *CacheLevel1) GetContentNodeosIdByPermlink(permlink string, flags ...bool) (int64, error) {
	id, err := cache.GetContentNodeosIdFromCache(permlink)
	if err == nil && id > 0 {
		return id, nil
	}

	id, err = cache.Level2.GetContentNodeosIdByPermlink(permlink)
	if err != nil {
		app.Error.Println(err)
		return -1, err
	}
	cache.SaveContentNodeosIdToCache(permlink, id)

	return id, nil
}

func (cache *CacheLevel1) GetContentRewardNodeos(id int64) (float64, error) {
	return cache.Level2.GetContentRewardNodeos(id)
}

func (cache *CacheLevel1) loadRewardsForArticles(list *[]*cache_level2.Article) {
	for _, content := range *list {
		if content == nil {
			continue
		}

		if content.NodeosId <= 1 {
			continue
		}

		var err error
		content.ValGolos, err = cache.GetContentRewardNodeos(content.NodeosId)
		if err != nil {
			app.Debug.Printf("Error wher GetContentRewardNodeos for nodeosId %d: %s", content.NodeosId, err)
			content.ValGolos = 0
		}
	}
}

func (cache *CacheLevel1) loadRewardsForComments(list *[]*cache_level2.Comment) {
	for _, content := range *list {
		if content == nil {
			continue
		}

		if content.NodeosId <= 1 {
			continue
		}

		var err error
		content.ValGolos, err = cache.GetContentRewardNodeos(content.NodeosId)
		if err != nil {
			app.Error.Printf("Error wher GetContentRewardNodeos for nodeosId %d: %s", content.NodeosId, err)
			content.ValGolos = 0
		}
	}
}

func (cache *CacheLevel1) UpdateContent(content *operations.UpdateMessageOp) error {
	id, err := cache.GetContentId(content.Data.Id.Author, content.Data.Id.Permlink)
	if err != nil {
		return errors.Wrap(err, "UpdateContent GetContentId")
	}

	err = cache.Level2.UpdateContent(content, id)
	if err != nil {
		return errors.Wrap(err, "UpdateContent")
	}

	err = cache.Level2.SaveTagsFromOperation(content.Data.JsonMetadata, id)
	if err != nil {
		return errors.Wrap(err, "UpdateContent SaveTagsFromOperation")
	}

	return nil
}
