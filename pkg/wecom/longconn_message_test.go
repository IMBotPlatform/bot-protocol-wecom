package wecom

import (
	"encoding/json"
	"testing"
)

func TestBuildLongConnSubscribeRequest(t *testing.T) {
	req := BuildLongConnSubscribeRequest("req-1", "bot-id", "secret")
	if req.Cmd != LongConnCmdSubscribe {
		t.Fatalf("unexpected cmd: %s", req.Cmd)
	}
	if req.Headers.RequestID != "req-1" {
		t.Fatalf("unexpected req_id: %s", req.Headers.RequestID)
	}

	body, ok := req.Body.(LongConnSubscribeBody)
	if !ok {
		t.Fatalf("unexpected body type: %T", req.Body)
	}
	if body.BotID != "bot-id" || body.Secret != "secret" {
		t.Fatalf("unexpected subscribe body: %+v", body)
	}
}

func TestBuildLongConnPingRequestJSON(t *testing.T) {
	req := BuildLongConnPingRequest("req-ping")
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal ping request: %v", err)
	}
	if string(data) != `{"cmd":"ping","headers":{"req_id":"req-ping"}}` {
		t.Fatalf("unexpected json: %s", string(data))
	}
}

func TestBuildLongConnSendMarkdownRequest(t *testing.T) {
	req := BuildLongConnSendMarkdownRequest("req-send", "chat-1", "hello")
	if req.Cmd != LongConnCmdSendMsg {
		t.Fatalf("unexpected cmd: %s", req.Cmd)
	}

	body, ok := req.Body.(LongConnPushMessage)
	if !ok {
		t.Fatalf("unexpected body type: %T", req.Body)
	}
	if body.ChatID != "chat-1" || body.MsgType != "markdown" {
		t.Fatalf("unexpected push body: %+v", body)
	}
	if body.Markdown == nil || body.Markdown.Content != "hello" {
		t.Fatalf("unexpected markdown payload: %+v", body.Markdown)
	}
}

func TestLongConnRawFrameUnmarshalBody(t *testing.T) {
	frame := LongConnRawFrame{
		Cmd: LongConnCmdMsgCallback,
		Headers: LongConnHeaders{
			RequestID: "req-callback",
		},
		Body: json.RawMessage(`{"msgid":"MSGID","msgtype":"text","text":{"content":"hello"}}`),
	}

	var msg Message
	if err := frame.UnmarshalBody(&msg); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if msg.MsgID != "MSGID" || msg.MsgType != "text" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.Text == nil || msg.Text.Content != "hello" {
		t.Fatalf("unexpected text payload: %+v", msg.Text)
	}
}

func TestLongConnRawFrameHasAckResult(t *testing.T) {
	okCode := 0
	frame := LongConnRawFrame{
		Headers: LongConnHeaders{
			RequestID: "req-ack",
		},
		ErrCode: &okCode,
		ErrMsg:  "ok",
	}
	if !frame.HasAckResult() {
		t.Fatal("ack frame should be detected")
	}
	if frame.IsCallback() {
		t.Fatal("ack frame should not be treated as callback")
	}
}
