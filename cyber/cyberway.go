package cyber

import (
	"encoding/json"
	"gitlab.com/stihi/stihi-backend/app/config"
	"gitlab.com/stihi/stihi-backend/app/random"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

/*
 	В данном модуле содержится код для взаимодействия с нодой cyberway для получения данных
*/

type Info struct {
	ServerVersion 				string 		`json:"server_version"`
	ChainId						string 		`json:"chain_id"`
	HeadBlockNum				int64		`json:"head_block_num"`
	LastIrreversibleBlockNum	int64		`json:"last_irreversible_block_num"`
	LastIrreversibleBlockId 	string 		`json:"last_irreversible_block_id"`
	HeadBlockId					string 		`json:"head_block_id"`
	HeadBlockTime				string 		`json:"head_block_time"`
	HeadBlockProducer			string		`json:"head_block_producer"`
	VirtualBlockCpuLimit		int64 		`json:"virtual_block_cpu_limit"`
	VirtualBlockNetLimit 		int64		`json:"virtual_block_net_limit"`
	BlockCpuLimit				int64		`json:"block_cpu_limit"`
	BlockNetLimit				int64		`json:"block_net_limit"`
	ServerVersionString			string 		`json:"server_version_string"`
}

var (
	Config 	*config.CyberwayConfig
)

func Init(cfg *config.CyberwayConfig) {
	Config = cfg
}

func BaseURL() string {
	return "http://" + Config.Host + ":" + strconv.FormatInt(int64(Config.Port), 10) + "/" + Config.Uri + "/"
}

func GetInfo() (*Info, error) {
	baseUrl := BaseURL()
	res, err := http.Get(baseUrl+"chain/get_info")
	if err != nil {
		return nil, err
	}

	info := Info{}
	jsDecoder := json.NewDecoder(res.Body)
	err = jsDecoder.Decode(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func GetBlockRaw(num int64) (string, error) {
	baseUrl := BaseURL()
	params := strings.NewReader(`{"block_num_or_id":`+strconv.FormatInt(num, 10)+`}`)
	res, err := http.Post(baseUrl+"chain/get_block", "application/json", params)
	if err != nil {
		return "", err
	}

	data, _ := ioutil.ReadAll(res.Body)
	return string(data), nil
}

func GenCyberUserId(prefix string) string {
	randLen := 12 - len(prefix)

	cyberNameCharset := random.CharsetAl + "12345"
	randStr := random.StringWithCharset(randLen, cyberNameCharset, len(cyberNameCharset))
	return prefix+randStr
}
