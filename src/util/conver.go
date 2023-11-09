package util

import (
	"encoding/json"
	"log"
)

// json字串 轉 map
func JsonToMap(jsonStr string) (map[string]string, error) {

	m := make(map[string]string)

	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		log.Printf("jsonToMap unmarshal err:%v", err)
		return m, err
	}

	return m, nil
}

// map 轉 json字串
func MapToJson(m map[string]string) (string, error) {

	jsonByte, err := json.Marshal(m)
	if err != nil {
		log.Printf("jsonToMap marshal err:%v", err)
		return "", err
	}

	return string(jsonByte[:]), err
}

// map 轉 json byte array
func MapToJsonByte(m map[string]interface{}) []byte {

	jsonByte, err := json.Marshal(m)
	if err != nil {
		log.Printf("jsonToMap marshal err:%v", err)
		return nil
	}

	return jsonByte
}
