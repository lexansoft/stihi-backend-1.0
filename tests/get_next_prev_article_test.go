package tests

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"testing"
)

func TestGetNextPrevArticles(t *testing.T) {
	InitTestArticlesNextPrev(dbConn)

	cacheL2 := cache_level2.CacheLevel2{
		QueryProcessor: dbConn,
	}

	/*
	fmt.Printf("npArticle1Id: %d\n", npArticle1Id)
	fmt.Printf("npArticle2Id: %d\n", npArticle2Id)
	fmt.Printf("npArticle3Id: %d\n", npArticle3Id)
	fmt.Printf("npArticle4Id: %d\n", npArticle4Id)
	fmt.Printf("npArticle5Id: %d\n", npArticle5Id)
	*/

	/*
		Исходная статья - в середине списка
	*/

	// Сортировка по id - без фильтров
	prevArt, nextArt, err := cacheL2.GetArticlePrevNext(npArticle2Id, "a.id", false, nil, nil, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle1Id, "ID - NO FILTERS: Should by prev article as 'article1'.")
	assert.Equal(t, nextArt.Id, npArticle3Id, "ID - NO FILTERS: Should by next article as 'article3'.")

	// Обратный порядок по id - без фильтров
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle2Id, "a.id", true, nil, nil, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle3Id, "ID DESC - NO FILTERS: Should by prev article as 'article3'.")
	assert.Equal(t, nextArt.Id, npArticle1Id, "ID DESC - NO FILTERS: Should by next article as 'article1'.")

	// Сортировка по времени - без фильтров
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle2Id, "a.time", false, nil, nil, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle1Id, "TIME - NO FILTERS: Should by prev article as 'article1'.")
	assert.Equal(t, nextArt.Id, npArticle3Id, "TIME - NO FILTERS: Should by next article as 'article3'.")

	// Обратный порядок по времени - без фильтров
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle2Id, "a.time", true, nil, nil, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle3Id, "TIME DESC - NO FILTERS: Should by prev article as 'article3'.")
	assert.Equal(t, nextArt.Id, npArticle1Id, "TIME DESC - NO FILTERS: Should by next article as 'article1'.")

	// Сортировка по id - фильтр по тэгу
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle3Id, "a.id", false, nil, nil, []string{"tag1"}, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle1Id, "ID - FILTER TAG: Should by prev article as 'article1'.")
	assert.Equal(t, nextArt.Id, npArticle5Id, "ID - FILTER TAG: Should by next article as 'article5'.")

	// Обратный порядок по id - фильтр по тэгу
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle3Id, "a.id", true, nil, nil, []string{"tag1"}, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle5Id, "ID DESC - FILTER TAG: Should by prev article as 'article5'.")
	assert.Equal(t, nextArt.Id, npArticle1Id, "ID DESC - FILTER TAG: Should by next article as 'article1'.")

	// Сортировка по времени - фильтр по тэгу
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle3Id, "a.time", false, nil, nil, []string{"tag1"}, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle1Id, "TIME - FILTER TAG: Should by prev article as 'article1'.")
	assert.Equal(t, nextArt.Id, npArticle5Id, "TIME - FILTER TAG: Should by next article as 'article5'.")

	// Обратный порядок по времени - фильтр по тэгу
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle3Id, "a.time", true, nil, nil, []string{"tag1"}, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle5Id, "TIME DESC - FILTER TAG: Should by prev article as 'article5'.")
	assert.Equal(t, nextArt.Id, npArticle1Id, "TIME DESC - FILTER TAG: Should by next article as 'article1'.")

	// Сортировка по id - фильтр по пользователю
	authorFilter1 := map[string]interface{}{ "a.author": "test-user1" }
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle2Id, "a.id", false, nil, authorFilter1, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle1Id, "ID - FILTER AUTHOR: Should by prev article as 'article1'.")
	assert.Equal(t, nextArt.Id, npArticle4Id, "ID - FILTER AUTHOR: Should by next article as 'article4'.")

	// Обратный порядок по id - фильтр по пользователю
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle2Id, "a.id", true, nil, authorFilter1, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle4Id, "ID DESC - FILTER AUTHOR: Should by prev article as 'article4'.")
	assert.Equal(t, nextArt.Id, npArticle1Id, "ID DESC - FILTER AUTHOR: Should by next article as 'article1'.")

	/*
		Исходная статья на краю списка
	*/

	// Сортировка по id - без фильтров - начало списка
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle1Id, "a.id", false, nil, nil, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, prevArt.Id < npArticle1Id || prevArt.Id <= 0, "ID - NO FILTERS: Should by prev article as id 0.")
	assert.Equal(t, nextArt.Id, npArticle2Id, "ID - NO FILTERS: Should by next article as 'article2'.")

	// Сортировка по id - без фильтров - конец списка
	prevArt, nextArt, err = cacheL2.GetArticlePrevNext(npArticle5Id, "a.id", false, nil, nil, nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, prevArt.Id, npArticle4Id, "ID - NO FILTERS: Should by prev article as 'article4'.")
	assert.True(t, nextArt.Id <= 0 || nextArt.Id > npArticle5Id, "ID - NO FILTERS: Should by next article as id 0.")
}
