package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderReportMarksSuccess(t *testing.T) {
	report := RenderGuardReport([]GuardEntry{
		{Name: "check_docs_consistency", Status: "ok", LogPath: ".ci-artifacts/check_docs_consistency.log"},
		{Name: "check_requirement_assets", Status: "ok", LogPath: ".ci-artifacts/check_requirement_assets.log"},
	}, "docs-sync-guard report")
	if !strings.Contains(report, "overall: `success`") {
		t.Fatalf("expected success report, got %q", report)
	}
	if !strings.Contains(report, "`check_requirement_assets`") {
		t.Fatalf("expected check entry in report, got %q", report)
	}
}

func TestLoadEntriesReadsJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.jsonl")
	if err := os.WriteFile(path, []byte("{\"name\":\"check_docs_sync_guard\",\"status\":\"failed\",\"log_path\":\".ci-artifacts/check_docs_sync_guard.log\"}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entries, err := LoadGuardEntries(path)
	if err != nil {
		t.Fatalf("LoadGuardEntries returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Status != "failed" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
