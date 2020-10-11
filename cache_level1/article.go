package cache_level1

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

// TODO: Периодически производить синхронизацию кэшированных данных о выплатах к статье через метод blockchain.SyncContent
func ArticleKey(id int64) string {
	return ArticlePrefix+":"+strconv.FormatInt(id, 10)
}

func (cache *CacheLevel1) SaveArticleFromOperation(op *operations.CreateMessageData, ts time.Time) (int64, error) {
	// Проверяем что статья содержит нужные тэги
	if !cache_level2.IsStihiContent(op.JsonMetadata) {
		// Сохраняем в кэш с id = -1
		_ = cache.saveContentIdLink(-1, op.Id.Author, op.Id.Permlink)
		return -1, errors.New(fmt.Sprintf("skip article - no stihi tags present: %+v", op.JsonMetadata))
	}

	content := cache_level2.CommentOperationToArticle(op)
	if cache.IsContentPresent(op.Id.Author, op.Id.Permlink) {
		// Если такая статья есть - обновляем
		return cache.UpdateArticle(content, ts)
	}

	id, err := cache.Level2.SaveArticleFromOperation(op, ts)
	if err != nil {
		return -1, err
	}

	_ = cache.saveContentIdLink(id, op.Id.Author, op.Id.Permlink)

	key := ArticleKey( id )

	content.Id = id
	content.Time = ts.String()

	err = cache.Save( key, content )
	if err != nil {
		return id, err
	}

	return id, nil
}

func (cache *CacheLevel1) UpdateArticle(content *cache_level2.Article, ts time.Time) (int64, error) {
	if !cache.IsContentPresent(content.Author, content.Permlink) {
		return -1, errors.New("skip update article - not exists: "+ content.Author+":"+ content.Permlink)
	}

	op := cache_level2.ArticleToCommentOperation(content)

	id, err := cache.Level2.SaveArticleFromOperation(op, ts)
	if err != nil {
		return -1, err
	}

	_ = cache.saveContentIdLink(id, content.Author, content.Permlink)

	key := ArticleKey( id )

	tags, err := cache_level2.ParseMeta(op.JsonMetadata)
	content.Metadata = map[string]interface{}{
		"tags": tags.Tags(),
	}

	err = cache.Save( key, content)
	if err != nil {
		return id, err
	}

	return id, nil
}

func (cache *CacheLevel1) GetArticle(id int64, rawFormat, adminMode bool) (*cache_level2.Article, error) {
	key := ArticleKey( id )

	art := &cache_level2.Article{}
	err := cache.GetObject(key, art)
	if err == nil {
		ttl, err := cache.RedisConn.GetTTL(key)
		if err == nil && ttl > -1 {
			return art, nil
		}
	}

	if !cache.Lock(key) {
		return nil, errors.New("can not create cache level 1 lock for: "+key)
	}
	defer cache.Unlock(key)

	art, err = cache.Level2.GetArticle(id, rawFormat, adminMode)
	if err != nil {
		return nil, err
	}

	list := make([]*cache_level2.Article, 0)
	list = append(list, art)

	cache.loadTagsForArticles(&list)
	cache.loadUserNamesForArticles(&list)
	cache.loadRewardsForArticles(&list)

	cache.SaveEx(key, art, time.Minute)

	return art, nil
}

func (cache *CacheLevel1) GetArticlePreview(id int64, adminMode bool) (*cache_level2.Article, error) {
	key := ArticleKey( id )

	op := &cache_level2.Article{}
	err := cache.GetObject(key, op)
	if err == nil {
		return op, nil
	}

	if !cache.Lock(key) {
		return nil, errors.New("can not create cache level 1 lock for: "+key)
	}
	defer cache.Unlock(key)

	op, err = cache.Level2.GetArticlePreview(id, adminMode)
	if err != nil {
		return nil, err
	}

	list := make([]*cache_level2.Article, 0)
	list = append(list, op)
	cache.loadTagsForArticles(&list)
	cache.loadUserNamesForArticles(&list)
	cache.loadRewardsForArticles(&list)

	cache.SaveEx(key, op, time.Minute)

	return op, nil
}

func (cache *CacheLevel1) GetArticlesListByIds(ids *[]int64) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetArticlesListByIds(ids)
	if err != nil {
		return nil, err
	}

	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

/*
	Get...ArticlesAfter - получение определенного количества статей после указанной
*/
func (cache *CacheLevel1) GetArticlesAfter(lastArticle int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool, filter string) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetArticlesAfter(lastArticle, count, tags, rubrics, notMat, adminMode, filter)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetFollowArticlesAfter(userId, lastArticle int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetFollowArticlesAfter(userId, lastArticle, count, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetActualArticlesAfter(lastArticle int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetActualArticlesAfter(lastArticle, count, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetPopularArticlesAfter(lastArticle int64, count int, period int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetPopularArticlesAfter(lastArticle, count, period, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetBlogArticlesAfter(lastArticle int64, count int, userId int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetBlogArticlesAfter(lastArticle, count, userId, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

/*
	Get...ArticlesBefore - получение всех статей перед указанной
*/
func (cache *CacheLevel1) GetArticlesBefore(firstArticle int64, tags []string, rubrics []string, notMat bool, adminMode bool, filter string) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetArticlesBefore(firstArticle, tags, rubrics, notMat, adminMode, filter)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetFollowArticlesBefore(userId, firstArticle int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetFollowArticlesBefore(userId, firstArticle, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetActualArticlesBefore(firstArticle int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetActualArticlesBefore(firstArticle, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetPopularArticlesBefore(firstArticle, period int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetPopularArticlesBefore(firstArticle, period, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetBlogArticlesBefore(firstArticle, userId int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetBlogArticlesBefore(firstArticle, userId, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

/*
	Get...LastArticles - получение определенного количества последних статей
*/
func (cache *CacheLevel1) GetLastArticles(count int, tags []string, rubrics []string, notMat bool, adminMode bool, filter string) (*[]*cache_level2.Article, error) {
	app.Debug.Printf("GetLastArticles start\n")
	list, err := cache.Level2.GetLastArticles(count, tags, rubrics, notMat, adminMode, filter)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetFollowLastArticles(userId int64, count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetFollowLastArticles(userId, count, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetActualLastArticles(count int, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetActualLastArticles(count, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetPopularLastArticles(count int, period int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetPopularLastArticles(count, period, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetBlogLastArticles(count int, userId int64, tags []string, rubrics []string, notMat bool, adminMode bool) (*[]*cache_level2.Article, error) {
	list, err := cache.Level2.GetBlogLastArticles(count, userId, tags, rubrics, notMat, adminMode)
	if err != nil {
		return nil, err
	}

	// Отдельно загружаем тэги статей
	cache.loadTagsForArticles(list)
	cache.loadUserNamesForArticles(list)
	cache.loadRewardsForArticles(list)

	return list, nil
}

func (cache *CacheLevel1) GetArticlePrevNext(articleId int64, orderField string, orderDesc bool, join []cache_level2.JoinLink, filterList map[string]interface{}, tags []string, rubrics []string, notMat bool, adminMode bool) (*cache_level2.NavigationArticlePoint, *cache_level2.NavigationArticlePoint, error) {
	return cache.Level2.GetArticlePrevNext(articleId, orderField, orderDesc, join, filterList, tags, rubrics, notMat, adminMode)
}
