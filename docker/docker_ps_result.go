package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
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

// GetID reads the JSON objects emitted by docker ps and requires one container.
func GetID(psCommandResult string) (string, error) {
	decoder := json.NewDecoder(strings.NewReader(psCommandResult))
	var id string
	for {
		var result PsCommandResult
		err := decoder.Decode(&result)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read docker ps JSON: %w", err)
		}
		if strings.TrimSpace(result.ID) == "" {
			return "", fmt.Errorf("docker ps JSON is missing a container ID")
		}
		if id != "" {
			return "", fmt.Errorf("multiple containers match (IDs %q and %q)", id, result.ID)
		}
		id = result.ID
	}
	if id == "" {
		return "", &ContainerNotFoundError{msg: "container not found."}
	}
	return id, nil
}

func UnmarshalPsCommandResult(data []byte) (PsCommandResult, error) {
	var result PsCommandResult

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
