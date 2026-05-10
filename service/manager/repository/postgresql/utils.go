package postgresql

import (
	"encoding/json"
	"fmt"
)

type stringSliceJSON []string

func (s *stringSliceJSON) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("stringSliceJSON.Scan: unsupported type %T", src)
	}
	if len(data) == 0 {
		*s = nil
		return nil
	}
	return json.Unmarshal(data, (*[]string)(s))
}

func marshalStringSlice(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	return json.Marshal(values)
}
