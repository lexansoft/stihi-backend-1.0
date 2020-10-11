package app

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/facebookgo/pidfile"
)

func FoundDupProcess() (bool, int) {
	// Определяем путь к своему исполняемому файлу
	selfCmdLine := strings.SplitN(GetCmdLine(os.Getpid()), " ", 2)

	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		Error.Printf("Cannot scan /proc: %s", err)
	}

	validPid := regexp.MustCompile(`^[0-9]+$`)
	for _, f := range files {
		if f.Mode().IsDir() {
			if validPid.MatchString(f.Name()) {
				appPid, _ := strconv.ParseInt(f.Name(), 10, 64)
				if os.Getpid() == int(appPid) {
					continue
				}
				cmdLine := strings.SplitN(GetCmdLine(int(appPid)), " ", 2)
				if cmdLine[0] == selfCmdLine[0] {
					fPid, _ := strconv.ParseInt(f.Name(), 10, 64)
					Debug.Printf("Found dup pid: %d", fPid)
					return true, int(fPid)
				}
			}
		}
	}

	Debug.Println("NOT found dup pid")
	return false, -1
}

func PidProcess(pidFileName string) {
	var err error
	pidfile.SetPidfilePath(pidFileName)
	oldPid, err := pidfile.Read()
	if err == nil {
		Error.Fatalln("Already running with pid:", oldPid)
		os.Exit(1)
	}
	/*
	if ok, pid := FoundDupProcess(); ok {
		Error.Fatalln("Already running with founded pid:", pid)
		os.Exit(1)
	}
	*/
	err = pidfile.Write()
	if err != nil {
		Error.Fatalln("Can't write pid in file:", err)
		os.Exit(1)
	}
}

func GetPidFromFile(pidFileName string) (int, error) {
	pidfile.SetPidfilePath(pidFileName)

	oldPid, err := pidfile.Read()
	if err != nil {
		return 0, err
	}

	return oldPid, nil
}

func GetCmdLine(pid int) string {
	pidStr := strconv.FormatInt(int64(pid), 10)
	procFsCmdLine := "/proc/" + pidStr + "/cmdline"

	content, err := ioutil.ReadFile(procFsCmdLine)
	if err != nil {
		Error.Print(err)
	}
	args := strings.Split(string(content), "\x00")
	cmdLine := strings.Join(args, " ")

	return cmdLine
}
