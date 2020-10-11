package operations

const (
	OpCreateMessageName = "createmssg"
	OpUpdateMessageName = "updatemssg"
	OpNewAccountName = "newaccount"
	OpUpdateAuthName = "updateauth"
	OpLinkAuthName = "linkauth"
	OpNewUserNameName = "newusername"
	OpUpVote = "upvote"
)

type BaseOperation struct {
	Name				string            			`json:"name"`
	Account				string         				`json:"account"`
	Authorization		[]*AuthInfoType   			`json:"authorization"`
	HexData				string						`json:"hex_data"`
	Data				*map[string]interface{}		`json:"data"`
}

func (op *BaseOperation) InitFromMap(source map[string]interface{}) *BaseOperation {
	op.Name, _ = source["name"].(string)
	op.Account, _ = source["account"].(string)
	op.HexData, _ = source["hex_data"].(string)

	authList, _ := source["authorization"].([]map[string]interface{})
	op.Authorization = make([]*AuthInfoType, len(authList), 0)
	for _, authRec := range authList {
		op.Authorization = append(op.Authorization, (&AuthInfoType{}).InitFromMap(authRec))
	}

	data, _ := source["data"].(map[string]interface{})
	op.Data = &data

	return op
}

type AuthInfoType struct {
	Actor		string 			`json:"actor"`
	Permission	string			`json:"permission"`
}

func (info *AuthInfoType) InitFromMap(source map[string]interface{}) *AuthInfoType {
	info.Actor, _ = source["actor"].(string)
	info.Permission, _ = source["permission"].(string)
	return info
}

type MessageIdType struct {
	Author		string			`json:"author"`
	Permlink	string			`json:"permlink"`
}

func (id *MessageIdType) InitFromMap(source map[string]interface{}) *MessageIdType {
	id.Author, _ = source["author"].(string)
	id.Permlink, _ = source["permlink"].(string)
	return id
}
