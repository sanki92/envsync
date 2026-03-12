package github

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func FetchSSHKeys(username string) ([]string, error) {
	url := fmt.Sprintf("https://github.com/%s.keys", username)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch keys for %s: %w", username, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub user %q not found or has no SSH keys", username)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub returned status %d for %s", resp.StatusCode, username)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	content := strings.TrimSpace(string(body))
	if content == "" {
		return nil, fmt.Errorf("no SSH keys found on github.com/%s", username)
	}

	var keys []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

func FindEd25519Key(keys []string) string {
	for _, k := range keys {
		if strings.HasPrefix(k, "ssh-ed25519 ") {
			return k
		}
	}
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}
