package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"babel-runtime/internal/app/wechatapp"
	"babel-runtime/internal/ops/collabmcp"
)

func main() {
	token := firstNonEmpty(os.Getenv("BABEL_WECHAT_TOKEN"), os.Getenv("WECHAT_TOKEN"))
	if token == "" {
		log.Fatal("missing BABEL_WECHAT_TOKEN or WECHAT_TOKEN")
	}

	port := firstNonEmpty(os.Getenv("BABEL_WECHAT_PORT"), os.Getenv("WECHAT_PORT"))
	if port == "" {
		port = "8080"
	}
	maxResumeSteps := 8
	if raw := os.Getenv("BABEL_WECHAT_MAX_RESUME_STEPS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatalf("invalid BABEL_WECHAT_MAX_RESUME_STEPS: %v", err)
		}
		maxResumeSteps = parsed
	}
	sceneCoreLibrary, sceneCoreSource, err := resolveSceneCoreLibraryConfig()
	if err != nil {
		log.Fatal(err)
	}

	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:                  token,
		MemoryRoot:             firstNonEmpty(os.Getenv("BABEL_WECHAT_MEMORY_ROOT"), ".codex-runtime/wechat-memory"),
		SceneCoreLibraryPath:   sceneCoreLibrary,
		SceneCoreLibrarySource: sceneCoreSource,
		MaxResumeSteps:         maxResumeSteps,
	})
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
	log.Printf("babel-wechatd listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func resolveSceneCoreLibraryConfig() (string, string, error) {
	path := firstNonEmpty(os.Getenv("BABEL_SCENE_CORE_LIBRARY"), os.Getenv("BABEL_CORE_SHARED_LIBRARY"))
	trimmed := strings.TrimSpace(path)
	switch trimmed {
	case "":
		return "", "local_default", nil
	case "@collab":
	default:
		return trimmed, "explicit_path", nil
	}

	artifact, ok, err := collabmcp.NewStore(collabmcp.DefaultStateDir()).LatestArtifact("scene_host_library")
	if err != nil {
		return "", "", fmt.Errorf("resolve scene host library from collab: %w", err)
	}
	if !ok {
		return "", "", fmt.Errorf("resolve scene host library from collab: no scene_host_library artifact found")
	}
	log.Printf("resolved scene host library from collab: %s", artifact.Path)
	return artifact.Path, "collab_artifact", nil
}
