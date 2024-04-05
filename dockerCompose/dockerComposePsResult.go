package dockerCompose

import (
	"encoding/json"
)

// `docker compose ps --format json` コマンドの実行結果スキーマ
//
// Example:
//
//	{
//	  "Command":"\"docker-entrypoint.s…\"",
//	  "CreatedAt":"2024-04-05 09:29:10 +0900 JST",
//	  "ExitCode":0,
//	  "Health":"",
//	  "ID":"1ab1fd63fb94",
//	  "Image":"node:18",
//	  "Labels":"...(snip)",
//	  "LocalVolumes":"1",
//	  "Mounts":"/run/desktop/m…,oasiz-mqtt-cli…",
//	  "Name":"oasiz-mqtt-client-app-1",
//	  "Names":"oasiz-mqtt-client-app-1",
//	  "Networks":"oasiz-mqtt-client_default",
//	  "Ports":"0.0.0.0:5173-\u003e5173/tcp",
//	  "Project":"oasiz-mqtt-client",
//	  "Publishers":[
//	    {
//	      "URL":"0.0.0.0",
//	      "TargetPort":5173,
//	      "PublishedPort":5173,
//	      "Protocol":"tcp"
//	    }
//	  ],
//	  "RunningFor":"4 seconds ago",
//	  "Service":"app",
//	  "Size":"0B",
//	  "State":"running",
//	  "Status":"Up 4 seconds"
//	}
type PsCommandResult struct {
	Project string `json:"Project"`
}

func GetProjectName(psCommandResult string) (string, error) {
	result, err := UnmarshalPsCommandResult([]byte(psCommandResult))
	if err != nil {
		return "", err
	}

	return result.Project, nil
}

func UnmarshalPsCommandResult(data []byte) (PsCommandResult, error) {
	var result PsCommandResult

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

