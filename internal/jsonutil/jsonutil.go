package jsonutil

import "encoding/json"

func Marshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
