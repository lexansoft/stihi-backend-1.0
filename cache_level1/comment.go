package cache_level1

import (
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"

	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

func CommentKey(id int64, rawFormat bool) string {
	raw := ":"
	if rawFormat {
		raw = ":raw:"
	}
	return CommentPrefix+raw+strconv.FormatInt(id, 10)
}

func (cache *CacheLevel1) SaveCommentFromOperation(op *operations.CreateMessageData, ts time.Time) (int64, error) {
	// Проверяем на "@@" в начале body и игнорируем если есть
	if strings.HasPrefix(op.Body, "@@") {
		return -1, errors.New("bad content in body - starting from @@: "+op.Id.Author+":"+op.Id.Permlink)
	}

	// Проверяем нет-ли уже такого коммента
	if cache.IsContentPresent(op.Id.Author, op.Id.Permlink) {
		// Если такой коммент уже есть - обновляем
		content := cache_level2.CommentOperationToComment(op)
		return cache.UpdateComment(content, ts)
	}

	// Проверяем есть-ли у нас в базе parent content
	if !cache.IsContentPresent(op.ParentId.Author, op.ParentId.Permlink) {
		return -1, errors.New("skip comment - parent content not exists: "+op.ParentId.Author+":"+op.ParentId.Permlink)
	}

	id, err := cache.Level2.SaveCommentFromOperation(op, ts, true)
	if err != nil {
		return -1, err
	}

	_ = cache.saveContentIdLink(id, op.Id.Author, op.Id.Permlink)

	// TODO: Проследить какой формат записывается в данном случае - raw или обработанный
	key := CommentKey( id, true )

	content := cache_level2.CommentOperationToComment(op)
	content.Id = id
	content.Time = ts.String()

	err = cache.Save( key, content )
	if err != nil {
		return id, err
	}

	return id, nil
}

func (cache *CacheLevel1) UpdateComment(content *cache_level2.Comment, ts time.Time) (int64, error) {
	// Проверяем на "@@" в начале body и игнорируем если есть
	if strings.HasPrefix(content.Body, "@@") {
		return -1, errors.New("bad content in body - starting from @@: "+content.Author+":"+content.Permlink)
	}

	// Проверяем нет-ли уже такого коммента
	if !cache.IsContentPresent(content.Author, content.Permlink) {
		return -1, errors.New("skip comment - not exists: "+ content.Author+":"+ content.Permlink)
	}

	// Проверяем есть-ли у нас в базе parent content
	if !cache.IsContentPresent(content.ParentAuthor, content.ParentPermlink) {
		return -1, errors.New("skip comment - parent content not exists: "+ content.ParentAuthor+":"+ content.ParentPermlink)
	}

	op := cache_level2.CommentToCommentOperation(content)

	id, err := cache.Level2.SaveCommentFromOperation(op, ts, false)
	if err != nil {
		return -1, err
	}

	cache.saveContentIdLink(id, content.Author, content.Permlink)

	key := CommentKey( id, true )

	tags, err := cache_level2.ParseMeta(op.JsonMetadata)
	content.Metadata = map[string]interface{}{
		"tags": tags.Tags(),
	}

	err = cache.Save( key, content )
	if err != nil {
		return id, err
	}

	return id, nil
}

func (cache *CacheLevel1) GetComment(id int64, rawFormat, adminMode bool) (*cache_level2.Comment, error) {
	key := CommentKey( id, rawFormat )

	op := &cache_level2.Comment{}
	err := cache.GetObject(key, op)
	if err == nil {
		return op, nil
	}

	if !cache.Lock(key) {
		return nil, errors.New("can not create cache level 1 lock for: "+key)
	}
	defer cache.Unlock(key)

	op, err = cache.Level2.GetComment(id, rawFormat, adminMode)
	if err != nil {
		return nil, err
	}

	cache.loadUserNamesForComments(&[]*cache_level2.Comment{ op })
	cache.loadRewardsForComments(&[]*cache_level2.Comment{ op })

	_ = cache.Save(key, op)

	return op, nil
}

func (cache *CacheLevel1) GetCommentsForContentFull(id int64, adminMode bool) (*[]*cache_level2.Comment, []int64, error) {
	list, ids, err := cache.Level2.GetCommentsForContentFull(id, adminMode)
	if err != nil {
		return nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, ids, nil
}

func (cache *CacheLevel1) GetCommentsForContent(id int64, adminMode bool) (*[]*cache_level2.Comment, []int64, error) {
	list, ids, err := cache.Level2.GetCommentsForContent(id, adminMode)
	if err != nil {
		return nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, ids, nil
}

func (cache *CacheLevel1) GetUserCommentsFull(userId int64, pagination cache_level2.PaginationParams, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	// Если в pagination значимый Id - получаем для него объект-коммент
	paginationComment, err := cache.getCommentByPagination(pagination, adminMode)
	if err != nil { return nil, nil, nil, err }

	list, l1, l2, err := cache.Level2.GetUserCommentsFull(userId, pagination, paginationComment, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}

func (cache *CacheLevel1) GetUserContentCommentsFull(userId int64, pagination cache_level2.PaginationParams, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	paginationComment, err := cache.getCommentByPagination(pagination, adminMode)
	if err != nil { return nil, nil, nil, err }

	list, l1, l2, err := cache.Level2.GetUserContentCommentsFull(userId, pagination, paginationComment, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}

func (cache *CacheLevel1) GetUserComments(userId int64, pagination cache_level2.PaginationParams, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	paginationComment, err := cache.getCommentByPagination(pagination, adminMode)
	if err != nil { return nil, nil, nil, err }

	list, l1, l2, err := cache.Level2.GetUserComments(userId, pagination, paginationComment, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}

func (cache *CacheLevel1) GetUserCommentsCount(userId int64, adminMode bool) (int64, error) {
	return cache.Level2.GetUserCommentsCount(userId, adminMode)
}

func (cache *CacheLevel1) GetUserContentComments(userId int64, pagination cache_level2.PaginationParams, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	paginationComment, err := cache.getCommentByPagination(pagination, adminMode)
	if err != nil { return nil, nil, nil, err }

	list, l1, l2, err := cache.Level2.GetUserContentComments(userId, pagination, paginationComment, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}

func (cache *CacheLevel1) GetUserContentCommentsCount(userId int64, adminMode bool) (int64, error) {
	return cache.Level2.GetUserContentCommentsCount(userId, adminMode)
}

func (cache *CacheLevel1) getCommentByPagination(pagination cache_level2.PaginationParams, adminMode bool) (*cache_level2.Comment, error) {
	var paginationComment *cache_level2.Comment
	if pagination.Mode == cache_level2.PaginationModeAfter || pagination.Mode == cache_level2.PaginationModeBefore {
		var err error
		paginationComment, err = cache.GetComment(pagination.Id, false, adminMode)
		if err != nil { return nil, err }

		// Переформатирование времени в нужном виде
		if paginationComment.Time != "" {
			list := strings.Split(paginationComment.Time, " ")
			paginationComment.Time = list[0]+" "+list[1]
		}
	}

	return paginationComment, nil
}

/* В кэше запоминаем:
	1. Тригерное время последнего обновления всех данных этого списка;
	2. Фактическое время последнего обновления данного списка.
	   Если данное время МЕНЬШЕ чем тригерное время - читаем все (ID из п.3 и список) из БД и устанавливаем данное время
	   в текущее;
	3. Первый закэшированный ID комментария. Если он есть - вместо GetAllCommentsLast используем GetAllCommentsAfter;
	4. Полный список объектов в плоском виде;
	5. Индекс: хэш map[<id comment>]<позиция в списке из п.4>
*/
func (cache *CacheLevel1) GetAllCommentsLast(count int, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	list, l1, l2, err := cache.Level2.GetAllCommentsLast(count, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}

func (cache *CacheLevel1) GetAllCommentsBefore(before int64, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	list, l1, l2, err := cache.Level2.GetAllCommentsBefore(before, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}

func (cache *CacheLevel1) GetAllCommentsAfter(after int64, count int, adminMode bool) (*[]*cache_level2.Comment, []int64, []int64, error) {
	list, l1, l2, err := cache.Level2.GetAllCommentsAfter(after, count, adminMode)
	if err != nil {
		return nil, nil, nil, err
	}

	cache.loadUserNamesForComments(list)
	cache.loadRewardsForComments(list)

	return list, l1, l2, nil
}
