package issuebridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultStateFile           = ".codex-runtime/issue_bridge_state.json"
	DefaultTokenFile           = ".codex-runtime/github-token.env"
	DefaultControlFile         = ".codex-runtime/thread_control.json"
	DefaultLockFile            = ".codex-runtime/issue_bridge.lock"
	DefaultEventsFile          = ".codex-runtime/issue_bridge_events.jsonl"
	DefaultManualLeaseDir      = ".codex-runtime/manual_leases"
	DefaultWorkerRegistryFile  = ".codex-runtime/claudecode_workers.json"
	DefaultWorkerPacketRoot    = ".codex-runtime/claudecode_workers"
	DefaultPollSeconds         = 15
	DefaultWatcherSession      = "codex_issue_watch"
	DefaultResumeSessionPrefix = "codex_issue_resume"
	DefaultAPIBaseURL          = "https://api.github.com"
)

type BridgeState struct {
	SchemaVersion               int    `json:"schema_version"`
	RepoFullName                string `json:"repo_full_name"`
	IssueNumber                 int    `json:"issue_number"`
	IssueURL                    string `json:"issue_url"`
	IssueTitle                  string `json:"issue_title"`
	DecisionRequest             string `json:"decision_request"`
	TerminalHandoff             string `json:"terminal_handoff"`
	HandoffStatus               string `json:"handoff_status,omitempty"`
	HandoffOpenedAtUTC          string `json:"handoff_opened_at_utc,omitempty"`
	HandoffConsumedAtUTC        string `json:"handoff_consumed_at_utc,omitempty"`
	ThreadID                    string `json:"thread_id"`
	Workdir                     string `json:"workdir"`
	OpenedAtUTC                 string `json:"opened_at_utc"`
	LastResumedCommentID        *int64 `json:"last_resumed_comment_id"`
	LastResumedCommentURL       string `json:"last_resumed_comment_url,omitempty"`
	LastResumedAtUTC            string `json:"last_resumed_at_utc,omitempty"`
	LastResumeTmuxSession       string `json:"last_resume_tmux_session,omitempty"`
	WatcherSessionName          string `json:"watcher_session_name,omitempty"`
	ClosedByActiveTerminalAtUTC string `json:"closed_by_active_terminal_at_utc,omitempty"`
}

type ThreadControl struct {
	SchemaVersion                  int    `json:"schema_version"`
	ThreadID                       string `json:"thread_id"`
	Owner                          string `json:"owner"`
	OwnerDetail                    string `json:"owner_detail"`
	ActiveTmuxSession              string `json:"active_tmux_session,omitempty"`
	LastTransitionAtUTC            string `json:"last_transition_at_utc"`
	LastManualClaimAtUTC           string `json:"last_manual_claim_at_utc,omitempty"`
	LastInterruptedAutoTmuxSession string `json:"last_interrupted_auto_tmux_session,omitempty"`
	LastAutoResumeAtUTC            string `json:"last_auto_resume_at_utc,omitempty"`
	LastManualReleaseAtUTC         string `json:"last_manual_release_at_utc,omitempty"`
}

type ManualLease struct {
	SchemaVersion int    `json:"schema_version"`
	ThreadID      string `json:"thread_id"`
	Entrypoint    string `json:"entrypoint"`
	SessionID     string `json:"session_id"`
	UpdatedAtUTC  string `json:"updated_at_utc"`
}

type ClaudeWorkerRegistry struct {
	SchemaVersion int            `json:"schema_version"`
	Workers       []ClaudeWorker `json:"workers"`
}

type ClaudeWorker struct {
	WorkerID         string   `json:"worker_id"`
	Status           string   `json:"status"`
	Lane             string   `json:"lane,omitempty"`
	TaskLevel        string   `json:"task_level,omitempty"`
	MaxFiles         int      `json:"max_files,omitempty"`
	MaxDeltaLines    int      `json:"max_delta_lines,omitempty"`
	ReadScope        []string `json:"read_scope,omitempty"`
	WriteScope       []string `json:"write_scope,omitempty"`
	TestCommands     []string `json:"test_commands,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	Model            string   `json:"model,omitempty"`
	TaskTitle        string   `json:"task_title,omitempty"`
	TaskSummary      string   `json:"task_summary,omitempty"`
	RepoFullName     string   `json:"repo_full_name,omitempty"`
	IssueNumber      int      `json:"issue_number,omitempty"`
	IssueURL         string   `json:"issue_url,omitempty"`
	ManagerThreadID  string   `json:"manager_thread_id,omitempty"`
	Workdir          string   `json:"workdir,omitempty"`
	PacketFile       string   `json:"packet_file,omitempty"`
	ReportFile       string   `json:"report_file,omitempty"`
	LastCommentFile  string   `json:"last_comment_file,omitempty"`
	LastNote         string   `json:"last_note,omitempty"`
	CreatedAtUTC     string   `json:"created_at_utc"`
	StartedAtUTC     string   `json:"started_at_utc,omitempty"`
	FinishedAtUTC    string   `json:"finished_at_utc,omitempty"`
	LastUpdatedAtUTC string   `json:"last_updated_at_utc"`
}

type Issue struct {
	Number   int    `json:"number"`
	HTMLURL  string `json:"html_url"`
	State    string `json:"state"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	ClosedAt string `json:"closed_at"`
}

type Comment struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

type CreateIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Assignees []string `json:"assignees,omitempty"`
}

var tmuxSessionExistsHook = tmuxSessionExists

func nowUTCISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func loadTextArg(direct, filePath string) (string, error) {
	if strings.TrimSpace(direct) != "" && strings.TrimSpace(filePath) != "" {
		return "", fmt.Errorf("只能二选一：直接传文本或传文件路径")
	}
	if strings.TrimSpace(filePath) != "" {
		payload, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(payload)), nil
	}
	if strings.TrimSpace(direct) != "" {
		return strings.TrimSpace(direct), nil
	}
	return "", fmt.Errorf("缺少必填文本参数")
}

func gitRemoteURL(workdir string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = workdir
	payload, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

func parseRepoFullName(remoteURL string) (string, error) {
	remote := strings.TrimSpace(remoteURL)
	switch {
	case strings.HasPrefix(remote, "git@github.com:"):
		remote = strings.TrimPrefix(remote, "git@github.com:")
	case strings.HasPrefix(remote, "https://github.com/"):
		remote = strings.TrimPrefix(remote, "https://github.com/")
	case strings.HasPrefix(remote, "http://github.com/"):
		remote = strings.TrimPrefix(remote, "http://github.com/")
	default:
		return "", fmt.Errorf("无法从 remote 解析 GitHub 仓库：%s", remoteURL)
	}
	remote = strings.TrimSuffix(remote, ".git")
	if strings.Count(remote, "/") != 1 {
		return "", fmt.Errorf("remote 不是 owner/name 形状：%s", remoteURL)
	}
	return remote, nil
}

func getRepoFullName(workdir, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	remoteURL, err := gitRemoteURL(workdir)
	if err != nil {
		return "", err
	}
	return parseRepoFullName(remoteURL)
}

func loadBridgeState(path string) (BridgeState, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return BridgeState{}, false, nil
		}
		return BridgeState{}, false, err
	}
	var state BridgeState
	if err := json.Unmarshal(payload, &state); err != nil {
		return BridgeState{}, false, err
	}
	if state.SchemaVersion == 0 {
		return BridgeState{}, false, nil
	}
	return state, true, nil
}

func saveBridgeState(path string, state BridgeState) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func loadThreadControl(path string) (ThreadControl, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ThreadControl{}, false, nil
		}
		return ThreadControl{}, false, err
	}
	var control ThreadControl
	if err := json.Unmarshal(payload, &control); err != nil {
		return ThreadControl{}, false, err
	}
	if control.SchemaVersion == 0 {
		return ThreadControl{}, false, nil
	}
	return control, true, nil
}

func saveThreadControl(path string, control ThreadControl) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(control, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func loadWorkerRegistry(path string) (ClaudeWorkerRegistry, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ClaudeWorkerRegistry{}, false, nil
		}
		return ClaudeWorkerRegistry{}, false, err
	}
	var registry ClaudeWorkerRegistry
	if err := json.Unmarshal(payload, &registry); err != nil {
		return ClaudeWorkerRegistry{}, false, err
	}
	if registry.SchemaVersion == 0 {
		return ClaudeWorkerRegistry{}, false, nil
	}
	return registry, true, nil
}

func saveWorkerRegistry(path string, registry ClaudeWorkerRegistry) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func manualLeasePath(workdir, leaseDir, entrypoint string) string {
	safeEntrypoint := strings.ReplaceAll(strings.TrimSpace(entrypoint), "/", "_")
	if safeEntrypoint == "" {
		safeEntrypoint = "manual"
	}
	return resolveInWorkdir(workdir, filepath.Join(leaseDir, safeEntrypoint+".json"))
}

func loadManualLease(path string) (ManualLease, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ManualLease{}, false, nil
		}
		return ManualLease{}, false, err
	}
	var lease ManualLease
	if err := json.Unmarshal(payload, &lease); err != nil {
		return ManualLease{}, false, err
	}
	if lease.SchemaVersion == 0 {
		return ManualLease{}, false, nil
	}
	return lease, true, nil
}

func saveManualLease(path string, lease ManualLease) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func eventHookCommand() string {
	return strings.TrimSpace(os.Getenv("BABEL_ISSUE_BRIDGE_EVENT_HOOK"))
}

func appendEvent(path, eventType string, dispatchHook bool, fields map[string]any) (map[string]any, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}
	entry := map[string]any{
		"schema_version":  1,
		"recorded_at_utc": nowUTCISO(),
		"event_type":      eventType,
	}
	for key, value := range fields {
		entry[key] = value
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	payload, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		return nil, err
	}
	if dispatchHook {
		dispatchEventHook(path, entry)
	}
	return entry, nil
}

func loadRecentEvents(path string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		return nil, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	events := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func dispatchEventHook(eventsPath string, entry map[string]any) {
	hook := eventHookCommand()
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
	_, _ = appendEvent(eventsPath, "event_hook_failed", false, fields)
}

func withLock(path string, fn func() error) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}()
	return fn()
}

func tmuxSessionExists(sessionName string) bool {
	if strings.TrimSpace(sessionName) == "" {
		return false
	}
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func killTmuxSession(sessionName string) bool {
	if !tmuxSessionExistsHook(sessionName) {
		return false
	}
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run() == nil
}

type ManualCleanupTarget struct {
	PID  int
	PGID int
	Args string
}

func listManualResumeProcesses() ([]ManualCleanupTarget, error) {
	cmd := exec.Command("ps", "-eo", "pid=,pgid=,args=")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseManualResumeProcesses(strings.NewReader(string(output)))
}

func parseManualResumeProcesses(reader io.Reader) ([]ManualCleanupTarget, error) {
	scanner := bufio.NewScanner(reader)
	var targets []ManualCleanupTarget
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, err
		}
		pgid, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		args := strings.Join(fields[2:], " ")
		targets = append(targets, ManualCleanupTarget{
			PID:  pid,
			PGID: pgid,
			Args: args,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func matchingManualCleanupTargets(processes []ManualCleanupTarget, threadID, entrypoint string, excludePID, excludePGID int) []ManualCleanupTarget {
	threadID = strings.TrimSpace(threadID)
	entrypoint = strings.TrimSpace(entrypoint)
	if entrypoint == "" {
		return nil
	}
	var matched []ManualCleanupTarget
	for _, process := range processes {
		if process.PID <= 0 || process.PID == excludePID {
			continue
		}
		if excludePGID > 0 && process.PGID == excludePGID {
			continue
		}
		if !strings.Contains(process.Args, "manual-resume") {
			continue
		}
		if !strings.Contains(process.Args, "--entrypoint "+entrypoint) {
			continue
		}
		if threadID != "" && !strings.Contains(process.Args, threadID) {
			continue
		}
		matched = append(matched, process)
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].PGID == matched[j].PGID {
			return matched[i].PID < matched[j].PID
		}
		return matched[i].PGID < matched[j].PGID
	})
	return matched
}

func cleanupManualTargets(targets []ManualCleanupTarget) []int {
	pgidSet := make(map[int]struct{})
	var pgids []int
	for _, target := range targets {
		if target.PGID <= 0 {
			continue
		}
		if _, ok := pgidSet[target.PGID]; ok {
			continue
		}
		pgidSet[target.PGID] = struct{}{}
		pgids = append(pgids, target.PGID)
	}
	for _, pgid := range pgids {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	}
	if len(pgids) > 0 {
		time.Sleep(2 * time.Second)
	}
	for _, pgid := range pgids {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
	sort.Ints(pgids)
	return pgids
}

func manualCleanupTargetPIDs(targets []ManualCleanupTarget) []int {
	if len(targets) == 0 {
		return []int{}
	}
	pids := make([]int, 0, len(targets))
	for _, target := range targets {
		pids = append(pids, target.PID)
	}
	sort.Ints(pids)
	return pids
}

func idleControlState(threadID string) ThreadControl {
	return ThreadControl{
		SchemaVersion:       1,
		ThreadID:            threadID,
		Owner:               "idle",
		OwnerDetail:         "idle",
		LastTransitionAtUTC: nowUTCISO(),
	}
}

func refreshControlState(control ThreadControl) ThreadControl {
	refreshed := control
	if refreshed.Owner != "watcher" {
		return refreshed
	}
	if tmuxSessionExistsHook(refreshed.ActiveTmuxSession) {
		return refreshed
	}
	idle := idleControlState(refreshed.ThreadID)
	idle.OwnerDetail = "watcher_session_finished"
	return idle
}

func loadAndRefreshControl(path, threadID string) (ThreadControl, error) {
	control, ok, err := loadThreadControl(path)
	if err != nil {
		return ThreadControl{}, err
	}
	if !ok {
		control = idleControlState(threadID)
	} else {
		control = refreshControlState(control)
		if control.ThreadID == "" && threadID != "" {
			control.ThreadID = threadID
		}
	}
	if err := saveThreadControl(path, control); err != nil {
		return ThreadControl{}, err
	}
	return control, nil
}

func manualControlState(threadID, ownerDetail, interruptedAutoSession string) ThreadControl {
	control := idleControlState(threadID)
	control.Owner = "manual"
	control.OwnerDetail = ownerDetail
	control.LastManualClaimAtUTC = nowUTCISO()
	if interruptedAutoSession != "" {
		control.LastInterruptedAutoTmuxSession = interruptedAutoSession
	}
	return control
}

func watcherControlState(threadID, watcherSessionName, activeTmuxSession string) ThreadControl {
	control := idleControlState(threadID)
	control.Owner = "watcher"
	control.OwnerDetail = watcherSessionName
	control.ActiveTmuxSession = activeTmuxSession
	control.LastAutoResumeAtUTC = nowUTCISO()
	return control
}

func loadTokenFile(path string) error {
	if githubToken() != "" {
		return nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(payload)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if key != "" && value != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func githubToken() string {
	if token := strings.TrimSpace(os.Getenv("BABEL_GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
}

func requireTokenForDefaultAPI(apiBaseURL string) error {
	if strings.TrimRight(apiBaseURL, "/") == DefaultAPIBaseURL && githubToken() == "" {
		return fmt.Errorf("默认 GitHub API 需要认证 token。请设置 BABEL_GITHUB_TOKEN 或 GITHUB_TOKEN。")
	}
	return nil
}

func githubLogin(client *GitHubClient) (string, error) {
	user, err := client.GetUser()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(user.Login) == "" {
		return "", fmt.Errorf("无法从 GitHub API 获取当前登录名")
	}
	return user.Login, nil
}

func renderIssueBody(report, decisionRequest, threadID, notifyLogin string) string {
	lines := []string{}
	if strings.TrimSpace(notifyLogin) != "" {
		lines = append(lines,
			fmt.Sprintf("@%s 请查看这个阶段 issue；如果要让我继续，请直接在这里评论下一步并关闭 issue。", notifyLogin),
			"",
		)
	}
	lines = append(lines,
		"## 阶段报告",
		"",
		strings.TrimSpace(report),
		"",
		"## 下一步决策请求",
		"",
		strings.TrimSpace(decisionRequest),
		"",
		"## 关闭回传说明",
		"",
		"请直接在本 issue 顶层评论下一步指令，然后关闭本 issue。",
		fmt.Sprintf("本机 watcher 会在检测到关闭后，对 Codex 线程 `%s` 执行 `codex resume`，继续同一条会话。", threadID),
	)
	return strings.Join(lines, "\n")
}

func renderTerminalHandoff(issueURL, decisionRequest string) string {
	return strings.Join([]string{
		fmt.Sprintf("当前阶段等待点：%s", issueURL),
		"如果你现在就在当前终端，直接回复我即可；我会关闭当前阶段 issue 并继续。",
		"如果你离开当前终端，请在该 issue 顶层评论下一步并关闭 issue。",
		"",
		"当前决策请求：",
		strings.TrimSpace(decisionRequest),
	}, "\n")
}

func resumeShellCommand(workdir, threadID, prompt, mode string) string {
	switch strings.TrimSpace(mode) {
	case "exec":
		return fmt.Sprintf(
			"cd %s && codex exec resume --dangerously-bypass-approvals-and-sandbox -c model_reasoning_effort=xhigh %s %s",
			shellQuote(workdir),
			threadID,
			shellQuote(prompt),
		)
	default:
		return fmt.Sprintf(
			"cd %s && codex --ask-for-approval never --sandbox danger-full-access -c model_reasoning_effort=xhigh resume %s %s",
			shellQuote(workdir),
			threadID,
			shellQuote(prompt),
		)
	}
}

func tmuxSessionName(prefix string, issueNumber int) string {
	return fmt.Sprintf("%s_%d_%d", prefix, issueNumber, time.Now().Unix())
}

func launchResumeInTmux(workdir, threadID, prompt string, issueNumber int, prefix, mode string) (string, error) {
	sessionName := tmuxSessionName(prefix, issueNumber)
	command := resumeShellCommand(workdir, threadID, prompt, mode)
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, command)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return sessionName, nil
}

func renderResumePrompt(state BridgeState, issue Issue, comment Comment) string {
	issueBody := strings.TrimSpace(issue.Body)
	if issueBody == "" {
		issueBody = "Issue 正文为空。"
	} else {
		issueBody = "Issue 正文仅作为上下文参考：\n" + issueBody
	}
	return strings.Join([]string{
		fmt.Sprintf("GitHub issue #%d 已关闭，请继续当前线程的下一步工作。", state.IssueNumber),
		fmt.Sprintf("Issue 标题：%s", strings.TrimSpace(issue.Title)),
		fmt.Sprintf("用户关闭前的最新评论如下：\n%s", strings.TrimSpace(comment.Body)),
		issueBody,
		fmt.Sprintf("请在工作目录 `%s` 继续执行。 保持既有仓库规则：小阶段结束前先提交并推送；如果阶段完成并需要等待用户决策，则创建下一条阶段 issue。", state.Workdir),
	}, "\n\n")
}

func renderAnnotatedTerminalCloseComment(replyText string) string {
	lines := []string{
		"当前活动终端已经收到用户下一步指令，本 issue 改由当前线程继续处理。",
		"watcher 不需要再拉起新的 `codex resume` 客户端。",
	}
	if text := strings.TrimSpace(replyText); text != "" {
		lines = append(lines, "", "用户在当前终端的指令：", text)
	}
	return strings.Join(lines, "\n")
}

func renderManualTakeoverCloseComment() string {
	return strings.Join([]string{
		"当前活动终端已经接管了这条线程，本 issue 不再作为等待点。",
		"watcher 不需要再为这次手动接管拉起新的 `codex resume` 客户端。",
		"",
		"用户已通过 Termux/服务器手动接管当前线程。",
	}, "\n")
}

func waitingHandoffFields(state *BridgeState) {
	state.HandoffStatus = "waiting"
	state.HandoffOpenedAtUTC = nowUTCISO()
	state.HandoffConsumedAtUTC = ""
}

func handoffIsWaiting(state BridgeState) bool {
	return state.HandoffStatus == "waiting"
}

func markHandoffConsumed(state *BridgeState, source string) {
	state.HandoffStatus = source
	state.HandoffConsumedAtUTC = nowUTCISO()
}

func handoffConsumedStatusForCloseReason(closeReason string) string {
	switch closeReason {
	case "manual_takeover":
		return "consumed_by_manual_takeover"
	case "manager_handoff":
		return "queued_for_manager_watcher"
	default:
		return "consumed_by_terminal_reply"
	}
}

func shouldMarkCreatedCommentConsumed(closeReason string) bool {
	return closeReason != "manager_handoff"
}

func markCommentConsumed(state *BridgeState, comment Comment) {
	state.LastResumedCommentID = &comment.ID
	state.LastResumedCommentURL = comment.HTMLURL
	state.LastResumedAtUTC = nowUTCISO()
}

func parseEventFields(items []string) (map[string]string, error) {
	fields := map[string]string{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("field 必须是 key=value 形状：%s", item)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("field key 不能为空：%s", item)
		}
		fields[key] = value
	}
	return fields, nil
}

func newestCommentToResume(comments []Comment, lastResumedCommentID *int64) *Comment {
	for i := len(comments) - 1; i >= 0; i-- {
		comment := comments[i]
		if strings.TrimSpace(comment.Body) == "" {
			continue
		}
		if lastResumedCommentID != nil && comment.ID == *lastResumedCommentID {
			return nil
		}
		return &comment
	}
	return nil
}

func resolveThreadID(explicit string, state *BridgeState) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	if state != nil && strings.TrimSpace(state.ThreadID) != "" {
		return strings.TrimSpace(state.ThreadID)
	}
	return strings.TrimSpace(os.Getenv("CODEX_THREAD_ID"))
}

func sortedKeys(entries map[string]any) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func truncateText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
