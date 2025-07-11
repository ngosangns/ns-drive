package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ReadFromFile(filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		file, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Printf("Warning: failed to close file: %v\n", err)
			}
		}()

		_, err = file.Write([]byte("[]"))
		if err != nil {
			return nil, err
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return byteValue, nil
}
