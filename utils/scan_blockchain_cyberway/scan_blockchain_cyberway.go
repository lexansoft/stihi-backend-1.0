package main

import (
	"database/sql"
	"flag"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/config"
	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
	"gitlab.com/stihi/stihi-backend/utils/scan_blockchain_cyberway/sb_cron"
)

/*
	TODO: Переделать сканирование блокчейна на сканирование cyber_events - операций из event-engine
	TODO: Отдельные счетчики И ПОТОКИ для каждого типа детектируемых операций

	При сбросе счетчика по одному из типов операций сканирование в его потоке начинается с начала данных.
	Остальные потоки обработки по другим операциям запускают сканирование со своих точек остановки.
*/

const (
	PidFileName = "/var/tmp/stihi_scan_blockchain_cyberway.pid"
	LogFileName = "/var/log/stihi/scan_blockchain_cyberway.log"

	SavedBlockStateId = "last_scan_block_cw"
	SavedFirstBlockStateId = "first_scan_block_cw"
	SavedBatchBlockStateId = "batch_scan_block_cw"
	SaveAfterBlocks = 10000
	SleepBlockTime	= 3 * time.Second
	BatchDBSize = 100
)

var (
	build				string
	batchMode			bool
	batchModeRestart	bool

	appConfigFileName   string
	dbConfigFileName	string
	mongoConfigFileName	string
	redisConfigFileName	string

	forceStop			bool
	Config				config.SBConfig

	tp		 			*app.TimeProfile
	timeByOperations 	map[string]int64

	DB					*cache_level1.CacheLevel1

	lastBlock			int64
	maxBlock			int64

	tpSaveVoteMutex		*sync.Mutex = &sync.Mutex{}
)

// Демон, который сканирует блокчейн из БД и собирает необходимые данные:
// 1. Пользователей (по регистрациям);
// 2. Статьи с определенными тэгами;
// 3. Комментарии к загруженным статьям;
// Остальные данные беруться бэкэндом прямо из mongodb базы ноды

func finish() {
	app.Info.Printf("Stop daemon with pid: %d", os.Getpid())
	forceStop = true
	os.Exit(0)
}

func main() {
	defer app.PanicHandle()

	if build != "" {
		app.Info.Printf("Build: %s", build)
	}

	var pidFile string
	flag.StringVar(&appConfigFileName, 		"config", 			"", "Application config file name")
	flag.StringVar(&dbConfigFileName, 		"db_config", 			"", "Db config file name")
	flag.StringVar(&mongoConfigFileName, 	"mongo_db_config", 	"", "MongoDb config file name")
	flag.StringVar(&redisConfigFileName, 	"redis_config", 		"", "Redis config file name")
	flag.StringVar(&pidFile, 				"pid", 				PidFileName, "Pid file name")
	flag.BoolVar(&batchMode,				"batch", 				false, "Run in batch mode")
	flag.BoolVar(&batchModeRestart,			"batch_restart",		false, "Run batch mode with restart scan")
	flag.Parse()

	if batchMode {
		app.Info.Println("Run in BATCH mode")
	} else {
		app.Info.Println("Run in ONLINE mode")
	}

	app.Debug.Printf("PID file: %s", pidFile)

	settings := app.AppStartSettings{
		AppName: "stihi_scan_blockchain_cyberway",
		AppConfigFile: appConfigFileName,
		Config: &Config,
		PidFile: pidFile,
		LogFile: LogFileName,
		FinishFunc: finish,
		Flags: app.AppStartFlags{
			Demonize: false,
			LogToConsole: true,
			DisableDupCheck: true,
		},
	}

	app.AppStart(&settings)
	defer os.Remove(pidFile)

	app.Info.Printf("Start daemon with pid: %d", os.Getpid())

	level1, err := cache_level1.New(redisConfigFileName, dbConfigFileName, mongoConfigFileName)
	if err != nil {
		app.Error.Fatalf("Error create DB: %s", err)
	}
	cache_level1.DB = level1
	DB = level1
	DB.Level2.MongoDBCheck()

	// Миграции
	DB.Level2.RunMigrations(false)

	// Отключаем Debug log если рабочий режим
	if Config.RPC.BlockchanName != "test" {
		app.DisableLogLevel(app.Debug)
	}

	// Обновляем у всех рубрик tag_name
	app.Info.Println("Process rubrics...")
	rows, err := DB.Level2.Query("SELECT id, name, raw_tag FROM rubrics")
	if err == nil {
		for rows.Next() {
			var id int64
			var name string
			var tagName sql.NullString
			err = rows.Scan(
				&id,
				&name,
				&tagName,
			)
			if err != nil {
				app.Error.Println(err)
			}

			if name != "" {
				tag := translit.EncodeTag(name)
				_, err = DB.Level2.Do("UPDATE rubrics SET raw_tag = $1 WHERE id = $2", tag, id)
				if err != nil {
					app.Error.Println(err)
				}
			}
		}
		rows.Close()
	} else {
		app.Error.Println(err)
	}
	app.Info.Println("done.")

	FillNodeosIdForContent()

	// RecodeForCyberName()

	sb_cron.Init(DB.Level2)

	mainLoop()

	sb_cron.Stop()
}

func mainLoop() {
	app.Info.Println("Main loop proc start...")

	app.Info.Printf("Block interval: %d\n", SleepBlockTime)

	LastIrreversibleBlockNum, _ := DB.Level2.LastBlockNum()

	if batchMode {
		lastBlock = getBatchSavedBlock()
		if Config.StartFromFirstBlock || batchModeRestart {
			lastBlock = 1
		}
		maxBlock = getFirstSavedBlock()
	} else {
		lastBlock = getSavedBlock()
		if lastBlock == 0 {
			lastBlock = LastIrreversibleBlockNum
		}

		// TODO: Убрать после отладки
		if Config.StartFromFirstBlock {
			lastBlock = 1
		}

		saveFirstBlock(lastBlock)
	}

	app.Info.Println("Start main loop...")

	saveCounter := SaveAfterBlocks
	tp = app.TimeProfileCreate()
	timeByOperations = make(map[string]int64)
	startedBlock := lastBlock

	for !forceStop {
		var err error

		LastIrreversibleBlockNum, err = DB.Level2.LastBlockNum()
		if err != nil {
			app.Error.Printf("Can not get LastBlockNum: %s", err)

			time.Sleep( SleepBlockTime )
			continue
		}

		// app.Debug.Printf("Block: %d / %d\n", lastBlock, LastIrreversibleBlockNum)

		for !forceStop && int64(LastIrreversibleBlockNum) - int64(lastBlock) > 0 {
			// app.Info.Println("Main loop level 2 start...")

			bTime := time.Now()
			blockList, err := DB.Level2.GetBlocks(lastBlock, BatchDBSize)
			addTime("GetBlocks", time.Now().Sub(bTime))
			if err != nil || blockList == nil {
				if len(err.Error()) > 1024 {
					app.Error.Printf("Can not read block: %d error: %s", lastBlock, err.Error()[:1024])
				} else {
					app.Error.Printf("Can not read block: %d error: %s", lastBlock, err)
				}

				time.Sleep(time.Duration(SleepBlockTime) * time.Second)
				continue
			}

			// app.Debug.Printf("Process block: %d", lastBlock)
			for _, block := range blockList {
				if processBlock(block, lastBlock) {
					saveCounter--
					if saveCounter <= 0 {
						speed := float64(SaveAfterBlocks) / tp.LocalDuration().Seconds()
						speedGlobal := float64(lastBlock-startedBlock) / tp.GlobalDuration().Seconds()
						app.Info.Printf("Current speed: %.2f / %.2f", speed, speedGlobal)
						app.Info.Printf("Times: %+v", timeByOperations)

						if batchMode {
							saveBatchBlock(lastBlock, int64(maxBlock))
						} else {
							saveBlock(lastBlock, int64(LastIrreversibleBlockNum))
						}

						saveCounter = SaveAfterBlocks

						// app.Debug.Printf("LastIrreversibleBlockNum = %d", int64(LastIrreversibleBlockNum))

						tp.Check()
					}
					lastBlock++
				}
			}

			// Если batch-режим, сканирование только до максимального блока
			if batchMode && lastBlock >= maxBlock {
				forceStop = true
			}
		}

		// Если используется ожидание - сохраняем после обработки каждого блока
		saveCounter = 0
		time.Sleep( SleepBlockTime )
	}

	app.Info.Println("Main loop finish")
}

// Возвращает true только в случае удачной обработки блока
func processBlock(pBlock *map[string]interface{}, blockNum int64) bool {
	block := *pBlock

	transactions, ok := block["transactions"].([]interface{})
	if !ok {
		return false
	}

	blockTimeStr, _ := block["timestamp"].(string)
	blockTime, err := time.Parse("2006-01-02T15:04:05.999", blockTimeStr)
	if err != nil {
		app.Error.Printf("Error parse block timestamp: %s", err)
		blockTime = time.Now()
	}

	// Process the transactions.
	for _, tx := range transactions {
		// app.Info.Printf("Process transaction: %d - %d", tx.RefBlockNum, tx.RefBlockPrefix)
		var ops []interface{};

		tr, ok := tx.(map[string]interface{})
		if !ok {
			continue
		}
		if trx, ok := tr["trx"].(map[string]interface{}); ok {
			if trans, ok := trx["transaction"].(map[string]interface{}); ok {
				ops, ok = trans["actions"].([]interface{})
				if !ok {
					continue
				}
			}
		}

		for _, op := range ops {
			// app.Info.Printf("Process operation: %d", operation.Type().Code())

			operation, ok := op.(map[string]interface{})
			if !ok {
				continue
			}

			switch operation["name"] {
			case operations.OpCreateMessageName:
				// app.Info.Printf("Process operation Message...")

				bTime := time.Now()
				comment := (&operations.CreateMessageOp{}).InitFromMap(operation)

				if comment.Data != nil && ( comment.Data.ParentId == nil || comment.Data.ParentId.Author == "" ) {
					_, err := DB.SaveArticleFromOperation(comment.Data, blockTime)
					if err != nil {
						if !strings.HasPrefix(err.Error(), "skip ") {
							app.Error.Printf("ERROR: Add article error: %s, block: %d", err, blockNum)
						}
					} else {
						// app.Debug.Printf("Article success added\n")
					}
				} else {
					_, err := DB.SaveCommentFromOperation(comment.Data, blockTime)
					if err != nil {
						if !strings.HasPrefix(err.Error(), "skip ") {
							app.Error.Printf("ERROR: Add comment error: %s, block: %d", err, blockNum)
						}
					} else {
						// app.Debug.Printf("Comment success added\n")
					}
				}
				addTime("CommentOperation", time.Now().Sub(bTime))

				// app.Info.Printf("Process operation Message done")
			case operations.OpUpdateMessageName:
				// app.Info.Printf("Process operation Message...")

				bTime := time.Now()
				content := (&operations.UpdateMessageOp{}).InitFromMap(operation)

				err = DB.UpdateContent(content)
				if err != nil {
					app.Error.Printf("ERROR: Add update content: %s", err)
				}

				addTime("CommentOperation", time.Now().Sub(bTime))

				// app.Info.Printf("Process operation Message done")
			case operations.OpNewUserNameName:
				// app.Info.Printf("Process operation NewUserName...")
				bTime := time.Now()
				name := (&operations.NewUserNameOp{}).InitFromMap(operation)

				err := DB.SaveNewUserNameFromOperation(name, blockTime)
				if err == nil {
					// app.Debug.Printf("User name success added\n")
				} else {
					if !strings.HasPrefix(err.Error(), "skip ") {
						app.Error.Printf("ERROR: Add user name error: %s, block: %d", err, blockNum)
					}
				}

				addTime("AccountNewUserName", time.Now().Sub(bTime))

				// app.Info.Printf("Process operation NewUserName done")
			case operations.OpNewAccountName:
				// app.Info.Printf("Process operation AccountCreate...")
				bTime := time.Now()
				account := (&operations.NewAccountOp{}).InitFromMap(operation)

				err := DB.SaveUserFromOperation(account, blockTime)
				if err == nil {
					/*
					err := DB.AddToHistoryAccountCreateOperation(account, tx, block)
					if err != nil {
						app.Error.Println(err)
					}
					*/

					// app.Debug.Printf("User success added\n")
				} else {
					if !strings.HasPrefix(err.Error(), "skip ") {
						app.Error.Printf("ERROR: Add user error: %s, block: %d", err, blockNum)
					}
				}
				addTime("AccountCreateOperation", time.Now().Sub(bTime))

				// app.Info.Printf("Process operation AccountCreate done")
			case operations.OpUpdateAuthName:
				// app.Info.Printf("Process operation UpdateAuth done")

				bTime := time.Now()
				auth := (&operations.UpdateAuthOp{}).InitFromMap(operation)

				err := DB.UpdateUserAuthFromOperation(auth, blockTime)
				if err == nil {
					/*
						err := DB.AddToHistoryAccountCreateOperation(account, tx, block)
						if err != nil {
							app.Error.Println(err)
						}
					*/

					// app.Debug.Printf("User new key success added\n")
				} else {
					if !strings.HasPrefix(err.Error(), "skip ") {
						app.Error.Printf("ERROR: Add user error: %s, block: %d", err, blockNum)
					}
				}
				addTime("AccountUpdateOperation", time.Now().Sub(bTime))

				// app.Info.Printf("Process operation UpdateAuth done")
			}
		}
	}

	return true
}

func getSavedBlock() (int64) {
	blkStr, err := DB.GetState(SavedBlockStateId)
	if err != nil || blkStr == "" {
		return 1
	}

	s, err := strconv.ParseInt(blkStr, 10, 64)
	if err != nil {
		return 1
	}
	return s
}

func getFirstSavedBlock() (int64) {
	blkStr, err := DB.GetState(SavedFirstBlockStateId)
	if err != nil || blkStr == "" {
		return 1
	}

	s, err := strconv.ParseInt(blkStr, 10, 64)
	if err != nil {
		return 1
	}
	return s
}

func getBatchSavedBlock() (int64) {
	blkStr, err := DB.GetState(SavedBatchBlockStateId)
	if err != nil || blkStr == "" {
		return 1
	}

	s, err := strconv.ParseInt(blkStr, 10, 64)
	if err != nil {
		return 1
	}
	return s
}

func saveBlock(blockNum int64, lastBlockNum int64) {
	app.Info.Printf("Save block num: %d / %d", blockNum, lastBlockNum)
	_ = DB.SaveState(SavedBlockStateId, strconv.FormatInt(blockNum, 10))
}

func saveFirstBlock(blockNum int64) {
	app.Info.Printf("Save first block num: %d", blockNum)
	_ = DB.SaveState(SavedFirstBlockStateId, strconv.FormatInt(blockNum, 10))
}

func saveBatchBlock(blockNum int64, lastBlockNum int64) {
	app.Info.Printf("Save batch block num: %d / %d", blockNum, lastBlockNum)
	_ = DB.SaveState(SavedBatchBlockStateId, strconv.FormatInt(blockNum, 10))
}

func addTime(key string, delta time.Duration) {
	cur, ok := timeByOperations[key]
	if ok {
		timeByOperations[key] = cur + delta.Nanoseconds()
	} else {
		timeByOperations[key] = delta.Nanoseconds()
	}
}

// Заполняем поле nodeos_id для контента, у которого оно не заполнено
func FillNodeosIdForContent() {
	app.Info.Println("Start fill NodeosId for content...")
	count := 1
	lastId := int64(0)
	for count > 0 {
		count = 0
		rows, err := DB.Level2.Query(`
			SELECT id, permlink 
			FROM content
			WHERE id > $1 AND (nodeos_id IS NULL OR nodeos_id <= 0)
			ORDER BY id
			LIMIT 1000
		`, lastId)
		if err != nil {
			app.Error.Printf("Error select for content with nodeos_id is null: %s", err)
			return
		}

		res := make(map[int64]int64)
		for rows.Next() {
			count++

			var id int64
			var permlink string

			err = rows.Scan(
				&id,
				&permlink,
			)
			if err != nil {
				rows.Close()
				app.Error.Printf("Error scan for content with nodeos_id is null: %s", err)
				return
			}

			lastId = id

			nodeosId, err := DB.Level2.GetContentNodeosIdByPermlink(permlink)
			if err != nil {
				rows.Close()
				app.Error.Printf("Error get nodeos_id from mongo: %s", err)
				continue
			}

			res[id] = nodeosId
		}
		rows.Close()

		if count > 0 {
			for id, nodeosId := range res {
				if nodeosId > 0 {
					_, err = DB.Level2.Do(`UPDATE content SET nodeos_id = $1 WHERE id = $2`, nodeosId, id)
					if err != nil {
						app.Error.Printf("Error when update nodeos_id: %s", err)
						return
					}
				}
			}
		}
	}
	app.Info.Println("Fill NodeosId for content done")
}

// Перекодирование таблицы пользователе на использование имен cyber в качестве основных
// В цикле по юзерам stihi.io:
// 1. Находим cyberName для юзера;
// 2. Запоминаем его gls имя;
// 3. Заменяем его имя в users и делаем синхронизацию для заполнения users_names и users_keys;
// 4. Заменяем в content все имена авторов и имена parent-авторов на cyber имя юзера
// Тот-же цикл по остальным юзерам
func RecodeForCyberName() {
	app.Debug.Println("Reorganize users names...")

	// Собираем юзеров stihi
	users := make(map[string]int64)
	rows, err := DB.Level2.Query(`SELECT id, name FROM users WHERE stihi_user`)
	if err != nil {
		app.Error.Fatalf("Get stihi user list error: %s\n", err)
	}
	for rows.Next() {
		var id int64
		var name string
		err := rows.Scan(
			&id,
			&name,
		)
		if err != nil {
			app.Error.Fatalf("Scan stihi user list error: %s\n", err)
		}

		users[name] = id
	}
	rows.Close()

	// Обрабатываем юзеров stihi
	for name, id := range users {
		cyberName := DB.GetNodeosName(name)

		if cyberName != name {
			app.Info.Printf("REORGANIZE: %d - %s -> %s", id, name, cyberName)
			ReorganazeForUser(id, name, cyberName)
		}
	}

	// Собираем остальных юзеров
	users = make(map[string]int64)
	rows, err = DB.Level2.Query(`SELECT id, name FROM users WHERE NOT stihi_user`)
	if err != nil {
		app.Error.Fatalf("Get stihi user list error: %s\n", err)
	}
	for rows.Next() {
		var id int64
		var name string
		err := rows.Scan(
			&id,
			&name,
		)
		if err != nil {
			app.Error.Fatalf("Scan NO stihi user list error: %s\n", err)
		}

		users[name] = id
	}
	rows.Close()

	// Обрабатываем юзеров
	for name, id := range users {
		cyberName := DB.GetNodeosName(name)

		if cyberName != name {
			app.Info.Printf("REORGANIZE: %d - %s -> %s", id, name, cyberName)
			ReorganazeForUser(id, name, cyberName)
		}
	}
	app.Debug.Println("Reorganize users names done")
}

func ReorganazeForUser(id int64, name, cyberName string) {
	// Заменяем имя в users и делаем синхронизацию юзера
	_, err := DB.Level2.Do(
		`UPDATE users SET name = $1 WHERE id = $2`,
		cyberName, id,
	)
	if err != nil {
		app.Error.Printf("Error update username to cyber: %s\n", err)
		return
	}

	// Добавляем имя в users_names
	_, err = DB.Level2.Do(
		`INSERT INTO users_names (user_id, creator, name) VALUES ($1, $2, $3)`,
		id, "gls", name,
	)
	if err != nil {
		app.Error.Printf("Error insert username: %s\n", err)
		return
	}

	// Заменяем имя автора в контенте на cyberName
	_, err = DB.Level2.Do(
		`UPDATE content SET author = $1 WHERE author = $2`,
		cyberName, name,
	)
	if err != nil {
		app.Error.Printf("Error update username to cyber: %s\n", err)
		return
	}

	// Заменяем имя parent автора на cyberName
	_, err = DB.Level2.Do(
		`UPDATE content SET parent_author = $1 WHERE parent_author = $2`,
		cyberName, name,
	)
	if err != nil {
		app.Error.Printf("Error update username to cyber: %s\n", err)
		return
	}

	// Заменяем имя в follows
	_, err = DB.Level2.Do(
		`UPDATE follows SET subscribed_for = $1 WHERE subscribed_for = $2`,
		cyberName, name,
	)
	if err != nil {
		app.Error.Printf("Error update username to cyber: %s\n", err)
		return
	}
}