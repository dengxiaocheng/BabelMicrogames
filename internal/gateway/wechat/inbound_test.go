package wechat_test

import (
	"testing"

	"babel-runtime/internal/gateway/wechat"
)

func TestParseInboundXML(t *testing.T) {
	raw := []byte(`<xml>
<ToUserName><![CDATA[gh_123]]></ToUserName>
<FromUserName><![CDATA[user_1]]></FromUserName>
<CreateTime>1711111111</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[/health]]></Content>
<MsgId>123456</MsgId>
</xml>`)

	env, err := wechat.ParseInboundXML(raw)
	if err != nil {
		t.Fatalf("ParseInboundXML returned error: %v", err)
	}
	if env.FromUserName != "user_1" || env.MsgType != "text" {
		t.Fatalf("unexpected envelope: %#v", env)
	}
}

func TestNormalizeInboundRoutesUnknownSlashCommand(t *testing.T) {
	msg := wechat.NormalizeInbound(wechat.InboundEnvelope{
		FromUserName: "user_1",
		MsgType:      "text",
		Content:      "/config",
		MsgID:        1001,
	})

	if msg.Route != wechat.RouteUnknown {
		t.Fatalf("expected unknown route, got %#v", msg)
	}
}

func TestNormalizeInboundRoutesFreeChat(t *testing.T) {
	msg := wechat.NormalizeInbound(wechat.InboundEnvelope{
		FromUserName: "user_2",
		MsgType:      "text",
		Content:      "你好",
		MsgID:        1002,
	})

	if msg.Route != wechat.RouteFreeChat {
		t.Fatalf("expected free chat route, got %#v", msg)
	}
}

func TestNormalizeInboundRoutesProjectConsult(t *testing.T) {
	msg := wechat.NormalizeInbound(wechat.InboundEnvelope{
		FromUserName: "user_3",
		MsgType:      "text",
		Content:      "/consult 帮我评估方案",
		MsgID:        1003,
	})

	if msg.Route != wechat.RouteProjectConsult {
		t.Fatalf("expected project consult route, got %#v", msg)
	}
}
