package redis_db

import (
	"fmt"
	"gopkg.in/redis.v4"
)

const (
	MaxOperationsBeforeQueueUpdates = 100
)

var (
	updateQueueSizesCounter int
)

// Load queue sizes for servers
func (conn *RedisConnection) QueueSizesUpdate(queue string, servers *[]RedisServer) {
	updateQueueSizesCounter = MaxOperationsBeforeQueueUpdates
	for i, server := range *servers {
		qLen, err := server.Connection.LLen(queue).Result()
		if err != nil {
			qLen = 0
		}

		(*servers)[i].QueueSizesMutex.Lock()
		(*servers)[i].QueueSizes[queue] = qLen
		(*servers)[i].QueueSizesMutex.Unlock()
	}
}

func (conn *RedisConnection) QueueSizesUpdateAll(queue string) {
	conn.QueueSizesUpdate(queue, &conn.MainServers)
	if conn.ReconfigureMode {
		conn.QueueSizesUpdate(queue, &conn.OldServers)
	}
}

func (conn *RedisConnection) CheckUpdateQueueSizes(queue string) {
	updateQueueSizesCounter--
	if updateQueueSizesCounter <= 0 {
		conn.QueueSizesUpdateAll(queue)
	}
}

func (conn *RedisConnection) MinQueueServerSearchBy(queue string, servers *[]RedisServer) RedisServer {
	var min int64
	var res RedisServer
	min = -1
	for _, server := range *servers {
		server.QueueSizesMutex.Lock()
		if min == -1 || server.QueueSizes[queue] <= min {
			res = server
			min = server.QueueSizes[queue]
		}
		server.QueueSizesMutex.Unlock()
	}
	return res
}

func (conn *RedisConnection) MinQueueMainServerSearch(queue string) RedisServer {
	conn.CheckUpdateQueueSizes(queue)
	return conn.MinQueueServerSearchBy(queue, &conn.MainServers)
}

func (conn *RedisConnection) MinQueueOldServerSearch(queue string) RedisServer {
	conn.CheckUpdateQueueSizes(queue)
	return conn.MinQueueServerSearchBy(queue, &conn.OldServers)
}

func (conn *RedisConnection) MaxQueueServerSearchBy(queue string, servers *[]RedisServer) (RedisServer, error) {
	var max int64
	var res RedisServer
	max = 0
	for _, server := range *servers {
		server.QueueSizesMutex.Lock()
		if server.QueueSizes[queue] > max {
			res = server
			max = server.QueueSizes[queue]
		}
		server.QueueSizesMutex.Unlock()
	}
	var err error
	if max == 0 {
		err = redis.Nil
	}
	return res, err
}

func (conn *RedisConnection) MaxQueueMainServerSearch(queue string) (RedisServer, error) {
	conn.CheckUpdateQueueSizes(queue)
	return conn.MaxQueueServerSearchBy(queue, &conn.MainServers)
}

func (conn *RedisConnection) MaxQueueOldServerSearch(queue string) (RedisServer, error) {
	conn.CheckUpdateQueueSizes(queue)
	return conn.MaxQueueServerSearchBy(queue, &conn.OldServers)
}

func (conn *RedisConnection) AddQueue(queue string, val string) error {
	minQueueServer := conn.MinQueueMainServerSearch(queue)
	incQueueSize(queue, &minQueueServer)
	return minQueueServer.Connection.RPush(queue, val).Err()
}

func (conn *RedisConnection) GetQueue(queue string) (string, error) {
	conn.QueueSizesUpdate(queue, &conn.MainServers)
	maxQueueServer, err := conn.MaxQueueMainServerSearch(queue)
	var val string
	if err == nil {
		val, err = maxQueueServer.Connection.LPop(queue).Result()
	}

	if err != nil {
		conn.QueueSizesUpdate(queue, &conn.OldServers)
		maxQueueOldServer, s_err := conn.MaxQueueOldServerSearch(queue)
		if s_err == nil {
			val, err = maxQueueOldServer.Connection.LPop(queue).Result()

			if err == nil {
				decQueueSize(queue, &maxQueueOldServer)
			}
		}
	} else {
		decQueueSize(queue, &maxQueueServer)
	}

	return val, err
}

func (conn *RedisConnection) GetQueueBatch(queue string, batch_size int64) ([]string, error) {
	mutex := conn.NewMutex(fmt.Sprintf("lock_queue_batch:%s", queue))
	mutex.Lock()
	defer mutex.Unlock()

	conn.QueueSizesUpdate(queue, &conn.MainServers)
	maxQueueServer, err := conn.MaxQueueMainServerSearch(queue)
	var val []string
	if err == nil {
		val, err = maxQueueServer.Connection.LRange(queue, 0, batch_size-1).Result()
	}

	if err != nil {
		conn.QueueSizesUpdate(queue, &conn.OldServers)
		maxQueueOldServer, s_err := conn.MaxQueueOldServerSearch(queue)
		if s_err == nil {
			val, err = maxQueueOldServer.Connection.LRange(queue, 0, batch_size-1).Result()

			if err == nil {
				maxQueueOldServer.Connection.LTrim(queue, int64(len(val)), -1)
				decQueueSizeBy(queue, &maxQueueOldServer, int64(len(val)))
			}
		}
	} else {
		maxQueueServer.Connection.LTrim(queue, int64(len(val)), -1)
		decQueueSizeBy(queue, &maxQueueServer, int64(len(val)))
	}

	return val, err
}

func (conn *RedisConnection) IsEmptyQueue(queue string) bool {
	conn.QueueSizesUpdate(queue, &conn.MainServers)
	maxQueueServer, err := conn.MaxQueueMainServerSearch(queue)
	lt_zero := true
	if err == nil {
		maxQueueServer.QueueSizesMutex.Lock()
		lt_zero = maxQueueServer.QueueSizes[queue] <= 0
		maxQueueServer.QueueSizesMutex.Unlock()
	}
	return err != nil || lt_zero
}

func (conn *RedisConnection) QueueSize(queue string) int64 {
	conn.QueueSizesUpdate(queue, &conn.MainServers)
	size := int64(0)
	for _, server := range conn.MainServers {
		server.QueueSizesMutex.Lock()
		size = size + server.QueueSizes[queue]
		server.QueueSizesMutex.Unlock()
	}
	return size
}

func (conn *RedisConnection) CleanQueueBy(queue string, servers *[]RedisServer) {
	for _, server := range *servers {
		server.Connection.Del(queue)
		server.QueueSizesMutex.Lock()
		server.QueueSizes[queue] = 0
		server.QueueSizesMutex.Unlock()
	}
}

func (conn *RedisConnection) CleanQueue(queue string) {
	conn.CleanQueueBy(queue, &conn.MainServers)
	if conn.ReconfigureMode {
		conn.CleanQueueBy(queue, &conn.OldServers)
	}
}

func incQueueSize(queue string, server *RedisServer) {
	server.QueueSizesMutex.Lock()
	server.QueueSizes[queue] += 1
	server.QueueSizesMutex.Unlock()
}

func decQueueSize(queue string, server *RedisServer) {
	decQueueSizeBy(queue, server, 1)
}

func decQueueSizeBy(queue string, server *RedisServer, size int64) {
	server.QueueSizesMutex.Lock()
	if server.QueueSizes[queue] <= 0 {
		server.QueueSizes[queue] = 0
	} else {
		server.QueueSizes[queue] -= size
	}
	server.QueueSizesMutex.Unlock()
}
