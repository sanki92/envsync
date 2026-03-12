package vault

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type EnvEntry struct {
	Key     string
	Value   string
	Comment string
}

func ReadEnvFile(path string) ([]EnvEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var entries []EnvEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			entries = append(entries, EnvEntry{Comment: line})
			continue
		}

		key, value := parseEnvLine(trimmed)
		entries = append(entries, EnvEntry{Key: key, Value: value})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return entries, nil
}

func WriteEnvFile(path string, entries []EnvEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, e := range entries {
		if e.Key == "" {
			fmt.Fprintln(w, e.Comment)
		} else {
			fmt.Fprintf(w, "%s=%s\n", e.Key, e.Value)
		}
	}
	return w.Flush()
}

func EnvMap(entries []EnvEntry) map[string]string {
	m := make(map[string]string)
	for _, e := range entries {
		if e.Key != "" {
			m[e.Key] = e.Value
		}
	}
	return m
}

func IsPublicKey(key string) bool {
	prefixes := []string{"NEXT_PUBLIC_", "REACT_APP_", "VITE_"}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func parseEnvLine(line string) (string, string) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return line, ""
	}

	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])


	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return key, value
}
