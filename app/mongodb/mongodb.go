package mongodb

import (
	"context"
	"gitlab.com/stihi/stihi-backend/app"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	EnvMongoDBFileConfig = "MONGO_DB_CONFIG"
)

var (
	Settings *Config
)

type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (cfg *Config) URI() string {
	authStr := ""
	if cfg.User != "" {
		authStr = cfg.User+":"+cfg.Password+"@"
	}

	return "mongodb://"+authStr+cfg.Host+":"+strconv.FormatInt(int64(cfg.Port), 10)
}

func LoadConfig() {
	if Settings != nil {
		return
	}

	InitFromFile(os.Getenv(EnvMongoDBFileConfig))
}

func InitFromFile(mongoConfigFileName string) {
	if Settings == nil {
		Settings = &Config{}
	}

	if mongoConfigFileName == "" {
		app.Error.Fatalf("MongoDb config file name required!")
	}

	_, err := os.Stat(mongoConfigFileName)
	if os.IsNotExist(err) {
		app.Error.Fatalf("MongoDb config file '%s' not exists.", mongoConfigFileName)
	}

	dat, err := ioutil.ReadFile(mongoConfigFileName)
	if err != nil {
		app.Error.Fatalln(err)
	}

	err = yaml.Unmarshal(dat, Settings)
	if err != nil {
		app.Error.Fatalf("error: %v", err)
	}
}

type Connection struct {
	Uri				string
	databaseName	string
	collectionName	string

	Client     *mongo.Client
	Database   *mongo.Database
	Collection *mongo.Collection

	ReopenTryCount	int
	ReopenDelay		time.Duration
}

func New() (*Connection, error) {
	connection := Connection{
		Uri: Settings.URI(),
		ReopenTryCount: 10,
		ReopenDelay: 10 * time.Second,
	}

	return &connection, nil
}

func (connection *Connection) Connect() error {
	var err error

	connection.Client, err = mongo.NewClient( options.Client().ApplyURI( connection.Uri ) )
	if err != nil {
		return err
	}

	err = connection.Client.Connect(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (connection *Connection) Close() {
	_ = connection.Client.Disconnect(context.Background())
}

func (connection *Connection) SetDB(name string) *Connection {
	connection.databaseName = name
	connection.Database = connection.Client.Database(name)
	return connection
}

func (connection *Connection) SetCollection(name string) *Connection {
	connection.collectionName = name
	connection.Collection = connection.Database.Collection(name)
	return connection
}

func (connection *Connection) Check() {
	if connection.Client == nil {
		err := connection.Connect()
		if err != nil {
			app.Error.Printf("Mongodb connect error: %s\n", err)
		}
	}

	err := connection.Client.Ping(context.Background(), nil)
	if err == nil {
		return
	}

	// Try reconnect in loop 10 times with delay 10 seconds
	counter := connection.ReopenTryCount
	for err != nil && counter > 0 {
		time.Sleep( connection.ReopenDelay )
		err = connection.Connect()
		counter--
	}
	if err != nil {
		log.Fatalf("Cannot reconnect to mongodb: %s", err)
	}

	if connection.Database != nil {
		connection.SetDB( connection.databaseName )
	}

	if connection.Collection != nil {
		connection.SetCollection( connection.collectionName )
	}
}