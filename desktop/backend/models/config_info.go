package models

import "encoding/json"

type ConfigInfo struct {
	WorkingDir string   `json:"working_dir"`
	Profiles   Profiles `json:"profiles"`
}

func (c ConfigInfo) ToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(c)
	if err != nil {
		return []byte{}, err
	}
	return jsonData, nil
}
