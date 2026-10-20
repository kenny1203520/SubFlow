package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile removes the optional --env-file argument and loads unset process
// variables from the file. Existing environment variables always take priority.
func LoadEnvFile(args []string) ([]string, error) {
	clean := make([]string, 0, len(args))
	file := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--env-file":
			if i+1 >= len(args) || args[i+1] == "" {
				return nil, fmt.Errorf("--env-file requires a path")
			}
			i++
			file = args[i]
		case strings.HasPrefix(arg, "--env-file="):
			file = strings.TrimPrefix(arg, "--env-file=")
			if file == "" {
				return nil, fmt.Errorf("--env-file requires a path")
			}
		default:
			clean = append(clean, arg)
		}
	}
	if file == "" {
		return clean, nil
	}
	input, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer input.Close()

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("invalid env file entry at line %d", lineNo)
		}
		value = strings.TrimSpace(value)
		if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
			quote := value[0]
			if end := strings.IndexByte(value[1:], quote); end >= 0 {
				end++
				rest := strings.TrimSpace(value[end+1:])
				if rest != "" && !strings.HasPrefix(rest, "#") {
					return nil, fmt.Errorf("invalid quoted env value at line %d", lineNo)
				}
				value = value[1:end]
			}
		} else if comment := strings.Index(value, " #"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return nil, fmt.Errorf("set env file value at line %d: %w", lineNo, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}
	return clean, nil
}
