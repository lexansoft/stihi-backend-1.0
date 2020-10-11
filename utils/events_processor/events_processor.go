package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/db"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"hash"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

/*
Read data from EventEngine cyberway node socket and save it for processing.
First line saving - simple log file.
Second line saving - postgresql database.

First line worked sync with read socket process.
Second line worked like N gorotines, parsed and saved data to database.
*/

const (
	PidFileName = "/var/tmp/stihi_events_processor_cyberway.pid"
	LogFileName = "/var/log/stihi/events_processor_cyberway.log"
)

var (
	build string

	connect     net.Conn
	dbTrxBatchSize = 100

	config EventsProcessorConfig
	appConfigFileName string
	dbConfigFileName string
	pidFile string

	forceStopRead bool
	forceStopWrite bool
	wgReadLoop    sync.WaitGroup = sync.WaitGroup{}
	wgPsqlLoop    sync.WaitGroup = sync.WaitGroup{}
)

type EventsProcessorConfig struct {
	Connection  string `yaml:"connection"`
	DataLogFile string `yaml:"data_log_file"`
	DBWorkers   int    `yaml:"db_workers_count"`
}

type DataLine struct {
	Line 		string
	Offset		int64
	LogFileTime	int64
	CheckSum	string
}

func finish() {
	app.Info.Printf("Stop daemon with pid: %d", os.Getpid())
	forceStopRead = true
	app.Info.Printf("Wait finish read loop...")
	wgReadLoop.Wait()
	app.Info.Printf("Read loop finished")
	forceStopWrite = true
	app.Info.Printf("Wait finish save loop...")
	wgPsqlLoop.Wait()
	app.Info.Printf("Save loop finished")
	app.Info.Printf("Finish")
	os.Exit(0)
}

func main() {
	defer app.PanicHandle()

	flag.StringVar(&appConfigFileName, 	"config", "", "Path for config file")
	flag.StringVar(&dbConfigFileName, 	"db_config", 			"", "Db config file name")
	flag.StringVar(&pidFile, 			"pid", 	PidFileName, "Pid file name")
	flag.Parse()

	if appConfigFileName == "" || dbConfigFileName == "" {
		flag.Usage()
		os.Exit(0)
	}

	settings := app.AppStartSettings{
		AppName: "stihi_events_processor_cyberway",
		AppConfigFile: appConfigFileName,
		Config: &config,
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

	app.Info.Printf("Start with pid: %d (build: %d)", os.Getpid(), build)

	db.InitFromFile(dbConfigFileName)

	// Migrations
	psql := db.New()
	if psql == nil {
		return
	}
	cacheLvl2 := cache_level2.CacheLevel2{
		QueryProcessor: psql,
	}
	cacheLvl2.RunMigrations(false)
	psql.Close()

	/*
	var err error
	var connType string
	if strings.HasPrefix(config.Connection, "unix") || strings.HasPrefix(config.Connection, "./") {
		// Unix socket
		connType = "unix"
	} else {
		// Tcp socket
		connType = "tcp"
	}
	connect, err = net.Dial(connType, config.Connection)
	if err != nil {
		app.Error.Panicf("Can not connect to %s socket %s: %s\n", connType, config.Connection, err)
	}
	*/
	reconnect()
	defer connect.Close()

	psqlDataChan := saveDataToDBLoop()

	wgReadLoop.Add(1)
	go func() {
		readSocketLoop(psqlDataChan)
		wgReadLoop.Done()
	}()

	app.Info.Printf("App running...")
	wgReadLoop.Wait()
	wgPsqlLoop.Wait()
	app.Info.Printf("App finished")
}

func readSocketLoop(psqlChan chan DataLine) {
	// dataLogFile, err := os.OpenFile(config.DataLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0644)
	dataLogFile, err := os.OpenFile(config.DataLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		app.Error.Panicf("Can not open data log file '%s': %s", config.DataLogFile, err)
	}
	defer dataLogFile.Close()

	var dataLine DataLine
	hashEngine := sha256.New()
	sockReader := bufio.NewReader(connect)
	for !forceStopRead {
		// Read line from socket
		dataLine.Line, err = sockReader.ReadString('\n')
		if err != nil {
			app.Error.Printf("Error read line from socket: %s", err)
			time.Sleep(time.Second)
			// reconnect()
			continue
		}

		// Write line to data log file
		logStat, _ := dataLogFile.Stat()
		stat := logStat.Sys().(*syscall.Stat_t)
		dataLine.LogFileTime = stat.Ctim.Nano()

		// Check rotate
		if logStat.Size() > 1 << (10 * 3) {
			// If file great then 1 Gb - rotate
			dataLogFile.Close()
			_ = os.Rename(config.DataLogFile, config.DataLogFile+"_"+strconv.FormatInt(time.Now().UnixNano(), 10))
			dataLogFile, err = os.OpenFile(config.DataLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				app.Error.Panicf("Can not open data log file '%s': %s", config.DataLogFile, err)
			}
			dataLine.LogFileTime = time.Now().UnixNano()
		}

		setDataCheckSum(&dataLine, &hashEngine)
		dataLine.Offset, _ = dataLogFile.Seek(0, io.SeekCurrent)

		n := len(dataLine.Line)
		nn, err := writeToDataLog(dataLogFile, &dataLine)
		if err != nil {
			app.Error.Printf("Error write line to data log file: %s", err)
		}
		if nn < n {
			app.Error.Printf("Error write size for line to data log file: %d bytes expected, %d bytes writed", n, nn - 1)
		}

		// Write line to database (parallel)
		if !forceStopRead {
			psqlChan <- dataLine
		}
	}

	app.Debug.Printf("Finish read data loop")
}

func saveDataToDBLoop() chan DataLine {
	dataChan := make(chan DataLine)
	for i := 0; i < config.DBWorkers; i++ {
		wgPsqlLoop.Add(1)
		go func(numProc int) {
			psqlDataProcess(dataChan, numProc)
			app.Debug.Printf("[%d] Finish psql loop", numProc)
			wgPsqlLoop.Done()
		}(i+1)
	}
	return dataChan
}

func psqlDataProcess(dataChan chan DataLine, numProcess int) {
	app.Info.Printf("[%d] Start psql thread...", numProcess)

	psql := db.New()
	if psql == nil {
		return
	}

	trx, err := psql.StartTransaction()
	if err != nil {
		app.Error.Printf("[%d] Can not start transaction: %s", numProcess, err)
		return
	}

	trxCounter := dbTrxBatchSize
	needCommit := false
	realCount := 0
	for !forceStopWrite {
		select {
		case data := <-dataChan:
			trxCounter--
			n, err := writeDataToPsql(trx, &data, numProcess)
			if err != nil {
				app.Error.Printf("[%d] Error write data to psql: %s\n[%s]\nstrlen = %d", numProcess, err, data.Line, len(data.Line))

				// Restart transaction if error
				needCommit = true
				trxCounter = 0

				// Repeat save data
				dataChan <- data
			} else {
				needCommit = true
				realCount += n
			}
		default:
			// Commit if no data in channel
			trxCounter = 0
			time.Sleep(200 * time.Millisecond)
		}

		if needCommit {
			if ( trxCounter <= 0 && realCount > 0 ) || trxCounter < -dbTrxBatchSize*5 {
				app.Info.Printf("[%d] Commit transaction real count: %d", numProcess, realCount)
				err = trx.CommitTransaction()
				if err != nil {
					app.Error.Printf("[%d] Can not commit transaction: %s", numProcess, err)
					_ = trx.RollbackTransaction()
				}

				realCount = 0
				trxCounter = dbTrxBatchSize
				needCommit = false
				trx, err = psql.StartTransaction()
				if err != nil {
					app.Error.Printf("[%d] Can not start transaction: %s", numProcess, err)
					return
				}
			}
		} else {
			trxCounter = dbTrxBatchSize
		}
	}

	if realCount > 0 {
		app.Info.Printf("[%d] Commit transaction real count: %d", numProcess, realCount)
		trx.CommitTransaction()
	}
}

// writeToDataLog save data to data log file
func writeToDataLog(file *os.File, data *DataLine) (int, error) {
	return file.WriteString(data.Line+"\n")
}

// writeDataToPsql parse and save data to Postgresql database
func writeDataToPsql(psql *db.Transaction, data *DataLine, numProcess int) (int, error) {
	// 1. Разбираем данные
	// 2. Фильтруем данные
	// 3. Сохраняем actions
	jsonData := map[string]interface{}{}
	err := json.Unmarshal([]byte(data.Line), &jsonData)
	if err != nil {
		app.Error.Printf("Error unmarhsul JSON data: %s", err)
		return 0, err
	}

	// Фильтруем по msg_type
	msgTypeI, ok := jsonData["msg_type"]
	if !ok {
		return 0, nil
	}
	msgType, ok := msgTypeI.(string)
	if !ok {
		return 0, nil
	}
	if msgType != "ApplyTrx" {
		return 0, nil
	}

	// Проверяем что нет "except" - ошибки
	_, ok = jsonData["except"]
	if ok {
		return 0, nil
	}

	// Цикл по actions
	actionsI, ok := jsonData["actions"]
	if !ok {
		return 0, nil
	}
	actions, ok := actionsI.([]interface{})
	if !ok {
		return 0, nil
	}

	actIdx := 1
	for _, actI := range actions {
		act := actI.(map[string]interface{})
		actName := act["action"].(string)

		// Ignore actions without usable information
		if actName == "onblock" || actName == "pick" || actName == "use" || actName == "providebw" {
			continue
		}

		actJson, err := json.Marshal(act)
		if err != nil {
			app.Error.Printf("Json marshal error: %s", err)
			actIdx++
			continue
		}

		id, err := psql.Insert(`
			INSERT INTO cyberway_actions
				(trx_id, block_num, block_time, action_idx, action)
			VALUES
				($1, $2, $3, $4, $5)
			ON CONFLICT 
				DO NOTHING`,
			jsonData["id"], jsonData["block_num"], jsonData["block_time"], actIdx, actJson,
		)
		if err != nil {
			app.Error.Printf("Error insert blockchain event to db: %s", err)
			return actIdx-1, err
		}

		app.Info.Printf("[%d] Add action %d - %s (%d)", numProcess, id, act["action"].(string), actIdx)

		actIdx++
	}

	return actIdx-1, nil
}

func setDataCheckSum(data *DataLine, hashEngine *hash.Hash) {
	var hash hash.Hash
	if hashEngine == nil {
		hash = sha256.New()
	} else {
		hash = *hashEngine
	}


	hash.Reset()
	hash.Write([]byte(data.Line))
	data.CheckSum = base64.StdEncoding.EncodeToString( hash.Sum( nil ) )
}

func reconnect() {
	if connect != nil {
		connect.Close()
	}

	var err error
	var connType string
	if strings.HasPrefix(config.Connection, "unix") || strings.HasPrefix(config.Connection, "./") {
		// Unix socket
		connType = "unix"
	} else {
		// Tcp socket
		connType = "tcp"
	}

	connect, err = net.Dial(connType, config.Connection)
	if err != nil {
		app.Error.Panicf("Can not connect to %s socket %s: %s\n", connType, config.Connection, err)
	}
}
