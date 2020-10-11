package db

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
	"gitlab.com/stihi/stihi-backend/app"
)

type Settings struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func LoadConfig() {
	if dbSettings != nil {
		return
	}

	InitFromFile(os.Getenv(EnvDbFileConfig))
}

func InitFromFile(dbConfigFileName string) {
	if dbSettings == nil {
		dbSettings = &Settings{}
	}

	if dbConfigFileName == "" {
		app.Error.Fatalf("Db config file name required!")
	}

	_, err := os.Stat(dbConfigFileName)
	if os.IsNotExist(err) {
		app.Error.Fatalf("Db config file '%s' not exists.", dbConfigFileName)
	}

	dat, err := ioutil.ReadFile(dbConfigFileName)
	if err != nil {
		app.Error.Fatalln(err)
	}

	err = yaml.Unmarshal(dat, dbSettings)
	if err != nil {
		app.Error.Fatalf("error: %v", err)
	}
}

func IsInListInt(list []int, id int) bool {
	for _, goodId := range list {
		if goodId == id {
			return true
		}
	}

	return false
}

func EscapedBy(source string, symbol string, code string) string {
	return strings.Replace(source, symbol, code, -1)
}

func Escaped(source string) string {
	return EscapedBy(source, "'", "''")
}

// Convert time:
// 2017-01-11T00:00:00Z -> 2017-01-11
// 2017-01-11T00:00:01Z -> 2017-01-11 00:00:01
func ConvertTime(src string) string {
	split := strings.Split(src, "T")
	if len(split) < 2 {
		split_s := regexp.MustCompile("[^\\d\\:\\-]+")
		split = split_s.Split(src, -1)
		if len(split) <= 1 {
			return src
		}
	}

	if split[1] == "00:00:00Z" || split[1] == "00:00:00" {
		return split[0]
	} else {
		time := split[1]
		time = strings.Replace(time, "Z", "", -1)
		return split[0] + " " + time
	}

	return src
}
