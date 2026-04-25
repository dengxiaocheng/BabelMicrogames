package issuebridge

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const (
	WorkerStatusQueued        = "queued"
	WorkerStatusRunning       = "running"
	WorkerStatusHandoffQueued = "handoff_queued"
	WorkerStatusDone          = "done"
	WorkerStatusRework        = "rework"
	WorkerStatusCancelled     = "cancelled"
	TaskLevelS                = "S"
	TaskLevelM                = "M"
	TaskLevelL                = "L"
)

func newWorkerRegistry() ClaudeWorkerRegistry {
	return ClaudeWorkerRegistry{
		SchemaVersion: 1,
		Workers:       []ClaudeWorker{},
	}
}

func workerStatusPriority(status string) int {
	switch strings.TrimSpace(status) {
	case WorkerStatusHandoffQueued:
		return 0
	case WorkerStatusRework:
		return 1
	case WorkerStatusRunning:
		return 2
	case WorkerStatusQueued:
		return 3
	default:
		return 9
	}
}

func workerLanePriority(lane string) int {
	switch strings.TrimSpace(lane) {
	case "foundation":
		return 0
	case "logic", "state":
		return 1
	case "content":
		return 2
	case "ui":
		return 3
	case "integration":
		return 4
	case "qa":
		return 5
	default:
		return 9
	}
}

func actionableWorkerStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case WorkerStatusQueued, WorkerStatusRunning, WorkerStatusHandoffQueued, WorkerStatusRework:
		return true
	default:
		return false
	}
}

func dispatchableWorkerStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case WorkerStatusQueued, WorkerStatusRework:
		return true
	default:
		return false
	}
}

func sortWorkersForQueue(workers []ClaudeWorker) {
	sort.SliceStable(workers, func(i, j int) bool {
		left := workers[i]
		right := workers[j]
		lp := workerStatusPriority(left.Status)
		rp := workerStatusPriority(right.Status)
		if lp != rp {
			return lp < rp
		}
		if strings.TrimSpace(left.LastUpdatedAtUTC) != strings.TrimSpace(right.LastUpdatedAtUTC) {
			return strings.TrimSpace(left.LastUpdatedAtUTC) < strings.TrimSpace(right.LastUpdatedAtUTC)
		}
		llp := workerLanePriority(left.Lane)
		rlp := workerLanePriority(right.Lane)
		if llp != rlp {
			return llp < rlp
		}
		return strings.TrimSpace(left.WorkerID) < strings.TrimSpace(right.WorkerID)
	})
}

func findWorkerIndex(workers []ClaudeWorker, workerID string) int {
	for idx, worker := range workers {
		if strings.TrimSpace(worker.WorkerID) == strings.TrimSpace(workerID) {
			return idx
		}
	}
	return -1
}

func upsertWorker(registry *ClaudeWorkerRegistry, worker ClaudeWorker) ClaudeWorker {
	if registry.SchemaVersion == 0 {
		*registry = newWorkerRegistry()
	}
	now := nowUTCISO()
	worker.WorkerID = strings.TrimSpace(worker.WorkerID)
	if strings.TrimSpace(worker.Status) == "" {
		worker.Status = WorkerStatusQueued
	}

	idx := findWorkerIndex(registry.Workers, worker.WorkerID)
	if idx < 0 {
		if strings.TrimSpace(worker.CreatedAtUTC) == "" {
			worker.CreatedAtUTC = now
		}
		worker.LastUpdatedAtUTC = now
		registry.Workers = append(registry.Workers, worker)
		return worker
	}

	current := registry.Workers[idx]
	if strings.TrimSpace(worker.CreatedAtUTC) == "" {
		worker.CreatedAtUTC = current.CreatedAtUTC
	}
	if strings.TrimSpace(worker.CreatedAtUTC) == "" {
		worker.CreatedAtUTC = now
	}
	if strings.TrimSpace(worker.SessionID) == "" {
		worker.SessionID = current.SessionID
	}
	if strings.TrimSpace(worker.Lane) == "" {
		worker.Lane = current.Lane
	}
	if strings.TrimSpace(worker.TaskLevel) == "" {
		worker.TaskLevel = current.TaskLevel
	}
	if worker.MaxFiles == 0 {
		worker.MaxFiles = current.MaxFiles
	}
	if worker.MaxDeltaLines == 0 {
		worker.MaxDeltaLines = current.MaxDeltaLines
	}
	if len(worker.ReadScope) == 0 {
		worker.ReadScope = append([]string{}, current.ReadScope...)
	}
	if len(worker.WriteScope) == 0 {
		worker.WriteScope = append([]string{}, current.WriteScope...)
	}
	if len(worker.TestCommands) == 0 {
		worker.TestCommands = append([]string{}, current.TestCommands...)
	}
	if strings.TrimSpace(worker.Model) == "" {
		worker.Model = current.Model
	}
	if strings.TrimSpace(worker.TaskTitle) == "" {
		worker.TaskTitle = current.TaskTitle
	}
	if strings.TrimSpace(worker.TaskSummary) == "" {
		worker.TaskSummary = current.TaskSummary
	}
	if strings.TrimSpace(worker.RepoFullName) == "" {
		worker.RepoFullName = current.RepoFullName
	}
	if worker.IssueNumber == 0 {
		worker.IssueNumber = current.IssueNumber
	}
	if strings.TrimSpace(worker.IssueURL) == "" {
		worker.IssueURL = current.IssueURL
	}
	if strings.TrimSpace(worker.ManagerThreadID) == "" {
		worker.ManagerThreadID = current.ManagerThreadID
	}
	if strings.TrimSpace(worker.Workdir) == "" {
		worker.Workdir = current.Workdir
	}
	if strings.TrimSpace(worker.PacketFile) == "" {
		worker.PacketFile = current.PacketFile
	}
	if strings.TrimSpace(worker.ReportFile) == "" {
		worker.ReportFile = current.ReportFile
	}
	if strings.TrimSpace(worker.LastCommentFile) == "" {
		worker.LastCommentFile = current.LastCommentFile
	}
	if strings.TrimSpace(worker.LastNote) == "" {
		worker.LastNote = current.LastNote
	}
	if strings.TrimSpace(worker.StartedAtUTC) == "" {
		worker.StartedAtUTC = current.StartedAtUTC
	}
	if strings.TrimSpace(worker.FinishedAtUTC) == "" {
		worker.FinishedAtUTC = current.FinishedAtUTC
	}
	worker.LastUpdatedAtUTC = now
	registry.Workers[idx] = worker
	return worker
}

func actionableWorkers(registry ClaudeWorkerRegistry) []ClaudeWorker {
	workers := []ClaudeWorker{}
	for _, worker := range registry.Workers {
		if actionableWorkerStatus(worker.Status) {
			workers = append(workers, worker)
		}
	}
	sortWorkersForQueue(workers)
	return workers
}

func dispatchableWorkers(registry ClaudeWorkerRegistry) []ClaudeWorker {
	workers := []ClaudeWorker{}
	for _, worker := range registry.Workers {
		if dispatchableWorkerStatus(worker.Status) {
			workers = append(workers, worker)
		}
	}
	sortWorkersForQueue(workers)
	return workers
}

func runningWorkers(registry ClaudeWorkerRegistry) []ClaudeWorker {
	workers := []ClaudeWorker{}
	for _, worker := range registry.Workers {
		if strings.TrimSpace(worker.Status) == WorkerStatusRunning {
			workers = append(workers, worker)
		}
	}
	sortWorkersForQueue(workers)
	return workers
}

func dispatchWorkerAllowed(candidate ClaudeWorker, registry ClaudeWorkerRegistry, maxRunning int, allowSameLane bool) bool {
	running := runningWorkers(registry)
	if maxRunning > 0 && len(running) >= maxRunning {
		return false
	}
	if allowSameLane {
		return true
	}
	candidateLane := strings.TrimSpace(candidate.Lane)
	if candidateLane == "" {
		return true
	}
	for _, worker := range running {
		if strings.TrimSpace(worker.Lane) == candidateLane {
			return false
		}
	}
	return true
}

func nextDispatchableWorker(registry ClaudeWorkerRegistry, includeActionable bool, maxRunning int, allowSameLane bool) (ClaudeWorker, bool) {
	return nextDispatchableWorkerWithPrefix(registry, includeActionable, maxRunning, allowSameLane, "")
}

func nextDispatchableWorkerWithPrefix(registry ClaudeWorkerRegistry, includeActionable bool, maxRunning int, allowSameLane bool, workerPrefix string) (ClaudeWorker, bool) {
	var candidates []ClaudeWorker
	if includeActionable {
		candidates = actionableWorkers(registry)
	} else {
		candidates = dispatchableWorkers(registry)
	}
	workerPrefix = strings.TrimSpace(workerPrefix)
	for _, worker := range candidates {
		if !dispatchableWorkerStatus(worker.Status) {
			continue
		}
		if workerPrefix != "" && !strings.HasPrefix(strings.TrimSpace(worker.WorkerID), workerPrefix) {
			continue
		}
		if dispatchWorkerAllowed(worker, registry, maxRunning, allowSameLane) {
			return worker, true
		}
	}
	return ClaudeWorker{}, false
}

func safeWorkerIDSegment(workerID string) string {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return "worker"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "\t", "_", "\n", "_", "\r", "_")
	return replacer.Replace(workerID)
}

func normalizeTaskLevel(level string) string {
	level = strings.ToUpper(strings.TrimSpace(level))
	switch level {
	case TaskLevelS, TaskLevelM, TaskLevelL:
		return level
	default:
		return ""
	}
}

func taskLevelDefaults(level string) (maxFiles, maxDeltaLines int) {
	switch normalizeTaskLevel(level) {
	case TaskLevelS:
		return 1, 200
	case TaskLevelM:
		return 3, 500
	case TaskLevelL:
		return 5, 800
	default:
		return 0, 0
	}
}

func applyTaskLevelDefaults(level string, maxFiles, maxDeltaLines int) (string, int, int) {
	level = normalizeTaskLevel(level)
	defaultFiles, defaultLines := taskLevelDefaults(level)
	if maxFiles <= 0 {
		maxFiles = defaultFiles
	}
	if maxDeltaLines <= 0 {
		maxDeltaLines = defaultLines
	}
	return level, maxFiles, maxDeltaLines
}

func workerArtifactsDir(workdir, packetRoot, workerID string) string {
	return resolveInWorkdir(workdir, filepath.Join(packetRoot, safeWorkerIDSegment(workerID)))
}

func workerPacketFilePath(workdir, packetRoot, workerID string) string {
	return filepath.Join(workerArtifactsDir(workdir, packetRoot, workerID), "packet.md")
}

func workerReportFilePath(workdir, packetRoot, workerID string) string {
	return filepath.Join(workerArtifactsDir(workdir, packetRoot, workerID), "report.md")
}

func markdownBulletList(items []string, empty string) string {
	if len(items) == 0 {
		return "- " + empty
	}
	var lines []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		lines = append(lines, "- "+item)
	}
	if len(lines) == 0 {
		return "- " + empty
	}
	return strings.Join(lines, "\n")
}

func appendUniqueNonEmpty(items []string, extra ...string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	for _, item := range extra {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func renderWorkerPacket(worker ClaudeWorker, goal string, acceptance, constraints, contextFiles, deliverables []string) string {
	title := strings.TrimSpace(worker.TaskTitle)
	if title == "" {
		title = strings.TrimSpace(worker.WorkerID)
	}
	summary := strings.TrimSpace(worker.TaskSummary)
	if summary == "" {
		summary = "待管理线程补充。"
	}
	goal = strings.TrimSpace(goal)
	if goal == "" {
		goal = "在不扩范围的前提下完成这个小任务，并把结果写回 report。"
	}
	level, maxFiles, maxDeltaLines := applyTaskLevelDefaults(worker.TaskLevel, worker.MaxFiles, worker.MaxDeltaLines)
	autoConstraints := []string{}
	if maxFiles > 0 {
		autoConstraints = append(autoConstraints, fmt.Sprintf("最多修改 %d 个文件。", maxFiles))
	}
	if maxDeltaLines > 0 {
		autoConstraints = append(autoConstraints, fmt.Sprintf("净新增或修改代码尽量不超过 %d 行。", maxDeltaLines))
	}
	if len(worker.WriteScope) > 0 {
		autoConstraints = append(autoConstraints, "只允许在 write scope 内改动文件；超出范围必须回交 manager。")
	}
	constraints = appendUniqueNonEmpty(constraints, autoConstraints...)

	sections := []string{
		fmt.Sprintf("# Worker Packet: %s", title),
		"",
		"## Worker Meta",
		fmt.Sprintf("- Worker ID: `%s`", strings.TrimSpace(worker.WorkerID)),
		fmt.Sprintf("- Lane: `%s`", strings.TrimSpace(worker.Lane)),
		fmt.Sprintf("- Task Level: `%s`", level),
		fmt.Sprintf("- Workdir: `%s`", strings.TrimSpace(worker.Workdir)),
		fmt.Sprintf("- Repo: `%s`", strings.TrimSpace(worker.RepoFullName)),
		fmt.Sprintf("- Manager Thread ID: `%s`", strings.TrimSpace(worker.ManagerThreadID)),
	}
	if worker.IssueNumber > 0 {
		sections = append(sections, fmt.Sprintf("- Waiting Issue: `#%d`", worker.IssueNumber))
	}
	if strings.TrimSpace(worker.IssueURL) != "" {
		sections = append(sections, fmt.Sprintf("- Issue URL: %s", strings.TrimSpace(worker.IssueURL)))
	}
	if strings.TrimSpace(worker.SessionID) != "" {
		sections = append(sections, fmt.Sprintf("- Claude Session ID: `%s`", strings.TrimSpace(worker.SessionID)))
	}
	if strings.TrimSpace(worker.Model) != "" {
		sections = append(sections, fmt.Sprintf("- Model: `%s`", strings.TrimSpace(worker.Model)))
	}

	sections = append(sections,
		"",
		"## Task",
		fmt.Sprintf("### Summary\n%s", summary),
		"",
		fmt.Sprintf("### Goal\n%s", goal),
		"",
		"## Budget",
		markdownBulletList([]string{
			fmt.Sprintf("Task Level: %s", level),
			conditionalBudgetLine("最多修改文件数", maxFiles),
			conditionalBudgetLine("最大净改动行数", maxDeltaLines),
		}, "未设置预算。"),
		"",
		"## Read Scope",
		markdownBulletList(worker.ReadScope, "未限制 read scope。"),
		"",
		"## Write Scope",
		markdownBulletList(worker.WriteScope, "未限制 write scope。"),
		"",
		"## Test Commands",
		markdownBulletList(worker.TestCommands, "未指定测试命令。"),
		"",
		"## Acceptance",
		markdownBulletList(acceptance, "由管理线程补充验收项。"),
		"",
		"## Constraints",
		markdownBulletList(constraints, "不要扩范围，不要顺手重构无关模块。"),
		"",
		"## Context Files",
		markdownBulletList(contextFiles, "无额外上下文文件。"),
		"",
		"## Deliverables",
		markdownBulletList(deliverables, "更新代码或文档，并把结果写进 report。"),
		"",
		"## Finish Protocol",
		fmt.Sprintf("1. 把结果写到 `%s`", strings.TrimSpace(worker.ReportFile)),
		fmt.Sprintf("2. 在仓库根目录执行：`sh scripts/claudecode_worker_finish.sh --worker-id %s`", strings.TrimSpace(worker.WorkerID)),
		"3. 不要自己做项目级归档或阶段切换，交给 Codex manager。",
		"",
	)
	return strings.Join(sections, "\n")
}

func renderWorkerReportTemplate(worker ClaudeWorker) string {
	title := strings.TrimSpace(worker.TaskTitle)
	if title == "" {
		title = strings.TrimSpace(worker.WorkerID)
	}
	level, maxFiles, maxDeltaLines := applyTaskLevelDefaults(worker.TaskLevel, worker.MaxFiles, worker.MaxDeltaLines)
	return strings.Join([]string{
		fmt.Sprintf("# Worker Report: %s", title),
		"",
		"## Summary",
		"- 待填写",
		"",
		"## Budget Check",
		fmt.Sprintf("- Task Level: %s", level),
		fmt.Sprintf("- File Budget: %s", renderBudgetValue(maxFiles, "未设置")),
		fmt.Sprintf("- Delta Line Budget: %s", renderBudgetValue(maxDeltaLines, "未设置")),
		"- Actual Changed Files: 待填写",
		"- Estimated Delta Lines: 待填写",
		"- Stayed Within Budget: 待填写",
		"",
		"## Files Changed",
		"- 待填写",
		"",
		"## Tests",
		markdownBulletList(worker.TestCommands, "未运行"),
		"",
		"## Scope Deviations",
		"- 待填写",
		"",
		"## Risks / Follow-ups",
		"- 待填写",
		"",
	}, "\n")
}

func conditionalBudgetLine(label string, value int) string {
	if value <= 0 {
		return ""
	}
	return fmt.Sprintf("%s: %d", label, value)
}

func renderBudgetValue(value int, fallback string) string {
	if value <= 0 {
		return fallback
	}
	return fmt.Sprintf("%d", value)
}
