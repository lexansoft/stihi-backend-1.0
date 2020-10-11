package operations

type UpdateAuthOp struct {
	BaseOperation

	Data		*UpdateAuthData			`json:"data"`
}

func (op *UpdateAuthOp) InitFromMap(source map[string]interface{}) *UpdateAuthOp {
	op.BaseOperation.InitFromMap(source)

	op.Data = (&UpdateAuthData{}).InitFromMap(*op.BaseOperation.Data)

	return op
}

type UpdateAuthData struct {
	Account		string					`json:"account"`
	Permission	string					`json:"permission"`
	Parent		string					`json:"parent"`

	Auth		*AccountKeyData			`json:"auth"`
}

func (d *UpdateAuthData) InitFromMap(source map[string]interface{}) *UpdateAuthData {
	d.Account, _ = source["account"].(string)
	d.Permission, _ = source["permission"].(string)
	d.Parent, _ = source["parent"].(string)

	auth, _ := source["auth"].(map[string]interface{})
	d.Auth = (&AccountKeyData{}).InitFromMap(auth)

	return d
}
