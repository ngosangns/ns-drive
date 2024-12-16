package utils

import (
	"bufio"
	"os"
	"strings"
)

func LoadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines or comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Split key and value by the first '='
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)

		// Set the environment variable
		os.Setenv(key, value)
	}

	return scanner.Err()
}
