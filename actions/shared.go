package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/SermoDigital/jose/jws"
	"github.com/pkg/errors"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strings"
	"unicode/utf8"
)

var (
	DB		*cache_level1.CacheLevel1
	DefaultLang = "ru"
	ErrorsBySubstrings = []SubstrError{
		SubstrError{
			SubStr: "Missing Active Authority",
			ErrCode: "authorize.wrong_key_or_password",
		},
		SubstrError{
			SubStr: "Missing Posting Authority",
			ErrCode: "authorize.wrong_key_or_password",
		},
		SubstrError{
			SubStr: "frozen",
			ErrCode: "content.frozen",
		},
		SubstrError{
			SubStr: "Voting weight is too small",
			ErrCode: "users.more_vote_power",
		},
		SubstrError{
			SubStr: "malformed private key",
			ErrCode: "authorize.wrong_key_or_password_format",
		},
		SubstrError{
			SubStr: "voter is on the list",
			ErrCode: "users.already_voting",
		},
		SubstrError{
			SubStr: "already voted",
			ErrCode: "users.already_voting",
		},
		SubstrError{
			SubStr: "once every",
			ErrCode: "content.comment_need_wait",
		},
		SubstrError{
			SubStr: "t exist in cashout window",
			ErrCode: "content.vote_time_exired",
		},
		SubstrError{
			SubStr: "Permlink doesn't exist",
			ErrCode: "content.absent",
		},
	}
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

type SubstrError struct {
	SubStr 		string
	ErrCode		string
}

type ParamTypes struct {
	Name string
	Type interface{}
}

func DecodeRequest(r *http.Request) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &params)
	if err != nil {
		return nil, err
	}

	return params, err
}

func DecodeGetRequest(r *http.Request) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	for key, val := range r.URL.Query() {
		params[key] = val
	}

	return params, nil
}

func CORSOptionsProcess(w *http.ResponseWriter, r *http.Request) bool {
	if strings.ToUpper(r.Method) == "OPTIONS" {
		(*w).Header().Set("Access-Control-Max-Age", "3600")
		(*w).Header().Set("Access-Control-Allow-Origin", Config.CORSOrigin)

		method := r.Header.Get("Access-Control-Request-Method")
		if method != "" {
			(*w).Header().Set("Access-Control-Allow-Method", method)
		}
		headers := r.Header.Get("Access-Control-Request-Headers")
		if headers != "" {
			(*w).Header().Set("Access-Control-Allow-Headers", headers)
		}

		return true
	}

	return false
}

func DoPlainResponse(w *http.ResponseWriter, html string) {
	// Заголовок для CORS
	(*w).Header().Set("Access-Control-Allow-Origin", Config.CORSOrigin)
	(*w).Header().Set("Access-Control-Allow-Headers", "origin, x-requested-with, content-type, accept, authorization")

	fmt.Fprintln(*w, html)
}


func DoJSONResponse(w *http.ResponseWriter, answer *map[string]interface{}, token *jws.JWS) {
	// TODO: Проверять JTW и выдавать новый если старый устаревает

	// Заголовок для CORS
	(*w).Header().Set("Access-Control-Allow-Origin", Config.CORSOrigin)
	(*w).Header().Set("Access-Control-Allow-Headers", "origin, x-requested-with, content-type, accept, authorization")

	jsonStr, err := json.Marshal(*answer)
	if err != nil {
		app.Error.Printf("JSON marshal error: %s", err)
		fmt.Fprintf(*w, `{"status":"error", "error":"%s"}`+"\n", err)
		return
	}

	fmt.Fprintln(*w, string(jsonStr))
}

func DoJSONResponseOK(w *http.ResponseWriter, token *jws.JWS) {
	answer := map[string]interface{}{
		"status": "ok",
	}

	DoJSONResponse(w, &answer, token)
}

func DoJSONError(w *http.ResponseWriter, err error, token *jws.JWS) {
	errStr := err.Error()
	stackTrace := ""
	if err, ok := err.(stackTracer); ok {
		stackTrace = fmt.Sprintf("%+v", err.StackTrace())
	}
	app.Error.Printf("DoJSONError: %s\nStack trace:\n%s\n", errStr, stackTrace)
	if len(errStr) > 5 && errStr[:5] == "l10n:" {
		list := strings.Split(errStr, ":")
		errLocalize := errors_l10n.New(DefaultLang, list[1])
		if errLocalize.Error() != "" {
			errStr = errLocalize.Error()
		}
	} else {
		// Конвертируем ошибку в читабельный вид по массиву ключевых строк
		for _, se := range ErrorsBySubstrings {
			if strings.Contains(errStr, se.SubStr) {
				errStr = errors_l10n.New(DefaultLang, se.ErrCode).Error()
				break
			}
		}
	}

	answer := map[string]interface{}{
		"status": "error",
		"error": errStr,
	}

	DoJSONResponse(w, &answer, token)
}

func IsParamType(w *http.ResponseWriter, name string, val interface{}, typeStr string) bool {
	if val == nil {
		return false
	}
	if reflect.TypeOf(val).String() != typeStr {
		return false
	}
	return true
}

func IsParam(val interface{}, typeStr string) bool {
	if val == nil {
		return false
	}
	if reflect.TypeOf(val).String() != typeStr {
		return false
	}
	return true
}

func getTags(metadata interface{}) ([]string) {
	var tags []string
	if metadata != nil {
		meta, ok := metadata.(map[string]interface{})
		if ok {
			mTags, ok := meta["tags"]
			if ok && mTags != nil {
				metaTags := mTags.([]interface{})
				tags = make([]string, len(metaTags))
				for i := range metaTags {
					tags[i] = metaTags[i].(string)
				}
			}
		}
	}
	return tags
}

func getMetaString(metadata interface{}, key string) (string) {
	if metadata != nil {
		meta, ok := metadata.(map[string]interface{})
		if ok {
			keyDataI, ok := meta[key]
			if ok && keyDataI != nil {
				keyData, ok := keyDataI.(string)
				if ok {
					return keyData
				}
			}
		}
	}
	return ""
}

func SetLang(r *http.Request) string {
	return DefaultLang
}

func sliceExcludeString(slice []string, str string) ([]string) {
	newList := make([]string, 0)

	for _, s := range slice {
		if s != str {
			newList = append(newList, s)
		}
	}

	return newList
}

func encodeStr(str string) (string, error) {
	if utf8.ValidString(str) {
		return str, nil
	} else {
		sr := strings.NewReader(str)
		tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
		buf, err := ioutil.ReadAll(tr)
		if err != err {
			return "", err
		}

		return string(buf), nil
	}
}

//ipRange - a structure that holds the start and end of a range of ip addresses
type ipRange struct {
	start net.IP
	end net.IP
}

// inRange - check to see if a given ip address is within a range given
func inRange(r ipRange, ipAddress net.IP) bool {
	// strcmp type byte comparison
	if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
		return true
	}
	return false
}

var privateRanges = []ipRange{
	ipRange{
		start: net.ParseIP("10.0.0.0"),
		end:   net.ParseIP("10.255.255.255"),
	},
	ipRange{
		start: net.ParseIP("100.64.0.0"),
		end:   net.ParseIP("100.127.255.255"),
	},
	ipRange{
		start: net.ParseIP("172.16.0.0"),
		end:   net.ParseIP("172.31.255.255"),
	},
	ipRange{
		start: net.ParseIP("192.0.0.0"),
		end:   net.ParseIP("192.0.0.255"),
	},
	ipRange{
		start: net.ParseIP("192.168.0.0"),
		end:   net.ParseIP("192.168.255.255"),
	},
	ipRange{
		start: net.ParseIP("198.18.0.0"),
		end:   net.ParseIP("198.19.255.255"),
	},
}


// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {
	// my use case is only concerned with ipv4 atm
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		// iterate over all our ranges
		for _, r := range privateRanges {
			// check if this ip is in a private range
			if inRange(r, ipAddress){
				return true
			}
		}
	}
	return false
}

func GetRealClientIP(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		// march from right to left until we get a public address
		// that will be the address right before our proxy.
		for i := len(addresses) -1 ; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			// header can contain spaces too, strip those out.
			realIP := net.ParseIP(ip)
			if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
				// bad address, go to next
				continue
			}
			return ip
		}
	}
	return ""
}

func ProvideBWRequired(userName string) bool {
	if Config.Golos.ProvideBWDays == 0 {
		return false
	} else if userName == Config.Golos.CreatorName {
		return false
	} else if Config.Golos.ProvideBWDays > 0 {
		userCreationAge := DB.GetUserCreationAge(userName)
		return userCreationAge < Config.Golos.ProvideBWDays
	}
	return true
}