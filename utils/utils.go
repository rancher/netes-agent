package utils

import (
	"encoding/json"
	"crypto/md5"
	"encoding/hex"
)

func ConvertByJSON(src, target interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}

func Hash(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}
