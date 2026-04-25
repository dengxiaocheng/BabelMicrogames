package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"babel-runtime/internal/core/types"
)

type operationalMemoryState struct {
	RuntimeID       string   `json:"runtime_id"`
	ExecutionID     string   `json:"execution_id"`
	UpdatedAtUnix   int64    `json:"updated_at_unix"`
	TaskIDs         []string `json:"task_ids,omitempty"`
	ArtifactIDs     []string `json:"artifact_ids,omitempty"`
	LatestReplyText string   `json:"latest_reply_text,omitempty"`
}

type sessionManifest struct {
	RuntimeID       string   `json:"runtime_id"`
	ExecutionID     string   `json:"execution_id"`
	UpdatedAtUnix   int64    `json:"updated_at_unix"`
	PreferredWorker string   `json:"preferred_worker"`
	PrimaryDocument string   `json:"primary_document"`
	AuxiliaryDocs   []string `json:"auxiliary_docs,omitempty"`
}

func writeOperationalMemory(root string, executionID string, tasks []types.AgentTaskSpec, artifacts []types.AgentArtifact, now time.Time) error {
	if root == "" || len(artifacts) == 0 {
		return nil
	}

	runtimeID := artifacts[0].RuntimeID
	if runtimeID == "" && len(tasks) > 0 {
		runtimeID = tasks[0].RuntimeID
	}
	if runtimeID == "" {
		return fmt.Errorf("missing runtime id for operational memory")
	}

	dir := filepath.Join(root, runtimeDirName(runtimeID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	reply := latestAssistantText(artifacts)
	taskLines := make([]string, 0, len(tasks))
	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.TaskID)
		taskLines = append(taskLines, fmt.Sprintf("- [%s] %s", task.TaskType, task.Input))
	}
	sort.Strings(taskIDs)
	sort.Strings(taskLines)

	artifactIDs := make([]string, 0, len(artifacts))
	factLines := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactIDs = append(artifactIDs, artifact.ArtifactID)
		if artifact.Body != "" {
			factLines = append(factLines, "- "+artifact.Body)
		}
	}
	sort.Strings(artifactIDs)
	sort.Strings(factLines)

	state := operationalMemoryState{
		RuntimeID:       runtimeID,
		ExecutionID:     executionID,
		UpdatedAtUnix:   now.Unix(),
		TaskIDs:         taskIDs,
		ArtifactIDs:     artifactIDs,
		LatestReplyText: reply,
	}
	stateBody, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	manifestBody, err := json.MarshalIndent(sessionManifest{
		RuntimeID:       runtimeID,
		ExecutionID:     executionID,
		UpdatedAtUnix:   now.Unix(),
		PreferredWorker: "claude_code",
		PrimaryDocument: "primary_context.md",
		AuxiliaryDocs: []string{
			"state.json",
			"recent_summary.md",
			"open_threads.md",
			"long_term_facts.md",
		},
	}, "", "  ")
	if err != nil {
		return err
	}

	primaryContext := buildPrimaryContext(runtimeID, executionID, reply, taskLines, factLines, now)

	// These files are intentionally derived and replaceable. They help future
	// workers regain context without becoming canonical runtime truth.
	files := map[string][]byte{
		"state.json":            append(stateBody, '\n'),
		"session_manifest.json": append(manifestBody, '\n'),
		"primary_context.md":    []byte(primaryContext),
		"recent_summary.md":     []byte(strings.TrimSpace(reply) + "\n"),
		"open_threads.md":       []byte(strings.Join(taskLines, "\n") + trailingNewline(taskLines)),
		"long_term_facts.md":    []byte(strings.Join(factLines, "\n") + trailingNewline(factLines)),
	}

	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runtimeDirName(runtimeID string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_")
	return replacer.Replace(runtimeID)
}

func latestAssistantText(artifacts []types.AgentArtifact) string {
	for i := len(artifacts) - 1; i >= 0; i-- {
		if artifacts[i].ArtifactType == "assistant_text" && artifacts[i].Body != "" {
			return artifacts[i].Body
		}
	}
	if len(artifacts) == 0 {
		return ""
	}
	return artifacts[len(artifacts)-1].Body
}

func trailingNewline(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return "\n"
}

func buildPrimaryContext(runtimeID, executionID, reply string, taskLines, factLines []string, now time.Time) string {
	sections := []string{
		"# Primary Context",
		"",
		fmt.Sprintf("- runtime: `%s`", runtimeID),
		fmt.Sprintf("- execution: `%s`", executionID),
		fmt.Sprintf("- updated_at_unix: `%d`", now.Unix()),
		"- preferred_worker: `claude_code`",
		"",
		"## Latest Reply",
		"",
		strings.TrimSpace(reply),
		"",
		"## Open Threads",
		"",
		strings.Join(taskLines, "\n"),
		"",
		"## Long Term Facts",
		"",
		strings.Join(factLines, "\n"),
		"",
	}
	return strings.Join(sections, "\n")
}
