package devcontainer

import (
	"encoding/json"
)

// `devcontainers up` コマンドの実行結果スキーマ
//
// Example: {"outcome":"success","containerId":"7278c789a975c34177e0b77d00477d5518c4fae4e66e6f0f9196561d5f895740","composeProjectName":"oasiz-mqtt-client","remoteUser":"root","remoteWorkspaceFolder":"/work"}
type UpCommandResult struct {
	Outcome               string `json:"outcome"`
	ContainerId           string `json:"containerId"`
	ComposeProjectName    string `json:"composeProjectName"`
	RemoteUser            string `json:"remoteUser"`
	RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
}

func GetContainerId(upCommandResult string) (string, error) {
	result, err := UnmarshalUpCommandResult([]byte(upCommandResult))
	if err != nil {
		return "", err
	}

	return result.ContainerId, nil
}

func UnmarshalUpCommandResult(data []byte) (UpCommandResult, error) {
	var result UpCommandResult

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
