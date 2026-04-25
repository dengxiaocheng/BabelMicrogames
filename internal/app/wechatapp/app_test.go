package wechatapp_test

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"babel-runtime/internal/app/wechatapp"
	"babel-runtime/internal/corehost"
)

func TestNewHandlerHealthz(t *testing.T) {
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: t.TempDir(),
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatalf("expected health body")
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if payload["scene_host_mode"] != "local_fallback" {
		t.Fatalf("expected local_fallback, got %v", payload["scene_host_mode"])
	}
	if payload["scene_host_source"] != "local_default" {
		t.Fatalf("expected local_default, got %v", payload["scene_host_source"])
	}
	if payload["scene_host_verified"] != true {
		t.Fatalf("expected verified true, got %v", payload["scene_host_verified"])
	}
}

func TestNewHandlerFailsWhenSharedLibraryIsInvalid(t *testing.T) {
	_, err := wechatapp.NewHandler(wechatapp.Config{
		Token:                "token",
		MemoryRoot:           t.TempDir(),
		SceneCoreLibraryPath: filepath.Join(t.TempDir(), "missing-babel-core.so"),
	})
	if err == nil {
		t.Fatalf("expected NewHandler to fail for missing shared library")
	}
	if !strings.Contains(err.Error(), "verify scene host library") {
		t.Fatalf("expected scene host error, got %v", err)
	}
}

func TestNewHandlerHealthzWithVerifiedSharedLibrary(t *testing.T) {
	libraryPath := buildFixtureLibrary(t)
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:                  "token",
		MemoryRoot:             t.TempDir(),
		SceneCoreLibraryPath:   libraryPath,
		SceneCoreLibrarySource: "explicit_path",
		Now:                    func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if payload["scene_host_mode"] != "shared_library" {
		t.Fatalf("expected shared_library, got %v", payload["scene_host_mode"])
	}
	if payload["scene_host_source"] != "explicit_path" {
		t.Fatalf("expected explicit_path, got %v", payload["scene_host_source"])
	}
	if payload["scene_host_verified"] != true {
		t.Fatalf("expected verified true, got %v", payload["scene_host_verified"])
	}
	if payload["scene_host_library"] != libraryPath {
		t.Fatalf("expected library path %q, got %v", libraryPath, payload["scene_host_library"])
	}
}

func TestNewHandlerMenuClickPersistsConsultRoute(t *testing.T) {
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: t.TempDir(),
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature("token", "1", "2"))

	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_menu]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MODE_CLAUDE]]></EventKey></xml>`,
	))
	clickRec := httptest.NewRecorder()
	handler.ServeHTTP(clickRec, clickReq)
	if clickRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", clickRec.Code)
	}

	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_menu]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[帮我看下运行时边界]]></Content><MsgId>126</MsgId></xml>`,
	))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)
	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}
	if !strings.Contains(textRec.Body.String(), "consult[1]: 帮我看下运行时边界") {
		t.Fatalf("expected consult route to persist, got %q", textRec.Body.String())
	}
}

func TestNewHandlerMenuClickPersistsSoloRoute(t *testing.T) {
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: t.TempDir(),
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature("token", "1", "2"))

	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_game]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MODE_GAME]]></EventKey></xml>`,
	))
	clickRec := httptest.NewRecorder()
	handler.ServeHTTP(clickRec, clickReq)
	if clickRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", clickRec.Code)
	}

	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_game]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[我去搬砖]]></Content><MsgId>127</MsgId></xml>`,
	))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)
	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}
	if !strings.Contains(textRec.Body.String(), "【单人角色】") {
		t.Fatalf("expected solo scene projection, got %q", textRec.Body.String())
	}
}

func TestNewHandlerFeedbackModeWritesFile(t *testing.T) {
	memoryRoot := t.TempDir()
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: memoryRoot,
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature("token", "1", "2"))

	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_feedback]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[FEEDBACK]]></EventKey></xml>`,
	))
	clickRec := httptest.NewRecorder()
	handler.ServeHTTP(clickRec, clickReq)

	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_feedback]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[这个地方再顺一点]]></Content><MsgId>301</MsgId></xml>`,
	))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)
	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}
	if !strings.Contains(textRec.Body.String(), "已收到反馈") {
		t.Fatalf("expected feedback confirmation, got %q", textRec.Body.String())
	}

	raw, err := os.ReadFile(filepath.Join(memoryRoot, "feedback", "feedback.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(raw), "这个地方再顺一点") {
		t.Fatalf("expected feedback content to be written, got %q", string(raw))
	}
}

func TestNewHandlerMultiplayerMenuRoutesRoom(t *testing.T) {
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: t.TempDir(),
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature("token", "1", "2"))

	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_room]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MP_START]]></EventKey></xml>`,
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "【联机】") {
		t.Fatalf("expected multiplayer status response, got %q", rec.Body.String())
	}
}

func TestNewHandlerSoloSceneWritesPrimaryContext(t *testing.T) {
	memoryRoot := t.TempDir()
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: memoryRoot,
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature("token", "1", "2"))

	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_scene_memory]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MODE_GAME]]></EventKey></xml>`,
	))
	clickRec := httptest.NewRecorder()
	handler.ServeHTTP(clickRec, clickReq)

	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_scene_memory]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[我去搬砖]]></Content><MsgId>401</MsgId></xml>`,
	))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)
	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}

	dir := filepath.Join(memoryRoot, "wechat:solo_scene:user_scene_memory")
	manifestRaw, err := os.ReadFile(filepath.Join(dir, "session_manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile session_manifest.json returned error: %v", err)
	}
	var manifest struct {
		PreferredWorker string `json:"preferred_worker"`
		PrimaryDocument string `json:"primary_document"`
		RulesetID       string `json:"ruleset_id"`
		PromptPackID    string `json:"prompt_pack_id"`
		GameplayAssetID string `json:"gameplay_asset_id"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("json.Unmarshal session manifest returned error: %v", err)
	}
	if manifest.PreferredWorker != "claude_code" {
		t.Fatalf("expected preferred worker claude_code, got %q", manifest.PreferredWorker)
	}
	if manifest.PrimaryDocument != "primary_context.md" {
		t.Fatalf("expected primary document primary_context.md, got %q", manifest.PrimaryDocument)
	}
	if manifest.RulesetID != "bootstrap.ruleset" || manifest.PromptPackID != "bootstrap.prompt_pack" || manifest.GameplayAssetID != "bootstrap.gameplay_asset" {
		t.Fatalf("unexpected requirement refs in scene manifest: %+v", manifest)
	}

	primaryContext, err := os.ReadFile(filepath.Join(dir, "primary_context.md"))
	if err != nil {
		t.Fatalf("ReadFile primary_context.md returned error: %v", err)
	}
	if !strings.Contains(string(primaryContext), "- mode: `solo_scene`") {
		t.Fatalf("expected solo scene primary context, got %q", string(primaryContext))
	}
	if !strings.Contains(string(primaryContext), "我去搬砖") {
		t.Fatalf("expected latest action in primary context, got %q", string(primaryContext))
	}
	if !strings.Contains(string(primaryContext), "- ruleset: `bootstrap.ruleset`") {
		t.Fatalf("expected requirement refs in primary context, got %q", string(primaryContext))
	}
}

func TestNewHandlerRoomSceneWritesPrimaryContext(t *testing.T) {
	memoryRoot := t.TempDir()
	handler, err := wechatapp.NewHandler(wechatapp.Config{
		Token:      "token",
		MemoryRoot: memoryRoot,
		Now:        func() time.Time { return time.Unix(123, 0) },
	})
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature("token", "1", "2"))

	joinA := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[player_a_memory]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MP_START]]></EventKey></xml>`,
	))
	joinARec := httptest.NewRecorder()
	handler.ServeHTTP(joinARec, joinA)

	joinB := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[player_b_memory]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MP_START]]></EventKey></xml>`,
	))
	joinBRec := httptest.NewRecorder()
	handler.ServeHTTP(joinBRec, joinB)
	if joinBRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", joinBRec.Code)
	}

	dir := filepath.Join(memoryRoot, "wechat:room_scene:main")
	primaryContext, err := os.ReadFile(filepath.Join(dir, "primary_context.md"))
	if err != nil {
		t.Fatalf("ReadFile primary_context.md returned error: %v", err)
	}
	if !strings.Contains(string(primaryContext), "- mode: `room_scene`") {
		t.Fatalf("expected room scene primary context, got %q", string(primaryContext))
	}
	if !strings.Contains(string(primaryContext), "player_a_memory") || !strings.Contains(string(primaryContext), "player_b_memory") {
		t.Fatalf("expected room player list in primary context, got %q", string(primaryContext))
	}
	if !strings.Contains(string(primaryContext), "- gameplay_asset: `bootstrap.gameplay_asset`") {
		t.Fatalf("expected gameplay asset ref in room primary context, got %q", string(primaryContext))
	}

	recentSummary, err := os.ReadFile(filepath.Join(dir, "recent_summary.md"))
	if err != nil {
		t.Fatalf("ReadFile recent_summary.md returned error: %v", err)
	}
	if !strings.Contains(string(recentSummary), "【联机】") {
		t.Fatalf("expected room summary in recent_summary.md, got %q", string(recentSummary))
	}
}

func buildSignature(token, timestamp, nonce string) string {
	values := []string{token, timestamp, nonce}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, "")))
	return hex.EncodeToString(sum[:])
}

func buildFixtureLibrary(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc not available")
	}

	soloRequest, soloResponse, err := corehost.FixturePair("solo_step")
	if err != nil {
		t.Fatalf("FixturePair(solo_step) returned error: %v", err)
	}
	roomRequest, roomResponse, err := corehost.FixturePair("room_step")
	if err != nil {
		t.Fatalf("FixturePair(room_step) returned error: %v", err)
	}

	soloRequestJSON, err := json.Marshal(soloRequest)
	if err != nil {
		t.Fatalf("json.Marshal solo request returned error: %v", err)
	}
	roomRequestJSON, err := json.Marshal(roomRequest)
	if err != nil {
		t.Fatalf("json.Marshal room request returned error: %v", err)
	}
	soloResponseJSON, err := corehost.MarshalFixture(soloResponse)
	if err != nil {
		t.Fatalf("MarshalFixture solo response returned error: %v", err)
	}
	roomResponseJSON, err := corehost.MarshalFixture(roomResponse)
	if err != nil {
		t.Fatalf("MarshalFixture room response returned error: %v", err)
	}

	source := fmt.Sprintf(`#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static const char* solo_request_match = %q;
static const char* room_request_match = %q;
static const char* solo_response = %q;
static const char* room_response = %q;

static int write_bytes(const char* src, uint8_t** out_bytes, size_t* out_len) {
	size_t len = strlen(src);
	uint8_t* buffer = (uint8_t*)malloc(len);
	if (buffer == NULL) {
		return 91;
	}
	memcpy(buffer, src, len);
	*out_bytes = buffer;
	*out_len = len;
	return 0;
}

int babel_sim_step(const uint8_t* request_bytes, size_t request_len, uint8_t** response_bytes, size_t* response_len) {
	char* request = (char*)malloc(request_len + 1);
	if (request == NULL) {
		return 92;
	}
	memcpy(request, request_bytes, request_len);
	request[request_len] = '\0';

	const char* response = NULL;
	if (strcmp(request, solo_request_match) == 0) {
		response = solo_response;
	} else if (strcmp(request, room_request_match) == 0) {
		response = room_response;
	}
	free(request);

	if (response == NULL) {
		return write_bytes("unknown fixture request", response_bytes, response_len) == 0 ? 2 : 93;
	}
	return write_bytes(response, response_bytes, response_len);
}

void babel_sim_free(void* ptr) {
	free(ptr);
}
`, string(soloRequestJSON), string(roomRequestJSON), string(soloResponseJSON), string(roomResponseJSON))

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "scene_core_stub.c")
	libraryPath := filepath.Join(tempDir, "libscene_core_stub.so")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	compile := exec.Command("cc", "-shared", "-fPIC", "-O2", "-o", libraryPath, sourcePath)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, output)
	}
	return libraryPath
}
