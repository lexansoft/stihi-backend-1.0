package operations

type CreateMessageOp struct {
	BaseOperation

	Data				*CreateMessageData 	`json:"data"`
}

type CreateMessageData struct {
	Id					*MessageIdType		`json:"message_id"`
	ParentId			*MessageIdType		`json:"parent_id"`

	Language			string				`json:"languagemssg"`
	Header				string				`json:"headermssg"`
	Body				string				`json:"bodymssg"`

	Tags				[]string			`json:"tags"`
	JsonMetadata		string				`json:"jsonmetadata"`

	TokenProp			int					`json:"tokenprop"`
	MaxPayout			*int				`json:"max_payout"`
	Beneficiaries		[]string			`json:"beneficiaries"`
	CuratorsPrcnt		int					`json:"curators_prcnt"`
	VestPayment			bool				`json:"vestpayment"`
}

func (op *CreateMessageOp) InitFromMap(source map[string]interface{}) *CreateMessageOp {
	op.BaseOperation.InitFromMap(source)

	op.Data = (&CreateMessageData{}).InitFromMap(*op.BaseOperation.Data)

	return op
}

func (d *CreateMessageData) InitFromMap(source map[string]interface{}) *CreateMessageData {
	msgId, _ := source["message_id"].(map[string]interface{})
	d.Id = (&MessageIdType{}).InitFromMap(msgId)

	msgParentId, _ := source["parent_id"].(map[string]interface{})
	d.ParentId = (&MessageIdType{}).InitFromMap(msgParentId)

	d.Language, _ = source["languagemssg"].(string)
	d.Header, _ = source["headermssg"].(string)
	d.Body, _ = source["bodymssg"].(string)

	d.Tags, _ = source["tags"].([]string)
	d.JsonMetadata, _ = source["jsonmetadata"].(string)

	d.TokenProp, _ = source["tokenprop"].(int)
	d.MaxPayout, _ = source["max_payout"].(*int)
	d.Beneficiaries, _ = source["beneficiaries"].([]string)
	d.CuratorsPrcnt, _ = source["curators_prcnt"].(int)
	d.VestPayment, _ = source["vestpayment"].(bool)

	return d
}