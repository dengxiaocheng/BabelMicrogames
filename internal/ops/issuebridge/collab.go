package issuebridge

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"babel-runtime/internal/ops/collabmcp"
)

func syncOnlineCollabHeartbeat(workdir, threadID, status, note string) error {
	repo := filepath.Base(workdir)
	if strings.TrimSpace(repo) == "" {
		repo = collabmcp.DefaultOnlineRepo
	}
	_, err := collabmcp.NewStore("").Heartbeat(collabmcp.HeartbeatInput{
		SessionID: collabmcp.DefaultOnlineSessionID,
		Repo:      repo,
		Role:      collabmcp.DefaultOnlineRole,
		Status:    strings.TrimSpace(status),
		Note:      strings.TrimSpace(note),
		ThreadID:  strings.TrimSpace(threadID),
		Commit:    gitHeadCommit(workdir),
	})
	return err
}

func syncOnlineCollabProgress(workdir, stage, summary string, changedPaths []string) error {
	repo := filepath.Base(workdir)
	if strings.TrimSpace(repo) == "" {
		repo = collabmcp.DefaultOnlineRepo
	}
	_, err := collabmcp.NewStore("").ReportProgress(collabmcp.ReportProgressInput{
		SessionID:    collabmcp.DefaultOnlineSessionID,
		Repo:         repo,
		Stage:        strings.TrimSpace(stage),
		Summary:      strings.TrimSpace(summary),
		ChangedPaths: changedPaths,
		Commit:       gitHeadCommit(workdir),
	})
	return err
}

func appendCollabSyncFailure(eventsPath, operation string, err error, fields map[string]any) {
	if err == nil {
		return
	}
	payload := map[string]any{
		"operation": operation,
		"error":     err.Error(),
	}
	for key, value := range fields {
		payload[key] = value
	}
	_, _ = appendEvent(eventsPath, "collab_sync_failed", true, payload)
}

func collabWaitingNote(issueNumber int, title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Sprintf("stage issue #%d waiting", issueNumber)
	}
	return fmt.Sprintf("stage issue #%d waiting: %s", issueNumber, title)
}

func gitHeadCommit(workdir string) string {
	if strings.TrimSpace(workdir) == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", workdir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
