package operations

import (
	"gitlab.com/stihi/stihi-backend/cyber/cyberdb"
	"strconv"
)

type NewAccountOp struct {
	BaseOperation

	Data				*NewAccountData		`json:"data"`
}

func (op *NewAccountOp) InitFromMap(source map[string]interface{}) *NewAccountOp {
	op.BaseOperation.InitFromMap(source)

	op.Data = (&NewAccountData{}).InitFromMap(*op.BaseOperation.Data)

	return op
}

type NewAccountData struct {
	Creator			string					`json:"creator"`
	Name			string					`json:"name"`

	Owner			*AccountKeyData			`json:"owner"`
	Active			*AccountKeyData			`json:"active"`
}

func (d *NewAccountData) InitFromMap(source map[string]interface{}) *NewAccountData {
	d.Creator, _ = source["creator"].(string)
	d.Name, _ = source["name"].(string)

	d.Owner  = (&AccountKeyData{}).InitFromMap(source["owner"].(map[string]interface{}))
	d.Active = (&AccountKeyData{}).InitFromMap(source["active"].(map[string]interface{}))

	return d
}

type AccountPermission struct {
	Permission 		string		`json:"permission"`
	Actor			string		`json:"actor"`
}

func (d *AccountPermission) InitFromMap(source map[string]interface{}) *AccountPermission {
	d.Permission, _ = source["permission"].(string)
	d.Actor, _ = source["actor"].(string)
	return d
}

type AccountListLine struct {
	Permission		*AccountPermission		`json:"permission"`
	Weight			int						`json:"weight"`
}

func (d *AccountListLine) InitFromMap(source map[string]interface{}) *AccountListLine {
	d.Weight, _ = source["weight"].(int)
	perm, _ := source["permission"].(map[string]interface{})
	d.Permission = (&AccountPermission{}).InitFromMap(perm)
	return d
}

type AccountKeyData struct {
	Threshold		int 					`json:"threshold"`
	Waits			[]*WaitsData			`json:"waits"`
	Accounts		[]*AccountListLine		`json:"accounts"`
	Keys			[]*KeyData				`json:"keys"`
}

func (d *AccountKeyData) InitFromMap(source map[string]interface{}) *AccountKeyData {
	d.Threshold, _ = source["threshold"].(int)

	waitsList, _ := source["waits"].([]map[string]interface{})
	d.Waits = make([]*WaitsData, len(waitsList), 0)
	for _, w := range waitsList {
		d.Waits = append(d.Waits, (&WaitsData{}).InitFromMap(w))
	}

	accList, _ := source["accounts"].([]map[string]interface{})
	d.Accounts = make([]*AccountListLine, len(accList), 0)
	for _, acc := range accList {
		d.Accounts = append(d.Accounts, (&AccountListLine{}).InitFromMap(acc))
	}

	keysList, _ := source["keys"].([]map[string]interface{})
	d.Keys = make([]*KeyData, len(keysList), 0)
	for _, k := range keysList {
		d.Keys = append(d.Keys, (&KeyData{}).InitFromMap(k))
	}

	return d
}

type WaitsData struct {
	WaitSec		int64						`json:"wait_sec"`
	Weight		int							`json:"weight"`
}

func (d *WaitsData) InitFromMap(source map[string]interface{}) *WaitsData {
	d.Weight, _ = source["weight"].(int)
	d.WaitSec, _ = source["wait_sec"].(int64)
	return d
}

type KeyData struct {
	Weight			int			`json:"weight"`
	Key				string		`json:"key"`
}

func (d *KeyData) InitFromMap(source map[string]interface{}) *KeyData {
	d.Weight, _ = source["weight"].(int)
	d.Key, _ = source["key"].(string)
	return d
}

func (d *KeyData) FromDB(key *cyberdb.KeyType) *KeyData {
	d.Key = key.Key

	weight, _ := strconv.ParseInt(key.Weight.String(), 10, 64)
	d.Weight = int(weight)

	return d
}