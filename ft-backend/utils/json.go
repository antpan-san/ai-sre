package utils

import (
	"encoding/json"
)

// MustMarshalJSON 将结构体转换为JSON字节数组，如果失败则panic
func MustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}