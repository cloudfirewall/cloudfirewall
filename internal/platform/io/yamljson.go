package io

import "encoding/json"

func UnmarshalByExt(_ string, data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func MarshalJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
