package wechat_test

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"babel-runtime/internal/agent"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/delivery"
	"babel-runtime/internal/gateway/wechat"
	"babel-runtime/internal/kernel"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/projection"
	"babel-runtime/internal/repository"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

type freeChatStub struct{}

func (freeChatStub) HandleFreeChat(ctx context.Context, msg wechat.NormalizedMessage) (string, error) {
	_ = ctx
	return "free ok: " + msg.Text, nil
}

type consultStub struct{}

func (consultStub) HandleProjectConsult(ctx context.Context, msg wechat.NormalizedMessage) (string, error) {
	_ = ctx
	return "consult ok: " + msg.Text, nil
}

type soloStub struct{}

func (soloStub) HandleSoloScene(ctx context.Context, msg wechat.NormalizedMessage) (string, error) {
	_ = ctx
	return "solo ok: " + msg.Text, nil
}

type roomStub struct{}

func (roomStub) HandleRoomScene(ctx context.Context, msg wechat.NormalizedMessage) (string, error) {
	_ = ctx
	return "room ok: " + msg.Text, nil
}

type feedbackStub struct{}

func (feedbackStub) HandleFeedback(ctx context.Context, msg wechat.NormalizedMessage) (string, error) {
	_ = ctx
	return "feedback ok: " + msg.Text, nil
}

func TestVerifySignature(t *testing.T) {
	signature := buildSignature("token", "1", "2")
	if !wechat.VerifySignature("token", "1", "2", signature) {
		t.Fatalf("expected signature to verify")
	}
}

func TestHandlerGETVerify(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("echostr", "hello")
	query.Set("signature", buildSignature(token, "1", "2"))

	req := httptest.NewRequest(http.MethodGet, "/wechat?"+query.Encode(), nil)
	rec := httptest.NewRecorder()

	handler := wechat.Handler{Token: token}
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "hello" {
		t.Fatalf("expected echo string, got %q", rec.Body.String())
	}
}

func TestHandlerPOSTFreeChat(t *testing.T) {
	token := "token"
	body := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_2]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[你好]]></Content><MsgId>124</MsgId></xml>`
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler := wechat.Handler{
		Token:    token,
		FreeChat: freeChatStub{},
	}
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "free ok: 你好") {
		t.Fatalf("expected free chat response body, got %q", rec.Body.String())
	}
}

func TestHandlerPOSTProjectConsult(t *testing.T) {
	token := "token"
	body := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_3]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[/consult 帮我评估方案]]></Content><MsgId>125</MsgId></xml>`
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler := wechat.Handler{
		Token:   token,
		Consult: consultStub{},
	}
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "consult ok: /consult 帮我评估方案") {
		t.Fatalf("expected project consult response body, got %q", rec.Body.String())
	}
}

func TestHandlerPOSTSoloSceneMenuClickSelectsGameMode(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	clickBody := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_game]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MODE_GAME]]></EventKey></xml>`
	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(clickBody))
	clickRec := httptest.NewRecorder()

	sessions := wechat.NewMemorySessionModeStore()
	handler := wechat.Handler{
		Token:    token,
		FreeChat: freeChatStub{},
		Consult:  consultStub{},
		Solo:     soloStub{},
		Room:     roomStub{},
		Feedback: feedbackStub{},
		Sessions: sessions,
	}
	handler.ServeHTTP(clickRec, clickReq)

	if clickRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", clickRec.Code)
	}
	if !strings.Contains(clickRec.Body.String(), "已切换到单人角色") {
		t.Fatalf("expected mode switch response, got %q", clickRec.Body.String())
	}

	textBody := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_game]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[我去搬砖]]></Content><MsgId>127</MsgId></xml>`
	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(textBody))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)

	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}
	if !strings.Contains(textRec.Body.String(), "solo ok: 我去搬砖") {
		t.Fatalf("expected solo scene response body, got %q", textRec.Body.String())
	}
}

func TestHandlerPOSTProjectConsultMenuClickSelectsConsultMode(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	clickBody := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_menu]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MODE_CLAUDE]]></EventKey></xml>`
	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(clickBody))
	clickRec := httptest.NewRecorder()

	sessions := wechat.NewMemorySessionModeStore()
	handler := wechat.Handler{
		Token:    token,
		FreeChat: freeChatStub{},
		Consult:  consultStub{},
		Solo:     soloStub{},
		Room:     roomStub{},
		Feedback: feedbackStub{},
		Sessions: sessions,
	}
	handler.ServeHTTP(clickRec, clickReq)

	if clickRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", clickRec.Code)
	}
	if !strings.Contains(clickRec.Body.String(), "已切换到项目咨询") {
		t.Fatalf("expected menu switch response, got %q", clickRec.Body.String())
	}

	textBody := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_menu]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[帮我看下运行时边界]]></Content><MsgId>126</MsgId></xml>`
	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(textBody))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)

	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}
	if !strings.Contains(textRec.Body.String(), "consult ok: 帮我看下运行时边界") {
		t.Fatalf("expected consult response body, got %q", textRec.Body.String())
	}
}

func TestHandlerPOSTLegacyAdminMenuClickReturnsRetiredMessage(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	body := `<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_admin]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MODE_CODEX]]></EventKey></xml>`
	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler := wechat.Handler{
		Token:    token,
		FreeChat: freeChatStub{},
		Consult:  consultStub{},
		Solo:     soloStub{},
		Room:     roomStub{},
		Feedback: feedbackStub{},
		Sessions: wechat.NewMemorySessionModeStore(),
	}
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "管理员菜单已下线") {
		t.Fatalf("expected retired admin menu response, got %q", rec.Body.String())
	}
}

func TestHandlerPOSTDecisionButtonUsesActiveSceneRoute(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	sessions := wechat.NewMemorySessionModeStore()
	sessions.SetSelectedRoute("user_scene", wechat.RouteSoloScene)
	handler := wechat.Handler{
		Token:    token,
		Solo:     soloStub{},
		Room:     roomStub{},
		Feedback: feedbackStub{},
		Sessions: sessions,
	}

	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_scene]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[OPT_1]]></EventKey></xml>`,
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "solo ok: 我选择①") {
		t.Fatalf("expected decision action routed to solo scene, got %q", rec.Body.String())
	}
}

func TestHandlerPOSTThoughtButtonUsesPrayerFoldedText(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	sessions := wechat.NewMemorySessionModeStore()
	sessions.SetSelectedRoute("user_scene", wechat.RouteSoloScene)
	handler := wechat.Handler{
		Token:    token,
		Solo:     soloStub{},
		Room:     roomStub{},
		Feedback: feedbackStub{},
		Sessions: sessions,
	}

	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_scene]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[LOOK_FAR]]></EventKey></xml>`,
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "默默祈求一点指引和庇佑") {
		t.Fatalf("expected prayer folded into thought text, got %q", rec.Body.String())
	}
}

func TestHandlerPOSTFeedbackModePersistsRoute(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	sessions := wechat.NewMemorySessionModeStore()
	handler := wechat.Handler{
		Token:    token,
		Feedback: feedbackStub{},
		Sessions: sessions,
	}

	clickReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_feedback]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[FEEDBACK]]></EventKey></xml>`,
	))
	clickRec := httptest.NewRecorder()
	handler.ServeHTTP(clickRec, clickReq)

	textReq := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_feedback]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[这个地方再顺一点]]></Content><MsgId>300</MsgId></xml>`,
	))
	textRec := httptest.NewRecorder()
	handler.ServeHTTP(textRec, textReq)

	if textRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", textRec.Code)
	}
	if !strings.Contains(textRec.Body.String(), "feedback ok: 这个地方再顺一点") {
		t.Fatalf("expected feedback route to persist, got %q", textRec.Body.String())
	}
}

func TestHandlerPOSTMultiplayerMenuClickRoutesRoom(t *testing.T) {
	token := "token"
	query := url.Values{}
	query.Set("timestamp", "1")
	query.Set("nonce", "2")
	query.Set("signature", buildSignature(token, "1", "2"))

	handler := wechat.Handler{
		Token:    token,
		Room:     roomStub{},
		Feedback: feedbackStub{},
		Sessions: wechat.NewMemorySessionModeStore(),
	}

	req := httptest.NewRequest(http.MethodPost, "/wechat?"+query.Encode(), strings.NewReader(
		`<xml><ToUserName><![CDATA[gh_1]]></ToUserName><FromUserName><![CDATA[user_room]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[CLICK]]></Event><EventKey><![CDATA[MP_START]]></EventKey></xml>`,
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "room ok: 联机") {
		t.Fatalf("expected room responder to receive multiplayer command, got %q", rec.Body.String())
	}
}

func TestNormalizeInboundSlashCommandRoutesUnknown(t *testing.T) {
	msg := wechat.NormalizeInbound(wechat.InboundEnvelope{
		FromUserName: "user_1",
		MsgType:      "text",
		Content:      "/config",
		MsgID:        1001,
	})
	if msg.Route != wechat.RouteUnknown {
		t.Fatalf("expected unknown route, got %#v", msg.Route)
	}
}

func TestNormalizeInboundNonTextRoutesUnknown(t *testing.T) {
	msg := wechat.NormalizeInbound(wechat.InboundEnvelope{
		FromUserName: "user_1",
		MsgType:      "event",
		Event:        "CLICK",
		EventKey:     "MODE_CLAUDE",
	})
	if msg.Route != wechat.RouteUnknown {
		t.Fatalf("expected unknown route, got %#v", msg.Route)
	}
}

func TestRuntimeBridgeRoundTripsFreeChatAndConsult(t *testing.T) {
	repo := repository.NewMemoryRepository()
	memoryRoot := t.TempDir()
	router, err := mode.NewStaticRouter(
		mode.FreeChatModule{},
		mode.ProjectConsultModule{},
	)
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:       repo,
		Router:     router,
		Supervisor: &agent.SimpleSupervisor{MemoryRoot: memoryRoot},
		Projector:  projection.SimpleProjector{DefaultTransport: "wechat", Now: func() time.Time { return time.Unix(123, 0) }},
		Dispatcher: delivery.QueueDispatcher{Now: func() time.Time { return time.Unix(124, 0) }},
		Now:        func() time.Time { return time.Unix(123, 0) },
	}
	bridge := wechat.RuntimeBridge{
		Repo:           repo,
		Kernel:         engine,
		MaxResumeSteps: 4,
		Now:            func() time.Time { return time.Unix(123, 0) },
	}

	freeReply, err := bridge.HandleFreeChat(context.Background(), wechat.NormalizedMessage{
		UserID:         "user_1",
		Text:           "你好",
		IdempotencyKey: "msg-1",
		Route:          wechat.RouteFreeChat,
	})
	if err != nil {
		t.Fatalf("HandleFreeChat returned error: %v", err)
	}
	if freeReply != "free chat[1]: 你好" {
		t.Fatalf("unexpected free chat reply %q", freeReply)
	}

	consultReply, err := bridge.HandleProjectConsult(context.Background(), wechat.NormalizedMessage{
		UserID:         "user_1",
		Text:           "/consult 帮我评估架构",
		IdempotencyKey: "msg-2",
		Route:          wechat.RouteProjectConsult,
	})
	if err != nil {
		t.Fatalf("HandleProjectConsult returned error: %v", err)
	}
	if consultReply != "consult[1]: 帮我评估架构" {
		t.Fatalf("unexpected consult reply %q", consultReply)
	}

	snapshot := repo.Snapshot()
	if got := snapshot.Runtimes["wechat:"+string(types.ModeFreeChat)+":user_1"].ModeID; got != types.ModeFreeChat {
		t.Fatalf("expected free chat runtime mode, got %q", got)
	}
	if got := snapshot.Runtimes["wechat:"+string(types.ModeProjectConsult)+":user_1"].ModeID; got != types.ModeProjectConsult {
		t.Fatalf("expected consult runtime mode, got %q", got)
	}
	if got := snapshot.Executions["wechat:"+string(types.ModeFreeChat)+":user_1:msg-1"].Stage; got != types.ExecutionDelivered {
		t.Fatalf("expected free chat execution delivered, got %q", got)
	}
	if got := snapshot.Executions["wechat:"+string(types.ModeProjectConsult)+":user_1:msg-2"].Stage; got != types.ExecutionDelivered {
		t.Fatalf("expected consult execution delivered, got %q", got)
	}
	if len(snapshot.Frames) != 2 {
		t.Fatalf("expected 2 projection frames, got %d", len(snapshot.Frames))
	}
	if len(snapshot.DeliveryJobs) != 2 {
		t.Fatalf("expected 2 delivery jobs, got %d", len(snapshot.DeliveryJobs))
	}
	freeMemory, err := os.ReadFile(filepath.Join(memoryRoot, "wechat:free_chat:user_1", "recent_summary.md"))
	if err != nil {
		t.Fatalf("ReadFile free chat memory returned error: %v", err)
	}
	if string(freeMemory) != "free chat[1]: 你好\n" {
		t.Fatalf("unexpected free chat memory %q", string(freeMemory))
	}
	consultMemory, err := os.ReadFile(filepath.Join(memoryRoot, "wechat:project_consult:user_1", "recent_summary.md"))
	if err != nil {
		t.Fatalf("ReadFile consult memory returned error: %v", err)
	}
	if string(consultMemory) != "consult[1]: 帮我评估架构\n" {
		t.Fatalf("unexpected consult memory %q", string(consultMemory))
	}
}

func TestRuntimeBridgeRoomSceneRequiresTwoPlayers(t *testing.T) {
	bridge, setNow := newRoomRuntimeBridge(t)
	setNow(time.Unix(200, 0))

	reply, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_a",
		Text:           "联机",
		IdempotencyKey: "room-1",
		Route:          wechat.RouteRoomScene,
	})
	if err != nil {
		t.Fatalf("HandleRoomScene returned error: %v", err)
	}
	if !strings.Contains(reply, "等待更多玩家加入") {
		t.Fatalf("expected waiting-for-more-players reply, got %q", reply)
	}
	if !strings.Contains(reply, "阶段=大厅等待") {
		t.Fatalf("expected lobby stage in reply, got %q", reply)
	}
	if !strings.Contains(reply, "玩家=player_a") {
		t.Fatalf("expected single player roster in reply, got %q", reply)
	}

	reply, err = bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_a",
		Text:           "我先行动",
		IdempotencyKey: "room-2",
		Route:          wechat.RouteRoomScene,
	})
	if err != nil {
		t.Fatalf("HandleRoomScene second call returned error: %v", err)
	}
	if strings.Contains(reply, "你刚才提交") {
		t.Fatalf("expected no submission before enough players join, got %q", reply)
	}

	reply, err = bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_b",
		Text:           "联机",
		IdempotencyKey: "room-3",
		Route:          wechat.RouteRoomScene,
	})
	if err != nil {
		t.Fatalf("HandleRoomScene third call returned error: %v", err)
	}
	if !strings.Contains(reply, "阶段=回合进行中") {
		t.Fatalf("expected round-open stage after second player joins, got %q", reply)
	}
	if !strings.Contains(reply, "最近房间事件：第1轮已开始") {
		t.Fatalf("expected room event summary after round opens, got %q", reply)
	}
}

func TestRuntimeBridgeRoomScenePrunesInactivePlayers(t *testing.T) {
	bridge, setNow := newRoomRuntimeBridge(t)
	setNow(time.Unix(300, 0))

	if _, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_a",
		Text:           "联机",
		IdempotencyKey: "join-a",
		Route:          wechat.RouteRoomScene,
	}); err != nil {
		t.Fatalf("join a returned error: %v", err)
	}
	if _, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_b",
		Text:           "联机",
		IdempotencyKey: "join-b",
		Route:          wechat.RouteRoomScene,
	}); err != nil {
		t.Fatalf("join b returned error: %v", err)
	}

	setNow(time.Unix(300+int64((2*time.Minute/time.Second))+5, 0))
	reply, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_c",
		Text:           "联机",
		IdempotencyKey: "join-c",
		Route:          wechat.RouteRoomScene,
	})
	if err != nil {
		t.Fatalf("join c returned error: %v", err)
	}
	if !strings.Contains(reply, "当前人数=1") {
		t.Fatalf("expected stale players pruned, got %q", reply)
	}
	if !strings.Contains(reply, "等待更多玩家加入") {
		t.Fatalf("expected waiting status after prune, got %q", reply)
	}
}

func TestRuntimeBridgeRoomSceneShowsSubmissionAndResolutionSummary(t *testing.T) {
	bridge, setNow := newRoomRuntimeBridge(t)
	setNow(time.Unix(400, 0))

	if _, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_a",
		Text:           "联机",
		IdempotencyKey: "summary-join-a",
		Route:          wechat.RouteRoomScene,
	}); err != nil {
		t.Fatalf("join a returned error: %v", err)
	}
	if _, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_b",
		Text:           "联机",
		IdempotencyKey: "summary-join-b",
		Route:          wechat.RouteRoomScene,
	}); err != nil {
		t.Fatalf("join b returned error: %v", err)
	}

	reply, err := bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_a",
		Text:           "我去抬木头",
		IdempotencyKey: "summary-act-a",
		Route:          wechat.RouteRoomScene,
	})
	if err != nil {
		t.Fatalf("submit a returned error: %v", err)
	}
	if !strings.Contains(reply, "已提交玩家=player_a") {
		t.Fatalf("expected submitted player summary, got %q", reply)
	}
	if !strings.Contains(reply, "待提交玩家=player_b") {
		t.Fatalf("expected waiting player summary, got %q", reply)
	}

	reply, err = bridge.HandleRoomScene(context.Background(), wechat.NormalizedMessage{
		UserID:         "player_b",
		Text:           "我去拉绳",
		IdempotencyKey: "summary-act-b",
		Route:          wechat.RouteRoomScene,
	})
	if err != nil {
		t.Fatalf("submit b returned error: %v", err)
	}
	if !strings.Contains(reply, "上轮结算=第1轮/2个行动") {
		t.Fatalf("expected last resolved summary, got %q", reply)
	}
	if !strings.Contains(reply, "当前轮次=2") {
		t.Fatalf("expected next round reopened, got %q", reply)
	}
}

func newRoomRuntimeBridge(t *testing.T) (wechat.RuntimeBridge, func(time.Time)) {
	t.Helper()

	repo := repository.NewMemoryRepository()
	memoryRoot := t.TempDir()
	currentNow := time.Unix(100, 0)
	nowFn := func() time.Time { return currentNow }
	router, err := mode.NewStaticRouter(
		mode.RoomSceneModule{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{
					TimeCore: timecore.SimpleCore{},
				},
			},
			Now: nowFn,
		},
	)
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:       repo,
		Router:     router,
		Supervisor: &agent.SimpleSupervisor{MemoryRoot: memoryRoot},
		Projector:  projection.SimpleProjector{DefaultTransport: "wechat", Now: nowFn},
		Dispatcher: delivery.QueueDispatcher{Now: nowFn},
		Now:        nowFn,
	}
	bridge := wechat.RuntimeBridge{
		Repo:           repo,
		Kernel:         engine,
		MemoryRoot:     memoryRoot,
		MaxResumeSteps: 4,
		Now:            nowFn,
	}
	return bridge, func(next time.Time) { currentNow = next }
}

func buildSignature(token, timestamp, nonce string) string {
	values := []string{token, timestamp, nonce}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, "")))
	return hex.EncodeToString(sum[:])
}
