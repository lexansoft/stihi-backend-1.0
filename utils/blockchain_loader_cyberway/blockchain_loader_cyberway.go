package main

import (
	"flag"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/config"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"gitlab.com/stihi/stihi-backend/cyber"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	PidFileName = "/var/tmp/stihi_blockchain_loader_cyberway.pid"
	LogFileName = "/var/log/stihi/blockchain_loader_cyberway.log"

	SaveAfterBlocks = 10000
)

var (
	build				string

	appConfigFileName   string
	dbConfigFileName	string
	redisConfigFileName	string

	forceStop			bool
	Config				config.SBConfig

	DB					*cache_level1.CacheLevel1

	lastBlock			int64
	SleepBlockTime		time.Duration = 3 * time.Second

	wgMainLoop			sync.WaitGroup = sync.WaitGroup{}
)

// Демон, который загружает блоки из блокчейна в БД postgresql

func finish() {
	app.Info.Printf("Stop daemon with pid: %d", os.Getpid())
	forceStop = true
	wgMainLoop.Wait()
	os.Exit(0)
}

func main() {
	defer app.PanicHandle()

	if build != "" {
		app.Info.Printf("Build: %s", build)
	}

	var pidFile string
	flag.StringVar(&appConfigFileName, 		"config", 			"", "Application config file name")
	flag.StringVar(&dbConfigFileName, 		"db_config", 		"", "Db config file name")
	flag.StringVar(&redisConfigFileName, 	"redis_config", 	"", "Redis config file name")
	flag.StringVar(&pidFile, 				"pid", 				PidFileName, "Pid file name")
	flag.Parse()

	app.Info.Println("Run in ONLINE mode")

	app.Debug.Printf("PID file: %s", pidFile)

	settings := app.AppStartSettings{
		AppName: "stihi_blockchain_loader_cyberway",
		AppConfigFile: appConfigFileName,
		Config: &Config,
		PidFile: pidFile,
		LogFile: LogFileName,
		FinishFunc: finish,
		Flags: app.AppStartFlags{
			Demonize: false,
			LogToConsole: true,
		},
	}

	app.AppStart(&settings)
	defer os.Remove(pidFile)

	app.Info.Printf("Start with pid: %d", os.Getpid())

	cyber.Init(&Config.Cyberway)

	level1, err := cache_level1.New(redisConfigFileName, dbConfigFileName, "")
	if err != nil {
		app.Error.Fatalf("Error create DB: %s", err)
	}
	cache_level1.DB = level1
	DB = level1

	// Миграции
	DB.Level2.RunMigrations(false)

	mainLoop()
}

func mainLoop() {
	app.Info.Println("Main loop proc start...")

	// Создаем пул горотин, которые будут загружать блоки
	props, err := cyber.GetInfo()
	if err != nil {
		app.Error.Panicf("Can not cyberway getInfo: %s", err)
	}

	lastBlock = getSavedBlock()
	if lastBlock == 0 {
		lastBlock = props.LastIrreversibleBlockNum
	}

	wgMainLoop.Add(Config.Cyberway.ProcsCount)

	for i := 0; i < Config.Cyberway.ProcsCount; i++ {
		go LoadProcess()
	}

	wgMainLoop.Wait()
}

func LoadProcess() {
	app.Info.Printf("Start loader process...")

	saveCounter := SaveAfterBlocks

	trx, _ := DB.Level2.StartTransaction()

	for !forceStop {
		props, err := cyber.GetInfo()
		if err != nil {
			app.Error.Printf("Can not getInfo: %s", err)

			time.Sleep( SleepBlockTime )
			continue
		}

		for !forceStop && int64(props.LastIrreversibleBlockNum) - int64(lastBlock) > 0 {
			block, err := cyber.GetBlockRaw(int64(lastBlock))
			if err != nil || block == "" {
				if len(err.Error()) > 1024 {
					app.Error.Printf("Can not read block: %d error: %s", lastBlock, err.Error()[:1024])
				} else {
					app.Error.Printf("Can not read block: %d error: %s", lastBlock, err)
				}

				// Если сообщение об ошибке о невозможности распарсить блок - пропускаем его
				if strings.Contains(err.Error(), "failed to unmarshal get_block response") ||
					strings.Contains(err.Error(), "cannot unmarshal" ) ||
					strings.Contains(err.Error(), "unmarshal error" ) {
					lastBlock++
				} else {
					app.EmailErrorf("Can not read block: %d error: %s", lastBlock, err)
				}

				time.Sleep( SleepBlockTime )
				continue
			}

			err = trx.SaveBlock(int64(lastBlock), block, `{}`)
			if err != nil {
				app.Error.Printf("DB SaveBlock error: %s", err)
				if strings.Contains(err.Error(), "current transaction is aborted") {
					_ = trx.RollbackTransaction()
					trx, _ = DB.Level2.StartTransaction()
				}
			} else {
				saveCounter--
				if saveCounter <= 0 {
					err = trx.CommitTransaction()
					if err != nil {
						app.Error.Printf("Commit error: %s", err)
						_ = trx.RollbackTransaction()
					} else {
						app.Info.Printf("Save block: %d / %d", lastBlock, props.LastIrreversibleBlockNum)
						saveCounter = SaveAfterBlocks
					}
					trx, _ = DB.Level2.StartTransaction()
				}
				lastBlock++
			}
		}

		err = trx.CommitTransaction()
		if err != nil {
			app.Error.Printf("Commit error: %s", err)
			_ = trx.RollbackTransaction()
		} else {
			app.Info.Printf("Save block: %d / %d", lastBlock, props.LastIrreversibleBlockNum)
			saveCounter = SaveAfterBlocks
		}
		trx, _ = DB.Level2.StartTransaction()

		time.Sleep( SleepBlockTime )
	}

	app.Info.Println("Main loop finish")
	wgMainLoop.Done()
}

func getSavedBlock() int64 {
	num, err := DB.Level2.LastBlockNum()
	if err != nil {
		app.Error.Println(err)
		return 1
	}
	return num
}
