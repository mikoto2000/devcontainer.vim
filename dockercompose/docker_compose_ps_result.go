package dockercompose

import (
	"encoding/json"
	"errors"
	"strings"
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
	output := strings.TrimSpace(psCommandResult)
	if output == "" {
		return "", nil
	}
	var project string
	if strings.HasPrefix(output, "[") {
		// Older Compose versions returned a JSON array.
		var results []PsCommandResult
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			return "", err
		}
		if len(results) == 0 {
			return "", nil
		}
		project = results[0].Project
	} else {
		line, _, _ := strings.Cut(output, "\n")
		result, err := UnmarshalPsCommandResult([]byte(line))
		if err != nil {
			return "", err
		}
		project = result.Project
	}
	if project == "" {
		return "", errors.New("docker compose ps returned a container without a project name")
	}
	return project, nil
}

func UnmarshalPsCommandResult(data []byte) (PsCommandResult, error) {
	var result PsCommandResult

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
