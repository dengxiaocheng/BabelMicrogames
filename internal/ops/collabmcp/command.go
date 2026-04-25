package collabmcp

import (
	"encoding/json"
	"flag"
	"fmt"
)

const (
	DefaultOnlineSessionID   = "online"
	DefaultOnlineRole        = "online-go-runtime"
	DefaultBabelCppSessionID = "babel-cpp"
	DefaultBabelCppRole      = "babel-cpp-core"
	DefaultOnlineRepo        = "babel-runtime"
	DefaultBabelRepo         = "Babel"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func runSetContract(r runtime, args []string) int {
	fs := flag.NewFlagSet("set-contract", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", DefaultOnlineSessionID, "更新契约的 session ID。")
	contractID := fs.String("contract-id", "", "契约 ID。")
	summary := fs.String("summary", "", "契约摘要。")
	var goSurfaces, cppSurfaces, sharedProtocols, requiredReads, sourceRefs stringListFlag
	fs.Var(&goSurfaces, "go-surface", "Go 侧负责的稳定面，可重复。")
	fs.Var(&cppSurfaces, "cpp-surface", "C++ 侧负责的稳定面，可重复。")
	fs.Var(&sharedProtocols, "shared-protocol", "共享协议，可重复。")
	fs.Var(&requiredReads, "required-read", "接手前必读引用，可重复。")
	fs.Var(&sourceRefs, "source-ref", "源引用，可重复。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).SetContract(SetContractInput{
		SessionID:       *sessionID,
		ContractID:      *contractID,
		Summary:         *summary,
		GoSurfaces:      goSurfaces,
		CppSurfaces:     cppSurfaces,
		SharedProtocols: sharedProtocols,
		RequiredReads:   requiredReads,
		SourceRefs:      sourceRefs,
	})
	return printMutation(r, result, err)
}

func runHeartbeat(r runtime, args []string) int {
	fs := flag.NewFlagSet("heartbeat", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "当前会话 ID。")
	repo := fs.String("repo", "", "当前仓库。")
	role := fs.String("role", "", "当前角色。")
	status := fs.String("status", "", "当前状态。")
	note := fs.String("note", "", "补充说明。")
	threadID := fs.String("thread-id", "", "当前 thread id。")
	commit := fs.String("commit", "", "最近 commit。")
	var scopes stringListFlag
	fs.Var(&scopes, "scope", "当前会话主动声明的 scope，可重复。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).Heartbeat(HeartbeatInput{
		SessionID: *sessionID,
		Repo:      *repo,
		Role:      *role,
		Status:    *status,
		Note:      *note,
		ThreadID:  *threadID,
		Commit:    *commit,
		Scopes:    scopes,
	})
	return printMutation(r, result, err)
}

func runClaimScope(r runtime, args []string) int {
	fs := flag.NewFlagSet("claim-scope", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "认领方 session ID。")
	repo := fs.String("repo", "", "当前仓库。")
	scope := fs.String("scope", "", "要认领的 scope。")
	note := fs.String("note", "", "补充说明。")
	ttlSeconds := fs.Int("ttl-seconds", 0, "可选；claim TTL 秒数。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).ClaimScope(ClaimScopeInput{
		SessionID:  *sessionID,
		Repo:       *repo,
		Scope:      *scope,
		Note:       *note,
		TTLSeconds: *ttlSeconds,
	})
	return printMutation(r, result, err)
}

func runReleaseScope(r runtime, args []string) int {
	fs := flag.NewFlagSet("release-scope", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "释放方 session ID。")
	scope := fs.String("scope", "", "要释放的 scope。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).ReleaseScope(ReleaseScopeInput{
		SessionID: *sessionID,
		Scope:     *scope,
	})
	return printMutation(r, result, err)
}

func runReportProgress(r runtime, args []string) int {
	fs := flag.NewFlagSet("report-progress", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "当前会话 ID。")
	repo := fs.String("repo", "", "当前仓库。")
	stage := fs.String("stage", "", "阶段名。")
	summary := fs.String("summary", "", "阶段摘要。")
	commit := fs.String("commit", "", "对应 commit。")
	var paths stringListFlag
	fs.Var(&paths, "path", "变更路径，可重复。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).ReportProgress(ReportProgressInput{
		SessionID:    *sessionID,
		Repo:         *repo,
		Stage:        *stage,
		Summary:      *summary,
		ChangedPaths: paths,
		Commit:       *commit,
	})
	return printMutation(r, result, err)
}

func runPublishArtifact(r runtime, args []string) int {
	fs := flag.NewFlagSet("publish-artifact", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "当前会话 ID。")
	repo := fs.String("repo", "", "当前仓库。")
	kind := fs.String("kind", "", "artifact 类型。")
	path := fs.String("path", "", "artifact 路径。")
	summary := fs.String("summary", "", "artifact 摘要。")
	commit := fs.String("commit", "", "对应 commit。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).PublishArtifact(PublishArtifactInput{
		SessionID: *sessionID,
		Repo:      *repo,
		Kind:      *kind,
		Path:      *path,
		Summary:   *summary,
		Commit:    *commit,
	})
	return printMutation(r, result, err)
}

func runPublishHandoff(r runtime, args []string) int {
	fs := flag.NewFlagSet("publish-handoff", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	fromSessionID := fs.String("from-session-id", "", "交出执行权的 session。")
	toSessionID := fs.String("to-session-id", "", "接手方 session。")
	repo := fs.String("repo", "", "主要仓库。")
	title := fs.String("title", "", "handoff 标题。")
	summary := fs.String("summary", "", "handoff 摘要。")
	commit := fs.String("commit", "", "对应 commit。")
	var requiredReads, paths stringListFlag
	fs.Var(&requiredReads, "required-read", "接手前必读引用，可重复。")
	fs.Var(&paths, "path", "关联路径，可重复。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).PublishHandoff(PublishHandoffInput{
		FromSessionID: *fromSessionID,
		ToSessionID:   *toSessionID,
		Repo:          *repo,
		Title:         *title,
		Summary:       *summary,
		RequiredReads: requiredReads,
		ChangedPaths:  paths,
		Commit:        *commit,
	})
	return printMutation(r, result, err)
}

func runAckHandoff(r runtime, args []string) int {
	fs := flag.NewFlagSet("ack-handoff", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "接手方 session。")
	handoffID := fs.String("handoff-id", "", "要确认的 handoff ID。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result, err := NewStore(*stateDir).AckHandoff(AckHandoffInput{
		SessionID: *sessionID,
		HandoffID: *handoffID,
	})
	return printMutation(r, result, err)
}

func printMutation(r runtime, result MutationResult, err error) int {
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	payload, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(r.Stderr, marshalErr)
		return 1
	}
	fmt.Fprintln(r.Stdout, string(payload))
	if !result.OK {
		return 1
	}
	return 0
}
