package redis_db

import (
	"github.com/dgryski/go-t1ha"
	"gitlab.com/stihi/stihi-backend/app"
)

func (conn *RedisConnection) mainShardServer(shardIndex uint64) RedisServer {
	return shardServer(shardIndex, conn.MainServers)
}

func (conn *RedisConnection) oldShardServer(shardIndex uint64) RedisServer {
	return shardServer(shardIndex, conn.OldServers)
}

func shardServer(shardIndex uint64, servers []RedisServer) RedisServer {
	if len(servers) <= 0 {
		app.Error.Fatalln("No redis servers in Settings.")
	}
	serverIndex := shardIndex % uint64(len(servers))
	return servers[serverIndex]
}

// Преобразование ключа в shard_index через хэширование (быстрое хэширование t1ha).
func ShardIndex(key string) uint64 {
	return t1ha.Sum64([]byte(key), 0)
}
