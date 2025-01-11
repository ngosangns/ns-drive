package utils

import (
	"bufio"
	"os"
	"strings"

	beConfig "desktop/backend/config"

	"github.com/caarlos0/env/v11"
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

func LoadConfigFromEnv() (*beConfig.Config, error) {
	// Load the .env file
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	err = LoadEnvFile(wd + "/.env")
	if err != nil {
		return nil, err
	}

	// Parse environment variables into the struct
	var config beConfig.Config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
