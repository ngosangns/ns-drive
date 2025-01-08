package models

import (
	"encoding/json"
	"io"
	"os"
)

type ConfigInfo struct {
	WorkingDir           string   `json:"working_dir"`
	SelectedProfileIndex uint     `json:"selected_profile_index"`
	Profiles             Profiles `json:"profiles"`
}

func (c ConfigInfo) ToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(c)
	if err != nil {
		return []byte{}, err
	}
	return jsonData, nil
}

func (c *ConfigInfo) ReadFromFile() error {
	if _, err := os.Stat(".profiles"); os.IsNotExist(err) {
		file, err := os.Create(".profiles")
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.Write([]byte("[]"))
		if err != nil {
			return err
		}
	}

	file, err := os.Open(".profiles")
	if err != nil {
		return err
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	profiles := Profiles{}

	err = json.Unmarshal(byteValue, &profiles)
	if err != nil {
		return err
	}

	c.Profiles = profiles

	return nil
}
