package wechat

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type InboundEnvelope struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgID        int64  `xml:"MsgId"`
	Event        string `xml:"Event"`
	EventKey     string `xml:"EventKey"`
}

type RouteKind string

const (
	RouteFreeChat       RouteKind = "free_chat"
	RouteProjectConsult RouteKind = "project_consult"
	RouteSoloScene      RouteKind = "solo_scene"
	RouteRoomScene      RouteKind = "room_scene"
	RouteFeedback       RouteKind = "feedback"
	RouteUnknown        RouteKind = "unknown"
)

type NormalizedMessage struct {
	UserID         string    `json:"user_id"`
	Text           string    `json:"text"`
	IdempotencyKey string    `json:"idempotency_key"`
	MsgType        string    `json:"msg_type"`
	Route          RouteKind `json:"route"`
}

func ParseInboundXML(raw []byte) (InboundEnvelope, error) {
	var env InboundEnvelope
	if err := xml.Unmarshal(raw, &env); err != nil {
		return InboundEnvelope{}, err
	}
	if env.FromUserName == "" {
		return InboundEnvelope{}, fmt.Errorf("missing from user")
	}
	return env, nil
}

func NormalizeInbound(env InboundEnvelope) NormalizedMessage {
	text := strings.TrimSpace(env.Content)
	if !strings.EqualFold(env.MsgType, "text") {
		return NormalizedMessage{
			UserID:         env.FromUserName,
			Text:           text,
			IdempotencyKey: fmt.Sprintf("%s:%d", env.FromUserName, env.MsgID),
			MsgType:        env.MsgType,
			Route:          RouteUnknown,
		}
	}
	route := RouteFreeChat
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(lower, "/consult ") || lower == "/consult" || strings.HasPrefix(lower, "consult:"):
		route = RouteProjectConsult
	case strings.HasPrefix(text, "/"):
		route = RouteUnknown
	}

	return NormalizedMessage{
		UserID:         env.FromUserName,
		Text:           text,
		IdempotencyKey: fmt.Sprintf("%s:%d", env.FromUserName, env.MsgID),
		MsgType:        env.MsgType,
		Route:          route,
	}
}
