package main

import (
	"flag"
	"gitlab.com/stihi/stihi-backend/actions"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/config"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/app/random"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"net/http"
	// Использовать эту строку для включения вывода отладочной информации по пути /debug/pprof/goroutine?debug=1
	// _ "net/http/pprof"
	"os"
	"strconv"
)

const (
	PidFileName = "/var/tmp/stihi_backend_cyberway.pid"
	LogFileName = "/var/log/stihi/backend_cyberway.log"
)

var (
	build				string

	appConfigFileName   string
	dbConfigFileName	string
	mongoConfigFileName	string
	redisConfigFileName	string

	forceStop			bool
	Config				config.BackendConfig
)

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
	flag.StringVar(&mongoConfigFileName, 	"mongo_db_config",	"", "MongoDb config file name")
	flag.StringVar(&redisConfigFileName, 	"redis_config", 		"", "Redis config file name")
	flag.StringVar(&pidFile, 				"pid", 				PidFileName, "Pid file name")
	flag.Parse()

	settings := app.AppStartSettings{
		AppName: "stihi_backend_cyberway",
		AppConfigFile: appConfigFileName,
		Config: &Config,
		PidFile: pidFile,
		LogFile: LogFileName,
		FinishFunc: finish,
		Flags: app.AppStartFlags{
			Demonize: false,
			LogToConsole: true,
		},
		EmailErrorTo: "andy@andyhost.ru",
		EmailErrorFrom: "no-reply@stihi.io",
		EmailErrorTitle: "ERROR: STIHI2",
		EmailPanicTitle: "PANIC: STIHI2",
	}

	app.AppStart(&settings)
	defer os.Remove(pidFile)

	app.Info.Printf("Start daemon with pid: %d", os.Getpid())

	level1, err := cache_level1.New(redisConfigFileName, dbConfigFileName, mongoConfigFileName)
	if err != nil {
		app.Error.Fatalf("Error create DB: %s", err)
	}
	actions.DB = level1
	cache_level1.DB = level1

	// Migrations
	level1.Level2.RunMigrations(false)

	err = jwt.Init(&Config.JWT)
	if err != nil {
		app.Error.Fatalf("Error init JWT: %s", err)
	}

	for _, langFile := range Config.L10NErrors {
		errors_l10n.LoadL10NBase(langFile.Lang, langFile.FileName)
	}

	actions.InitRoutes(&Config)

	err = http.ListenAndServe(Config.Listen.Address.String()+":"+strconv.FormatInt(int64(Config.Listen.Port), 10), http.HandlerFunc(logRequest))
	if err != nil {
		app.Info.Printf("Host: %s, Port: %s", Config.Listen.Address.String(), strconv.FormatInt(int64(Config.Listen.Port), 10))
		app.Error.Fatalf("Error run listen: %s", err)
	}
}

func logRequest(w http.ResponseWriter, r *http.Request) {
	defer app.PanicHandle()

	rid := random.String(8)
	tprofile := app.TimeProfileCreate()
	app.Debug.Printf("START: [%s] %s %s %s\n", rid, r.RemoteAddr, r.Method, r.URL)
	http.DefaultServeMux.ServeHTTP(w, r)
	app.Debug.Printf(" DONE: [%s] %s %s %s (%s)\n", rid, r.RemoteAddr, r.Method, r.URL, tprofile.GlobalDuration().String())
}
