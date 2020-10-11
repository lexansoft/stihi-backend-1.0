package operations

type NewUserNameOp struct {
	BaseOperation

	Data				*NewUserNameData		`json:"data"`
}

func (op *NewUserNameOp) InitFromMap(source map[string]interface{}) *NewUserNameOp {
	op.BaseOperation.InitFromMap(source)

	op.Data = (&NewUserNameData{}).InitFromMap(*op.BaseOperation.Data)

	return op
}

type NewUserNameData struct {
	Creator			string					`json:"creator"`
	Name			string					`json:"name"`
	Owner			string					`json:"owner"`
}

func (d *NewUserNameData) InitFromMap(source map[string]interface{}) *NewUserNameData {
	d.Creator, _ = source["creator"].(string)
	d.Name, _ = source["name"].(string)
	d.Owner  = source["owner"].(string)

	return d
}
