package devcontainer

import (
	"encoding/json"
)

// devcontainer.json のスキーマ(の一部)
type DevcontainerJSON struct {
	DockerComposeFile []string `json:"dockerComposeFile"`
}

func UnmarshalDevcontainerJSON(data []byte) (DevcontainerJSON, error) {
	var result DevcontainerJSON

	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
