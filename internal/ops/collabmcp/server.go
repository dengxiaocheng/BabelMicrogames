package collabmcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

const protocolVersion = "2024-11-05"

type runtime struct {
	Stdout io.Writer
	Stderr io.Writer
}

type Server struct {
	Store Store
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type toolCallResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	r := runtime{Stdout: stdout, Stderr: stderr}
	if len(args) == 0 {
		return runServe(r, stdin, args)
	}
	switch args[0] {
	case "serve":
		return runServe(r, stdin, args[1:])
	case "read-state":
		return runSnapshot(r, args[1:])
	case "snapshot":
		return runSnapshot(r, args[1:])
	case "events":
		return runEvents(r, args[1:])
	case "set-contract":
		return runSetContract(r, args[1:])
	case "heartbeat":
		return runHeartbeat(r, args[1:])
	case "claim-scope":
		return runClaimScope(r, args[1:])
	case "release-scope":
		return runReleaseScope(r, args[1:])
	case "report-progress":
		return runReportProgress(r, args[1:])
	case "publish-artifact":
		return runPublishArtifact(r, args[1:])
	case "publish-handoff":
		return runPublishHandoff(r, args[1:])
	case "ack-handoff":
		return runAckHandoff(r, args[1:])
	default:
		usage(stderr)
		return 2
	}
}

func runServe(r runtime, stdin io.Reader, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	server := Server{Store: NewStore(*stateDir)}
	if err := server.Serve(stdin, r.Stdout); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	return 0
}

func runSnapshot(r runtime, args []string) int {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	sessionID := fs.String("session-id", "", "可选；按目标 session 过滤 pending handoff。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	view, err := NewStore(*stateDir).Snapshot(ReadStateInput{SessionID: *sessionID})
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	payload, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	fmt.Fprintln(r.Stdout, string(payload))
	return 0
}

func runEvents(r runtime, args []string) int {
	fs := flag.NewFlagSet("events", flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	stateDir := fs.String("state-dir", DefaultStateDir(), "节点级协作状态目录。")
	tail := fs.Int("tail", 20, "读取最近多少条事件。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	events, err := LoadRecentEvents(filepath.Join(*stateDir, eventsFileName), *tail)
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	payload, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		fmt.Fprintln(r.Stderr, err)
		return 1
	}
	fmt.Fprintln(r.Stdout, string(payload))
	return 0
}

func usage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: babel-collab-mcp <serve|read-state|snapshot|events|set-contract|heartbeat|claim-scope|release-scope|report-progress|publish-artifact|publish-handoff|ack-handoff> [args]")
}

func (s Server) Serve(stdin io.Reader, stdout io.Writer) error {
	reader := bufio.NewReader(stdin)
	for {
		payload, err := readFramedMessage(reader)
		if err != nil {
			return err
		}
		var request rpcRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			if err := writeResponse(stdout, rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "invalid JSON"},
			}); err != nil {
				return err
			}
			continue
		}
		response, shouldRespond := s.handleRequest(request)
		if !shouldRespond {
			continue
		}
		if err := writeResponse(stdout, response); err != nil {
			return err
		}
	}
}

func (s Server) handleRequest(request rpcRequest) (rpcResponse, bool) {
	response := rpcResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}
	switch request.Method {
	case "initialize":
		response.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "babel-collab-mcp",
				"version": "0.1.0",
			},
		}
		return response, true
	case "ping":
		response.Result = map[string]any{}
		return response, true
	case "notifications/initialized":
		return rpcResponse{}, false
	case "tools/list":
		response.Result = map[string]any{"tools": toolDefinitions()}
		return response, true
	case "tools/call":
		result, err := s.handleToolCall(request.Params)
		if err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
		} else {
			response.Result = result
		}
		return response, true
	case "resources/list":
		response.Result = map[string]any{
			"resources": []map[string]any{
				{
					"uri":      "collab://state/current",
					"name":     "当前协作状态",
					"mimeType": "application/json",
				},
				{
					"uri":      "collab://contract/current",
					"name":     "当前协作契约",
					"mimeType": "application/json",
				},
			},
		}
		return response, true
	case "resources/read":
		result, err := s.handleResourceRead(request.Params)
		if err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
		} else {
			response.Result = result
		}
		return response, true
	default:
		response.Error = &rpcError{Code: -32601, Message: "method not found"}
		return response, true
	}
}

func (s Server) handleToolCall(raw json.RawMessage) (toolCallResult, error) {
	var params toolCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return toolCallResult{}, fmt.Errorf("tools/call 参数无效: %w", err)
	}
	switch params.Name {
	case "set_contract":
		var input SetContractInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("set_contract 参数无效: %w", err)
		}
		result, err := s.Store.SetContract(input)
		return toToolResult(result), err
	case "read_state":
		var input ReadStateInput
		if len(params.Arguments) > 0 {
			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return toolCallResult{}, fmt.Errorf("read_state 参数无效: %w", err)
			}
		}
		view, err := s.Store.Snapshot(input)
		if err != nil {
			return toolCallResult{}, err
		}
		return toolCallResult{
			Content:           []toolContent{{Type: "text", Text: "已返回当前协作状态快照。"}},
			StructuredContent: view,
		}, nil
	case "heartbeat":
		var input HeartbeatInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("heartbeat 参数无效: %w", err)
		}
		result, err := s.Store.Heartbeat(input)
		return toToolResult(result), err
	case "claim_scope":
		var input ClaimScopeInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("claim_scope 参数无效: %w", err)
		}
		result, err := s.Store.ClaimScope(input)
		return toToolResult(result), err
	case "release_scope":
		var input ReleaseScopeInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("release_scope 参数无效: %w", err)
		}
		result, err := s.Store.ReleaseScope(input)
		return toToolResult(result), err
	case "report_progress":
		var input ReportProgressInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("report_progress 参数无效: %w", err)
		}
		result, err := s.Store.ReportProgress(input)
		return toToolResult(result), err
	case "publish_artifact":
		var input PublishArtifactInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("publish_artifact 参数无效: %w", err)
		}
		result, err := s.Store.PublishArtifact(input)
		return toToolResult(result), err
	case "publish_handoff":
		var input PublishHandoffInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("publish_handoff 参数无效: %w", err)
		}
		result, err := s.Store.PublishHandoff(input)
		return toToolResult(result), err
	case "ack_handoff":
		var input AckHandoffInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return toolCallResult{}, fmt.Errorf("ack_handoff 参数无效: %w", err)
		}
		result, err := s.Store.AckHandoff(input)
		return toToolResult(result), err
	default:
		return toolCallResult{}, fmt.Errorf("未知工具: %s", params.Name)
	}
}

func (s Server) handleResourceRead(raw json.RawMessage) (map[string]any, error) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("resources/read 参数无效: %w", err)
	}
	switch params.URI {
	case "collab://state/current":
		view, err := s.Store.Snapshot(ReadStateInput{})
		if err != nil {
			return nil, err
		}
		return resourceText(params.URI, view)
	case "collab://contract/current":
		view, err := s.Store.Snapshot(ReadStateInput{})
		if err != nil {
			return nil, err
		}
		return resourceText(params.URI, map[string]any{"contract": view.Contract})
	default:
		return nil, fmt.Errorf("未知资源: %s", params.URI)
	}
}

func resourceText(uri string, payload any) (map[string]any, error) {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"contents": []map[string]any{
			{
				"uri":      uri,
				"mimeType": "application/json",
				"text":     string(raw),
			},
		},
	}, nil
}

func toToolResult(result MutationResult) toolCallResult {
	content := []toolContent{{Type: "text", Text: result.Message}}
	return toolCallResult{
		Content:           content,
		StructuredContent: result,
		IsError:           !result.OK,
	}
}

func toolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "set_contract",
			Description: "写入当前 Go/C++ 协作边界与必读引用，避免两个会话靠隐式聊天上下文对齐。",
			InputSchema: objectSchema(
				requiredProps("session_id", "contract_id", "summary"),
				stringProp("session_id", "更新契约的 session 标识。"),
				stringProp("contract_id", "当前边界契约的稳定 ID。"),
				stringProp("summary", "当前边界的简述。"),
				stringArrayProp("go_surfaces", "Go 侧负责的稳定面。"),
				stringArrayProp("cpp_surfaces", "C++ 侧负责的稳定面。"),
				stringArrayProp("shared_protocols", "双方共享的协议或数据面。"),
				stringArrayProp("required_reads", "接手前必须读取的文档或提交。"),
				stringArrayProp("source_refs", "对应的仓库文档、issue、commit 等引用。"),
			),
		},
		{
			Name:        "read_state",
			Description: "读取当前结构化协作状态，而不是假设另一个会话共享了同一段隐式上下文。",
			InputSchema: objectSchema(
				nil,
				stringProp("session_id", "可选；只筛出与这个 session 相关的 pending handoff。"),
			),
		},
		{
			Name:        "heartbeat",
			Description: "更新某个会话的当前状态、线程、仓库和已认领 scope。",
			InputSchema: objectSchema(
				requiredProps("session_id"),
				stringProp("session_id", "当前会话 ID。"),
				stringProp("repo", "会话当前主要工作的仓库。"),
				stringProp("role", "例如 online-go-runtime 或 babel-cpp-core。"),
				stringProp("status", "例如 active、waiting、blocked。"),
				stringProp("note", "当前阶段的简短说明。"),
				stringProp("thread_id", "对应的 Codex thread id。"),
				stringProp("commit", "最近的 commit。"),
				stringArrayProp("scopes", "当前会话主动声明的 scope 列表。"),
			),
		},
		{
			Name:        "claim_scope",
			Description: "认领目录、模块或子系统，避免 Go 和 C++ 会话同时改同一块代码。",
			InputSchema: objectSchema(
				requiredProps("session_id", "scope"),
				stringProp("session_id", "认领方 session ID。"),
				stringProp("repo", "当前仓库。"),
				stringProp("scope", "要认领的目录、模块或子系统名。"),
				stringProp("note", "补充说明。"),
				intProp("ttl_seconds", "可选；claim 的超时秒数。"),
			),
		},
		{
			Name:        "release_scope",
			Description: "释放之前认领的 scope。",
			InputSchema: objectSchema(
				requiredProps("session_id", "scope"),
				stringProp("session_id", "释放方 session ID。"),
				stringProp("scope", "要释放的 scope。"),
			),
		},
		{
			Name:        "report_progress",
			Description: "记录阶段进度、变更范围和 commit，供另一会话按需读取。",
			InputSchema: objectSchema(
				requiredProps("session_id", "summary"),
				stringProp("session_id", "当前会话 ID。"),
				stringProp("repo", "当前仓库。"),
				stringProp("stage", "阶段名。"),
				stringProp("summary", "本次阶段进度摘要。"),
				stringArrayProp("changed_paths", "本阶段变更的关键路径。"),
				stringProp("commit", "本阶段 commit。"),
			),
		},
		{
			Name:        "publish_artifact",
			Description: "发布结构化产物信息，例如 Babel/C++ 编出的共享库路径，让另一会话直接消费，而不是只靠聊天描述。",
			InputSchema: objectSchema(
				requiredProps("session_id", "kind", "path"),
				stringProp("session_id", "当前会话 ID。"),
				stringProp("repo", "当前仓库。"),
				stringProp("kind", "artifact 类型，例如 scene_host_library。"),
				stringProp("path", "artifact 的绝对路径或稳定路径。"),
				stringProp("summary", "artifact 的简要说明。"),
				stringProp("commit", "产出 artifact 对应的提交。"),
			),
		},
		{
			Name:        "publish_handoff",
			Description: "发布显式 handoff，告诉另一会话它接下来该读什么、接哪里继续做。",
			InputSchema: objectSchema(
				requiredProps("from_session_id", "title", "summary"),
				stringProp("from_session_id", "交出执行权的 session。"),
				stringProp("to_session_id", "接手的目标 session；留空表示任何会话都可接。"),
				stringProp("repo", "handoff 主要针对的仓库。"),
				stringProp("title", "handoff 标题。"),
				stringProp("summary", "handoff 摘要。"),
				stringArrayProp("required_reads", "接手前必须读取的文档、issue 或 commit。"),
				stringArrayProp("changed_paths", "本次 handoff 关联的关键路径。"),
				stringProp("commit", "handoff 对应的提交。"),
			),
		},
		{
			Name:        "ack_handoff",
			Description: "确认某条 handoff 已被当前会话接手。",
			InputSchema: objectSchema(
				requiredProps("session_id", "handoff_id"),
				stringProp("session_id", "接手方 session。"),
				stringProp("handoff_id", "要确认的 handoff ID。"),
			),
		},
	}
}

func objectSchema(required []string, properties ...map[string]any) map[string]any {
	props := map[string]any{}
	for _, property := range properties {
		for key, value := range property {
			props[key] = value
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func requiredProps(names ...string) []string {
	return names
}

func stringProp(name, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "string", "description": description}}
}

func stringArrayProp(name, description string) map[string]any {
	return map[string]any{name: map[string]any{
		"type":        "array",
		"description": description,
		"items":       map[string]any{"type": "string"},
	}}
}

func intProp(name, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "integer", "description": description}}
}

func readFramedMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && contentLength == -1 {
				return nil, io.EOF
			}
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			length, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, err
			}
			contentLength = length
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeResponse(writer io.Writer, response rpcResponse) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "Content-Length: %d\r\n\r\n", len(payload))
	buffer.Write(payload)
	_, err = writer.Write(buffer.Bytes())
	return err
}
