package collabmcp

import (
	"encoding/json"
	"testing"
)

func TestHandleInitialize(t *testing.T) {
	server := Server{Store: NewStore(t.TempDir())}
	response, shouldRespond := server.handleRequest(rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	})
	if !shouldRespond {
		t.Fatalf("expected initialize to respond")
	}
	if response.Error != nil {
		t.Fatalf("unexpected initialize error: %+v", response.Error)
	}
	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected initialize result type: %T", response.Result)
	}
	if result["protocolVersion"] != protocolVersion {
		t.Fatalf("unexpected protocol version: %+v", result)
	}
}

func TestHandleToolsCallClaimScope(t *testing.T) {
	server := Server{Store: NewStore(t.TempDir())}
	params := map[string]any{
		"name": "claim_scope",
		"arguments": map[string]any{
			"session_id": "online",
			"scope":      "internal/kernel",
			"repo":       "babel-runtime",
		},
	}
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	result, err := server.handleToolCall(raw)
	if err != nil {
		t.Fatalf("handleToolCall returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result)
	}
	structured, ok := result.StructuredContent.(MutationResult)
	if !ok {
		t.Fatalf("unexpected structured content type: %T", result.StructuredContent)
	}
	if !structured.OK || structured.Claim == nil || structured.Claim.Scope != "internal/kernel" {
		t.Fatalf("unexpected mutation result: %+v", structured)
	}
}

func TestHandleResourceReadState(t *testing.T) {
	server := Server{Store: NewStore(t.TempDir())}
	_, err := server.Store.Heartbeat(HeartbeatInput{
		SessionID: "online",
		Repo:      "babel-runtime",
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}
	raw := json.RawMessage(`{"uri":"collab://state/current"}`)
	result, err := server.handleResourceRead(raw)
	if err != nil {
		t.Fatalf("handleResourceRead returned error: %v", err)
	}
	contents, ok := result["contents"].([]map[string]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("unexpected contents: %+v", result["contents"])
	}
	if contents[0]["uri"] != "collab://state/current" {
		t.Fatalf("unexpected resource uri: %+v", contents[0])
	}
}
