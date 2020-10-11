package errors_l10n

import (
	"fmt"
	"strings"
	"io/ioutil"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	l10nDB map[string]interface{}
	defaultLang = "en"
)

type ErrorL10N struct {
	lang 	string
	code   	string
	params 	[]interface{}
	err 	error
}

func LoadL10NBase(lang, filename string) error {
	if l10nDB == nil {
		l10nDB = make(map[string]interface{})
	}

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.New("Error read lang file '"+filename+"': "+err.Error())
	}

	l10nDB[lang] = make(map[interface{}]interface{})
	err = yaml.Unmarshal(yamlFile, l10nDB[lang])
	if err != nil {
		return errors.New("Error parsing lang file '"+filename+"': "+err.Error())
	}

	return nil
}

func SetDefaultLang(lang string) {
	defaultLang = lang
}

func New(lang, code string, params ...interface{}) (*ErrorL10N) {
	err := ErrorL10N{
		err: errors.New(code),
	}
	err.SetLang(lang)
	err.SetCode(code)
	err.SetParams(params...)

	return &err
}

func (err *ErrorL10N) StackTrace() errors.StackTrace {
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	if e, ok := err.err.(stackTracer); ok {
		return e.StackTrace()
	}

	return errors.StackTrace{}
}

func (err *ErrorL10N) Error() string {
	format := getElementFromDB(err.lang, err.code)
	if format == "" {
		format = getElementFromDB(defaultLang, err.code)
	}

	var str string
	if err.params != nil && len(err.params) > 0 {
		// Генерируем строку с учетом параметров
		str = fmt.Sprintf(format, err.params...)
	} else {
		str = format
	}

	return str
}

func (err *ErrorL10N) SetCode(code string) {
	err.code = code
}

func (err *ErrorL10N) SetLang(lang string) {
	err.lang = lang
}

func (err *ErrorL10N) SetParams(params ...interface{}) {
	err.params = params
}

func getElementFromDB(lang, code string) string {
	base := l10nDB[lang]
	if base == nil {
		base = l10nDB[defaultLang]
	}
	if base == nil {
		return "Unknown lang: "+lang
	}

	curNode := base.(map[interface{}]interface{})
	path := strings.Split(code, ".")
	for _, elem := range path {
		val, ok := curNode[elem]
		if ok {
			switch val.(type) {
			case map[interface{}]interface{}:
				curNode = val.(map[interface{}]interface{})
			case string:
				return val.(string)
			}
		} else {
			return "-- not found code '"+code+"' for lang '"+lang+"' --"
		}
	}

	return "-- not found code '"+code+"' for lang '"+lang+"' --"
}