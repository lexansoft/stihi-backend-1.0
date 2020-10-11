package blockchain

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
)

//ContentMetadata type from parameter JSON
type ContentMetadata map[string]interface{}

//UnmarshalJSON unpacking the JSON parameter in the ContentMetadata type.
func (op *ContentMetadata) UnmarshalJSON(p []byte) error {
	var raw map[string]interface{}

	str, errUnq := strconv.Unquote(string(p))
	if errUnq != nil {
		return errUnq
	}
	if str == "" || str == "\"\"" {
		return nil
	}

	if err := json.Unmarshal([]byte(str), &raw); err != nil {
		return errors.Wrap(err, "ERROR: ContentMedata unmarshal error")
	}

	*op = raw

	return nil
}

//MarshalJSON function for packing the ContentMetadata type in JSON.
func (op *ContentMetadata) MarshalJSON() ([]byte, error) {
	ans, err := json.Marshal(*op)
	if err != nil {
		return []byte{}, err
	}
	return []byte(strconv.Quote(string(ans))), nil
}
