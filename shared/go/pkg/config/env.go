package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile loads environment variables from a .env file
// This is useful for development environments
func LoadEnvFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		// .env file is optional, don't error if it doesn't exist
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid line %d in env file: %s", lineNumber, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Set environment variable (only if not already set)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading env file: %w", err)
	}

	return nil
}

// LoadEnvFiles loads multiple .env files in order
// Later files override earlier ones
func LoadEnvFiles(filePaths ...string) error {
	for _, filePath := range filePaths {
		if err := LoadEnvFile(filePath); err != nil {
			return err
		}
	}
	return nil
}

// MustLoadEnvFile loads a .env file and panics on error
func MustLoadEnvFile(filePath string) {
	if err := LoadEnvFile(filePath); err != nil {
		panic(fmt.Sprintf("failed to load env file: %v", err))
	}
}

// GetEnvOrPanic gets an environment variable or panics if not set
func GetEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

// RequireEnvVars checks that all required environment variables are set
func RequireEnvVars(keys ...string) error {
	missing := make([]string, 0)

	for _, key := range keys {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// EnvVarInfo holds information about an environment variable
type EnvVarInfo struct {
	Key          string
	DefaultValue string
	Required     bool
	Description  string
}

// ValidateEnvVars validates that all required environment variables are set
// and returns information about what's missing
func ValidateEnvVars(vars []EnvVarInfo) ([]string, error) {
	missing := make([]string, 0)
	warnings := make([]string, 0)

	for _, v := range vars {
		value := os.Getenv(v.Key)

		if value == "" {
			if v.Required {
				missing = append(missing, fmt.Sprintf("%s (required): %s", v.Key, v.Description))
			} else if v.DefaultValue == "" {
				warnings = append(warnings, fmt.Sprintf("%s (optional): %s", v.Key, v.Description))
			}
		}
	}

	if len(missing) > 0 {
		return warnings, fmt.Errorf("missing required environment variables:\n%s",
			strings.Join(missing, "\n"))
	}

	return warnings, nil
}

// GetAllEnvVars returns all environment variables as a map
func GetAllEnvVars() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

// FilterEnvVars returns environment variables matching a prefix
func FilterEnvVars(prefix string) map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 && strings.HasPrefix(pair[0], prefix) {
			env[pair[0]] = pair[1]
		}
	}
	return env
}
