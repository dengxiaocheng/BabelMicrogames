package collabmcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaimScopeRejectsDifferentOwner(t *testing.T) {
	store := NewStore(t.TempDir())
	first, err := store.ClaimScope(ClaimScopeInput{
		SessionID: "online",
		Repo:      "babel-runtime",
		Scope:     "internal/kernel",
	})
	if err != nil {
		t.Fatalf("ClaimScope returned error: %v", err)
	}
	if !first.OK {
		t.Fatalf("expected initial claim to succeed: %+v", first)
	}

	second, err := store.ClaimScope(ClaimScopeInput{
		SessionID: "babel-cpp",
		Repo:      "Babel",
		Scope:     "internal/kernel",
	})
	if err != nil {
		t.Fatalf("ClaimScope returned error: %v", err)
	}
	if second.OK {
		t.Fatalf("expected conflicting claim to fail: %+v", second)
	}
	if second.Conflicting == nil || second.Conflicting.SessionID != "online" {
		t.Fatalf("unexpected conflicting claim: %+v", second.Conflicting)
	}
}

func TestPublishAndAckHandoff(t *testing.T) {
	store := NewStore(t.TempDir())
	published, err := store.PublishHandoff(PublishHandoffInput{
		FromSessionID: "online",
		ToSessionID:   "babel-cpp",
		Repo:          "Babel",
		Title:         "同步 C++ surface",
		Summary:       "请按当前桥接协议重构 deterministic core。",
	})
	if err != nil {
		t.Fatalf("PublishHandoff returned error: %v", err)
	}
	if !published.OK || published.Handoff == nil {
		t.Fatalf("unexpected publish result: %+v", published)
	}

	view, err := store.Snapshot(ReadStateInput{SessionID: "babel-cpp"})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.PendingHandoffs) != 1 || view.PendingHandoffs[0].ID != published.Handoff.ID {
		t.Fatalf("unexpected pending handoffs: %+v", view.PendingHandoffs)
	}

	acked, err := store.AckHandoff(AckHandoffInput{
		SessionID: "babel-cpp",
		HandoffID: published.Handoff.ID,
	})
	if err != nil {
		t.Fatalf("AckHandoff returned error: %v", err)
	}
	if !acked.OK || acked.Handoff == nil || acked.Handoff.Status != "acked" {
		t.Fatalf("unexpected ack result: %+v", acked)
	}
}

func TestSetContractAndSnapshotExposeProgress(t *testing.T) {
	store := NewStore(t.TempDir())
	contract, err := store.SetContract(SetContractInput{
		SessionID:       "online",
		ContractID:      "go-cpp-boundary-v1",
		Summary:         "Go 负责 orchestration，C++ 负责 deterministic core。",
		GoSurfaces:      []string{"kernel", "repository"},
		CppSurfaces:     []string{"time_core", "settlement_core"},
		SharedProtocols: []string{"execution io", "deterministic state schema"},
	})
	if err != nil {
		t.Fatalf("SetContract returned error: %v", err)
	}
	if !contract.OK || contract.Contract == nil {
		t.Fatalf("unexpected contract result: %+v", contract)
	}

	progress, err := store.ReportProgress(ReportProgressInput{
		SessionID:    "online",
		Repo:         "babel-runtime",
		Stage:        "collab-mcp",
		Summary:      "已经落地最小 MCP 服务端。",
		ChangedPaths: []string{"cmd/babel-collab-mcp", "internal/ops/collabmcp"},
	})
	if err != nil {
		t.Fatalf("ReportProgress returned error: %v", err)
	}
	if !progress.OK || progress.Progress == nil {
		t.Fatalf("unexpected progress result: %+v", progress)
	}

	view, err := store.Snapshot(ReadStateInput{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if view.Contract == nil || view.Contract.ContractID != "go-cpp-boundary-v1" {
		t.Fatalf("unexpected contract in view: %+v", view.Contract)
	}
	if len(view.RecentProgress) != 1 || view.RecentProgress[0].Stage != "collab-mcp" {
		t.Fatalf("unexpected progress view: %+v", view.RecentProgress)
	}
}

func TestPublishArtifactAndResolveLatest(t *testing.T) {
	store := NewStore(t.TempDir())
	published, err := store.PublishArtifact(PublishArtifactInput{
		SessionID: "babel-cpp",
		Repo:      "Babel",
		Kind:      "scene_host_library",
		Path:      "/tmp/libbabel_scene_core.so",
		Summary:   "最小共享库",
		Commit:    "abc1234",
	})
	if err != nil {
		t.Fatalf("PublishArtifact returned error: %v", err)
	}
	if !published.OK || published.Artifact == nil {
		t.Fatalf("unexpected artifact result: %+v", published)
	}

	artifact, ok, err := store.LatestArtifact("scene_host_library")
	if err != nil {
		t.Fatalf("LatestArtifact returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected latest artifact")
	}
	if artifact.Path != "/tmp/libbabel_scene_core.so" {
		t.Fatalf("unexpected artifact path: %+v", artifact)
	}

	view, err := store.Snapshot(ReadStateInput{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if len(view.RecentArtifacts) != 1 || view.RecentArtifacts[0].Kind != "scene_host_library" {
		t.Fatalf("unexpected artifact view: %+v", view.RecentArtifacts)
	}
}

func TestAppendCollabEventDispatchesOptionalHook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	previous := os.Getenv(defaultEventHookEnv)
	if err := os.Setenv(defaultEventHookEnv, "cat >/dev/null"); err != nil {
		t.Fatalf("Setenv returned error: %v", err)
	}
	defer func() {
		if previous == "" {
			_ = os.Unsetenv(defaultEventHookEnv)
		} else {
			_ = os.Setenv(defaultEventHookEnv, previous)
		}
	}()

	appendCollabEvent(path, "session_heartbeat", map[string]any{"session_id": "online"})
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(payload), "session_heartbeat") {
		t.Fatalf("unexpected event payload: %s", string(payload))
	}
}
