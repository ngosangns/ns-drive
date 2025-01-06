package models

import (
	"encoding/json"
	"io"
	"os"
)

type Profile struct {
	Name            string   `json:"name"`
	From            string   `json:"from"`
	To              string   `json:"to"`
	IncludedPaths   []string `json:"included_paths"`
	ExcludedPaths   []string `json:"excluded_paths"`
	Bandwidth       int      `json:"bandwidth" default:"5M"`
	Parallel        int      `json:"parallel" default:"16"`
	ParallelChecker int      `json:"parallel_checker" default:"16"`
	BackupPath      string   `json:"backup_path"`
	CachePath       string   `json:"cache_path"`
}

type Profiles []Profile

func (profiles Profiles) ToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(profiles)
	if err != nil {
		return []byte{}, err
	}
	return jsonData, nil
}

func (profiles Profiles) ReadFromFile() error {
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

	err = json.Unmarshal(byteValue, &profiles)
	if err != nil {
		return err
	}

	return nil
}
