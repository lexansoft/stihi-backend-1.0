package app

import (
	"fmt"
	"gitlab.com/stihi/stihi-backend/app/sendmail"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/VividCortex/godaemon"
	"gopkg.in/yaml.v2"
)

var (
	appSettings		*AppStartSettings
	stopHup         chan bool
)

type AppStartFlags struct {
	Demonize		bool
	LogToConsole	bool
	DisableDupCheck		bool
}

type AppStartSettings struct {
	AppName			string
	AppConfigFile	string
	PidFile			string
	LogFile			string

	Config			interface{}
	FinishFunc		func()

	Flags			AppStartFlags

	EmailErrorTo	string
	EmailErrorFrom	string
	EmailErrorTitle	string
	EmailPanicTitle	string
}

func AppStart(settings *AppStartSettings) {
	appSettings = settings

	// Обработка команд из параметров командной строки
	CheckCommands()

	// Демонизация через запуск копии приложения в фоне
	if appSettings.Flags.Demonize {
		godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	}

	// Инициализация лока в лог-файл
	if !appSettings.Flags.LogToConsole {
		InitLogFile(appSettings.AppName, appSettings.LogFile)
	}

	// setLimits()

	// Защита приложения от двойного запсука
	// Необходимо указывать здесь что-бы в pid-файле лежал реальный pid
	if !appSettings.Flags.DisableDupCheck {
		PidProcess(appSettings.PidFile)
	}

	// Обработка сигналов, получаемых процессом
	stopHup = make(chan bool, 1)
	sigExit, sigHup, sigUsr2 := SignalsInit()
	go SignalsProcess(sigExit)
	go SignalHup(sigHup)
	go SignalNoop(sigUsr2)

	LoadFromConfig(appSettings.AppConfigFile, appSettings.Config)
}

func CheckCommands() {
	if len(os.Args) <= 1 {
		return
	}

	command := os.Args[len(os.Args)-1]

	switch command {
	case "stop":
		commandStop()
		os.Exit(0)
	case "restart":
		commandStop()
		time.Sleep(time.Second)
		fmt.Print("Init restart...\n")
	case "status":
		statusCommand()
		os.Exit(0)
	}
}

func commandStop() {
	oldPid, err := GetPidFromFile(appSettings.PidFile)
	if oldPid == os.Getpid() {
		return
	}
	if err == nil {
		p, err := os.FindProcess(oldPid)

		if err == nil {
			fmt.Printf("Process: %+v\n", p)
			err = p.Signal(syscall.SIGQUIT)
			if err != nil {
				fmt.Printf("Error when stop process %d: %s\n", oldPid, err)
				stopByFoundedPid()
				os.Remove(appSettings.PidFile)
			} else {
				fmt.Print("Process successfull stoped.\n")
			}
		}
	} else {
		fmt.Printf("Error read pid file: %s\n", err)
		stopByFoundedPid()
	}
}

func stopByFoundedPid() {
	if isDup, oldPid := FoundDupProcess(); isDup {
		fmt.Printf("Stop by founded pid: %d\n", oldPid)
		proc, _ := os.FindProcess(oldPid)
		proc.Signal(syscall.SIGQUIT)
	}
}

func statusCommand() {
	oldPid, err := GetPidFromFile(appSettings.PidFile)
	if err == nil {
		p, err := os.FindProcess(oldPid)
		err = p.Signal(syscall.SIGUSR2)
		if err != nil {
			fmt.Printf("Pid present, but process error: %s\n", err)
			os.Remove(appSettings.PidFile)
		} else {
			fmt.Printf("Process running with pid: %d\n", oldPid)
		}
	} else {
		fmt.Print("Process not found.\n")
	}
}

func SignalsInit() (chan os.Signal, chan os.Signal, chan os.Signal) {
	sigChan := make(chan os.Signal, 1)
	sigHup := make(chan os.Signal, 1)
	sigUsr2 := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	signal.Notify(sigHup,
		syscall.SIGHUP)
	signal.Notify(sigUsr2,
		syscall.SIGUSR2)
	return sigChan, sigHup, sigUsr2
}

func SignalsProcess(signals chan os.Signal) {
	<-signals

	Info.Println("Detect stop command. Please, wait...")

	os.Remove(appSettings.PidFile)

	appSettings.FinishFunc()
}

func SignalHup(signals chan os.Signal) {
	for {
		select {
		case <-signals:
			if !appSettings.Flags.LogToConsole {
				Info.Println("Detect HUP signal. Rotate log...")
				InitLogFile(appSettings.AppName, appSettings.LogFile)
				Info.Println("Rotate log done.")
			}
		case <-stopHup:
			return
		}
	}
}

func SignalNoop(signals chan os.Signal) {
	for {
		select {
		case <-signals:
		case <-stopHup:
			return
		}
	}
}

func LoadFromConfig(confFile string, config interface{}) {
	_, err := os.Stat(confFile)
	if os.IsNotExist(err) {
		Error.Fatalf("Config file '%s' not exists.", confFile)
	}

	dat, err := ioutil.ReadFile(confFile)
	if err != nil {
		Error.Fatalln(err)
	}

	err = yaml.Unmarshal(dat, config)
	if err != nil {
		Error.Fatalf("error: %v", err)
	}
}

func setLimits() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		Error.Println("Error Getting Rlimit ", err)
	}
	fmt.Println(rLimit)
	rLimit.Max = 100000
	rLimit.Cur = 100000
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		Error.Println("Error Setting Rlimit ", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		Error.Println("Error Getting Rlimit ", err)
	}
	Info.Println("Rlimit Final", rLimit)
}

func PanicHandle() {
	if x := recover(); x != nil {
		body := fmt.Sprintf("RUNTIME PANIC: %v\n%s\n", x, debug.Stack())

		if appSettings.EmailErrorTo != "" {
			sendmail.SendMail(appSettings.EmailErrorFrom, appSettings.EmailErrorTo, appSettings.EmailPanicTitle, body)
		}

		Error.Println(body)

		// Не останавливаем выполнение сервиса при панике
		// panic(x)
	}
}


type TimeProfile struct {
	firstTime		time.Time
	lastTime		time.Time
	blockTime		time.Duration
}

func TimeProfileCreate() *TimeProfile {
	tp := TimeProfile{
		firstTime: time.Now(),
		lastTime: time.Now(),
		blockTime: 0,
	}
	return &tp
}

func (tp *TimeProfile) StartBlock() {
	tp.lastTime = time.Now()
}

func (tp *TimeProfile) StopBlock() {
	tp.blockTime += time.Now().Sub(tp.lastTime)
}

func (tp *TimeProfile) Check() string {
	global := tp.GlobalDuration()
	local := tp.LocalDuration()
	tp.lastTime = time.Now()
	return fmt.Sprintf("TimeProfile: global = %s, local = %s", global.String(), local.String())
}

func (tp *TimeProfile) GlobalDuration() time.Duration {
	return time.Now().Sub(tp.firstTime)
}

func (tp *TimeProfile) LocalDuration() time.Duration {
	return time.Now().Sub(tp.lastTime)
}
