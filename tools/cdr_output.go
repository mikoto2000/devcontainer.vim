package tools

import (
	"encoding/json"
)

// `clipboard-data-receiver` コマンドの標準出力スキーマ
//
// Example:
//
//	{
//	  "pid": 1234,
//	  "address": "0.0.0.0",
//	  "port": 5678
//	}
type CdrOutput struct {
	Pid     int    `json:"pid"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func GetProcessInfo(cdrOutput string) (int, string, int, error) {
	result, err := UnmarshalCdrOutput([]byte(cdrOutput))
	if err != nil {
		return 0, "", 0, err
	}

	return result.Pid, result.Address, result.Port, nil
}

func UnmarshalCdrOutput(data []byte) (CdrOutput, error) {
	var result CdrOutput

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
