package devcontainer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ReadConfigurationError struct {
	msg string
}

func (e *ReadConfigurationError) Error() string {
	return e.msg
}

// `devcontainers read-configuration` コマンドの実行結果スキーマ
//
//	Example:	{
//							"configuration":{
//								"name":"development environment",
//								"dockerComposeFile":[
//									"../docker-compose.yaml"
//								],
//								"service":"app",
//								"workspaceFolder":"/work",
//								"remoteUser":"root",
//								"configFilePath": {
//									"$mid":1,
//									"fsPath":"/home/mikoto/project/oasiz-mqtt-client/.devcontainer/devcontainer.json",
//									"path":"/home/mikoto/project/oasiz-mqtt-client/.devcontainer/devcontainer.json",
//									"scheme":"vscode-fileHost"
//								}
//							},
//							"workspace": {
//								"workspaceFolder":"/work"
//							}
//						}
type ReadConfigurationCommandResult struct {
	Configuration Configuration `json:"configuration"`
}

type Configuration struct {
	DevcontainerJSON
	ForwardPorts   json.RawMessage `json:"forwardPorts,omitempty"`
	ConfigFilePath ConfigFilePath  `json:"configFilePath"`
}

type ConfigFilePath struct {
	FsPath string `json:"fsPath"`
}

type ForwardConfig struct {
	Host string
	Port string
}

const forwardHostLabel = `[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?`

var forwardHostPattern = regexp.MustCompile(`^` + forwardHostLabel + `(\.` + forwardHostLabel + `)*\.?$`)

// readConfigurationCommandResult から forwardPorts の情報を取得します。
func GetForwardPorts(readConfigurationCommandResult string) ([]ForwardConfig, error) {
	result, err := UnmarshalReadConfigurationCommandResult([]byte(readConfigurationCommandResult))
	if err != nil {
		return []ForwardConfig{}, &ReadConfigurationError{msg: "`devcontainer read-configuration` の出力パースに失敗しました。`.devcontainer.json が存在することと、 docker エンジンが起動していることを確認してください。"}
	}

	rawPorts := result.Configuration.ForwardPorts
	if rawPorts == nil {
		return nil, nil
	}
	var ports []json.RawMessage
	if err := json.Unmarshal(rawPorts, &ports); err != nil || ports == nil {
		return nil, &ReadConfigurationError{msg: "forwardPorts must be an array"}
	}

	forwardConfigs := make([]ForwardConfig, 0, len(ports))
	for i, raw := range ports {
		host, port := "localhost", string(raw)
		if strings.HasPrefix(port, `"`) {
			var address string
			if err := json.Unmarshal(raw, &address); err != nil {
				return nil, &ReadConfigurationError{msg: fmt.Sprintf("forwardPorts[%d]: invalid address", i)}
			}
			var found bool
			host, port, found = strings.Cut(address, ":")
			if !found || len(strings.TrimSuffix(host, ".")) > 253 || !forwardHostPattern.MatchString(host) {
				return nil, &ReadConfigurationError{msg: fmt.Sprintf("forwardPorts[%d]: expected a valid host:port address", i)}
			}
		}
		number, err := parseForwardPort(port)
		if err != nil {
			return nil, &ReadConfigurationError{msg: fmt.Sprintf("forwardPorts[%d]: %v", i, err)}
		}
		forwardConfigs = append(forwardConfigs, ForwardConfig{Host: host, Port: strconv.FormatUint(number, 10)})
	}

	return forwardConfigs, nil
}

func parseForwardPort(port string) (uint64, error) {
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || len(port) > 5 {
		return 0, fmt.Errorf("expected a decimal port from 0 to 65535")
	}
	return number, nil
}

func GetConfigFilePath(readConfigurationCommandResult string) (string, error) {
	result, err := UnmarshalReadConfigurationCommandResult([]byte(readConfigurationCommandResult))
	if err != nil {
		return "", &ReadConfigurationError{msg: "`devcontainer read-configuration` の出力パースに失敗しました。`.devcontainer.json が存在することと、 docker エンジンが起動していることを確認してください。"}
	}

	return result.Configuration.ConfigFilePath.FsPath, nil
}

func UnmarshalReadConfigurationCommandResult(data []byte) (ReadConfigurationCommandResult, error) {
	var result ReadConfigurationCommandResult

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
