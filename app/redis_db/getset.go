package redis_db

import (
	"gopkg.in/redis.v4"
	"strconv"
	"time"
)

func (conn *RedisConnection) Get(key string) (string, error) {
	shardIndex := ShardIndex(key)

	return conn.GetShard(shardIndex, key)
}

func (conn *RedisConnection) GetShard(shardIndex uint64, key string) (string, error) {
	mainServer := conn.mainShardServer(shardIndex)
	val, err := mainServer.Connection.Get(key).Result()

	if err != nil && err.Error() == redis.Nil.Error() && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			val, err = oldServer.Connection.Get(key).Result()
			if err == nil {
				// Move to main
				mainServer.Connection.Set(key, val, 0)
				oldServer.Connection.Del(key)
			}
		}
	}

	return val, err
}

func (conn *RedisConnection) Set(key string, val string) error {
	shardIndex := ShardIndex(key)

	return conn.SetShard(shardIndex, key, val)
}


func (conn *RedisConnection) SetExpire(key string, expire time.Duration) error {
	shardIndex := ShardIndex(key)

	return conn.SetExpireShard(shardIndex, key, expire)
}

func (conn *RedisConnection) GetTTL(key string) (int64, error) {
	shardIndex := ShardIndex(key)

	return conn.GetTTLShard(shardIndex, key)
}

func (conn *RedisConnection) GetTTLShard(shardIndex uint64, key string) (int64, error) {
	mainServer := conn.mainShardServer(shardIndex)
	res, err := mainServer.Connection.PTTL(key).Result()

	if err == nil && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			oldServer.Connection.Del(key)
		}
	}

	if err != nil {
		return -1, err
	}

	return res.Milliseconds(), nil
}

func (conn *RedisConnection) SetShard(shardIndex uint64, key string, val string) error {
	mainServer := conn.mainShardServer(shardIndex)
	_, err := mainServer.Connection.Set(key, val, 0).Result()

	if err == nil && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			oldServer.Connection.Del(key)
		}
	}

	return err
}

func (conn *RedisConnection) SetExpireShard(shardIndex uint64, key string, expire time.Duration) error {
	mainServer := conn.mainShardServer(shardIndex)
	_, err := mainServer.Connection.Expire(key, expire).Result()

	if err == nil && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			oldServer.Connection.Del(key)
		}
	}

	return err
}

func (conn *RedisConnection) Del(key string) error {
	shardIndex := ShardIndex(key)

	return conn.DelShard(shardIndex, key)
}

func (conn *RedisConnection) DelShard(shardIndex uint64, key string) error {
	mainServer := conn.mainShardServer(shardIndex)
	_, err := mainServer.Connection.Del(key).Result()

	if conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			oldServer.Connection.Del(key)
		}
	}

	return err
}

func (conn *RedisConnection) RPush(key string, val string) error {
	shardIndex := ShardIndex(key)

	return conn.RPushShard(shardIndex, key, val)
}

func (conn *RedisConnection) RPushShard(shardIndex uint64, key string, val string) error {
	mainServer := conn.mainShardServer(shardIndex)
	if conn.ReconfigureMode {
		// Check exists key and if not exists - copy from old
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) &&
			!mainServer.Connection.Exists(key).Val() &&
			oldServer.Connection.Exists(key).Val() {
			old_list, err := oldServer.Connection.LRange(key, 0, -1).Result()
			if err == nil {
				for _, s := range old_list {
					mainServer.Connection.RPush(key, s)
				}
			}
		}
	}

	_, err := mainServer.Connection.RPush(key, val).Result()

	if err == nil && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			oldServer.Connection.Del(key)
		}
	}

	return err
}

func (conn *RedisConnection) LPop(key string) (string, error) {
	shardIndex := ShardIndex(key)

	return conn.LPopShard(shardIndex, key)
}

func (conn *RedisConnection) LPopShard(shardIndex uint64, key string) (string, error) {
	mainServer := conn.mainShardServer(shardIndex)
	val, err := mainServer.Connection.LPop(key).Result()

	if err != nil && err.Error() == redis.Nil.Error() && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			val, err = oldServer.Connection.LPop(key).Result()
		}
	}

	return val, err
}

func (conn *RedisConnection) LRange(key string, start int64, finish int64) ([]string, error) {
	shardIndex := ShardIndex(key)

	return conn.LRangeShard(shardIndex, key, start, finish)
}

func (conn *RedisConnection) LRangeShard(shardIndex uint64, key string, start int64, finish int64) ([]string, error) {
	mainServer := conn.mainShardServer(shardIndex)
	val, err := mainServer.Connection.LRange(key, start, finish).Result()

	if ((err != nil && err.Error() == redis.Nil.Error()) || len(val) == 0) && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			val, err = oldServer.Connection.LRange(key, start, finish).Result()
			if err == nil {
				// Move to main
				old_list, err1 := oldServer.Connection.LRange(key, 0, -1).Result()
				if err1 == nil {
					for _, s := range old_list {
						mainServer.Connection.RPush(key, s)
					}
				}
				oldServer.Connection.Del(key)
			}
		}
	}

	return val, err
}

func (conn *RedisConnection) LRem(key string, remValue string) error {
	shardIndex := ShardIndex(key)

	return conn.LRemShard(shardIndex, key, remValue)
}

func (conn *RedisConnection) LRemShard(shardIndex uint64, key string, remValue string) error {
	mainServer := conn.mainShardServer(shardIndex)
	err := mainServer.Connection.LRem(key, 0, remValue).Err()

	if (err != nil && err.Error() == redis.Nil.Error()) && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			err = oldServer.Connection.LRem(key, 0, remValue).Err()
		}
	}

	return err
}

func (conn *RedisConnection) Exists(key string) bool {
	shardIndex := ShardIndex(key)

	return conn.ExistsShard(shardIndex, key)
}

func (conn *RedisConnection) ExistsShard(shardIndex uint64, key string) bool {
	mainServer := conn.mainShardServer(shardIndex)
	res, err := mainServer.Connection.Exists(key).Result()

	if (err != nil && err.Error() == redis.Nil.Error()) && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			res, err = oldServer.Connection.Exists(key).Result()
		}
	}

	return err == nil && res
}

func (conn *RedisConnection) LLen(key string) int64 {
	shardIndex := ShardIndex(key)

	return conn.LLenShard(shardIndex, key)
}

func (conn *RedisConnection) LLenShard(shardIndex uint64, key string) int64 {
	mainServer := conn.mainShardServer(shardIndex)
	res, err := mainServer.Connection.LLen(key).Result()

	if (err != nil && err.Error() == redis.Nil.Error()) && conn.ReconfigureMode {
		oldServer := conn.oldShardServer(shardIndex)
		if !serversEquals(mainServer, oldServer) {
			res, err = oldServer.Connection.LLen(key).Result()
		}
	}

	if err != nil {
		res = 0
	}

	return res
}

func (conn *RedisConnection) Incr(key string) (uint64, error) {
	shardIndex := ShardIndex(key)

	return conn.IncrShard(shardIndex, key)
}

func (conn *RedisConnection) IncrShard(shardIndex uint64, key string) (uint64, error) {
	mainServer := conn.mainShardServer(shardIndex)
	id, err := mainServer.Connection.Incr(key).Result()
	return uint64(id), err
}

func (conn *RedisConnection) Decr(key string) (uint64, error) {
	shardIndex := ShardIndex(key)

	return conn.DecrShard(shardIndex, key)
}

func (conn *RedisConnection) DecrShard(shardIndex uint64, key string) (uint64, error) {
	mainServer := conn.mainShardServer(shardIndex)
	id, err := mainServer.Connection.Decr(key).Result()
	return uint64(id), err
}

func (conn *RedisConnection) NewID(key string) (uint64, error) {
	id, err := conn.MainServers[0].Connection.Incr(key).Result()
	return uint64(id), err
}

func (conn *RedisConnection) GetID(key string) (uint64, error) {
	id, err := conn.MainServers[0].Connection.Get(key).Result()
	if err != nil {
		return 0, err
	}

	id_int, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, err
	}

	return id_int, err
}

func (conn *RedisConnection) SetID(key string, id uint64) error {
	err := conn.MainServers[0].Connection.Set(key, strconv.FormatUint(id, 10), -1).Err()
	return err
}
