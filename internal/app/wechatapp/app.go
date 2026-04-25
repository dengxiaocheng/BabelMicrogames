package wechatapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"babel-runtime/internal/agent"
	"babel-runtime/internal/corehost"
	"babel-runtime/internal/delivery"
	"babel-runtime/internal/gateway/wechat"
	"babel-runtime/internal/kernel"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/projection"
	"babel-runtime/internal/repository"
	"babel-runtime/internal/requirementregistry"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

type Config struct {
	Token                  string
	MemoryRoot             string
	SceneCoreLibraryPath   string
	SceneCoreLibrarySource string
	MaxResumeSteps         int
	Now                    func() time.Time
}

type SceneHostStatus struct {
	Mode            string
	Source          string
	LibraryPath     string
	Verified        bool
	ContractVersion string
}

func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("missing wechat token")
	}

	repo := repository.NewMemoryRepository()
	sceneHost, sceneHostStatus, err := buildSceneHost(cfg.SceneCoreLibraryPath, cfg.SceneCoreLibrarySource)
	if err != nil {
		return nil, err
	}
	router, err := mode.NewStaticRouter(
		mode.FreeChatModule{Now: cfg.Now},
		mode.ProjectConsultModule{Now: cfg.Now},
		mode.SoloSceneModule{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{
					TimeCore: timecore.SimpleCore{},
				},
			},
			Host: sceneHost,
			Now:  cfg.Now,
		},
		mode.RoomSceneModule{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{
					TimeCore: timecore.SimpleCore{},
				},
			},
			Host: sceneHost,
			Now:  cfg.Now,
		},
	)
	if err != nil {
		return nil, err
	}

	engine := kernel.SimpleEngine{
		Repo:       repo,
		Router:     router,
		Supervisor: &agent.SimpleSupervisor{MemoryRoot: cfg.MemoryRoot},
		Projector:  projection.SimpleProjector{DefaultTransport: "wechat", Now: cfg.Now},
		Dispatcher: delivery.QueueDispatcher{Now: cfg.Now},
		Requirements: requirementregistry.FilesystemRegistry{
			RepoRoot: repoRoot(),
		},
		Now: cfg.Now,
	}
	bridge := wechat.RuntimeBridge{
		Repo:           repo,
		Kernel:         engine,
		MemoryRoot:     cfg.MemoryRoot,
		MaxResumeSteps: cfg.MaxResumeSteps,
		Now:            cfg.Now,
	}
	sessions := wechat.NewMemorySessionModeStore()
	wechatHandler := &wechat.Handler{
		Token:    cfg.Token,
		FreeChat: bridge,
		Consult:  bridge,
		Solo:     bridge,
		Room:     bridge,
		Feedback: wechat.FileFeedbackResponder{Root: filepath.Join(cfg.MemoryRoot, "feedback"), Now: cfg.Now},
		Sessions: sessions,
		Now:      cfg.Now,
	}

	mux := http.NewServeMux()
	mux.Handle("/wechat", wechatHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                     true,
			"service":                "babel-wechatd",
			"time":                   now(cfg.Now).Unix(),
			"scene_host_mode":        sceneHostStatus.Mode,
			"scene_host_source":      sceneHostStatus.Source,
			"scene_host_verified":    sceneHostStatus.Verified,
			"scene_host_library":     sceneHostStatus.LibraryPath,
			"scene_host_contract":    sceneHostStatus.ContractVersion,
			"scene_host_integration": sceneHostStatus.Mode != "",
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	return mux, nil
}

func NewServer(cfg Config) (*http.Server, error) {
	handler, err := NewHandler(cfg)
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}, nil
}

func Health(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func now(fn func() time.Time) time.Time {
	if fn != nil {
		return fn()
	}
	return time.Now()
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func buildSceneHost(sharedLibraryPath, source string) (corehost.SceneHost, SceneHostStatus, error) {
	if sharedLibraryPath == "" {
		return nil, SceneHostStatus{
			Mode:            "local_fallback",
			Source:          defaultSceneHostSource(source),
			Verified:        true,
			ContractVersion: corehost.SceneHostContractVersion,
		}, nil
	}
	if err := corehost.VerifyFixtureLibrary(context.Background(), sharedLibraryPath); err != nil {
		return nil, SceneHostStatus{}, fmt.Errorf("verify scene host library: %w", err)
	}
	host, err := corehost.NewSharedLibrarySceneHost(sharedLibraryPath)
	if err != nil {
		return nil, SceneHostStatus{}, fmt.Errorf("build scene host: %w", err)
	}
	return host, SceneHostStatus{
		Mode:            "shared_library",
		Source:          defaultSceneHostSource(source),
		LibraryPath:     sharedLibraryPath,
		Verified:        true,
		ContractVersion: corehost.SceneHostContractVersion,
	}, nil
}

func defaultSceneHostSource(source string) string {
	if source == "" {
		return "local_default"
	}
	return source
}
