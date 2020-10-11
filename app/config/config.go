package config

import (
	"net"
	"strconv"
)

type RPCConfig struct {
	Host 			string 		`yaml:"host"`
	Port 			int 		`yaml:"port"`
	BlockchanName	string 		`yaml:"chain"`
}

func (cfg RPCConfig) BaseURL(proto string) string {
	return proto+"://"+cfg.Host+":"+strconv.FormatInt(int64(cfg.Port), 10)
}

// Параметр StartFromFirstBlock учитывается только в batch-режиме
type SBConfig struct {
	StartFromFirstBlock	bool 				`yaml:"start_from_first_block"`
	RPC 				RPCConfig 			`yaml:"rpc,flow"`
	Cyberway			CyberwayConfig		`yaml:"cyberway"`
}

type PortListener struct {
	Address net.IP `yaml:"host"`
	Port    int    `yaml:"port"`
}

type JWTConfig struct {
	PrivateKeyPath string `yaml:"private_key_path"`
	PublicKeyPath  string `yaml:"public_key_path"`
	Issuer         string `yaml:"issuer"`
	Expire         uint   `yaml:"expire"`
	RenewTime	   uint	  `yaml:"renew_time"`
}

type Fee struct {
	Amount		   float64 `yaml:"amount"`
	Symbol		   string `yaml:"symbol"`
}

type DelegationConfig struct {
	From			string	`yaml:"from"`
	Key				string	`yaml:"key"`
	Permission		string	`yaml:"permission"`
	Value			int	`yaml:"value"`
}

type GolosConfig struct {
	CreatorName       string `yaml:"creator"`
	CreatorKey  	  string `yaml:"creator_key"`
	CreatorPermission string `yaml:"creator_permission"`
	ProvideBWDays	  int	 `yaml:"provide_bw_days"`		// 0 - not use, -1 - use any time
	PaymentsTo        string `yaml:"payments_to"`
	Delegation	  	  DelegationConfig `yaml:"delegation"`
}

type L10NErrorFile struct {
	Lang 	 string		`yaml:"lang"`
	FileName string		`yaml:"file_name"`
}

type SharepayConfig struct {
	URL		string		`yaml:"url"`
	SPID	string		`yaml:"spid"`
}

type CyberwayConfig struct {
	Host 			string 		`yaml:"host"`
	Port 			int 		`yaml:"port"`
	Uri 			string 		`yaml:"uri"`
	ProcsCount		int			`yaml:"procs_count"`
}

type BackendConfig struct {
	CORSOrigin		string				`yaml:"cors_origin"`
	Golos			GolosConfig			`yaml:"golos,flow"`
	Listen			PortListener		`yaml:"listen,flow"`
	JWT				JWTConfig			`yaml:"jwt,flow"`
	RPC 			RPCConfig 			`yaml:"rpc,flow"`
	Sharpay			SharepayConfig		`yaml:"sharpay,flow"`
	L10NErrors		[]L10NErrorFile		`yaml:"l10n_errors,flow"`
	Cyberway		CyberwayConfig		`yaml:"cyberway"`
}
