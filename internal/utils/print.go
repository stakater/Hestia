package utils

import (
	"encoding/json"
	"fmt"
)

func FormatResource(obj interface{}) string {
	jsonData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v\n", obj)
	}

	return string(jsonData)
}
