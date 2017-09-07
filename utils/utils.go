package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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
