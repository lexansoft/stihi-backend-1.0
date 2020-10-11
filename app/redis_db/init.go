package redis_db

import (
	"io/ioutil"
	"os"
	"sync"

	"gopkg.in/redis.v4"
	"gopkg.in/yaml.v2"
	"gitlab.com/stihi/stihi-backend/app"
	"github.com/pkg/errors"
)

const (
	EnvRedisFileConfig = "REDIS_CONFIG"
)

func New(configFileName string) (*RedisConnection, error) {
	conn := &RedisConnection{}

	err := conn.InitFromFile(configFileName)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (conn *RedisConnection) InitFromEnv() (error) {
	redisConfigFile := os.Getenv(EnvRedisFileConfig)
	if redisConfigFile == "" {
		return errors.New("redis config file name required ("+EnvRedisFileConfig+" environment)")
	}
	return conn.InitFromFile(redisConfigFile)
}

func (conn *RedisConnection) InitFromFile(confFile string) (error) {
	_, err := os.Stat(confFile)
	if os.IsNotExist(err) {
		return errors.New("redis config file '"+confFile+"' not exists.")
	}

	err = conn.ReadSettings(confFile)
	if err != nil {
		return err
	}

	err = conn.RedisInit()
	if err != nil {
		return err
	}

	return nil
}

func (conn *RedisConnection) ReadSettings(fileName string) (error) {
	dat, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(dat, conn)
	if err != nil {
		return err
	}

	return nil
}

func (conn *RedisConnection) RedisInit() (error) {
	if len(conn.MainServers) <= 0 {
		return errors.New("no main servers in config")
	}

	redisServersConnect(&conn.MainServers)
	queueSizesMutexInit(&conn.MainServers)
	redisServerQueueSizesInit(&conn.MainServers)

	if conn.ReconfigureMode {
		if len(conn.OldServers) <= 0 {
			return errors.New("no old servers in config for reconfigure mode")
		}

		redisServersConnect(&conn.OldServers)
		queueSizesMutexInit(&conn.OldServers)
		redisServerQueueSizesInit(&conn.OldServers)
	}

	return nil
}

func redisServersConnect(servers *[]RedisServer) {
	for i, server := range *servers {
		(*servers)[i].Connection = redis.NewClient(&redis.Options{
			Addr:     server.Addr,
			Password: server.Password,
			DB:       server.Db,
		})

		_, err := (*servers)[i].Connection.Ping().Result()

		if err != nil {
			(*servers)[i].Connection = nil
			app.Error.Fatalln("Error connection to Redis server " + server.Addr)
		}
	}
}

func queueSizesMutexInit(servers *[]RedisServer) {
	for i := range *servers {
		if (*servers)[i].QueueSizes == nil {
			(*servers)[i].QueueSizesMutex = &sync.Mutex{}
		}
	}
}

func redisServerQueueSizesInit(servers *[]RedisServer) {
	for i := range *servers {
		if (*servers)[i].QueueSizes == nil {
			(*servers)[i].QueueSizes = make(map[string]int64)
		}
	}
}

func serversEquals(server1 RedisServer, server2 RedisServer) bool {
	return server1.Addr == server2.Addr && server1.Db == server2.Db
}
