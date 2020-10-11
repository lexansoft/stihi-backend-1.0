// +build !develop

package app

import (
	"fmt"
	"gitlab.com/stihi/stihi-backend/app/sendmail"
	"io/ioutil"
	. "log"
	"os"
	"runtime/debug"
	"strconv"
)

var (
	format = Ldate | Ltime | Llongfile

	Info  = New(os.Stdout, "INFO: ", format)
	Debug = New(os.Stdout, "DEBUG: ", format)
	Error = New(os.Stderr, "ERROR: ", format)

	fileLog *os.File
)

func InitLogFile(appName string, logFile string) {
	oldFile := fileLog

	fLog, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		Error.Fatalf("Can not open log file: %s", err)
	}

	fileLog = fLog

	pid := strconv.FormatInt(int64(os.Getpid()), 10)
	Info = New(fLog, pid+": INFO ("+appName+"): ", format)
	Debug = New(fLog, pid+": DEBUG ("+appName+"): ", format)
	Error = New(fLog, pid+": ERROR ("+appName+"): ", format)
	os.Stdout = fLog
	os.Stderr = fLog

	if oldFile != nil {
		oldFile.Close()
	}
}

func DisableLogLevel(logger *Logger) {
	logger.SetOutput(ioutil.Discard)
}

func EmailErrorf(format string, params... interface{}) {
	str := fmt.Sprintf(format, params...)

	Error.Println(str)

	if appSettings.EmailErrorTo == "" {
		Error.Println("EmailErrorTo is empty - can not send error to email")
	}

	str += fmt.Sprintf("\n\nSTACK:\n\n%s\n", debug.Stack())

	sendmail.SendMail(appSettings.EmailErrorFrom, appSettings.EmailErrorTo, appSettings.EmailErrorTitle, str)
}
