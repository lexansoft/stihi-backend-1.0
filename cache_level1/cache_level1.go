package cache_level1

import (
	"encoding/json"
	"sync"
	"time"

	"gitlab.com/stihi/stihi-backend/app/redis_db"
	"gitlab.com/stihi/stihi-backend/cache_level2"
)

type CacheLevel1 struct {
	Level2 *cache_level2.CacheLevel2

	RedisConn	*redis_db.RedisConnection
}

type CL1Transaction struct {
	CacheLevel1
}

const (
	ContentIdPrefix           = "content_id"
	ContentByIdPrefix         = "content_by_id"
	ContentNodeosIdPrefix     = "content_nodeos_id"
	ContentIdByNodeosIdPrefix = "content_id_by_nodeos_id"
	ContentNodeosIdByIdPrefix = "content_nodeos_id_by_id"
	ArticlePrefix             = "article"
	CommentPrefix             = "comment"
	UserPrefix                = "user"
	UserHistoryPrefix         = "user_history"
	UserIdPrefix              = "user_id"

	LockPrefix			= "lock"
)

var (
	DB		*CacheLevel1

	locks	map[string]bool
	mutex	*sync.Mutex
)

func (cache *CacheLevel1) StartTransaction() (*CL1Transaction, error) {
	trans, err := cache.Level2.StartTransaction()
	if err != nil {
		return nil, err
	}

	newCache := CL1Transaction{
		CacheLevel1{
			Level2: trans,
			RedisConn: cache.RedisConn,
		},
	}

	return &newCache, nil
}

func (trans *CL1Transaction) CommitTransaction() error {
	return trans.Level2.CommitTransaction()
}

func (trans *CL1Transaction) RollbackTransaction() error {
	return trans.Level2.RollbackTransaction()
}

func New(redisConfigFileName, dbConfigFileName, mongoConfigFileName string) (*CacheLevel1, error) {
	cacheL1 := &CacheLevel1{}

	cacheL2, err := cache_level2.New(dbConfigFileName, mongoConfigFileName)
	if err != nil {
		return nil, err
	}
	cacheL1.Level2 = cacheL2

	redisDb, err := redis_db.New(redisConfigFileName)
	if err != nil {
		return nil, err
	}
	cacheL1.RedisConn = redisDb

	mutex = &sync.Mutex{}
	locks = make(map[string]bool)

	return cacheL1, nil
}

func (cache *CacheLevel1) Save(key string, obj interface{}) error {
	opJson, err := json.Marshal(obj)
	if err != nil {
		return err
	}


	err = cache.RedisConn.Set(key, string(opJson))
	return err
}

func (cache *CacheLevel1) Clean(key string) error {
	err := cache.RedisConn.Del(key)
	return err
}

// Сохранение в кэше со временем "протухания"
func (cache *CacheLevel1) SaveEx(key string, obj interface{}, expire time.Duration) error {
	err := cache.Save(key, obj)
	if err != nil {
		return err
	}
	err = cache.RedisConn.SetExpire(key, expire)
	return err
}

func (cache *CacheLevel1) GetObject(key string, obj interface{}) error {
	dataJson, err := cache.RedisConn.Get(key)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(dataJson), obj)
	return err
}

func (cache *CacheLevel1) Lock(key string) bool {
	lockKey := LockPrefix+":"+key
	for {
		if getLocalLock(lockKey) {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		break
	}

	redisMutex := cache.RedisConn.NewMutex(lockKey)
	if !redisMutex.Lock() {
		return false
	}

	setLocalLock(lockKey)
	return true
}

func (cache *CacheLevel1) Unlock(key string) {
	lockKey := LockPrefix+":"+key
	delLocalLock(lockKey)

	redisMutex := cache.RedisConn.NewMutex(lockKey)
	redisMutex.Unlock()
}

func (cache *CacheLevel1) LockNoWait(key string) bool {
	lockKey := LockPrefix+":"+key
	if getLocalLock(lockKey) {
		return false
	}

	redisMutex := cache.RedisConn.NewMutex(lockKey)
	redisMutex.NoWait = true
	if !redisMutex.Lock() {
		return false
	}

	setLocalLock(lockKey)
	return true
}

func getLocalLock(lockKey string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	lock, ok := locks[lockKey]
	if !ok {
		return false
	}

	return lock
}

func setLocalLock(lockKey string) {
	mutex.Lock()
	defer mutex.Unlock()

	locks[lockKey] = true
}

func delLocalLock(lockKey string) {
	mutex.Lock()
	defer mutex.Unlock()

	delete(locks, lockKey)
}