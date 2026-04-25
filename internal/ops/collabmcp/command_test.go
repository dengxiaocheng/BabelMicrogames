package collabmcp

import (
	"bytes"
	"strings"
	"testing"
)

func TestHeartbeatSubcommandWritesState(t *testing.T) {
	stateDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{
		"heartbeat",
		"--state-dir", stateDir,
		"--session-id", DefaultBabelCppSessionID,
		"--repo", DefaultBabelRepo,
		"--role", DefaultBabelCppRole,
		"--status", "manual-active",
		"--thread-id", "thread-1",
		"--note", "entered via termux_m",
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, stderr.String())
	}
	view, err := NewStore(stateDir).Snapshot(ReadStateInput{SessionID: DefaultBabelCppSessionID})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.Sessions) != 1 {
		t.Fatalf("unexpected sessions: %+v", view.Sessions)
	}
	if view.Sessions[0].Status != "manual-active" || view.Sessions[0].Role != DefaultBabelCppRole {
		t.Fatalf("unexpected session state: %+v", view.Sessions[0])
	}
}

func TestPublishAndAckHandoffSubcommands(t *testing.T) {
	stateDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{
		"publish-handoff",
		"--state-dir", stateDir,
		"--from-session-id", DefaultOnlineSessionID,
		"--to-session-id", DefaultBabelCppSessionID,
		"--repo", DefaultBabelRepo,
		"--title", "实现 C++ deterministic core",
		"--summary", "请接手 settlement core 的重构。",
		"--required-read", "docs/operations/COLLAB_MCP.md",
		"--path", "src/",
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("publish-handoff failed: %d stderr=%s", exitCode, stderr.String())
	}
	view, err := NewStore(stateDir).Snapshot(ReadStateInput{SessionID: DefaultBabelCppSessionID})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.PendingHandoffs) != 1 {
		t.Fatalf("unexpected pending handoffs: %+v", view.PendingHandoffs)
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = Main([]string{
		"ack-handoff",
		"--state-dir", stateDir,
		"--session-id", DefaultBabelCppSessionID,
		"--handoff-id", view.PendingHandoffs[0].ID,
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("ack-handoff failed: %d stderr=%s", exitCode, stderr.String())
	}
	view, err = NewStore(stateDir).Snapshot(ReadStateInput{SessionID: DefaultBabelCppSessionID})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.PendingHandoffs) != 0 {
		t.Fatalf("expected no pending handoffs after ack, got %+v", view.PendingHandoffs)
	}
}

func TestPublishArtifactSubcommand(t *testing.T) {
	stateDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{
		"publish-artifact",
		"--state-dir", stateDir,
		"--session-id", DefaultBabelCppSessionID,
		"--repo", DefaultBabelRepo,
		"--kind", "scene_host_library",
		"--path", "/tmp/libbabel_scene_core.so",
		"--summary", "最小共享库",
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("publish-artifact failed: %d stderr=%s", exitCode, stderr.String())
	}
	view, err := NewStore(stateDir).Snapshot(ReadStateInput{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.RecentArtifacts) != 1 || view.RecentArtifacts[0].Path != "/tmp/libbabel_scene_core.so" {
		t.Fatalf("unexpected artifacts: %+v", view.RecentArtifacts)
	}
}
