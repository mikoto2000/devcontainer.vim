package devcontainer

import (
	"encoding/json"
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
	ForwardPorts   []any          `json:"forwardPorts"`
	ConfigFilePath ConfigFilePath `json:"configFilePath"`
}

type ConfigFilePath struct {
	FsPath string `json:"fsPath"`
}

type ForwardConfig struct {
	Host string
	Port string
}

// readConfigurationCommandResult から forwardPorts の情報を取得します。
func GetForwardPorts(readConfigurationCommandResult string) ([]ForwardConfig, error) {
	result, err := UnmarshalReadConfigurationCommandResult([]byte(readConfigurationCommandResult))
	if err != nil {
		return []ForwardConfig{}, &ReadConfigurationError{msg: "`devcontainer read-configuration` の出力パースに失敗しました。`.devcontainer.json が存在することと、 docker エンジンが起動していることを確認してください。"}
	}

	forwardPorts := result.Configuration.ForwardPorts

	forwardConfigs := make([]ForwardConfig, len(forwardPorts))

	for i, forwardPort := range forwardPorts {
		switch v := forwardPort.(type) {
		case int:
			forwardConfigs[i] =
				ForwardConfig{
					Host: "localhost",
					Port: strconv.Itoa(v),
				}
		case float64:
			forwardConfigs[i] =
				ForwardConfig{
					Host: "localhost",
					Port: strconv.Itoa(int(v)),
				}
		case string:
			item := strings.Split(v, ":")
			forwardConfigs[i] =
				ForwardConfig{
					Host: item[0],
					Port: item[1],
				}
		default:
			// do nothing
		}
	}

	return forwardConfigs, nil
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
