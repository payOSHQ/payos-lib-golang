package apijson

import "encoding/json"

// ConvertInterface converts an interface{} to a specific type using JSON marshaling
func ConvertInterface(src interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
