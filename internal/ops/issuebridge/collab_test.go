package issuebridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"babel-runtime/internal/ops/collabmcp"
)

func TestSyncOnlineCollabHeartbeat(t *testing.T) {
	stateDir := t.TempDir()
	previous := os.Getenv("BABEL_COLLAB_STATE_DIR")
	if err := os.Setenv("BABEL_COLLAB_STATE_DIR", stateDir); err != nil {
		t.Fatalf("Setenv returned error: %v", err)
	}
	defer func() {
		if previous == "" {
			_ = os.Unsetenv("BABEL_COLLAB_STATE_DIR")
		} else {
			_ = os.Setenv("BABEL_COLLAB_STATE_DIR", previous)
		}
	}()

	workdir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("Abs returned error: %v", err)
	}
	if err := syncOnlineCollabHeartbeat(workdir, "thread-1", "waiting", "stage issue opened"); err != nil {
		t.Fatalf("syncOnlineCollabHeartbeat returned error: %v", err)
	}

	view, err := collabmcp.NewStore(stateDir).Snapshot(collabmcp.ReadStateInput{SessionID: collabmcp.DefaultOnlineSessionID})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.Sessions) != 1 {
		t.Fatalf("expected one session, got %+v", view.Sessions)
	}
	session := view.Sessions[0]
	if session.SessionID != collabmcp.DefaultOnlineSessionID {
		t.Fatalf("unexpected session id: %+v", session)
	}
	if session.Status != "waiting" || session.LastThreadID != "thread-1" {
		t.Fatalf("unexpected session state: %+v", session)
	}
	if strings.TrimSpace(session.LastCommit) == "" {
		t.Fatalf("unexpected commit: %q", session.LastCommit)
	}
}
