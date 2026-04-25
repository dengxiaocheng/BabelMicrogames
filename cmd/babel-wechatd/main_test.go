package main

import (
	"os"
	"testing"

	"babel-runtime/internal/ops/collabmcp"
)

func TestResolveSceneCoreLibraryConfigFromCollab(t *testing.T) {
	stateDir := t.TempDir()
	store := collabmcp.NewStore(stateDir)
	_, err := store.PublishArtifact(collabmcp.PublishArtifactInput{
		SessionID: "babel-cpp",
		Repo:      "Babel",
		Kind:      "scene_host_library",
		Path:      "/tmp/libbabel_scene_core.so",
		Summary:   "test artifact",
		Commit:    "abc1234",
	})
	if err != nil {
		t.Fatalf("PublishArtifact returned error: %v", err)
	}

	previousStateDir, hadStateDir := os.LookupEnv("BABEL_COLLAB_STATE_DIR")
	previousPath, hadPath := os.LookupEnv("BABEL_SCENE_CORE_LIBRARY")
	t.Setenv("BABEL_COLLAB_STATE_DIR", stateDir)
	t.Setenv("BABEL_SCENE_CORE_LIBRARY", "@collab")
	defer func() {
		if hadStateDir {
			_ = os.Setenv("BABEL_COLLAB_STATE_DIR", previousStateDir)
		} else {
			_ = os.Unsetenv("BABEL_COLLAB_STATE_DIR")
		}
		if hadPath {
			_ = os.Setenv("BABEL_SCENE_CORE_LIBRARY", previousPath)
		} else {
			_ = os.Unsetenv("BABEL_SCENE_CORE_LIBRARY")
		}
	}()

	path, source, err := resolveSceneCoreLibraryConfig()
	if err != nil {
		t.Fatalf("resolveSceneCoreLibraryConfig returned error: %v", err)
	}
	if path != "/tmp/libbabel_scene_core.so" {
		t.Fatalf("expected collab artifact path, got %q", path)
	}
	if source != "collab_artifact" {
		t.Fatalf("expected collab_artifact source, got %q", source)
	}
}

func TestResolveSceneCoreLibraryConfigExplicitPath(t *testing.T) {
	t.Setenv("BABEL_SCENE_CORE_LIBRARY", "/tmp/libbabel_scene_core.so")
	path, source, err := resolveSceneCoreLibraryConfig()
	if err != nil {
		t.Fatalf("resolveSceneCoreLibraryConfig returned error: %v", err)
	}
	if path != "/tmp/libbabel_scene_core.so" {
		t.Fatalf("expected explicit path, got %q", path)
	}
	if source != "explicit_path" {
		t.Fatalf("expected explicit_path source, got %q", source)
	}
}

func TestResolveSceneCoreLibraryConfigCollabMissingArtifactFails(t *testing.T) {
	t.Setenv("BABEL_COLLAB_STATE_DIR", t.TempDir())
	t.Setenv("BABEL_SCENE_CORE_LIBRARY", "@collab")
	if _, _, err := resolveSceneCoreLibraryConfig(); err == nil {
		t.Fatalf("expected missing collab artifact to fail")
	}
}
