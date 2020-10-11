package redis_db

import (
	"time"

	"gopkg.in/redis.v4"
)

type RedisMutex struct {
	Name   string
	Server *redis.Client
	Delay  time.Duration
	Expiry time.Duration
	Try    int
	NoWait bool
}

func (mutex *RedisMutex) Lock() bool {
	var locked bool

	if mutex.Delay == 0 {
		mutex.Delay = 100 * time.Millisecond
	}

	if mutex.Try == 0 {
		mutex.Try = 50
	}

	counter := mutex.Try

	start := true
	locked = true
	for start || (!locked && counter > 0) {
		if !locked {
			time.Sleep(mutex.Delay)
		}

		locked = mutex.Server.SetNX(mutex.Name, "1", mutex.Expiry).Val()

		if mutex.NoWait {
			return locked
		}

		start = false
		counter--
	}

	return locked
}

func (mutex *RedisMutex) Unlock() {
	mutex.Server.Del(mutex.Name)
}

func (conn *RedisConnection) NewMutex(name string) *RedisMutex {
	mutex := RedisMutex{
		Name:   name,
		Server: conn.MainServers[0].Connection,
	}

	return &mutex
}
