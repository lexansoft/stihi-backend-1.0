package cache_level1

import (
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cache_level2"
)

func (cache *CacheLevel1) GetInvitesList( count int ) (*[]*cache_level2.Invite, error) {
	list, err := cache.Level2.GetInvitesList(count)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}

	// Загружаем информацию об авторах
	for i, inv := range *list {
		nodeosName := cache.GetNodeosName(inv.AuthorName)
		user, err := cache.GetUserInfoByName(nodeosName)
		if err != nil {
			app.Error.Println(err)
		} else if (*list)[i] != nil && user != nil {
			(*list)[i].Author = *user
		}
	}

	return list, nil
}

func (cache *CacheLevel1) CreateInvite( login, payData string ) error {
	return cache.Level2.CreateInvite( login, payData )
}
