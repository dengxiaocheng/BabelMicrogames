package wechat

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type FreeChatResponder interface {
	HandleFreeChat(ctx context.Context, msg NormalizedMessage) (string, error)
}

type ProjectConsultResponder interface {
	HandleProjectConsult(ctx context.Context, msg NormalizedMessage) (string, error)
}

type SoloSceneResponder interface {
	HandleSoloScene(ctx context.Context, msg NormalizedMessage) (string, error)
}

type RoomSceneResponder interface {
	HandleRoomScene(ctx context.Context, msg NormalizedMessage) (string, error)
}

type FeedbackResponder interface {
	HandleFeedback(ctx context.Context, msg NormalizedMessage) (string, error)
}

type Handler struct {
	Token    string
	FreeChat FreeChatResponder
	Consult  ProjectConsultResponder
	Solo     SoloSceneResponder
	Room     RoomSceneResponder
	Feedback FeedbackResponder
	Sessions SessionModeStore
	Now      func() time.Time
}

type outboundEnvelope struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   cdata    `xml:"ToUserName"`
	FromUserName cdata    `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      cdata    `xml:"MsgType"`
	Content      cdata    `xml:"Content"`
}

type cdata struct {
	Text string `xml:",cdata"`
}

func VerifySignature(token, timestamp, nonce, signature string) bool {
	values := []string{token, timestamp, nonce}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, "")))
	return hex.EncodeToString(sum[:]) == strings.ToLower(signature)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerify(w, r)
	case http.MethodPost:
		h.handleInbound(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if !VerifySignature(h.Token, query.Get("timestamp"), query.Get("nonce"), query.Get("signature")) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}
	_, _ = io.WriteString(w, query.Get("echostr"))
}

func (h Handler) handleInbound(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if !VerifySignature(h.Token, query.Get("timestamp"), query.Get("nonce"), query.Get("signature")) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	env, err := ParseInboundXML(raw)
	if err != nil {
		http.Error(w, "invalid xml", http.StatusBadRequest)
		return
	}
	if reply, handled, err := h.handleMenuClick(r.Context(), env); err != nil {
		http.Error(w, "menu action failed", http.StatusInternalServerError)
		return
	} else if handled {
		if err := writeXMLReply(w, env, reply, h.now()); err != nil {
			http.Error(w, "failed to write reply", http.StatusInternalServerError)
		}
		return
	}
	msg := NormalizeInbound(env)
	msg = h.applySelectedRoute(msg)

	reply, err := h.respond(r.Context(), msg)
	if err != nil {
		http.Error(w, "route handler failed", http.StatusInternalServerError)
		return
	}

	if err := writeXMLReply(w, env, reply, h.now()); err != nil {
		http.Error(w, "failed to write reply", http.StatusInternalServerError)
		return
	}
}

func (h Handler) handleMenuClick(ctx context.Context, env InboundEnvelope) (string, bool, error) {
	if !strings.EqualFold(env.MsgType, "event") || !strings.EqualFold(env.Event, "CLICK") {
		return "", false, nil
	}

	switch env.EventKey {
	case "MODE_CLAUDE":
		if h.Sessions != nil {
			h.Sessions.SetSelectedRoute(env.FromUserName, RouteProjectConsult)
		}
		return "已切换到项目咨询，直接发送问题即可", true, nil
	case "MODE_FREE_CHAT":
		if h.Sessions != nil {
			h.Sessions.SetSelectedRoute(env.FromUserName, RouteFreeChat)
		}
		return "已切换到自由聊天，直接发送消息即可", true, nil
	case "MODE_GAME":
		if h.Sessions != nil {
			h.Sessions.SetSelectedRoute(env.FromUserName, RouteSoloScene)
		}
		return "已切换到单人角色，直接描述你的行动即可", true, nil
	case "MP_START":
		if h.Sessions != nil {
			h.Sessions.SetSelectedRoute(env.FromUserName, RouteRoomScene)
		}
		reply, err := h.respond(ctx, NormalizedMessage{
			UserID:         env.FromUserName,
			Text:           "联机",
			IdempotencyKey: fmt.Sprintf("%s:mp_start:%d", env.FromUserName, env.CreateTime),
			MsgType:        "event",
			Route:          RouteRoomScene,
		})
		return reply, true, err
	case "FEEDBACK":
		if h.Sessions != nil {
			h.Sessions.SetSelectedRoute(env.FromUserName, RouteFeedback)
		}
		return "已切换到反馈，直接输入你的建议或需求即可。", true, nil
	case "OPT_1", "OPT_2", "OPT_3", "OPT_4", "OPT_5":
		route, ok := h.activeSceneRoute(env.FromUserName)
		if !ok {
			return "请先点「单人角色」或「联机」进入对应模式。", true, nil
		}
		reply, err := h.respond(ctx, NormalizedMessage{
			UserID:         env.FromUserName,
			Text:           decisionButtonText(env.EventKey),
			IdempotencyKey: fmt.Sprintf("%s:%s:%d", env.FromUserName, env.EventKey, env.CreateTime),
			MsgType:        "event",
			Route:          route,
		})
		return reply, true, err
	case "LOOK_TOWER", "SLACK_OFF", "LOOK_FAR", "MISS_HOME", "THINK_FAMILY":
		route, ok := h.activeSceneRoute(env.FromUserName)
		if !ok {
			return "请先点「单人角色」或「联机」进入对应模式。", true, nil
		}
		reply, err := h.respond(ctx, NormalizedMessage{
			UserID:         env.FromUserName,
			Text:           thoughtButtonText(env.EventKey),
			IdempotencyKey: fmt.Sprintf("%s:%s:%d", env.FromUserName, env.EventKey, env.CreateTime),
			MsgType:        "event",
			Route:          route,
		})
		return reply, true, err
	case "MODE_CODEX", "TOGGLE_ADMIN",
		"DEAD_CODE", "COVERAGE", "DOC_CONSISTENCY", "HEALTH", "PROCESS_AUDIT",
		"BUILD_TEST", "GIT_SYNC_PUSH", "NEXT_TASK", "CMD_STATS", "CMD_SESSIONS":
		return "管理员菜单已下线，请使用玩家菜单。", true, nil
	case "PRAY":
		return "祈祷已并入「想 -> 眺望远方」中。", true, nil
	default:
		return "unsupported command", true, nil
	}
}

func (h Handler) applySelectedRoute(msg NormalizedMessage) NormalizedMessage {
	if h.Sessions == nil || msg.Route != RouteFreeChat {
		return msg
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" || strings.HasPrefix(text, "/") {
		return msg
	}
	route, ok := h.Sessions.GetSelectedRoute(msg.UserID)
	if !ok {
		return msg
	}
	switch route {
	case RouteProjectConsult, RouteFreeChat, RouteSoloScene, RouteRoomScene, RouteFeedback:
		msg.Route = route
	}
	return msg
}

func (h Handler) activeSceneRoute(userID string) (RouteKind, bool) {
	if h.Sessions == nil {
		return RouteUnknown, false
	}
	route, ok := h.Sessions.GetSelectedRoute(userID)
	if !ok {
		return RouteUnknown, false
	}
	switch route {
	case RouteSoloScene, RouteRoomScene:
		return route, true
	default:
		return RouteUnknown, false
	}
}

func (h Handler) respond(ctx context.Context, msg NormalizedMessage) (string, error) {
	switch msg.Route {
	case RouteFreeChat:
		if h.FreeChat == nil {
			return "free chat queued", nil
		}
		return h.FreeChat.HandleFreeChat(ctx, msg)
	case RouteProjectConsult:
		if h.Consult == nil {
			return "project consult queued", nil
		}
		return h.Consult.HandleProjectConsult(ctx, msg)
	case RouteSoloScene:
		if h.Solo == nil {
			return "solo scene queued", nil
		}
		return h.Solo.HandleSoloScene(ctx, msg)
	case RouteRoomScene:
		if h.Room == nil {
			return "联机功能正在接入中。", nil
		}
		return h.Room.HandleRoomScene(ctx, msg)
	case RouteFeedback:
		if h.Feedback == nil {
			return "反馈入口正在接入中。", nil
		}
		return h.Feedback.HandleFeedback(ctx, msg)
	default:
		return "unsupported command", nil
	}
}

func decisionButtonText(key string) string {
	label := strings.TrimPrefix(key, "OPT_")
	switch label {
	case "1":
		return "我选择①"
	case "2":
		return "我选择②"
	case "3":
		return "我选择③"
	case "4":
		return "我选择④"
	case "5":
		return "我选择⑤"
	default:
		return "我选择"
	}
}

func thoughtButtonText(key string) string {
	switch key {
	case "LOOK_TOWER":
		return "[自由行动]抬头看看巴别塔建到多高了，看到进度心里踏实些"
	case "SLACK_OFF":
		return "[自由行动]偷偷摸鱼，找个角落躲起来跟工友闲聊喝口水"
	case "LOOK_FAR":
		return "[自由行动]站在高处眺望远方，望着地平线外的世界出神，也在心里默默祈求一点指引和庇佑"
	case "MISS_HOME":
		return "[自由行动]想念远方的故乡，不知父母亲人过得怎样"
	case "THINK_FAMILY":
		return "[自由行动]思考自己在巴别塔的家庭和生活，想想未来的打算"
	default:
		return "[自由行动]"
	}
}

func (h Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func writeXMLReply(w http.ResponseWriter, env InboundEnvelope, reply string, now time.Time) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	out := outboundEnvelope{
		ToUserName:   cdata{Text: env.FromUserName},
		FromUserName: cdata{Text: env.ToUserName},
		CreateTime:   now.Unix(),
		MsgType:      cdata{Text: "text"},
		Content:      cdata{Text: reply},
	}
	data, err := xml.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal outbound xml: %w", err)
	}
	_, err = w.Write(data)
	return err
}
