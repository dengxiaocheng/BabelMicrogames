package controlplane

import (
	"context"
	"fmt"
	"strings"
)

type AdminCommand string

const (
	CommandHealth AdminCommand = "health"
	CommandConfig AdminCommand = "config"
	CommandReload AdminCommand = "reload_config"
	CommandUnknown AdminCommand = "unknown"
)

type AdminResponse struct {
	Command AdminCommand `json:"command"`
	Text    string       `json:"text"`
}

type AdminService struct {
	Health interface {
		Snapshot(ctx context.Context) (HealthStatus, error)
	}
	Config interface {
		Snapshot() RuntimeConfig
	}
}

func ParseAdminCommand(text string) AdminCommand {
	normalized := strings.TrimSpace(strings.ToLower(text))
	switch normalized {
	case "/health", "health":
		return CommandHealth
	case "/config", "config":
		return CommandConfig
	case "/reload_config", "reload_config":
		return CommandReload
	default:
		return CommandUnknown
	}
}

func (s AdminService) Handle(ctx context.Context, text string) (AdminResponse, error) {
	cmd := ParseAdminCommand(text)
	switch cmd {
	case CommandHealth:
		if s.Health == nil {
			return AdminResponse{}, fmt.Errorf("missing health service")
		}
		status, err := s.Health.Snapshot(ctx)
		if err != nil {
			return AdminResponse{}, err
		}
		return AdminResponse{
			Command: cmd,
			Text:    fmt.Sprintf("ok=%t version=%s env=%s solo=%d rooms=%d stale=%d", status.OK, status.ConfigVersion, status.Environment, status.SoloSessions, status.Rooms, status.StaleCheckpoints),
		}, nil
	case CommandConfig:
		if s.Config == nil {
			return AdminResponse{}, fmt.Errorf("missing config store")
		}
		cfg := s.Config.Snapshot()
		return AdminResponse{
			Command: cmd,
			Text:    fmt.Sprintf("version=%s free_chat=%t project_consult=%t multiplayer=%t", cfg.Version, cfg.FreeChatEnabled, cfg.ProjectConsultEnabled, cfg.MultiplayerEnabled),
		}, nil
	case CommandReload:
		return AdminResponse{
			Command: cmd,
			Text:    "reload requested",
		}, nil
	default:
		return AdminResponse{
			Command: CommandUnknown,
			Text:    "unknown admin command",
		}, nil
	}
}
