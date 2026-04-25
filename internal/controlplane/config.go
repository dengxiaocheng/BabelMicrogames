package controlplane

import "sync"

type RuntimeConfig struct {
	Version                string `json:"version"`
	Environment            string `json:"environment"`
	FreeChatEnabled        bool   `json:"free_chat_enabled"`
	ProjectConsultEnabled  bool   `json:"project_consult_enabled"`
	MultiplayerEnabled     bool   `json:"multiplayer_enabled"`
	HotReloadEnabled       bool   `json:"hot_reload_enabled"`
	DefaultSoloPromptPack  string `json:"default_solo_prompt_pack"`
	DefaultRoomPromptPack  string `json:"default_room_prompt_pack"`
}

type ConfigStore struct {
	mu     sync.RWMutex
	config RuntimeConfig
}

func NewConfigStore(config RuntimeConfig) *ConfigStore {
	return &ConfigStore{config: config}
}

func (s *ConfigStore) Snapshot() RuntimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *ConfigStore) Replace(config RuntimeConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}
