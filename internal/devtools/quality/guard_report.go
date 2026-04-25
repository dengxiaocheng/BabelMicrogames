package quality

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

type GuardEntry struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	LogPath string `json:"log_path"`
}

func LoadGuardEntries(path string) ([]GuardEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := []GuardEntry{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry GuardEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func RenderGuardReport(entries []GuardEntry, title string) string {
	overall := "success"
	for _, entry := range entries {
		if entry.Status != "ok" {
			overall = "failed"
			break
		}
	}

	lines := []string{
		"## " + title,
		"",
		"- overall: `" + overall + "`",
		"",
		"| check | status | log |",
		"| --- | --- | --- |",
	}
	for _, entry := range entries {
		name := entry.Name
		if name == "" {
			name = "unknown"
		}
		status := entry.Status
		if status == "" {
			status = "unknown"
		}
		lines = append(lines, "| `"+name+"` | `"+status+"` | `"+entry.LogPath+"` |")
	}

	failed := []GuardEntry{}
	for _, entry := range entries {
		if entry.Status != "ok" {
			failed = append(failed, entry)
		}
	}
	if len(failed) > 0 {
		lines = append(lines, "", "### failed checks", "")
		for _, entry := range failed {
			name := entry.Name
			if name == "" {
				name = "unknown"
			}
			lines = append(lines, "- `"+name+"`: see `"+entry.LogPath+"`")
		}
	}

	return strings.Join(lines, "\n") + "\n"
}
