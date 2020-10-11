package operations

type UpdateMessageOp struct {
	BaseOperation

	Data				*UpdateMessageData 	`json:"data"`
}

type UpdateMessageData struct {
	Id					*MessageIdType		`json:"message_id"`

	Language			string				`json:"languagemssg"`
	Header				string				`json:"headermssg"`
	Body				string				`json:"bodymssg"`

	Tags				[]string			`json:"tags"`
	JsonMetadata		string				`json:"jsonmetadata"`
}

func (op *UpdateMessageOp) InitFromMap(source map[string]interface{}) *UpdateMessageOp {
	op.BaseOperation.InitFromMap(source)

	op.Data = (&UpdateMessageData{}).InitFromMap(*op.BaseOperation.Data)

	return op
}

func (d *UpdateMessageData) InitFromMap(source map[string]interface{}) *UpdateMessageData {
	msgId, _ := source["message_id"].(map[string]interface{})
	d.Id = (&MessageIdType{}).InitFromMap(msgId)

	d.Language, _ = source["languagemssg"].(string)
	d.Header, _ = source["headermssg"].(string)
	d.Body, _ = source["bodymssg"].(string)

	d.Tags, _ = source["tags"].([]string)
	d.JsonMetadata, _ = source["jsonmetadata"].(string)

	return d
}