package issuebridge

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func Main(args []string, stdout, stderr io.Writer) int {
	return (&Runner{Stdout: stdout, Stderr: stderr}).Run(args)
}

func (r *Runner) Run(args []string) int {
	if len(args) == 0 {
		r.usage()
		return 2
	}
	switch args[0] {
	case "open-stage":
		return r.runOpenStage(args[1:])
	case "start-watcher":
		return r.runStartWatcher(args[1:])
	case "stop-watcher":
		return r.runStopWatcher(args[1:])
	case "watch":
		return r.runWatch(args[1:])
	case "status":
		return r.runStatus(args[1:])
	case "handoff":
		return r.runHandoff(args[1:])
	case "worker-register":
		return r.runWorkerRegister(args[1:])
	case "worker-packet":
		return r.runWorkerPacket(args[1:])
	case "worker-next":
		return r.runWorkerNext(args[1:])
	case "worker-start":
		return r.runWorkerStart(args[1:])
	case "worker-finish":
		return r.runWorkerFinish(args[1:])
	case "worker-queue":
		return r.runWorkerQueue(args[1:])
	case "worker-set-status":
		return r.runWorkerSetStatus(args[1:])
	case "close-active":
		return r.runCloseActive(args[1:])
	case "manager-handoff":
		return r.runManagerHandoff(args[1:])
	case "claim-manual":
		return r.runClaimManual(args[1:])
	case "release-manual":
		return r.runReleaseManual(args[1:])
	case "cleanup-manual":
		return r.runCleanupManual(args[1:])
	case "touch-manual-lease":
		return r.runTouchManualLease(args[1:])
	case "manual-resume":
		return r.runManualResume(args[1:])
	case "events":
		return r.runEvents(args[1:])
	case "log-event":
		return r.runLogEvent(args[1:])
	default:
		r.usage()
		return 2
	}
}

func (r *Runner) runOpenStage(args []string) int {
	fs := flag.NewFlagSet("open-stage", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	title := fs.String("title", "", "")
	report := fs.String("report", "", "")
	reportFile := fs.String("report-file", "", "")
	decisionRequest := fs.String("decision-request", "", "")
	decisionRequestFile := fs.String("decision-request-file", "", "")
	threadID := fs.String("thread-id", "", "")
	repo := fs.String("repo", "", "")
	workdir := fs.String("workdir", ".", "")
	stateFile := fs.String("state-file", DefaultStateFile, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	apiBaseURL := fs.String("api-base-url", DefaultAPIBaseURL, "")
	watcherSessionName := fs.String("watcher-session-name", DefaultWatcherSession, "")
	dryRun := fs.Bool("dry-run", false, "")
	assignSelf := fs.Bool("assign-self", true, "")
	noAssignSelf := fs.Bool("no-assign-self", false, "")
	mentionAssignee := fs.Bool("mention-assignee", true, "")
	noMentionAssignee := fs.Bool("no-mention-assignee", false, "")
	notifyLogin := fs.String("notify-login", "", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *noAssignSelf {
		*assignSelf = false
	}
	if *noMentionAssignee {
		*mentionAssignee = false
	}
	if strings.TrimSpace(*title) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --title")
		return 2
	}

	absoluteWorkdir, err := filepath.Abs(*workdir)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if err := loadTokenFile(resolveInWorkdir(absoluteWorkdir, *tokenFile)); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	reportText, err := loadTextArg(*report, resolveOptionalInWorkdir(absoluteWorkdir, *reportFile))
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	decisionRequestText, err := loadTextArg(*decisionRequest, resolveOptionalInWorkdir(absoluteWorkdir, *decisionRequestFile))
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	repoFullName, err := getRepoFullName(absoluteWorkdir, *repo)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	resolvedThreadID := resolveThreadID(*threadID, nil)
	if resolvedThreadID == "" {
		fmt.Fprintln(r.Stderr, "缺少 thread id。请传 --thread-id 或设置 CODEX_THREAD_ID。")
		return 1
	}

	login := strings.TrimSpace(*notifyLogin)
	client := &GitHubClient{BaseURL: *apiBaseURL, Token: githubToken()}
	if !*dryRun && (*assignSelf || *mentionAssignee) && login == "" {
		login, err = githubLogin(client)
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
	}
	body := renderIssueBody(reportText, decisionRequestText, resolvedThreadID, conditionalString(*mentionAssignee, login))

	if *dryRun {
		output := map[string]any{
			"repo_full_name":   repoFullName,
			"thread_id":        resolvedThreadID,
			"title":            *title,
			"body":             body,
			"decision_request": decisionRequestText,
			"state_file":       resolveInWorkdir(absoluteWorkdir, *stateFile),
		}
		r.printJSON(output)
		return 0
	}

	if err := requireTokenForDefaultAPI(*apiBaseURL); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	statePath := resolveInWorkdir(absoluteWorkdir, *stateFile)
	lockPath := resolveInWorkdir(absoluteWorkdir, *lockFile)
	eventsPath := resolveInWorkdir(absoluteWorkdir, *eventsFile)
	openedIssueNumber := 0
	var state BridgeState
	err = withLock(lockPath, func() error {
		payload := CreateIssueRequest{Title: *title, Body: body}
		if *assignSelf && login != "" {
			payload.Assignees = []string{login}
		}
		issue, err := client.CreateIssue(repoFullName, payload)
		if err != nil {
			return err
		}
		state = BridgeState{
			SchemaVersion:      1,
			RepoFullName:       repoFullName,
			IssueNumber:        issue.Number,
			IssueURL:           issue.HTMLURL,
			IssueTitle:         *title,
			DecisionRequest:    decisionRequestText,
			TerminalHandoff:    renderTerminalHandoff(issue.HTMLURL, decisionRequestText),
			ThreadID:           resolvedThreadID,
			Workdir:            absoluteWorkdir,
			OpenedAtUTC:        nowUTCISO(),
			WatcherSessionName: *watcherSessionName,
		}
		openedIssueNumber = issue.Number
		waitingHandoffFields(&state)
		if err := saveBridgeState(statePath, state); err != nil {
			return err
		}
		_, err = appendEvent(eventsPath, "stage_issue_opened", true, map[string]any{
			"repo_full_name":       repoFullName,
			"issue_number":         issue.Number,
			"issue_url":            issue.HTMLURL,
			"thread_id":            resolvedThreadID,
			"watcher_session_name": *watcherSessionName,
		})
		return err
	})
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	appendCollabSyncFailure(eventsPath, "open-stage-heartbeat", syncOnlineCollabHeartbeat(absoluteWorkdir, resolvedThreadID, "waiting", collabWaitingNote(openedIssueNumber, *title)), map[string]any{
		"thread_id":    resolvedThreadID,
		"issue_number": openedIssueNumber,
	})
	appendCollabSyncFailure(eventsPath, "open-stage-progress", syncOnlineCollabProgress(absoluteWorkdir, "stage-boundary", fmt.Sprintf("已进入 waiting：%s", strings.TrimSpace(*title)), nil), map[string]any{
		"thread_id":    resolvedThreadID,
		"issue_number": openedIssueNumber,
	})
	r.printJSON(state)
	return 0
}

func (r *Runner) runStartWatcher(args []string) int {
	fs := flag.NewFlagSet("start-watcher", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	sessionName := fs.String("session-name", DefaultWatcherSession, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	tokenPath := resolveInWorkdir(workdir, *tokenFile)
	if strings.TrimSpace(os.Getenv("BABEL_GITHUB_TOKEN")) == "" &&
		strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) == "" {
		if _, err := os.Stat(tokenPath); err != nil {
			fmt.Fprintf(r.Stderr, "缺少 GitHub token。请先配置 %s 或在环境里设置 BABEL_GITHUB_TOKEN/GITHUB_TOKEN。\n", tokenPath)
			return 1
		}
	}
	command := buildWatchCommand(workdir, fs.Args())
	if tmuxSessionExistsHook(*sessionName) {
		_, _ = appendEvent(resolveInWorkdir(workdir, DefaultEventsFile), "watcher_process_start_skipped", true, map[string]any{
			"session_name": *sessionName,
		})
		r.println("watcher 已在运行: " + *sessionName)
		return 0
	}
	cmd := exec.Command("tmux", "new-session", "-d", "-s", *sessionName, command)
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	_, _ = appendEvent(resolveInWorkdir(workdir, DefaultEventsFile), "watcher_process_started", true, map[string]any{
		"session_name": *sessionName,
		"command":      command,
	})
	r.println("watcher 已启动: " + *sessionName)
	return 0
}

func (r *Runner) runStopWatcher(args []string) int {
	fs := flag.NewFlagSet("stop-watcher", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	sessionName := fs.String("session-name", DefaultWatcherSession, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	eventsPath := resolveInWorkdir(workdir, DefaultEventsFile)
	if !tmuxSessionExistsHook(*sessionName) {
		_, _ = appendEvent(eventsPath, "watcher_process_stop_skipped", true, map[string]any{
			"session_name": *sessionName,
		})
		r.println("watcher 不存在: " + *sessionName)
		return 0
	}
	if err := exec.Command("tmux", "kill-session", "-t", *sessionName).Run(); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	_, _ = appendEvent(eventsPath, "watcher_process_stopped", true, map[string]any{
		"session_name": *sessionName,
	})
	r.println("watcher 已停止: " + *sessionName)
	return 0
}

func (r *Runner) runStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateFile := fs.String("state-file", DefaultStateFile, "")
	controlFile := fs.String("control-file", DefaultControlFile, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	apiBaseURL := fs.String("api-base-url", DefaultAPIBaseURL, "")
	fetchIssue := fs.Bool("fetch-issue", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	if err := loadTokenFile(resolveInWorkdir(workdir, *tokenFile)); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	statePath := resolveInWorkdir(workdir, *stateFile)
	state, ok, err := loadBridgeState(statePath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if !ok {
		fmt.Fprintf(r.Stdout, "state file 不存在或为空：%s\n", statePath)
		return 1
	}
	r.printJSON(state)
	control, err := loadAndRefreshControl(resolveInWorkdir(workdir, *controlFile), state.ThreadID)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.println("")
	r.printJSON(map[string]any{"thread_control": control})
	if *fetchIssue {
		if err := requireTokenForDefaultAPI(*apiBaseURL); err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		client := &GitHubClient{BaseURL: *apiBaseURL, Token: githubToken()}
		issue, err := client.GetIssue(state.RepoFullName, state.IssueNumber)
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		r.println("")
		r.printJSON(map[string]any{
			"issue_state":     issue.State,
			"issue_title":     issue.Title,
			"issue_closed_at": issue.ClosedAt,
			"issue_url":       issue.HTMLURL,
		})
	}
	return 0
}

func (r *Runner) runHandoff(args []string) int {
	fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateFile := fs.String("state-file", DefaultStateFile, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	state, ok, err := loadBridgeState(resolveInWorkdir(workdir, *stateFile))
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if !ok {
		fmt.Fprintf(r.Stderr, "state file 不存在或为空：%s\n", resolveInWorkdir(workdir, *stateFile))
		return 1
	}
	if !handoffIsWaiting(state) {
		fmt.Fprintln(r.Stderr, "当前没有等待中的 terminal handoff")
		return 1
	}
	if strings.TrimSpace(state.TerminalHandoff) == "" {
		fmt.Fprintln(r.Stderr, "当前 state 没有 terminal handoff 信息")
		return 1
	}
	r.println(state.TerminalHandoff)
	return 0
}

func loadRegistryForCommand(path string) (ClaudeWorkerRegistry, error) {
	registry, exists, err := loadWorkerRegistry(path)
	if err != nil {
		return ClaudeWorkerRegistry{}, err
	}
	if !exists {
		return newWorkerRegistry(), nil
	}
	return registry, nil
}

func fillWorkerFromState(worker *ClaudeWorker, statePath string) {
	state, ok, err := loadBridgeState(statePath)
	if err != nil || !ok {
		return
	}
	if strings.TrimSpace(worker.RepoFullName) == "" {
		worker.RepoFullName = state.RepoFullName
	}
	if worker.IssueNumber == 0 {
		worker.IssueNumber = state.IssueNumber
	}
	if strings.TrimSpace(worker.IssueURL) == "" {
		worker.IssueURL = state.IssueURL
	}
	if strings.TrimSpace(worker.ManagerThreadID) == "" {
		worker.ManagerThreadID = state.ThreadID
	}
	if strings.TrimSpace(worker.Workdir) == "" {
		worker.Workdir = state.Workdir
	}
}

func (r *Runner) runWorkerRegister(args []string) int {
	fs := flag.NewFlagSet("worker-register", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	stateFile := fs.String("state-file", DefaultStateFile, "")
	workerID := fs.String("worker-id", "", "")
	status := fs.String("status", WorkerStatusQueued, "")
	lane := fs.String("lane", "", "")
	taskLevel := fs.String("task-level", "", "")
	maxFiles := fs.Int("max-files", 0, "")
	maxDeltaLines := fs.Int("max-delta-lines", 0, "")
	sessionID := fs.String("session-id", "", "")
	model := fs.String("model", "", "")
	taskTitle := fs.String("task-title", "", "")
	taskSummary := fs.String("task-summary", "", "")
	repoFullName := fs.String("repo-full-name", "", "")
	issueNumber := fs.Int("issue-number", 0, "")
	issueURL := fs.String("issue-url", "", "")
	managerThreadID := fs.String("manager-thread-id", "", "")
	var readScope stringSlice
	var writeScope stringSlice
	var testCommands stringSlice
	fs.Var(&readScope, "read-scope", "")
	fs.Var(&writeScope, "write-scope", "")
	fs.Var(&testCommands, "test-command", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workerID) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --worker-id")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	normalizedTaskLevel, resolvedMaxFiles, resolvedMaxDeltaLines := applyTaskLevelDefaults(*taskLevel, *maxFiles, *maxDeltaLines)
	worker := ClaudeWorker{
		WorkerID:        *workerID,
		Status:          *status,
		Lane:            strings.TrimSpace(*lane),
		TaskLevel:       normalizedTaskLevel,
		MaxFiles:        resolvedMaxFiles,
		MaxDeltaLines:   resolvedMaxDeltaLines,
		ReadScope:       []string(readScope),
		WriteScope:      []string(writeScope),
		TestCommands:    []string(testCommands),
		SessionID:       strings.TrimSpace(*sessionID),
		Model:           strings.TrimSpace(*model),
		TaskTitle:       strings.TrimSpace(*taskTitle),
		TaskSummary:     strings.TrimSpace(*taskSummary),
		RepoFullName:    strings.TrimSpace(*repoFullName),
		IssueNumber:     *issueNumber,
		IssueURL:        strings.TrimSpace(*issueURL),
		ManagerThreadID: strings.TrimSpace(*managerThreadID),
		Workdir:         workdir,
	}
	fillWorkerFromState(&worker, resolveInWorkdir(workdir, *stateFile))
	worker = upsertWorker(&registry, worker)
	if err := saveWorkerRegistry(registryPath, registry); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(worker)
	return 0
}

func (r *Runner) runWorkerPacket(args []string) int {
	fs := flag.NewFlagSet("worker-packet", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	stateFile := fs.String("state-file", DefaultStateFile, "")
	packetRoot := fs.String("packet-root", DefaultWorkerPacketRoot, "")
	workerID := fs.String("worker-id", "", "")
	lane := fs.String("lane", "", "")
	taskLevel := fs.String("task-level", "", "")
	maxFiles := fs.Int("max-files", 0, "")
	maxDeltaLines := fs.Int("max-delta-lines", 0, "")
	sessionID := fs.String("session-id", "", "")
	model := fs.String("model", "", "")
	taskTitle := fs.String("task-title", "", "")
	taskSummary := fs.String("task-summary", "", "")
	taskSummaryFile := fs.String("task-summary-file", "", "")
	goal := fs.String("goal", "", "")
	goalFile := fs.String("goal-file", "", "")
	repoFullName := fs.String("repo-full-name", "", "")
	issueNumber := fs.Int("issue-number", 0, "")
	issueURL := fs.String("issue-url", "", "")
	managerThreadID := fs.String("manager-thread-id", "", "")
	resetReport := fs.Bool("reset-report", false, "")
	var acceptance stringSlice
	var constraints stringSlice
	var contextFiles stringSlice
	var deliverables stringSlice
	var readScope stringSlice
	var writeScope stringSlice
	var testCommands stringSlice
	fs.Var(&acceptance, "acceptance", "")
	fs.Var(&constraints, "constraint", "")
	fs.Var(&contextFiles, "context-file", "")
	fs.Var(&deliverables, "deliverable", "")
	fs.Var(&readScope, "read-scope", "")
	fs.Var(&writeScope, "write-scope", "")
	fs.Var(&testCommands, "test-command", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workerID) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --worker-id")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	taskSummaryText, err := loadTextArg(*taskSummary, resolveOptionalInWorkdir(workdir, *taskSummaryFile))
	if err != nil && (strings.TrimSpace(*taskSummary) != "" || strings.TrimSpace(*taskSummaryFile) != "") {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	goalText, err := loadTextArg(*goal, resolveOptionalInWorkdir(workdir, *goalFile))
	if err != nil && (strings.TrimSpace(*goal) != "" || strings.TrimSpace(*goalFile) != "") {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	status := WorkerStatusQueued
	if idx := findWorkerIndex(registry.Workers, *workerID); idx >= 0 {
		status = registry.Workers[idx].Status
	}
	normalizedTaskLevel, resolvedMaxFiles, resolvedMaxDeltaLines := applyTaskLevelDefaults(*taskLevel, *maxFiles, *maxDeltaLines)
	worker := ClaudeWorker{
		WorkerID:        *workerID,
		Status:          status,
		Lane:            strings.TrimSpace(*lane),
		TaskLevel:       normalizedTaskLevel,
		MaxFiles:        resolvedMaxFiles,
		MaxDeltaLines:   resolvedMaxDeltaLines,
		ReadScope:       []string(readScope),
		WriteScope:      []string(writeScope),
		TestCommands:    []string(testCommands),
		SessionID:       strings.TrimSpace(*sessionID),
		Model:           strings.TrimSpace(*model),
		TaskTitle:       strings.TrimSpace(*taskTitle),
		TaskSummary:     strings.TrimSpace(taskSummaryText),
		RepoFullName:    strings.TrimSpace(*repoFullName),
		IssueNumber:     *issueNumber,
		IssueURL:        strings.TrimSpace(*issueURL),
		ManagerThreadID: strings.TrimSpace(*managerThreadID),
		Workdir:         workdir,
		PacketFile:      workerPacketFilePath(workdir, *packetRoot, *workerID),
		ReportFile:      workerReportFilePath(workdir, *packetRoot, *workerID),
	}
	fillWorkerFromState(&worker, resolveInWorkdir(workdir, *stateFile))
	worker = upsertWorker(&registry, worker)

	if err := ensureParentDir(worker.PacketFile); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	packetText := renderWorkerPacket(worker, goalText, []string(acceptance), []string(constraints), []string(contextFiles), []string(deliverables))
	if err := os.WriteFile(worker.PacketFile, []byte(packetText), 0o644); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if *resetReport || !fileExists(worker.ReportFile) {
		reportText := renderWorkerReportTemplate(worker)
		if err := os.WriteFile(worker.ReportFile, []byte(reportText), 0o644); err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
	}
	if err := saveWorkerRegistry(registryPath, registry); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(worker)
	return 0
}

func (r *Runner) runWorkerNext(args []string) int {
	fs := flag.NewFlagSet("worker-next", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	allActionable := fs.Bool("all-actionable", false, "")
	shellFormat := fs.Bool("shell", false, "")
	maxRunning := fs.Int("max-running", 1, "")
	allowSameLane := fs.Bool("allow-same-lane", false, "")
	workerPrefix := fs.String("worker-prefix", "", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	worker, ok := nextDispatchableWorkerWithPrefix(registry, *allActionable, *maxRunning, *allowSameLane, *workerPrefix)
	if !ok {
		fmt.Fprintln(r.Stderr, "没有可分配的 worker")
		return 1
	}
	if *shellFormat {
		r.println(
			"WORKER_ID="+shellQuote(worker.WorkerID),
			"STATUS="+shellQuote(worker.Status),
			"LANE="+shellQuote(worker.Lane),
			"TASK_LEVEL="+shellQuote(worker.TaskLevel),
			"MAX_FILES="+shellQuote(fmt.Sprintf("%d", worker.MaxFiles)),
			"MAX_DELTA_LINES="+shellQuote(fmt.Sprintf("%d", worker.MaxDeltaLines)),
			"TASK_TITLE="+shellQuote(worker.TaskTitle),
			"TASK_SUMMARY="+shellQuote(worker.TaskSummary),
			"SESSION_ID="+shellQuote(worker.SessionID),
			"MODEL="+shellQuote(worker.Model),
			"PACKET_FILE="+shellQuote(worker.PacketFile),
			"REPORT_FILE="+shellQuote(worker.ReportFile),
			"WORKDIR="+shellQuote(worker.Workdir),
		)
		return 0
	}
	r.printJSON(worker)
	return 0
}

func (r *Runner) runWorkerStart(args []string) int {
	fs := flag.NewFlagSet("worker-start", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	stateFile := fs.String("state-file", DefaultStateFile, "")
	workerID := fs.String("worker-id", "", "")
	lane := fs.String("lane", "", "")
	taskLevel := fs.String("task-level", "", "")
	maxFiles := fs.Int("max-files", 0, "")
	maxDeltaLines := fs.Int("max-delta-lines", 0, "")
	sessionID := fs.String("session-id", "", "")
	model := fs.String("model", "", "")
	taskTitle := fs.String("task-title", "", "")
	taskSummary := fs.String("task-summary", "", "")
	repoFullName := fs.String("repo-full-name", "", "")
	issueNumber := fs.Int("issue-number", 0, "")
	issueURL := fs.String("issue-url", "", "")
	managerThreadID := fs.String("manager-thread-id", "", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workerID) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --worker-id")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	normalizedTaskLevel, resolvedMaxFiles, resolvedMaxDeltaLines := applyTaskLevelDefaults(*taskLevel, *maxFiles, *maxDeltaLines)
	worker := ClaudeWorker{
		WorkerID:        *workerID,
		Status:          WorkerStatusRunning,
		Lane:            strings.TrimSpace(*lane),
		TaskLevel:       normalizedTaskLevel,
		MaxFiles:        resolvedMaxFiles,
		MaxDeltaLines:   resolvedMaxDeltaLines,
		SessionID:       strings.TrimSpace(*sessionID),
		Model:           strings.TrimSpace(*model),
		TaskTitle:       strings.TrimSpace(*taskTitle),
		TaskSummary:     strings.TrimSpace(*taskSummary),
		RepoFullName:    strings.TrimSpace(*repoFullName),
		IssueNumber:     *issueNumber,
		IssueURL:        strings.TrimSpace(*issueURL),
		ManagerThreadID: strings.TrimSpace(*managerThreadID),
		Workdir:         workdir,
		StartedAtUTC:    nowUTCISO(),
	}
	fillWorkerFromState(&worker, resolveInWorkdir(workdir, *stateFile))
	worker = upsertWorker(&registry, worker)
	if strings.TrimSpace(worker.StartedAtUTC) == "" {
		worker.StartedAtUTC = nowUTCISO()
		worker = upsertWorker(&registry, worker)
	}
	if err := saveWorkerRegistry(registryPath, registry); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(worker)
	return 0
}

func (r *Runner) runWorkerFinish(args []string) int {
	fs := flag.NewFlagSet("worker-finish", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	workerID := fs.String("worker-id", "", "")
	comment := fs.String("comment", "", "")
	commentFile := fs.String("comment-file", "", "")
	annotateComment := fs.Bool("annotate-comment", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workerID) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --worker-id")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	idx := findWorkerIndex(registry.Workers, *workerID)
	if idx < 0 {
		fmt.Fprintln(r.Stderr, "worker 不存在")
		return 1
	}
	worker := registry.Workers[idx]
	resolvedCommentFile := strings.TrimSpace(*commentFile)
	if resolvedCommentFile == "" && strings.TrimSpace(*comment) == "" && strings.TrimSpace(worker.ReportFile) != "" {
		resolvedCommentFile = strings.TrimSpace(worker.ReportFile)
	}
	worker.Status = WorkerStatusHandoffQueued
	worker.FinishedAtUTC = nowUTCISO()
	worker.LastCommentFile = resolvedCommentFile
	registry.Workers[idx] = worker
	worker = upsertWorker(&registry, worker)
	if err := saveWorkerRegistry(registryPath, registry); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}

	forwarded := []string{}
	if strings.TrimSpace(*comment) != "" {
		forwarded = append(forwarded, "--comment", *comment)
	}
	if resolvedCommentFile != "" {
		forwarded = append(forwarded, "--comment-file", resolvedCommentFile)
	}
	if *annotateComment {
		forwarded = append(forwarded, "--annotate-comment")
	}
	return r.runManagerHandoff(forwarded)
}

func (r *Runner) runWorkerQueue(args []string) int {
	fs := flag.NewFlagSet("worker-queue", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	all := fs.Bool("all", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if *all {
		workers := append([]ClaudeWorker{}, registry.Workers...)
		sortWorkersForQueue(workers)
		r.printJSON(workers)
		return 0
	}
	r.printJSON(actionableWorkers(registry))
	return 0
}

func (r *Runner) runWorkerSetStatus(args []string) int {
	fs := flag.NewFlagSet("worker-set-status", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	registryFile := fs.String("registry-file", DefaultWorkerRegistryFile, "")
	workerID := fs.String("worker-id", "", "")
	status := fs.String("status", "", "")
	note := fs.String("note", "", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*workerID) == "" || strings.TrimSpace(*status) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --worker-id 或 --status")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	registryPath := resolveInWorkdir(workdir, *registryFile)
	registry, err := loadRegistryForCommand(registryPath)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	idx := findWorkerIndex(registry.Workers, *workerID)
	if idx < 0 {
		fmt.Fprintln(r.Stderr, "worker 不存在")
		return 1
	}
	worker := registry.Workers[idx]
	worker.Status = strings.TrimSpace(*status)
	worker.LastNote = strings.TrimSpace(*note)
	if worker.Status == WorkerStatusDone || worker.Status == WorkerStatusCancelled {
		if strings.TrimSpace(worker.FinishedAtUTC) == "" {
			worker.FinishedAtUTC = nowUTCISO()
		}
	}
	registry.Workers[idx] = worker
	worker = upsertWorker(&registry, worker)
	if err := saveWorkerRegistry(registryPath, registry); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(worker)
	return 0
}

func (r *Runner) runCloseActive(args []string) int {
	fs := flag.NewFlagSet("close-active", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateFile := fs.String("state-file", DefaultStateFile, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	apiBaseURL := fs.String("api-base-url", DefaultAPIBaseURL, "")
	comment := fs.String("comment", "", "")
	commentFile := fs.String("comment-file", "", "")
	closeReason := fs.String("close-reason", "terminal_reply", "")
	annotateComment := fs.Bool("annotate-comment", false, "")
	force := fs.Bool("force", false, "")
	dryRun := fs.Bool("dry-run", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	if err := loadTokenFile(resolveInWorkdir(workdir, *tokenFile)); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if err := requireTokenForDefaultAPI(*apiBaseURL); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	client := &GitHubClient{BaseURL: *apiBaseURL, Token: githubToken()}
	statePath := resolveInWorkdir(workdir, *stateFile)
	lockPath := resolveInWorkdir(workdir, *lockFile)
	eventsPath := resolveInWorkdir(workdir, *eventsFile)
	closedThreadID := ""
	closedIssueNumber := 0
	var output map[string]any
	err := withLock(lockPath, func() error {
		state, ok, err := loadBridgeState(statePath)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("state file 不存在或为空：%s", statePath)
		}
		issue, err := client.GetIssue(state.RepoFullName, state.IssueNumber)
		if err != nil {
			return err
		}
		if issue.State == "closed" {
			_, _ = appendEvent(eventsPath, "active_issue_close_skipped_already_closed", true, map[string]any{
				"issue_number": state.IssueNumber,
				"issue_url":    issue.HTMLURL,
			})
			output = map[string]any{
				"issue_number":    state.IssueNumber,
				"issue_state":     "closed",
				"comment_created": false,
			}
			return nil
		}
		if !handoffIsWaiting(state) && !*force {
			return fmt.Errorf("当前没有等待中的 handoff；这条终端消息不应消费阶段 issue")
		}
		closedThreadID = state.ThreadID
		closedIssueNumber = state.IssueNumber

		commentText := ""
		switch *closeReason {
		case "manual_takeover":
			commentText = renderManualTakeoverCloseComment()
		default:
			if strings.TrimSpace(*comment) != "" || strings.TrimSpace(*commentFile) != "" {
				commentText, err = loadTextArg(*comment, resolveOptionalInWorkdir(workdir, *commentFile))
				if err != nil {
					return err
				}
				if *annotateComment {
					commentText = renderAnnotatedTerminalCloseComment(commentText)
				}
			}
		}

		if *dryRun {
			output = map[string]any{
				"issue_number": state.IssueNumber,
				"issue_url":    issue.HTMLURL,
				"comment":      emptyToNil(commentText),
				"close":        true,
			}
			return nil
		}

		var createdComment *Comment
		if strings.TrimSpace(commentText) != "" {
			comment, err := client.CreateIssueComment(state.RepoFullName, state.IssueNumber, commentText)
			if err != nil {
				return err
			}
			createdComment = &comment
			if shouldMarkCreatedCommentConsumed(*closeReason) {
				markCommentConsumed(&state, comment)
				if err := saveBridgeState(statePath, state); err != nil {
					return err
				}
			}
		}
		if err := client.CloseIssue(state.RepoFullName, state.IssueNumber); err != nil {
			return err
		}
		markHandoffConsumed(&state, handoffConsumedStatusForCloseReason(*closeReason))
		state.ClosedByActiveTerminalAtUTC = nowUTCISO()
		if err := saveBridgeState(statePath, state); err != nil {
			return err
		}
		fields := map[string]any{
			"issue_number":    state.IssueNumber,
			"issue_url":       issue.HTMLURL,
			"close_reason":    *closeReason,
			"handoff_status":  state.HandoffStatus,
			"comment_created": createdComment != nil,
		}
		if createdComment != nil {
			fields["comment_id"] = createdComment.ID
			fields["comment_url"] = createdComment.HTMLURL
		}
		_, _ = appendEvent(eventsPath, "active_issue_closed", true, fields)
		output = map[string]any{
			"issue_number":    state.IssueNumber,
			"issue_url":       issue.HTMLURL,
			"comment_created": createdComment != nil,
			"comment_url":     conditionalCommentURL(createdComment),
			"issue_state":     "closed",
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if !*dryRun && strings.TrimSpace(closedThreadID) != "" {
		switch *closeReason {
		case "manager_handoff":
			appendCollabSyncFailure(eventsPath, "manager-handoff-heartbeat", syncOnlineCollabHeartbeat(workdir, closedThreadID, "waiting", fmt.Sprintf("manager handoff queued from issue #%d", closedIssueNumber)), map[string]any{
				"thread_id":    closedThreadID,
				"issue_number": closedIssueNumber,
				"close_reason": *closeReason,
			})
		case "manual_takeover":
		default:
			appendCollabSyncFailure(eventsPath, "close-active-heartbeat", syncOnlineCollabHeartbeat(workdir, closedThreadID, "active", fmt.Sprintf("terminal handoff consumed from issue #%d", closedIssueNumber)), map[string]any{
				"thread_id":    closedThreadID,
				"issue_number": closedIssueNumber,
				"close_reason": *closeReason,
			})
		}
	}
	r.printJSON(output)
	return 0
}

func (r *Runner) runManagerHandoff(args []string) int {
	fs := flag.NewFlagSet("manager-handoff", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateFile := fs.String("state-file", DefaultStateFile, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	apiBaseURL := fs.String("api-base-url", DefaultAPIBaseURL, "")
	comment := fs.String("comment", "", "")
	commentFile := fs.String("comment-file", "", "")
	annotateComment := fs.Bool("annotate-comment", false, "")
	dryRun := fs.Bool("dry-run", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	forwarded := []string{
		"--state-file", *stateFile,
		"--token-file", *tokenFile,
		"--lock-file", *lockFile,
		"--events-file", *eventsFile,
		"--api-base-url", *apiBaseURL,
		"--close-reason", "manager_handoff",
	}
	if strings.TrimSpace(*comment) != "" {
		forwarded = append(forwarded, "--comment", *comment)
	}
	if strings.TrimSpace(*commentFile) != "" {
		forwarded = append(forwarded, "--comment-file", *commentFile)
	}
	if *annotateComment {
		forwarded = append(forwarded, "--annotate-comment")
	}
	if *dryRun {
		forwarded = append(forwarded, "--dry-run")
	}
	return r.runCloseActive(forwarded)
}

func (r *Runner) runManualResume(args []string) int {
	fs := flag.NewFlagSet("manual-resume", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	threadID := fs.String("thread-id", "", "")
	entrypoint := fs.String("entrypoint", "manual_cli", "")
	stateFile := fs.String("state-file", DefaultStateFile, "")
	controlFile := fs.String("control-file", DefaultControlFile, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	leaseDir := fs.String("lease-dir", DefaultManualLeaseDir, "")
	leaseSessionID := fs.String("lease-session-id", "", "")
	leaseTTLSeconds := fs.Int("lease-ttl-seconds", 8, "")
	leaseCheckSeconds := fs.Int("lease-check-seconds", 2, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	resolvedThreadID := resolveThreadID(*threadID, nil)
	if resolvedThreadID == "" {
		fmt.Fprintln(r.Stderr, "缺少 thread id。请传 --thread-id 或设置 CODEX_THREAD_ID。")
		return 1
	}
	statePath := resolveInWorkdir(workdir, *stateFile)
	controlPath := resolveInWorkdir(workdir, *controlFile)
	tokenPath := resolveInWorkdir(workdir, *tokenFile)
	lockPath := resolveInWorkdir(workdir, *lockFile)
	eventsPath := resolveInWorkdir(workdir, *eventsFile)
	leasePath := manualLeasePath(workdir, *leaseDir, *entrypoint)

	_, _ = appendEvent(eventsPath, "manual_resume_invoked", true, map[string]any{
		"thread_id":  resolvedThreadID,
		"entrypoint": *entrypoint,
	})
	appendCollabSyncFailure(eventsPath, "manual-resume-heartbeat-start", syncOnlineCollabHeartbeat(workdir, resolvedThreadID, "manual-active", fmt.Sprintf("manual resume via %s", *entrypoint)), map[string]any{
		"thread_id":  resolvedThreadID,
		"entrypoint": *entrypoint,
	})

	cleanupRunner := &Runner{Stdout: io.Discard, Stderr: io.Discard}
	_ = cleanupRunner.Run([]string{
		"cleanup-manual",
		"--thread-id", resolvedThreadID,
		"--entrypoint", *entrypoint,
		"--control-file", controlPath,
		"--lock-file", lockPath,
		"--events-file", eventsPath,
		"--exclude-pid", fmt.Sprintf("%d", os.Getpid()),
		"--exclude-pgid", fmt.Sprintf("%d", syscall.Getpgrp()),
	})

	claimRunner := &Runner{Stdout: io.Discard, Stderr: io.Discard}
	if claimRunner.Run([]string{
		"claim-manual",
		"--thread-id", resolvedThreadID,
		"--state-file", statePath,
		"--control-file", controlPath,
		"--lock-file", lockPath,
		"--events-file", eventsPath,
		"--owner-detail", *entrypoint,
	}) != 0 {
		fmt.Fprintln(r.Stderr, "manual claim 失败")
		return 1
	}

	if err := loadTokenFile(tokenPath); err == nil {
		closeRunner := &Runner{Stdout: io.Discard, Stderr: io.Discard}
		_ = closeRunner.Run([]string{
			"close-active",
			"--state-file", statePath,
			"--token-file", tokenPath,
			"--lock-file", lockPath,
			"--events-file", eventsPath,
			"--close-reason", "manual_takeover",
		})
	}

	defer func() {
		releaseRunner := &Runner{Stdout: io.Discard, Stderr: io.Discard}
		_ = releaseRunner.Run([]string{
			"release-manual",
			"--control-file", controlPath,
			"--lock-file", lockPath,
			"--events-file", eventsPath,
			"--reason", "manual_session_finished",
		})
		_, _ = appendEvent(eventsPath, "manual_resume_finished", true, map[string]any{
			"thread_id":  resolvedThreadID,
			"entrypoint": *entrypoint,
		})
		appendCollabSyncFailure(eventsPath, "manual-resume-heartbeat-finish", syncOnlineCollabHeartbeat(workdir, resolvedThreadID, "idle", fmt.Sprintf("manual session finished via %s", *entrypoint)), map[string]any{
			"thread_id":  resolvedThreadID,
			"entrypoint": *entrypoint,
		})
	}()

	cmd := exec.Command("codex", append([]string{
		"--ask-for-approval", "never",
		"--sandbox", "danger-full-access",
		"-c", "model_reasoning_effort=xhigh",
		"-C", workdir,
		"resume",
		resolvedThreadID,
	}, fs.Args()...)...)
	cmd.Dir = workdir
	cmd.Stdin = os.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signalCh)

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	leaseExpiredCh := make(chan string, 1)
	if strings.TrimSpace(*leaseSessionID) != "" && *leaseTTLSeconds > 0 {
		go monitorManualLease(leasePath, resolvedThreadID, *entrypoint, *leaseSessionID, time.Duration(*leaseTTLSeconds)*time.Second, time.Duration(maxInt(*leaseCheckSeconds, 1))*time.Second, leaseExpiredCh)
	}

	for {
		select {
		case sig := <-signalCh:
			if cmd.Process == nil {
				continue
			}
			signalValue, ok := sig.(syscall.Signal)
			if !ok {
				signalValue = syscall.SIGTERM
			}
			signalProcessGroup(cmd.Process.Pid, signalValue)
			go forceKillProcessGroupAfter(cmd.Process.Pid, 2*time.Second)
		case reason := <-leaseExpiredCh:
			_, _ = appendEvent(eventsPath, "manual_lease_expired", true, map[string]any{
				"thread_id":    resolvedThreadID,
				"entrypoint":   *entrypoint,
				"lease_path":   leasePath,
				"lease_reason": reason,
			})
			if cmd.Process != nil {
				signalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
				go forceKillProcessGroupAfter(cmd.Process.Pid, 2*time.Second)
			}
		case err := <-waitCh:
			if strings.TrimSpace(*leaseSessionID) != "" {
				clearMatchingManualLease(leasePath, *leaseSessionID)
			}
			if err == nil {
				return 0
			}
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
	}
}

func monitorManualLease(leasePath, threadID, entrypoint, sessionID string, ttl, interval time.Duration, expiredCh chan<- string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		lease, exists, err := loadManualLease(leasePath)
		if err != nil || !exists {
			select {
			case expiredCh <- "missing":
			default:
			}
			return
		}
		if strings.TrimSpace(lease.ThreadID) != strings.TrimSpace(threadID) || strings.TrimSpace(lease.Entrypoint) != strings.TrimSpace(entrypoint) {
			select {
			case expiredCh <- "mismatch":
			default:
			}
			return
		}
		if strings.TrimSpace(lease.SessionID) != strings.TrimSpace(sessionID) {
			select {
			case expiredCh <- "session_replaced":
			default:
			}
			return
		}
		updatedAt, err := time.Parse("2006-01-02T15:04:05Z", lease.UpdatedAtUTC)
		if err != nil {
			select {
			case expiredCh <- "invalid_timestamp":
			default:
			}
			return
		}
		if time.Since(updatedAt) > ttl {
			select {
			case expiredCh <- "ttl_expired":
			default:
			}
			return
		}
	}
}

func clearMatchingManualLease(path, sessionID string) {
	lease, exists, err := loadManualLease(path)
	if err != nil || !exists {
		return
	}
	if strings.TrimSpace(lease.SessionID) != strings.TrimSpace(sessionID) {
		return
	}
	_ = os.Remove(path)
}

func signalProcessGroup(pid int, sig syscall.Signal) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, sig)
}

func forceKillProcessGroupAfter(pid int, delay time.Duration) {
	time.Sleep(delay)
	signalProcessGroup(pid, syscall.SIGKILL)
}

func (r *Runner) runClaimManual(args []string) int {
	fs := flag.NewFlagSet("claim-manual", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	threadID := fs.String("thread-id", "", "")
	stateFile := fs.String("state-file", DefaultStateFile, "")
	controlFile := fs.String("control-file", DefaultControlFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	ownerDetail := fs.String("owner-detail", "manual_cli", "")
	interruptAuto := fs.Bool("interrupt-auto", true, "")
	noInterruptAuto := fs.Bool("no-interrupt-auto", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *noInterruptAuto {
		*interruptAuto = false
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	state, _, err := loadBridgeState(resolveInWorkdir(workdir, *stateFile))
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	resolvedThreadID := resolveThreadID(*threadID, &state)
	if resolvedThreadID == "" {
		fmt.Fprintln(r.Stderr, "缺少 thread id。请传 --thread-id、准备好 state file，或设置 CODEX_THREAD_ID。")
		return 1
	}
	controlPath := resolveInWorkdir(workdir, *controlFile)
	lockPath := resolveInWorkdir(workdir, *lockFile)
	eventsPath := resolveInWorkdir(workdir, *eventsFile)
	var control ThreadControl
	err = withLock(lockPath, func() error {
		current, err := loadAndRefreshControl(controlPath, resolvedThreadID)
		if err != nil {
			return err
		}
		interruptedAutoSession := ""
		if *interruptAuto && current.Owner == "watcher" {
			if killTmuxSession(current.ActiveTmuxSession) {
				interruptedAutoSession = current.ActiveTmuxSession
			}
		}
		control = manualControlState(resolvedThreadID, *ownerDetail, interruptedAutoSession)
		if err := saveThreadControl(controlPath, control); err != nil {
			return err
		}
		_, err = appendEvent(eventsPath, "manual_claimed", true, map[string]any{
			"thread_id":                resolvedThreadID,
			"owner_detail":             *ownerDetail,
			"interrupted_auto_session": emptyToNil(interruptedAutoSession),
		})
		return err
	})
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(control)
	return 0
}

func (r *Runner) runReleaseManual(args []string) int {
	fs := flag.NewFlagSet("release-manual", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	controlFile := fs.String("control-file", DefaultControlFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	reason := fs.String("reason", "manual_session_finished", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	controlPath := resolveInWorkdir(workdir, *controlFile)
	lockPath := resolveInWorkdir(workdir, *lockFile)
	eventsPath := resolveInWorkdir(workdir, *eventsFile)
	var output ThreadControl
	err := withLock(lockPath, func() error {
		control, ok, err := loadThreadControl(controlPath)
		if err != nil {
			return err
		}
		if !ok {
			output = idleControlState("")
			return nil
		}
		control = refreshControlState(control)
		if control.Owner != "manual" {
			output = control
			return saveThreadControl(controlPath, control)
		}
		idle := idleControlState(control.ThreadID)
		idle.OwnerDetail = *reason
		idle.LastManualReleaseAtUTC = nowUTCISO()
		if err := saveThreadControl(controlPath, idle); err != nil {
			return err
		}
		_, err = appendEvent(eventsPath, "manual_released", true, map[string]any{
			"thread_id": control.ThreadID,
			"reason":    *reason,
		})
		output = idle
		return err
	})
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(output)
	return 0
}

func (r *Runner) runCleanupManual(args []string) int {
	fs := flag.NewFlagSet("cleanup-manual", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	threadID := fs.String("thread-id", "", "")
	entrypoint := fs.String("entrypoint", "", "")
	controlFile := fs.String("control-file", DefaultControlFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	excludePID := fs.Int("exclude-pid", 0, "")
	excludePGID := fs.Int("exclude-pgid", 0, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*entrypoint) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --entrypoint")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	controlPath := resolveInWorkdir(workdir, *controlFile)
	lockPath := resolveInWorkdir(workdir, *lockFile)
	eventsPath := resolveInWorkdir(workdir, *eventsFile)
	targetThreadID := strings.TrimSpace(*threadID)

	var result map[string]any
	err := withLock(lockPath, func() error {
		processes, err := listManualResumeProcesses()
		if err != nil {
			return err
		}
		targets := matchingManualCleanupTargets(processes, targetThreadID, *entrypoint, *excludePID, *excludePGID)
		killed := cleanupManualTargets(targets)
		result = map[string]any{
			"entrypoint":   *entrypoint,
			"thread_id":    emptyToNil(targetThreadID),
			"exclude_pid":  *excludePID,
			"exclude_pgid": *excludePGID,
			"matched_pids": manualCleanupTargetPIDs(targets),
			"killed_pgids": killed,
		}

		control, ok, err := loadThreadControl(controlPath)
		if err != nil {
			return err
		}
		if ok && control.Owner == "manual" && control.OwnerDetail == *entrypoint {
			idle := idleControlState(targetThreadID)
			if idle.ThreadID == "" {
				idle.ThreadID = control.ThreadID
			}
			idle.OwnerDetail = "cleanup_manual_takeover"
			idle.LastManualReleaseAtUTC = nowUTCISO()
			if err := saveThreadControl(controlPath, idle); err != nil {
				return err
			}
		}
		if len(killed) > 0 {
			_, err = appendEvent(eventsPath, "manual_cleanup", true, map[string]any{
				"entrypoint":   *entrypoint,
				"thread_id":    emptyToNil(targetThreadID),
				"exclude_pid":  *excludePID,
				"exclude_pgid": *excludePGID,
				"matched_pids": manualCleanupTargetPIDs(targets),
				"killed_pgids": killed,
			})
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if result == nil {
		result = map[string]any{
			"entrypoint":   *entrypoint,
			"thread_id":    emptyToNil(targetThreadID),
			"exclude_pid":  *excludePID,
			"exclude_pgid": *excludePGID,
			"matched_pids": []int{},
			"killed_pgids": []int{},
		}
	}
	r.printJSON(result)
	return 0
}

func (r *Runner) runTouchManualLease(args []string) int {
	fs := flag.NewFlagSet("touch-manual-lease", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	threadID := fs.String("thread-id", "", "")
	entrypoint := fs.String("entrypoint", "manual_cli", "")
	sessionID := fs.String("session-id", "", "")
	leaseDir := fs.String("lease-dir", DefaultManualLeaseDir, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*sessionID) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --session-id")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	resolvedThreadID := resolveThreadID(*threadID, nil)
	if resolvedThreadID == "" {
		fmt.Fprintln(r.Stderr, "缺少 thread id。请传 --thread-id 或设置 CODEX_THREAD_ID。")
		return 1
	}
	leasePath := manualLeasePath(workdir, *leaseDir, *entrypoint)
	lease := ManualLease{
		SchemaVersion: 1,
		ThreadID:      resolvedThreadID,
		Entrypoint:    *entrypoint,
		SessionID:     *sessionID,
		UpdatedAtUTC:  nowUTCISO(),
	}
	if err := saveManualLease(leasePath, lease); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	r.printJSON(map[string]any{
		"lease_path": leasePath,
		"session_id": *sessionID,
		"thread_id":  resolvedThreadID,
		"entrypoint": *entrypoint,
	})
	return 0
}

func (r *Runner) runWatch(args []string) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateFile := fs.String("state-file", DefaultStateFile, "")
	controlFile := fs.String("control-file", DefaultControlFile, "")
	tokenFile := fs.String("token-file", DefaultTokenFile, "")
	lockFile := fs.String("lock-file", DefaultLockFile, "")
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	apiBaseURL := fs.String("api-base-url", DefaultAPIBaseURL, "")
	pollSeconds := fs.Int("poll-seconds", DefaultPollSeconds, "")
	watcherSessionName := fs.String("watcher-session-name", DefaultWatcherSession, "")
	resumeSessionPrefix := fs.String("resume-session-prefix", DefaultResumeSessionPrefix, "")
	resumeMode := fs.String("resume-mode", "interactive", "")
	once := fs.Bool("once", false, "")
	resumeDryRun := fs.Bool("resume-dry-run", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *resumeMode != "interactive" && *resumeMode != "exec" {
		fmt.Fprintln(r.Stderr, "invalid --resume-mode: use interactive or exec")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	if err := loadTokenFile(resolveInWorkdir(workdir, *tokenFile)); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if err := requireTokenForDefaultAPI(*apiBaseURL); err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	client := &GitHubClient{BaseURL: *apiBaseURL, Token: githubToken()}
	statePath := resolveInWorkdir(workdir, *stateFile)
	controlPath := resolveInWorkdir(workdir, *controlFile)
	lockPath := resolveInWorkdir(workdir, *lockFile)
	eventsPath := resolveInWorkdir(workdir, *eventsFile)

	for {
		state, ok, err := loadBridgeState(statePath)
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		if !ok {
			message := fmt.Sprintf("state file 不存在，等待中：%s", statePath)
			if *once {
				fmt.Fprintln(r.Stderr, message)
				return 1
			}
			r.println(message)
			time.Sleep(time.Duration(*pollSeconds) * time.Second)
			continue
		}

		control, err := loadAndRefreshControl(controlPath, state.ThreadID)
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		if control.Owner == "manual" {
			if *once {
				r.println("当前线程处于 manual 接管状态")
				return 0
			}
			time.Sleep(time.Duration(*pollSeconds) * time.Second)
			continue
		}
		if control.Owner == "watcher" && tmuxSessionExistsHook(control.ActiveTmuxSession) {
			if *once {
				r.println("watcher 已有活跃自动恢复会话")
				return 0
			}
			time.Sleep(time.Duration(*pollSeconds) * time.Second)
			continue
		}

		issue, err := client.GetIssue(state.RepoFullName, state.IssueNumber)
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		if issue.State != "closed" {
			if *once {
				r.println("issue 尚未关闭")
				return 0
			}
			time.Sleep(time.Duration(*pollSeconds) * time.Second)
			continue
		}
		comments, err := client.ListIssueComments(state.RepoFullName, state.IssueNumber)
		if err != nil {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		comment := newestCommentToResume(comments, state.LastResumedCommentID)
		if comment == nil {
			if *once {
				r.println("issue 已关闭，但没有新的可恢复评论")
				return 0
			}
			time.Sleep(time.Duration(*pollSeconds) * time.Second)
			continue
		}

		sleepAfterLock := false
		launchedThreadID := ""
		launchedIssueNumber := 0
		launchedCommentID := int64(0)
		err = withLock(lockPath, func() error {
			latestState, ok, err := loadBridgeState(statePath)
			if err != nil {
				return err
			}
			if !ok {
				if *once {
					r.println("state file 不存在")
					return errExit{code: 1}
				}
				sleepAfterLock = true
				return nil
			}
			if latestState.IssueNumber != state.IssueNumber {
				if *once {
					r.println("当前活动 issue 已变化")
					return errExit{code: 0}
				}
				sleepAfterLock = true
				return nil
			}
			latestControl, err := loadAndRefreshControl(controlPath, latestState.ThreadID)
			if err != nil {
				return err
			}
			if latestControl.Owner == "manual" {
				if *once {
					r.println("当前线程处于 manual 接管状态")
					return errExit{code: 0}
				}
				sleepAfterLock = true
				return nil
			}
			if latestControl.Owner == "watcher" && tmuxSessionExistsHook(latestControl.ActiveTmuxSession) {
				if *once {
					r.println("watcher 已有活跃自动恢复会话")
					return errExit{code: 0}
				}
				sleepAfterLock = true
				return nil
			}
			latestComment := newestCommentToResume(comments, latestState.LastResumedCommentID)
			if latestComment == nil {
				if *once {
					r.println("issue 已关闭，但评论已被当前终端消费")
					return errExit{code: 0}
				}
				sleepAfterLock = true
				return nil
			}

			prompt := renderResumePrompt(latestState, issue, *latestComment)
			sessionName := "(dry-run)"
			if *resumeDryRun {
				r.println(resumeShellCommand(latestState.Workdir, latestState.ThreadID, prompt, *resumeMode))
			} else {
				sessionName, err = launchResumeInTmux(latestState.Workdir, latestState.ThreadID, prompt, latestState.IssueNumber, *resumeSessionPrefix, *resumeMode)
				if err != nil {
					return err
				}
				r.printJSON(map[string]any{
					"issue_number": latestState.IssueNumber,
					"comment_id":   latestComment.ID,
					"tmux_session": sessionName,
				})
			}

			markCommentConsumed(&latestState, *latestComment)
			markHandoffConsumed(&latestState, "consumed_by_watcher")
			latestState.LastResumeTmuxSession = sessionName
			launchedThreadID = latestState.ThreadID
			launchedIssueNumber = latestState.IssueNumber
			launchedCommentID = latestComment.ID
			if err := saveBridgeState(statePath, latestState); err != nil {
				return err
			}
			_, err = appendEvent(eventsPath, "watcher_resume_launched", true, map[string]any{
				"issue_number":   latestState.IssueNumber,
				"issue_url":      issue.HTMLURL,
				"comment_id":     latestComment.ID,
				"comment_url":    latestComment.HTMLURL,
				"tmux_session":   sessionName,
				"thread_id":      latestState.ThreadID,
				"resume_dry_run": *resumeDryRun,
				"resume_mode":    *resumeMode,
			})
			if err != nil {
				return err
			}
			if !*resumeDryRun {
				if err := saveThreadControl(controlPath, watcherControlState(latestState.ThreadID, *watcherSessionName, sessionName)); err != nil {
					return err
				}
			}
			return nil
		})
		var exitErr errExit
		if err != nil && !asErrExit(err, &exitErr) {
			fmt.Fprintln(r.Stderr, err)
			return 1
		}
		if exitErr.set {
			return exitErr.code
		}
		if *once {
			return 0
		}
		if strings.TrimSpace(launchedThreadID) != "" {
			appendCollabSyncFailure(eventsPath, "watch-heartbeat", syncOnlineCollabHeartbeat(workdir, launchedThreadID, "auto-active", fmt.Sprintf("watcher resumed from issue #%d comment %d", launchedIssueNumber, launchedCommentID)), map[string]any{
				"thread_id":    launchedThreadID,
				"issue_number": launchedIssueNumber,
				"comment_id":   launchedCommentID,
			})
		}
		if sleepAfterLock {
			time.Sleep(time.Duration(*pollSeconds) * time.Second)
			continue
		}
		time.Sleep(time.Duration(*pollSeconds) * time.Second)
	}
}

func (r *Runner) runEvents(args []string) int {
	fs := flag.NewFlagSet("events", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	tail := fs.Int("tail", 20, "")
	raw := fs.Bool("raw", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	eventsPath := resolveInWorkdir(workdir, *eventsFile)
	events, err := loadRecentEvents(eventsPath, *tail)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if len(events) == 0 {
		fmt.Fprintf(r.Stdout, "没有可显示的事件：%s\n", eventsPath)
		return 0
	}
	if *raw {
		for _, event := range events {
			raw, err := json.Marshal(event)
			if err != nil {
				fmt.Fprintln(r.Stderr, err)
				return 1
			}
			r.println(string(raw))
		}
		return 0
	}
	r.printJSON(events)
	return 0
}

func (r *Runner) runLogEvent(args []string) int {
	fs := flag.NewFlagSet("log-event", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	eventsFile := fs.String("events-file", DefaultEventsFile, "")
	eventType := fs.String("event-type", "", "")
	quiet := fs.Bool("quiet", false, "")
	var fields stringSlice
	fs.Var(&fields, "field", "")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*eventType) == "" {
		fmt.Fprintln(r.Stderr, "缺少 --event-type")
		return 2
	}
	workdir := mustGetwd(r.Stderr)
	if workdir == "" {
		return 1
	}
	parsedFields, err := parseEventFields(fields)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	eventFields := map[string]any{}
	for key, value := range parsedFields {
		eventFields[key] = value
	}
	entry, err := appendEvent(resolveInWorkdir(workdir, *eventsFile), *eventType, true, eventFields)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	if !*quiet {
		r.printJSON(entry)
	}
	return 0
}

func (r *Runner) usage() {
	fmt.Fprintln(r.Stderr, "usage: babel-issue-bridge <open-stage|start-watcher|stop-watcher|watch|status|handoff|worker-register|worker-packet|worker-next|worker-start|worker-finish|worker-queue|worker-set-status|close-active|manager-handoff|claim-manual|release-manual|cleanup-manual|touch-manual-lease|manual-resume|events|log-event> [args]")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *Runner) printJSON(value any) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return
	}
	r.println(string(payload))
}

func (r *Runner) println(lines ...string) {
	for _, line := range lines {
		fmt.Fprintln(r.Stdout, line)
	}
}

func mustGetwd(stderr io.Writer) string {
	workdir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ""
	}
	return workdir
}

func resolveInWorkdir(workdir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workdir, path)
}

func resolveOptionalInWorkdir(workdir, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return resolveInWorkdir(workdir, path)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func conditionalString(condition bool, value string) string {
	if condition {
		return value
	}
	return ""
}

func conditionalCommentURL(comment *Comment) any {
	if comment == nil {
		return nil
	}
	return comment.HTMLURL
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func buildWatchCommand(workdir string, args []string) string {
	command := "cd " + shellQuote(workdir) + " && go run ./cmd/babel-issue-bridge watch"
	for _, arg := range args {
		command += " " + shellQuote(arg)
	}
	return command
}

type errExit struct {
	code int
	set  bool
}

func (e errExit) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

func asErrExit(err error, target *errExit) bool {
	if err == nil {
		return false
	}
	exit, ok := err.(errExit)
	if !ok {
		return false
	}
	*target = exit
	target.set = true
	return true
}
