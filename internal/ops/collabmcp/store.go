package collabmcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	stateSchemaVersion    = 1
	defaultStateDirEnv    = "BABEL_COLLAB_STATE_DIR"
	defaultStateDirSuffix = ".codex-runtime/collab"
	defaultEventHookEnv   = "BABEL_COLLAB_EVENT_HOOK"
	stateFileName         = "state.json"
	lockFileName          = "lock"
	eventsFileName        = "events.jsonl"
	maxHandoffs           = 100
	maxProgressEntries    = 100
	maxArtifacts          = 100
)

type Store struct {
	StateDir string
}

type PersistentState struct {
	SchemaVersion int                     `json:"schema_version"`
	UpdatedAtUTC  string                  `json:"updated_at_utc"`
	Contract      *ContractState          `json:"contract,omitempty"`
	Sessions      map[string]SessionState `json:"sessions,omitempty"`
	Claims        []ScopeClaim            `json:"claims,omitempty"`
	Handoffs      []Handoff               `json:"handoffs,omitempty"`
	Progress      []ProgressEntry         `json:"progress,omitempty"`
	Artifacts     []ArtifactEntry         `json:"artifacts,omitempty"`
}

type ContractState struct {
	SchemaVersion   int      `json:"schema_version"`
	ContractID      string   `json:"contract_id"`
	Summary         string   `json:"summary"`
	GoSurfaces      []string `json:"go_surfaces,omitempty"`
	CppSurfaces     []string `json:"cpp_surfaces,omitempty"`
	SharedProtocols []string `json:"shared_protocols,omitempty"`
	RequiredReads   []string `json:"required_reads,omitempty"`
	SourceRefs      []string `json:"source_refs,omitempty"`
	UpdatedBy       string   `json:"updated_by"`
	UpdatedAtUTC    string   `json:"updated_at_utc"`
}

type SessionState struct {
	SchemaVersion      int      `json:"schema_version"`
	SessionID          string   `json:"session_id"`
	Repo               string   `json:"repo,omitempty"`
	Role               string   `json:"role,omitempty"`
	Status             string   `json:"status,omitempty"`
	Note               string   `json:"note,omitempty"`
	LastThreadID       string   `json:"last_thread_id,omitempty"`
	LastCommit         string   `json:"last_commit,omitempty"`
	LastHeartbeatAtUTC string   `json:"last_heartbeat_at_utc"`
	ClaimedScopes      []string `json:"claimed_scopes,omitempty"`
}

type ScopeClaim struct {
	Scope        string `json:"scope"`
	SessionID    string `json:"session_id"`
	Repo         string `json:"repo,omitempty"`
	Note         string `json:"note,omitempty"`
	ClaimedAtUTC string `json:"claimed_at_utc"`
	ExpiresAtUTC string `json:"expires_at_utc,omitempty"`
}

type Handoff struct {
	ID             string   `json:"id"`
	FromSessionID  string   `json:"from_session_id"`
	ToSessionID    string   `json:"to_session_id,omitempty"`
	Repo           string   `json:"repo,omitempty"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	RequiredReads  []string `json:"required_reads,omitempty"`
	ChangedPaths   []string `json:"changed_paths,omitempty"`
	Commit         string   `json:"commit,omitempty"`
	Status         string   `json:"status"`
	CreatedAtUTC   string   `json:"created_at_utc"`
	AckedBySession string   `json:"acked_by_session,omitempty"`
	AckedAtUTC     string   `json:"acked_at_utc,omitempty"`
}

type ProgressEntry struct {
	ID            string   `json:"id"`
	SessionID     string   `json:"session_id"`
	Repo          string   `json:"repo,omitempty"`
	Stage         string   `json:"stage,omitempty"`
	Summary       string   `json:"summary"`
	ChangedPaths  []string `json:"changed_paths,omitempty"`
	Commit        string   `json:"commit,omitempty"`
	ReportedAtUTC string   `json:"reported_at_utc"`
}

type ArtifactEntry struct {
	ID             string `json:"id"`
	SessionID      string `json:"session_id"`
	Repo           string `json:"repo,omitempty"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
	Summary        string `json:"summary,omitempty"`
	Commit         string `json:"commit,omitempty"`
	PublishedAtUTC string `json:"published_at_utc"`
}

type StateView struct {
	SchemaVersion   int             `json:"schema_version"`
	StateDir        string          `json:"state_dir"`
	UpdatedAtUTC    string          `json:"updated_at_utc,omitempty"`
	Contract        *ContractState  `json:"contract,omitempty"`
	Sessions        []SessionState  `json:"sessions,omitempty"`
	Claims          []ScopeClaim    `json:"claims,omitempty"`
	PendingHandoffs []Handoff       `json:"pending_handoffs,omitempty"`
	RecentHandoffs  []Handoff       `json:"recent_handoffs,omitempty"`
	RecentProgress  []ProgressEntry `json:"recent_progress,omitempty"`
	RecentArtifacts []ArtifactEntry `json:"recent_artifacts,omitempty"`
}

type SetContractInput struct {
	SessionID       string   `json:"session_id"`
	ContractID      string   `json:"contract_id"`
	Summary         string   `json:"summary"`
	GoSurfaces      []string `json:"go_surfaces"`
	CppSurfaces     []string `json:"cpp_surfaces"`
	SharedProtocols []string `json:"shared_protocols"`
	RequiredReads   []string `json:"required_reads"`
	SourceRefs      []string `json:"source_refs"`
}

type HeartbeatInput struct {
	SessionID string   `json:"session_id"`
	Repo      string   `json:"repo"`
	Role      string   `json:"role"`
	Status    string   `json:"status"`
	Note      string   `json:"note"`
	ThreadID  string   `json:"thread_id"`
	Commit    string   `json:"commit"`
	Scopes    []string `json:"scopes"`
}

type ClaimScopeInput struct {
	SessionID  string `json:"session_id"`
	Repo       string `json:"repo"`
	Scope      string `json:"scope"`
	Note       string `json:"note"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type ReleaseScopeInput struct {
	SessionID string `json:"session_id"`
	Scope     string `json:"scope"`
}

type ReportProgressInput struct {
	SessionID    string   `json:"session_id"`
	Repo         string   `json:"repo"`
	Stage        string   `json:"stage"`
	Summary      string   `json:"summary"`
	ChangedPaths []string `json:"changed_paths"`
	Commit       string   `json:"commit"`
}

type PublishArtifactInput struct {
	SessionID string `json:"session_id"`
	Repo      string `json:"repo"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Summary   string `json:"summary"`
	Commit    string `json:"commit"`
}

type PublishHandoffInput struct {
	FromSessionID string   `json:"from_session_id"`
	ToSessionID   string   `json:"to_session_id"`
	Repo          string   `json:"repo"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	RequiredReads []string `json:"required_reads"`
	ChangedPaths  []string `json:"changed_paths"`
	Commit        string   `json:"commit"`
}

type AckHandoffInput struct {
	SessionID string `json:"session_id"`
	HandoffID string `json:"handoff_id"`
}

type ReadStateInput struct {
	SessionID string `json:"session_id"`
}

type MutationResult struct {
	OK          bool           `json:"ok"`
	Message     string         `json:"message"`
	Contract    *ContractState `json:"contract,omitempty"`
	Session     *SessionState  `json:"session,omitempty"`
	Claim       *ScopeClaim    `json:"claim,omitempty"`
	Handoff     *Handoff       `json:"handoff,omitempty"`
	Progress    *ProgressEntry `json:"progress,omitempty"`
	Artifact    *ArtifactEntry `json:"artifact,omitempty"`
	Conflicting *ScopeClaim    `json:"conflicting_claim,omitempty"`
	View        *StateView     `json:"state,omitempty"`
}

func DefaultStateDir() string {
	if explicit := strings.TrimSpace(os.Getenv(defaultStateDirEnv)); explicit != "" {
		return explicit
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, defaultStateDirSuffix)
	}
	return defaultStateDirSuffix
}

func NewStore(stateDir string) Store {
	if strings.TrimSpace(stateDir) == "" {
		stateDir = DefaultStateDir()
	}
	return Store{StateDir: stateDir}
}

func (s Store) statePath() string {
	return filepath.Join(s.StateDir, stateFileName)
}

func (s Store) lockPath() string {
	return filepath.Join(s.StateDir, lockFileName)
}

func (s Store) eventsPath() string {
	return filepath.Join(s.StateDir, eventsFileName)
}

func (s Store) Snapshot(input ReadStateInput) (StateView, error) {
	var view StateView
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		view = renderStateView(s.StateDir, *state, input.SessionID)
		return false, nil
	})
	return view, err
}

func (s Store) SetContract(input SetContractInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" {
			result = MutationResult{OK: false, Message: "session_id 不能为空"}
			return false, nil
		}
		if strings.TrimSpace(input.ContractID) == "" {
			result = MutationResult{OK: false, Message: "contract_id 不能为空"}
			return false, nil
		}
		if strings.TrimSpace(input.Summary) == "" {
			result = MutationResult{OK: false, Message: "summary 不能为空"}
			return false, nil
		}
		now := nowUTC()
		contract := &ContractState{
			SchemaVersion:   stateSchemaVersion,
			ContractID:      strings.TrimSpace(input.ContractID),
			Summary:         strings.TrimSpace(input.Summary),
			GoSurfaces:      normalizeSlice(input.GoSurfaces),
			CppSurfaces:     normalizeSlice(input.CppSurfaces),
			SharedProtocols: normalizeSlice(input.SharedProtocols),
			RequiredReads:   normalizeSlice(input.RequiredReads),
			SourceRefs:      normalizeSlice(input.SourceRefs),
			UpdatedBy:       strings.TrimSpace(input.SessionID),
			UpdatedAtUTC:    now,
		}
		state.Contract = contract
		state.UpdatedAtUTC = now
		appendCollabEvent(s.eventsPath(), "contract_updated", map[string]any{
			"session_id":  input.SessionID,
			"contract_id": contract.ContractID,
		})
		view := renderStateView(s.StateDir, *state, input.SessionID)
		result = MutationResult{OK: true, Message: "contract 已更新", Contract: contract, View: &view}
		return true, nil
	})
	return result, err
}

func (s Store) Heartbeat(input HeartbeatInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" {
			result = MutationResult{OK: false, Message: "session_id 不能为空"}
			return false, nil
		}
		session := ensureSession(state, input.SessionID)
		session.Repo = strings.TrimSpace(input.Repo)
		session.Role = strings.TrimSpace(input.Role)
		session.Status = strings.TrimSpace(input.Status)
		session.Note = strings.TrimSpace(input.Note)
		session.LastThreadID = strings.TrimSpace(input.ThreadID)
		session.LastCommit = strings.TrimSpace(input.Commit)
		session.LastHeartbeatAtUTC = nowUTC()
		if len(input.Scopes) > 0 {
			session.ClaimedScopes = normalizeSlice(input.Scopes)
		} else {
			session.ClaimedScopes = claimedScopesForSession(state.Claims, input.SessionID)
		}
		state.Sessions[input.SessionID] = *session
		state.UpdatedAtUTC = session.LastHeartbeatAtUTC
		appendCollabEvent(s.eventsPath(), "session_heartbeat", map[string]any{
			"session_id": input.SessionID,
			"status":     session.Status,
		})
		view := renderStateView(s.StateDir, *state, input.SessionID)
		result = MutationResult{OK: true, Message: "heartbeat 已更新", Session: session, View: &view}
		return true, nil
	})
	return result, err
}

func (s Store) ClaimScope(input ClaimScopeInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.Scope) == "" {
			result = MutationResult{OK: false, Message: "session_id 和 scope 都不能为空"}
			return false, nil
		}
		scope := strings.TrimSpace(input.Scope)
		now := nowUTC()
		for i := range state.Claims {
			if state.Claims[i].Scope != scope {
				continue
			}
			if state.Claims[i].SessionID != input.SessionID {
				conflict := state.Claims[i]
				result = MutationResult{
					OK:          false,
					Message:     fmt.Sprintf("scope %q 已被 %s 占用", scope, conflict.SessionID),
					Conflicting: &conflict,
				}
				return false, nil
			}
			state.Claims[i].Repo = strings.TrimSpace(input.Repo)
			state.Claims[i].Note = strings.TrimSpace(input.Note)
			state.Claims[i].ClaimedAtUTC = now
			state.Claims[i].ExpiresAtUTC = expiresAt(now, input.TTLSeconds)
			rebuildSessionClaims(state)
			appendCollabEvent(s.eventsPath(), "scope_claim_refreshed", map[string]any{
				"session_id": input.SessionID,
				"scope":      scope,
			})
			view := renderStateView(s.StateDir, *state, input.SessionID)
			claim := state.Claims[i]
			result = MutationResult{OK: true, Message: "scope 认领已刷新", Claim: &claim, View: &view}
			state.UpdatedAtUTC = now
			return true, nil
		}
		claim := ScopeClaim{
			Scope:        scope,
			SessionID:    strings.TrimSpace(input.SessionID),
			Repo:         strings.TrimSpace(input.Repo),
			Note:         strings.TrimSpace(input.Note),
			ClaimedAtUTC: now,
			ExpiresAtUTC: expiresAt(now, input.TTLSeconds),
		}
		state.Claims = append(state.Claims, claim)
		rebuildSessionClaims(state)
		state.UpdatedAtUTC = now
		appendCollabEvent(s.eventsPath(), "scope_claimed", map[string]any{
			"session_id": input.SessionID,
			"scope":      scope,
		})
		view := renderStateView(s.StateDir, *state, input.SessionID)
		result = MutationResult{OK: true, Message: "scope 已认领", Claim: &claim, View: &view}
		return true, nil
	})
	return result, err
}

func (s Store) ReleaseScope(input ReleaseScopeInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.Scope) == "" {
			result = MutationResult{OK: false, Message: "session_id 和 scope 都不能为空"}
			return false, nil
		}
		scope := strings.TrimSpace(input.Scope)
		now := nowUTC()
		for i := range state.Claims {
			if state.Claims[i].Scope != scope || state.Claims[i].SessionID != input.SessionID {
				continue
			}
			released := state.Claims[i]
			state.Claims = append(state.Claims[:i], state.Claims[i+1:]...)
			rebuildSessionClaims(state)
			state.UpdatedAtUTC = now
			appendCollabEvent(s.eventsPath(), "scope_released", map[string]any{
				"session_id": input.SessionID,
				"scope":      scope,
			})
			view := renderStateView(s.StateDir, *state, input.SessionID)
			result = MutationResult{OK: true, Message: "scope 已释放", Claim: &released, View: &view}
			return true, nil
		}
		result = MutationResult{OK: false, Message: fmt.Sprintf("session %s 未持有 scope %q", input.SessionID, scope)}
		return false, nil
	})
	return result, err
}

func (s Store) ReportProgress(input ReportProgressInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.Summary) == "" {
			result = MutationResult{OK: false, Message: "session_id 和 summary 都不能为空"}
			return false, nil
		}
		now := nowUTC()
		entry := ProgressEntry{
			ID:            makeEntryID("progress"),
			SessionID:     strings.TrimSpace(input.SessionID),
			Repo:          strings.TrimSpace(input.Repo),
			Stage:         strings.TrimSpace(input.Stage),
			Summary:       strings.TrimSpace(input.Summary),
			ChangedPaths:  normalizeSlice(input.ChangedPaths),
			Commit:        strings.TrimSpace(input.Commit),
			ReportedAtUTC: now,
		}
		state.Progress = append(state.Progress, entry)
		if len(state.Progress) > maxProgressEntries {
			state.Progress = append([]ProgressEntry(nil), state.Progress[len(state.Progress)-maxProgressEntries:]...)
		}
		session := ensureSession(state, input.SessionID)
		session.Repo = strings.TrimSpace(input.Repo)
		if session.Status == "" {
			session.Status = "active"
		}
		session.LastCommit = strings.TrimSpace(input.Commit)
		session.LastHeartbeatAtUTC = now
		state.Sessions[input.SessionID] = *session
		state.UpdatedAtUTC = now
		appendCollabEvent(s.eventsPath(), "progress_reported", map[string]any{
			"session_id": input.SessionID,
			"stage":      entry.Stage,
		})
		view := renderStateView(s.StateDir, *state, input.SessionID)
		result = MutationResult{OK: true, Message: "progress 已记录", Progress: &entry, View: &view}
		return true, nil
	})
	return result, err
}

func (s Store) PublishArtifact(input PublishArtifactInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.Kind) == "" || strings.TrimSpace(input.Path) == "" {
			result = MutationResult{OK: false, Message: "session_id、kind 和 path 都不能为空"}
			return false, nil
		}
		now := nowUTC()
		entry := ArtifactEntry{
			ID:             makeEntryID("artifact"),
			SessionID:      strings.TrimSpace(input.SessionID),
			Repo:           strings.TrimSpace(input.Repo),
			Kind:           strings.TrimSpace(input.Kind),
			Path:           strings.TrimSpace(input.Path),
			Summary:        strings.TrimSpace(input.Summary),
			Commit:         strings.TrimSpace(input.Commit),
			PublishedAtUTC: now,
		}
		state.Artifacts = append(state.Artifacts, entry)
		if len(state.Artifacts) > maxArtifacts {
			state.Artifacts = append([]ArtifactEntry(nil), state.Artifacts[len(state.Artifacts)-maxArtifacts:]...)
		}
		session := ensureSession(state, input.SessionID)
		if session.Repo == "" {
			session.Repo = strings.TrimSpace(input.Repo)
		}
		session.Status = "artifact-published"
		session.LastCommit = strings.TrimSpace(input.Commit)
		session.LastHeartbeatAtUTC = now
		state.Sessions[input.SessionID] = *session
		state.UpdatedAtUTC = now
		appendCollabEvent(s.eventsPath(), "artifact_published", map[string]any{
			"session_id": input.SessionID,
			"kind":       entry.Kind,
			"path":       entry.Path,
		})
		view := renderStateView(s.StateDir, *state, input.SessionID)
		result = MutationResult{OK: true, Message: "artifact 已发布", Artifact: &entry, View: &view}
		return true, nil
	})
	return result, err
}

func (s Store) LatestArtifact(kind string) (ArtifactEntry, bool, error) {
	view, err := s.Snapshot(ReadStateInput{})
	if err != nil {
		return ArtifactEntry{}, false, err
	}
	for _, artifact := range view.RecentArtifacts {
		if strings.TrimSpace(kind) == "" || artifact.Kind == strings.TrimSpace(kind) {
			return artifact, true, nil
		}
	}
	return ArtifactEntry{}, false, nil
}

func (s Store) PublishHandoff(input PublishHandoffInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.FromSessionID) == "" {
			result = MutationResult{OK: false, Message: "from_session_id 不能为空"}
			return false, nil
		}
		if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Summary) == "" {
			result = MutationResult{OK: false, Message: "title 和 summary 都不能为空"}
			return false, nil
		}
		now := nowUTC()
		handoff := Handoff{
			ID:            makeEntryID("handoff"),
			FromSessionID: strings.TrimSpace(input.FromSessionID),
			ToSessionID:   strings.TrimSpace(input.ToSessionID),
			Repo:          strings.TrimSpace(input.Repo),
			Title:         strings.TrimSpace(input.Title),
			Summary:       strings.TrimSpace(input.Summary),
			RequiredReads: normalizeSlice(input.RequiredReads),
			ChangedPaths:  normalizeSlice(input.ChangedPaths),
			Commit:        strings.TrimSpace(input.Commit),
			Status:        "pending",
			CreatedAtUTC:  now,
		}
		state.Handoffs = append(state.Handoffs, handoff)
		if len(state.Handoffs) > maxHandoffs {
			state.Handoffs = append([]Handoff(nil), state.Handoffs[len(state.Handoffs)-maxHandoffs:]...)
		}
		session := ensureSession(state, input.FromSessionID)
		if session.Repo == "" {
			session.Repo = strings.TrimSpace(input.Repo)
		}
		session.Status = "handoff-published"
		session.LastCommit = strings.TrimSpace(input.Commit)
		session.LastHeartbeatAtUTC = now
		state.Sessions[input.FromSessionID] = *session
		state.UpdatedAtUTC = now
		appendCollabEvent(s.eventsPath(), "handoff_published", map[string]any{
			"handoff_id":      handoff.ID,
			"from_session_id": handoff.FromSessionID,
			"to_session_id":   handoff.ToSessionID,
		})
		view := renderStateView(s.StateDir, *state, input.FromSessionID)
		result = MutationResult{OK: true, Message: "handoff 已发布", Handoff: &handoff, View: &view}
		return true, nil
	})
	return result, err
}

func (s Store) AckHandoff(input AckHandoffInput) (MutationResult, error) {
	var result MutationResult
	err := s.withStateLock(func(state *PersistentState) (bool, error) {
		if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.HandoffID) == "" {
			result = MutationResult{OK: false, Message: "session_id 和 handoff_id 都不能为空"}
			return false, nil
		}
		now := nowUTC()
		for i := len(state.Handoffs) - 1; i >= 0; i-- {
			if state.Handoffs[i].ID != input.HandoffID {
				continue
			}
			if state.Handoffs[i].Status == "acked" {
				handoff := state.Handoffs[i]
				view := renderStateView(s.StateDir, *state, input.SessionID)
				result = MutationResult{OK: true, Message: "handoff 之前已确认", Handoff: &handoff, View: &view}
				return false, nil
			}
			if state.Handoffs[i].ToSessionID != "" && state.Handoffs[i].ToSessionID != input.SessionID {
				result = MutationResult{OK: false, Message: fmt.Sprintf("handoff %s 不是发给 %s", input.HandoffID, input.SessionID)}
				return false, nil
			}
			state.Handoffs[i].Status = "acked"
			state.Handoffs[i].AckedBySession = strings.TrimSpace(input.SessionID)
			state.Handoffs[i].AckedAtUTC = now
			session := ensureSession(state, input.SessionID)
			session.Status = "handoff-acked"
			session.LastHeartbeatAtUTC = now
			state.Sessions[input.SessionID] = *session
			state.UpdatedAtUTC = now
			appendCollabEvent(s.eventsPath(), "handoff_acked", map[string]any{
				"handoff_id": input.HandoffID,
				"session_id": input.SessionID,
			})
			handoff := state.Handoffs[i]
			view := renderStateView(s.StateDir, *state, input.SessionID)
			result = MutationResult{OK: true, Message: "handoff 已确认", Handoff: &handoff, View: &view}
			return true, nil
		}
		result = MutationResult{OK: false, Message: fmt.Sprintf("找不到 handoff %s", input.HandoffID)}
		return false, nil
	})
	return result, err
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func expiresAt(start string, ttlSeconds int) string {
	if ttlSeconds <= 0 {
		return ""
	}
	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return ""
	}
	return startTime.Add(time.Duration(ttlSeconds) * time.Second).UTC().Format(time.RFC3339)
}

func ensureSession(state *PersistentState, sessionID string) *SessionState {
	if state.Sessions == nil {
		state.Sessions = map[string]SessionState{}
	}
	session, ok := state.Sessions[sessionID]
	if !ok {
		session = SessionState{
			SchemaVersion:      stateSchemaVersion,
			SessionID:          sessionID,
			LastHeartbeatAtUTC: nowUTC(),
		}
	}
	session.SchemaVersion = stateSchemaVersion
	return &session
}

func claimedScopesForSession(claims []ScopeClaim, sessionID string) []string {
	scopes := make([]string, 0)
	for _, claim := range claims {
		if claim.SessionID == sessionID {
			scopes = append(scopes, claim.Scope)
		}
	}
	sort.Strings(scopes)
	return scopes
}

func rebuildSessionClaims(state *PersistentState) {
	if state.Sessions == nil {
		return
	}
	for sessionID, session := range state.Sessions {
		session.ClaimedScopes = claimedScopesForSession(state.Claims, sessionID)
		state.Sessions[sessionID] = session
	}
}

func normalizeSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	if len(result) == 0 {
		return nil
	}
	return result
}

func makeEntryID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}

func renderStateView(stateDir string, state PersistentState, sessionID string) StateView {
	view := StateView{
		SchemaVersion: stateSchemaVersion,
		StateDir:      stateDir,
		UpdatedAtUTC:  state.UpdatedAtUTC,
		Contract:      state.Contract,
		Claims:        append([]ScopeClaim(nil), state.Claims...),
	}
	view.Sessions = make([]SessionState, 0, len(state.Sessions))
	for _, session := range state.Sessions {
		view.Sessions = append(view.Sessions, session)
	}
	sort.Slice(view.Sessions, func(i, j int) bool { return view.Sessions[i].SessionID < view.Sessions[j].SessionID })
	sort.Slice(view.Claims, func(i, j int) bool {
		if view.Claims[i].Scope == view.Claims[j].Scope {
			return view.Claims[i].SessionID < view.Claims[j].SessionID
		}
		return view.Claims[i].Scope < view.Claims[j].Scope
	})
	for i := len(state.Handoffs) - 1; i >= 0; i-- {
		handoff := state.Handoffs[i]
		view.RecentHandoffs = append(view.RecentHandoffs, handoff)
		if handoff.Status == "pending" && handoffRelevantToSession(handoff, sessionID) {
			view.PendingHandoffs = append(view.PendingHandoffs, handoff)
		}
	}
	for i := len(state.Progress) - 1; i >= 0; i-- {
		view.RecentProgress = append(view.RecentProgress, state.Progress[i])
	}
	for i := len(state.Artifacts) - 1; i >= 0; i-- {
		view.RecentArtifacts = append(view.RecentArtifacts, state.Artifacts[i])
	}
	return view
}

func handoffRelevantToSession(handoff Handoff, sessionID string) bool {
	if strings.TrimSpace(sessionID) == "" {
		return true
	}
	if handoff.ToSessionID == "" {
		return true
	}
	return handoff.ToSessionID == sessionID
}

func (s Store) withStateLock(fn func(state *PersistentState) (bool, error)) error {
	if err := os.MkdirAll(s.StateDir, 0o755); err != nil {
		return err
	}
	lockFile, err := os.OpenFile(s.lockPath(), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer lockFile.Close()
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	state, err := s.loadState()
	if err != nil {
		return err
	}
	shouldSave, err := fn(&state)
	if err != nil {
		return err
	}
	if !shouldSave {
		return nil
	}
	state.SchemaVersion = stateSchemaVersion
	if strings.TrimSpace(state.UpdatedAtUTC) == "" {
		state.UpdatedAtUTC = nowUTC()
	}
	return s.saveState(state)
}

func (s Store) loadState() (PersistentState, error) {
	payload, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return PersistentState{
				SchemaVersion: stateSchemaVersion,
				Sessions:      map[string]SessionState{},
			}, nil
		}
		return PersistentState{}, err
	}
	var state PersistentState
	if err := json.Unmarshal(payload, &state); err != nil {
		return PersistentState{}, err
	}
	if state.SchemaVersion == 0 {
		state.SchemaVersion = stateSchemaVersion
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionState{}
	}
	return state, nil
}

func (s Store) saveState(state PersistentState) error {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.statePath() + ".tmp"
	if err := os.WriteFile(tmpPath, append(payload, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.statePath())
}

func appendCollabEvent(path, eventType string, fields map[string]any) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	entry := map[string]any{
		"schema_version":  stateSchemaVersion,
		"recorded_at_utc": nowUTC(),
		"event_type":      eventType,
	}
	for key, value := range fields {
		entry[key] = value
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = file.Write(append(raw, '\n'))
	dispatchCollabEventHook(path, entry)
}

func LoadRecentEvents(path string, tail int) ([]map[string]any, error) {
	if tail <= 0 {
		tail = 20
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0, tail)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if len(lines) == tail {
			copy(lines, lines[1:])
			lines[len(lines)-1] = line
		} else {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	events := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("解析事件失败: %w", err)
		}
		events = append(events, event)
	}
	return events, nil
}

func collabEventHookCommand() string {
	return strings.TrimSpace(os.Getenv(defaultEventHookEnv))
}

func dispatchCollabEventHook(eventsPath string, entry map[string]any) {
	hook := collabEventHookCommand()
	if hook == "" {
		return
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-lc", hook)
	cmd.Dir = filepath.Dir(eventsPath)
	cmd.Stdin = strings.NewReader(string(payload))
	output, err := cmd.CombinedOutput()
	if err == nil {
		return
	}
	fields := map[string]any{
		"hook_command":      hook,
		"source_event_type": entry["event_type"],
	}
	if ctx.Err() == context.DeadlineExceeded {
		fields["error"] = ctx.Err().Error()
	} else {
		fields["error"] = err.Error()
	}
	stderr := strings.TrimSpace(string(output))
	if stderr != "" {
		fields["stderr"] = truncateText(stderr, 500)
	}
	appendCollabEventWithoutHook(eventsPath, "event_hook_failed", fields)
}

func appendCollabEventWithoutHook(path, eventType string, fields map[string]any) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	entry := map[string]any{
		"schema_version":  stateSchemaVersion,
		"recorded_at_utc": nowUTC(),
		"event_type":      eventType,
	}
	for key, value := range fields {
		entry[key] = value
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = file.Write(append(raw, '\n'))
}

func truncateText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
