package issuebridge

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshControlStateReleasesFinishedWatcherSession(t *testing.T) {
	previous := tmuxSessionExistsHook
	tmuxSessionExistsHook = func(string) bool { return false }
	defer func() { tmuxSessionExistsHook = previous }()

	control := ThreadControl{
		SchemaVersion:       1,
		ThreadID:            "thread-1",
		Owner:               "watcher",
		OwnerDetail:         "codex_issue_watch",
		ActiveTmuxSession:   "codex_issue_resume_1_1",
		LastTransitionAtUTC: nowUTCISO(),
	}
	refreshed := refreshControlState(control)
	if refreshed.Owner != "idle" {
		t.Fatalf("expected idle owner, got %q", refreshed.Owner)
	}
	if refreshed.OwnerDetail != "watcher_session_finished" {
		t.Fatalf("unexpected owner detail: %q", refreshed.OwnerDetail)
	}
	if refreshed.ActiveTmuxSession != "" {
		t.Fatalf("expected cleared active tmux session, got %q", refreshed.ActiveTmuxSession)
	}
	if refreshed.ThreadID != "thread-1" {
		t.Fatalf("unexpected thread id: %q", refreshed.ThreadID)
	}
}

func TestManualControlStateRecordsInterruptedAutoSession(t *testing.T) {
	control := manualControlState("thread-2", "termux_s", "codex_issue_resume_2_99")
	if control.Owner != "manual" {
		t.Fatalf("unexpected owner: %q", control.Owner)
	}
	if control.OwnerDetail != "termux_s" {
		t.Fatalf("unexpected owner detail: %q", control.OwnerDetail)
	}
	if control.ThreadID != "thread-2" {
		t.Fatalf("unexpected thread id: %q", control.ThreadID)
	}
	if control.LastInterruptedAutoTmuxSession != "codex_issue_resume_2_99" {
		t.Fatalf("unexpected interrupted session: %q", control.LastInterruptedAutoTmuxSession)
	}
}

func TestRenderIssueBodyMentionsNotifyLogin(t *testing.T) {
	body := renderIssueBody("阶段报告正文", "下一步请求", "thread-3", "dengxiaocheng")
	if !strings.Contains(body, "@dengxiaocheng") {
		t.Fatalf("expected notify login mention, got %q", body)
	}
	if !strings.Contains(body, "thread-3") {
		t.Fatalf("expected thread id in body, got %q", body)
	}
}

func TestRenderAnnotatedTerminalCloseCommentIncludesReply(t *testing.T) {
	body := renderAnnotatedTerminalCloseComment("继续实现 close-active")
	if !strings.Contains(body, "当前活动终端已经收到用户下一步指令") {
		t.Fatalf("missing close comment prefix: %q", body)
	}
	if !strings.Contains(body, "继续实现 close-active") {
		t.Fatalf("missing user reply: %q", body)
	}
}

func TestRenderManualTakeoverCloseComment(t *testing.T) {
	body := renderManualTakeoverCloseComment()
	if !strings.Contains(body, "当前活动终端已经接管了这条线程") {
		t.Fatalf("unexpected body: %q", body)
	}
	if !strings.Contains(body, "用户已通过 Termux/服务器手动接管当前线程") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestRenderTerminalHandoff(t *testing.T) {
	body := renderTerminalHandoff("https://github.com/example/repo/issues/1", "请选择下一步方向。")
	if !strings.Contains(body, "当前阶段等待点：https://github.com/example/repo/issues/1") {
		t.Fatalf("unexpected handoff body: %q", body)
	}
	if !strings.Contains(body, "请选择下一步方向。") {
		t.Fatalf("unexpected handoff body: %q", body)
	}
}

func TestWaitingHandoffFieldsMarkStateWaiting(t *testing.T) {
	state := BridgeState{}
	waitingHandoffFields(&state)
	if state.HandoffStatus != "waiting" {
		t.Fatalf("unexpected handoff status: %q", state.HandoffStatus)
	}
	if !handoffIsWaiting(state) {
		t.Fatalf("expected state to be waiting")
	}
}

func TestMarkHandoffConsumed(t *testing.T) {
	state := BridgeState{HandoffStatus: "waiting"}
	markHandoffConsumed(&state, "consumed_by_terminal_reply")
	if state.HandoffStatus != "consumed_by_terminal_reply" {
		t.Fatalf("unexpected handoff status: %q", state.HandoffStatus)
	}
	if state.HandoffConsumedAtUTC == "" {
		t.Fatalf("expected handoff consumed timestamp")
	}
}

func TestHandoffConsumedStatusForCloseReason(t *testing.T) {
	if got := handoffConsumedStatusForCloseReason("manual_takeover"); got != "consumed_by_manual_takeover" {
		t.Fatalf("unexpected manual status: %q", got)
	}
	if got := handoffConsumedStatusForCloseReason("manager_handoff"); got != "queued_for_manager_watcher" {
		t.Fatalf("unexpected manager status: %q", got)
	}
	if got := handoffConsumedStatusForCloseReason("terminal_reply"); got != "consumed_by_terminal_reply" {
		t.Fatalf("unexpected default status: %q", got)
	}
}

func TestShouldMarkCreatedCommentConsumed(t *testing.T) {
	if shouldMarkCreatedCommentConsumed("manager_handoff") {
		t.Fatalf("manager_handoff should keep the new comment resumable")
	}
	if !shouldMarkCreatedCommentConsumed("terminal_reply") {
		t.Fatalf("terminal replies should consume the new comment")
	}
}

func TestUpsertWorkerPreservesExistingFields(t *testing.T) {
	registry := newWorkerRegistry()
	first := upsertWorker(&registry, ClaudeWorker{
		WorkerID:      "worker-a",
		Status:        WorkerStatusQueued,
		Lane:          "ui",
		TaskLevel:     TaskLevelM,
		MaxFiles:      3,
		MaxDeltaLines: 500,
		ReadScope:     []string{"docs/", "src/ui/"},
		WriteScope:    []string{"src/ui/"},
		TestCommands:  []string{"go test ./..."},
		SessionID:     "session-1",
		TaskTitle:     "first",
		PacketFile:    "/tmp/packet.md",
		ReportFile:    "/tmp/report.md",
		CreatedAtUTC:  "2026-04-24T00:00:00Z",
	})
	second := upsertWorker(&registry, ClaudeWorker{
		WorkerID: "worker-a",
		Status:   WorkerStatusRunning,
	})
	if first.CreatedAtUTC != second.CreatedAtUTC {
		t.Fatalf("expected created_at preserved, got %q -> %q", first.CreatedAtUTC, second.CreatedAtUTC)
	}
	if second.SessionID != "session-1" {
		t.Fatalf("expected session id preserved, got %q", second.SessionID)
	}
	if second.Lane != "ui" {
		t.Fatalf("expected lane preserved, got %q", second.Lane)
	}
	if second.TaskLevel != TaskLevelM {
		t.Fatalf("expected task level preserved, got %q", second.TaskLevel)
	}
	if second.MaxFiles != 3 || second.MaxDeltaLines != 500 {
		t.Fatalf("expected budgets preserved, got max_files=%d max_delta_lines=%d", second.MaxFiles, second.MaxDeltaLines)
	}
	if len(second.ReadScope) != 2 || len(second.WriteScope) != 1 || len(second.TestCommands) != 1 {
		t.Fatalf("expected scopes preserved, got %+v", second)
	}
	if second.TaskTitle != "first" {
		t.Fatalf("expected task title preserved, got %q", second.TaskTitle)
	}
	if second.PacketFile != "/tmp/packet.md" {
		t.Fatalf("expected packet file preserved, got %q", second.PacketFile)
	}
	if second.ReportFile != "/tmp/report.md" {
		t.Fatalf("expected report file preserved, got %q", second.ReportFile)
	}
}

func TestActionableWorkersSortsByPriority(t *testing.T) {
	registry := newWorkerRegistry()
	registry.Workers = []ClaudeWorker{
		{WorkerID: "queued-1", Status: WorkerStatusQueued, LastUpdatedAtUTC: "2026-04-24T00:00:03Z"},
		{WorkerID: "handoff-1", Status: WorkerStatusHandoffQueued, LastUpdatedAtUTC: "2026-04-24T00:00:04Z"},
		{WorkerID: "running-1", Status: WorkerStatusRunning, LastUpdatedAtUTC: "2026-04-24T00:00:02Z"},
		{WorkerID: "done-1", Status: WorkerStatusDone, LastUpdatedAtUTC: "2026-04-24T00:00:01Z"},
	}
	workers := actionableWorkers(registry)
	if len(workers) != 3 {
		t.Fatalf("expected 3 actionable workers, got %d", len(workers))
	}
	if workers[0].WorkerID != "handoff-1" {
		t.Fatalf("expected handoff first, got %q", workers[0].WorkerID)
	}
	if workers[1].WorkerID != "running-1" {
		t.Fatalf("expected running second, got %q", workers[1].WorkerID)
	}
	if workers[2].WorkerID != "queued-1" {
		t.Fatalf("expected queued third, got %q", workers[2].WorkerID)
	}
}

func TestDispatchableWorkersIgnoresRunningAndHandoff(t *testing.T) {
	registry := newWorkerRegistry()
	registry.Workers = []ClaudeWorker{
		{WorkerID: "queued-1", Status: WorkerStatusQueued, LastUpdatedAtUTC: "2026-04-24T00:00:03Z"},
		{WorkerID: "rework-1", Status: WorkerStatusRework, LastUpdatedAtUTC: "2026-04-24T00:00:02Z"},
		{WorkerID: "running-1", Status: WorkerStatusRunning, LastUpdatedAtUTC: "2026-04-24T00:00:01Z"},
		{WorkerID: "handoff-1", Status: WorkerStatusHandoffQueued, LastUpdatedAtUTC: "2026-04-24T00:00:00Z"},
	}
	workers := dispatchableWorkers(registry)
	if len(workers) != 2 {
		t.Fatalf("expected 2 dispatchable workers, got %d", len(workers))
	}
	if workers[0].WorkerID != "rework-1" {
		t.Fatalf("expected rework first, got %q", workers[0].WorkerID)
	}
	if workers[1].WorkerID != "queued-1" {
		t.Fatalf("expected queued second, got %q", workers[1].WorkerID)
	}
}

func TestApplyTaskLevelDefaults(t *testing.T) {
	level, maxFiles, maxDeltaLines := applyTaskLevelDefaults("m", 0, 0)
	if level != TaskLevelM {
		t.Fatalf("expected normalized level M, got %q", level)
	}
	if maxFiles != 3 || maxDeltaLines != 500 {
		t.Fatalf("unexpected defaults for M: files=%d lines=%d", maxFiles, maxDeltaLines)
	}
	level, maxFiles, maxDeltaLines = applyTaskLevelDefaults("S", 2, 150)
	if level != TaskLevelS || maxFiles != 2 || maxDeltaLines != 150 {
		t.Fatalf("expected explicit overrides preserved, got level=%q files=%d lines=%d", level, maxFiles, maxDeltaLines)
	}
}

func TestNextDispatchableWorkerRespectsMaxRunning(t *testing.T) {
	registry := newWorkerRegistry()
	registry.Workers = []ClaudeWorker{
		{WorkerID: "running-1", Status: WorkerStatusRunning, Lane: "ui", LastUpdatedAtUTC: "2026-04-24T00:00:01Z"},
		{WorkerID: "queued-1", Status: WorkerStatusQueued, Lane: "logic", LastUpdatedAtUTC: "2026-04-24T00:00:02Z"},
	}
	_, ok := nextDispatchableWorker(registry, false, 1, false)
	if ok {
		t.Fatalf("expected no dispatchable worker when max-running is reached")
	}
	worker, ok := nextDispatchableWorker(registry, false, 2, false)
	if !ok || worker.WorkerID != "queued-1" {
		t.Fatalf("expected queued-1 when max-running allows another worker, got %+v ok=%v", worker, ok)
	}
}

func TestNextDispatchableWorkerRespectsLaneLock(t *testing.T) {
	registry := newWorkerRegistry()
	registry.Workers = []ClaudeWorker{
		{WorkerID: "running-ui", Status: WorkerStatusRunning, Lane: "ui", LastUpdatedAtUTC: "2026-04-24T00:00:01Z"},
		{WorkerID: "queued-ui", Status: WorkerStatusQueued, Lane: "ui", LastUpdatedAtUTC: "2026-04-24T00:00:02Z"},
		{WorkerID: "queued-logic", Status: WorkerStatusQueued, Lane: "logic", LastUpdatedAtUTC: "2026-04-24T00:00:03Z"},
	}
	worker, ok := nextDispatchableWorker(registry, false, 2, false)
	if !ok || worker.WorkerID != "queued-logic" {
		t.Fatalf("expected lane-safe worker, got %+v ok=%v", worker, ok)
	}
	worker, ok = nextDispatchableWorker(registry, false, 2, true)
	if !ok || worker.WorkerID != "queued-ui" {
		t.Fatalf("expected same-lane worker when lane lock disabled, got %+v ok=%v", worker, ok)
	}
}

func TestRunWorkerRegisterCreatesRegistryEntry(t *testing.T) {
	tmp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.runWorkerRegister([]string{"--worker-id", "worker-a", "--task-title", "task-a"}); code != 0 {
		t.Fatalf("runWorkerRegister returned code %d stderr=%q", code, stderr.String())
	}
	registry, exists, err := loadWorkerRegistry(filepath.Join(tmp, ".codex-runtime", "claudecode_workers.json"))
	if err != nil {
		t.Fatalf("loadWorkerRegistry returned error: %v", err)
	}
	if !exists || len(registry.Workers) != 1 {
		t.Fatalf("expected 1 worker in registry, got exists=%v workers=%d", exists, len(registry.Workers))
	}
	if registry.Workers[0].WorkerID != "worker-a" || registry.Workers[0].Status != WorkerStatusQueued {
		t.Fatalf("unexpected worker: %+v", registry.Workers[0])
	}
}

func TestRunWorkerPacketCreatesArtifactsAndRegistryFields(t *testing.T) {
	tmp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &Runner{Stdout: &stdout, Stderr: &stderr}
	args := []string{
		"--worker-id", "worker-a",
		"--lane", "ui",
		"--task-level", "M",
		"--task-title", "配给日 UI",
		"--task-summary", "补一轮 UI 调整",
		"--goal", "完成发粮界面的首屏交互。",
		"--read-scope", "docs/",
		"--write-scope", "src/ui/",
		"--test-command", "go test ./internal/ops/issuebridge",
		"--acceptance", "按钮状态正确",
		"--constraint", "不要改无关场景",
		"--deliverable", "更新代码并写 report",
	}
	if code := runner.runWorkerPacket(args); code != 0 {
		t.Fatalf("runWorkerPacket returned code %d stderr=%q", code, stderr.String())
	}

	registry, exists, err := loadWorkerRegistry(filepath.Join(tmp, ".codex-runtime", "claudecode_workers.json"))
	if err != nil {
		t.Fatalf("loadWorkerRegistry returned error: %v", err)
	}
	if !exists || len(registry.Workers) != 1 {
		t.Fatalf("expected 1 worker in registry, got exists=%v workers=%d", exists, len(registry.Workers))
	}
	worker := registry.Workers[0]
	if !strings.HasSuffix(worker.PacketFile, filepath.Join(".codex-runtime", "claudecode_workers", "worker-a", "packet.md")) {
		t.Fatalf("unexpected packet file: %q", worker.PacketFile)
	}
	if worker.Lane != "ui" {
		t.Fatalf("unexpected lane: %q", worker.Lane)
	}
	if worker.TaskLevel != TaskLevelM || worker.MaxFiles != 3 || worker.MaxDeltaLines != 500 {
		t.Fatalf("unexpected task budget defaults: %+v", worker)
	}
	if len(worker.ReadScope) != 1 || len(worker.WriteScope) != 1 || len(worker.TestCommands) != 1 {
		t.Fatalf("unexpected worker scopes: %+v", worker)
	}
	if !strings.HasSuffix(worker.ReportFile, filepath.Join(".codex-runtime", "claudecode_workers", "worker-a", "report.md")) {
		t.Fatalf("unexpected report file: %q", worker.ReportFile)
	}
	packetPayload, err := os.ReadFile(worker.PacketFile)
	if err != nil {
		t.Fatalf("ReadFile packet returned error: %v", err)
	}
	packet := string(packetPayload)
	if !strings.Contains(packet, "# Worker Packet: 配给日 UI") {
		t.Fatalf("packet missing title: %q", packet)
	}
	if !strings.Contains(packet, "按钮状态正确") {
		t.Fatalf("packet missing acceptance: %q", packet)
	}
	if !strings.Contains(packet, "Task Level: `M`") {
		t.Fatalf("packet missing task level: %q", packet)
	}
	if !strings.Contains(packet, "最多修改文件数: 3") {
		t.Fatalf("packet missing file budget: %q", packet)
	}
	if !strings.Contains(packet, "src/ui/") {
		t.Fatalf("packet missing write scope: %q", packet)
	}
	if !strings.Contains(packet, "go test ./internal/ops/issuebridge") {
		t.Fatalf("packet missing test command: %q", packet)
	}
	reportPayload, err := os.ReadFile(worker.ReportFile)
	if err != nil {
		t.Fatalf("ReadFile report returned error: %v", err)
	}
	report := string(reportPayload)
	if !strings.Contains(report, "# Worker Report: 配给日 UI") {
		t.Fatalf("unexpected report template: %q", report)
	}
	if !strings.Contains(report, "Budget Check") || !strings.Contains(report, "File Budget: 3") {
		t.Fatalf("report missing budget section: %q", report)
	}
}

func TestRunWorkerNextShellReturnsDispatchableWorker(t *testing.T) {
	tmp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	registry := newWorkerRegistry()
	upsertWorker(&registry, ClaudeWorker{WorkerID: "running-1", Status: WorkerStatusRunning, Lane: "ui"})
	upsertWorker(&registry, ClaudeWorker{WorkerID: "queued-1", Status: WorkerStatusQueued, Lane: "logic", TaskTitle: "队列任务", PacketFile: "/tmp/packet.md"})
	upsertWorker(&registry, ClaudeWorker{WorkerID: "rework-1", Status: WorkerStatusRework, Lane: "ui", TaskTitle: "返工任务", PacketFile: "/tmp/rework.md"})
	if err := saveWorkerRegistry(filepath.Join(tmp, ".codex-runtime", "claudecode_workers.json"), registry); err != nil {
		t.Fatalf("saveWorkerRegistry returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.runWorkerNext([]string{"--shell", "--max-running", "2"}); code != 0 {
		t.Fatalf("runWorkerNext returned code %d stderr=%q", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "WORKER_ID='queued-1'") {
		t.Fatalf("unexpected worker selection: %q", output)
	}
	if !strings.Contains(output, "LANE='logic'") {
		t.Fatalf("unexpected lane output: %q", output)
	}
	if !strings.Contains(output, "PACKET_FILE='/tmp/packet.md'") {
		t.Fatalf("unexpected packet file: %q", output)
	}
}

func TestRunWorkerNextShellAllowsSameLaneWhenRequested(t *testing.T) {
	tmp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	registry := newWorkerRegistry()
	upsertWorker(&registry, ClaudeWorker{WorkerID: "running-1", Status: WorkerStatusRunning, Lane: "ui"})
	upsertWorker(&registry, ClaudeWorker{WorkerID: "rework-1", Status: WorkerStatusRework, Lane: "ui", TaskTitle: "返工任务", PacketFile: "/tmp/rework.md"})
	if err := saveWorkerRegistry(filepath.Join(tmp, ".codex-runtime", "claudecode_workers.json"), registry); err != nil {
		t.Fatalf("saveWorkerRegistry returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.runWorkerNext([]string{"--shell", "--max-running", "2", "--allow-same-lane"}); code != 0 {
		t.Fatalf("runWorkerNext returned code %d stderr=%q", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "WORKER_ID='rework-1'") {
		t.Fatalf("unexpected worker selection with same-lane enabled: %q", output)
	}
}

func TestRunWorkerNextShellRespectsWorkerPrefix(t *testing.T) {
	tmp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	registry := newWorkerRegistry()
	upsertWorker(&registry, ClaudeWorker{WorkerID: "oldproj-foundation", Status: WorkerStatusQueued, Lane: "foundation"})
	upsertWorker(&registry, ClaudeWorker{WorkerID: "newproj-foundation", Status: WorkerStatusQueued, Lane: "foundation"})
	if err := saveWorkerRegistry(filepath.Join(tmp, ".codex-runtime", "claudecode_workers.json"), registry); err != nil {
		t.Fatalf("saveWorkerRegistry returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.runWorkerNext([]string{"--shell", "--worker-prefix", "newproj-"}); code != 0 {
		t.Fatalf("runWorkerNext returned code %d stderr=%q", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "WORKER_ID='newproj-foundation'") {
		t.Fatalf("unexpected worker selection with prefix: %q", output)
	}
}

func TestRunWorkerSetStatusRemovesFromActionableQueue(t *testing.T) {
	tmp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() { _ = os.Chdir(previous) }()

	registry := newWorkerRegistry()
	upsertWorker(&registry, ClaudeWorker{WorkerID: "worker-a", Status: WorkerStatusQueued})
	if err := saveWorkerRegistry(filepath.Join(tmp, ".codex-runtime", "claudecode_workers.json"), registry); err != nil {
		t.Fatalf("saveWorkerRegistry returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.runWorkerSetStatus([]string{"--worker-id", "worker-a", "--status", WorkerStatusDone}); code != 0 {
		t.Fatalf("runWorkerSetStatus returned code %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.runWorkerQueue([]string{}); code != 0 {
		t.Fatalf("runWorkerQueue returned code %d stderr=%q", code, stderr.String())
	}
	var workers []ClaudeWorker
	if err := json.Unmarshal(stdout.Bytes(), &workers); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v output=%q", err, stdout.String())
	}
	if len(workers) != 0 {
		t.Fatalf("expected empty actionable queue, got %+v", workers)
	}
}

func TestAppendEventAndLoadRecentEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if _, err := appendEvent(path, "watcher_process_started", false, map[string]any{"session_name": "codex_issue_watch"}); err != nil {
		t.Fatalf("appendEvent returned error: %v", err)
	}
	if _, err := appendEvent(path, "manual_claimed", false, map[string]any{"thread_id": "thread-1"}); err != nil {
		t.Fatalf("appendEvent returned error: %v", err)
	}
	events, err := loadRecentEvents(path, 2)
	if err != nil {
		t.Fatalf("loadRecentEvents returned error: %v", err)
	}
	if got := events[0]["event_type"]; got != "watcher_process_started" {
		t.Fatalf("unexpected first event type: %v", got)
	}
	if got := events[1]["thread_id"]; got != "thread-1" {
		t.Fatalf("unexpected second event thread id: %v", got)
	}
}

func TestParseEventFields(t *testing.T) {
	fields, err := parseEventFields([]string{"session_name=codex_issue_watch", "thread_id=thread-1"})
	if err != nil {
		t.Fatalf("parseEventFields returned error: %v", err)
	}
	if fields["session_name"] != "codex_issue_watch" {
		t.Fatalf("unexpected session_name: %q", fields["session_name"])
	}
	if fields["thread_id"] != "thread-1" {
		t.Fatalf("unexpected thread_id: %q", fields["thread_id"])
	}
}

func TestAppendEventDispatchesOptionalHook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	previous := os.Getenv("BABEL_ISSUE_BRIDGE_EVENT_HOOK")
	if err := os.Setenv("BABEL_ISSUE_BRIDGE_EVENT_HOOK", "cat >/dev/null"); err != nil {
		t.Fatalf("Setenv returned error: %v", err)
	}
	defer func() {
		if previous == "" {
			_ = os.Unsetenv("BABEL_ISSUE_BRIDGE_EVENT_HOOK")
		} else {
			_ = os.Setenv("BABEL_ISSUE_BRIDGE_EVENT_HOOK", previous)
		}
	}()

	if _, err := appendEvent(path, "watcher_process_started", true, map[string]any{"session_name": "codex_issue_watch"}); err != nil {
		t.Fatalf("appendEvent returned error: %v", err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Count(strings.TrimSpace(string(payload)), "\n")+1 != 1 {
		t.Fatalf("expected only one event line, got %q", string(payload))
	}
}

func TestBuildWatchCommandIncludesQuotedArgs(t *testing.T) {
	command := buildWatchCommand("/tmp/work dir", []string{"--once", "--resume-session-prefix", "issue resume"})
	if !strings.Contains(command, "cd '/tmp/work dir' && go run ./cmd/babel-issue-bridge watch") {
		t.Fatalf("unexpected command prefix: %q", command)
	}
	if !strings.Contains(command, "'issue resume'") {
		t.Fatalf("expected quoted arg in command: %q", command)
	}
}

func TestResumeShellCommandExecMode(t *testing.T) {
	command := resumeShellCommand("/tmp/work dir", "thread-a", "继续", "exec")
	if !strings.Contains(command, "codex exec resume --dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("expected exec resume command, got %q", command)
	}
	if !strings.Contains(command, "thread-a") {
		t.Fatalf("expected thread resume, got %q", command)
	}
	if strings.Contains(command, "--mode") {
		t.Fatalf("did not expect unsupported --mode flag: %q", command)
	}
}

func TestResumeShellCommandInteractiveMode(t *testing.T) {
	command := resumeShellCommand("/tmp/work dir", "thread-a", "继续", "interactive")
	if !strings.Contains(command, "codex --ask-for-approval never --sandbox danger-full-access") {
		t.Fatalf("expected interactive resume command, got %q", command)
	}
	if strings.Contains(command, "codex exec") {
		t.Fatalf("did not expect exec mode command: %q", command)
	}
}

func TestParseManualResumeProcesses(t *testing.T) {
	processes, err := parseManualResumeProcesses(strings.NewReader(`
  100 100 /home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-a --entrypoint termux_s
  101 100 node /home/openclaw/.nvm/versions/node/v22.22.1/bin/codex resume thread-a
  200 200 /home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-b --entrypoint termux_m
`))
	if err != nil {
		t.Fatalf("parseManualResumeProcesses returned error: %v", err)
	}
	if len(processes) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(processes))
	}
	if processes[0].PID != 100 || processes[0].PGID != 100 {
		t.Fatalf("unexpected first process: %+v", processes[0])
	}
	if !strings.Contains(processes[2].Args, "--entrypoint termux_m") {
		t.Fatalf("unexpected args: %q", processes[2].Args)
	}
}

func TestMatchingManualCleanupTargets(t *testing.T) {
	processes := []ManualCleanupTarget{
		{PID: 100, PGID: 100, Args: "/home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-a --entrypoint termux_s"},
		{PID: 101, PGID: 100, Args: "node /home/openclaw/.nvm/versions/node/v22.22.1/bin/codex resume thread-a"},
		{PID: 200, PGID: 200, Args: "/home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-a --entrypoint termux_s"},
		{PID: 300, PGID: 300, Args: "/home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-b --entrypoint termux_m"},
	}
	targets := matchingManualCleanupTargets(processes, "thread-a", "termux_s", 200, 200)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].PID != 100 || targets[0].PGID != 100 {
		t.Fatalf("unexpected target: %+v", targets[0])
	}
	if got := manualCleanupTargetPIDs(targets); len(got) != 1 || got[0] != 100 {
		t.Fatalf("unexpected pids: %v", got)
	}
}

func TestMatchingManualCleanupTargetsSkipsExcludedProcessGroup(t *testing.T) {
	processes := []ManualCleanupTarget{
		{PID: 400, PGID: 400, Args: "/home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-a --entrypoint termux_m"},
		{PID: 401, PGID: 400, Args: "sh -lc ... manual-resume --thread-id thread-a --entrypoint termux_m"},
		{PID: 500, PGID: 500, Args: "/home/openclaw/babel-runtime/.codex-runtime/bin/babel-issue-bridge manual-resume --thread-id thread-a --entrypoint termux_m"},
	}
	targets := matchingManualCleanupTargets(processes, "thread-a", "termux_m", 0, 400)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target after pgid exclusion, got %d", len(targets))
	}
	if targets[0].PGID != 500 {
		t.Fatalf("unexpected remaining pgid: %+v", targets[0])
	}
}

func TestSaveAndLoadManualLease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manual_leases", "termux_m.json")
	lease := ManualLease{
		SchemaVersion: 1,
		ThreadID:      "thread-1",
		Entrypoint:    "termux_m",
		SessionID:     "session-1",
		UpdatedAtUTC:  nowUTCISO(),
	}
	if err := saveManualLease(path, lease); err != nil {
		t.Fatalf("saveManualLease returned error: %v", err)
	}
	got, exists, err := loadManualLease(path)
	if err != nil {
		t.Fatalf("loadManualLease returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected lease to exist")
	}
	if got.SessionID != "session-1" || got.ThreadID != "thread-1" || got.Entrypoint != "termux_m" {
		t.Fatalf("unexpected lease: %+v", got)
	}
}

func TestTouchManualLeaseWritesLeaseFile(t *testing.T) {
	workdir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()
	runner := &Runner{Stdout: io.Discard, Stderr: io.Discard}
	if code := runner.Run([]string{
		"touch-manual-lease",
		"--thread-id", "thread-2",
		"--entrypoint", "termux_s",
		"--session-id", "session-2",
		"--lease-dir", ".codex-runtime/manual_leases",
	}); code != 0 {
		t.Fatalf("touch-manual-lease returned code %d", code)
	}
	leasePath := filepath.Join(workdir, ".codex-runtime", "manual_leases", "termux_s.json")
	lease, exists, err := loadManualLease(leasePath)
	if err != nil {
		t.Fatalf("loadManualLease returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected lease file at %s", leasePath)
	}
	if lease.SessionID != "session-2" || lease.ThreadID != "thread-2" {
		t.Fatalf("unexpected lease: %+v", lease)
	}
}
