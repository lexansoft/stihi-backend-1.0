package cache_level1

import (
	"github.com/pkg/errors"
	"gopkg.in/redis.v4"
)

func StateKey(id string) string {
	return "state_data:"+id
}

func (cache *CacheLevel1) SaveState(id string, value string) error {
	err := cache.Level2.SaveState(id, value)
	if err != nil {
		return err
	}

	key := StateKey( id )
	err = cache.RedisConn.Set(key, value)
	if err != nil {
		return err
	}

	return nil
}

func (cache *CacheLevel1) GetState(id string) (string, error) {
	key := StateKey(id)

	val, err := cache.RedisConn.Get(key)
	if err == nil {
		return val, nil
	}
	if err != nil && err != redis.Nil {
		return "", err
	}

	if !cache.Lock(key) {
		return "", errors.New("can not create cache level 1 lock for: "+key)
	}
	defer cache.Unlock(key)

	val, err = cache.Level2.GetState(id)
	if err != nil {
		return "", err
	}

	return val, nil
}
