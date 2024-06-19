package docker

import (
	"encoding/json"
)

// `docker ps --format json` コマンドの実行結果スキーマ
//
// Example:
//
//	{
//	}
type PsCommandResult struct {
	ID string `json:"ID"`
}

func GetID(psCommandResult string) (string, error) {
	result, err := UnmarshalPsCommandResult([]byte(psCommandResult))
	if err != nil {
		return "", err
	}

	return result.ID, nil
}

func UnmarshalPsCommandResult(data []byte) (PsCommandResult, error) {
	var result PsCommandResult

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
