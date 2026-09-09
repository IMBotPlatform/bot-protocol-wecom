package wecom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test the actual JSON/WebSocket/ACK path, without real credentials or WeCom.
func longConnFixture(t *testing.T, handler Handler, callbacks []LongConnRequest) (*LongConnBot, <-chan capturedLongConnRequest, <-chan error) {
	t.Helper()
	frames := make(chan capturedLongConnRequest, 128)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var req capturedLongConnRequest
			if conn.ReadJSON(&req) != nil {
				return
			}
			frames <- req
			ack := map[string]any{"headers": req.Headers, "errcode": 0}
			if req.Cmd == LongConnCmdUploadMediaInit {
				ack["body"] = map[string]string{"upload_id": "upload"}
			}
			if req.Cmd == LongConnCmdUploadMediaFinish {
				ack["body"] = map[string]string{"media_id": "image-1", "type": "image"}
			}
			if conn.WriteJSON(ack) != nil {
				return
			}
			if req.Cmd == LongConnCmdSubscribe {
				for _, cb := range callbacks {
					if conn.WriteJSON(cb) != nil {
						return
					}
				}
			}
		}
	}))
	bot, err := NewLongConnBotWithOptions("fixture-bot", "fixture-secret", handler, LongConnOptions{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), RequestTimeout: time.Second, PingInterval: 20 * time.Millisecond, ReconnectInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- bot.Start(context.Background()) }()
	t.Cleanup(func() { bot.Close(); server.Close() })
	return bot, frames, done
}
func nextLongFrame(t *testing.T, frames <-chan capturedLongConnRequest, cmd string) capturedLongConnRequest {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case f := <-frames:
			if f.Cmd == cmd {
				return f
			}
		case <-timer.C:
			t.Fatalf("missing %s", cmd)
			return capturedLongConnRequest{}
		}
	}
}
func waitLongReady(t *testing.T, b *LongConnBot) {
	t.Helper()
	for until := time.Now().Add(time.Second); time.Now().Before(until); {
		if b.Ready() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("not ready")
}
func TestLongConnCallbackIdentityDedupAndSynchronousPush(t *testing.T) {
	var count atomic.Int32
	observed := make(chan Context, 2)
	handler := HandlerFunc(func(c Context) <-chan Chunk {
		count.Add(1)
		observed <- c
		// A synchronous application send must not deadlock the ACK reader.
		err := c.LongConn.SendMarkdownWithChatType(c.Message.From.UserID, LongConnChatTypeSingle, "pushed")
		out := make(chan Chunk, 1)
		if err != nil {
			out <- Chunk{Content: "push failed", IsFinal: true}
		} else {
			out <- Chunk{Content: "help", IsFinal: true}
		}
		close(out)
		return out
	})
	cb := NewLongConnRequest(LongConnCmdMsgCallback, "req-1", map[string]any{"msgid": "msg-1", "chattype": "single", "from": map[string]string{"userid": "user-1"}, "msgtype": "text", "text": map[string]string{"content": "/help"}})
	bot, frames, _ := longConnFixture(t, handler, []LongConnRequest{cb, cb})
	waitLongReady(t, bot)
	c := <-observed
	if c.StreamID != "msg-1" || c.RequestID != "req-1" || c.Bot != nil || c.LongConn != bot {
		t.Fatal("lost callback identity")
	}
	push := nextLongFrame(t, frames, LongConnCmdSendMsg)
	var p LongConnPushMessage
	json.Unmarshal(push.Body, &p)
	if p.ChatID != "user-1" || p.ChatType != LongConnChatTypeSingle {
		t.Fatal("wrong single-chat target")
	}
	reply := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	var stream StreamReply
	json.Unmarshal(reply.Body, &stream)
	if stream.Stream.Content != "help" || !stream.Stream.Finish {
		t.Fatal("pipeline failed")
	}
	time.Sleep(30 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("duplicate callback executed")
	}
	bot.Close()
}
func TestLongConnReplacedStopsAndRedactsAuthFailure(t *testing.T) {
	cb := NewLongConnRequest(LongConnCmdEventCallback, "disconnected", map[string]any{"msgtype": "event", "event": map[string]string{"eventtype": "disconnected_event"}})
	bot, _, done := longConnFixture(t, nil, []LongConnRequest{cb})
	select {
	case err := <-done:
		if !errors.Is(err, ErrLongConnReplaced) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("replaced bot reconnected")
	}
	if bot.Ready() {
		t.Fatal("replaced bot ready")
	}
	err := (&longConnAPIError{cmd: LongConnCmdSubscribe, requestID: "private-id", errCode: 400, errMsg: "private-secret"}).Error()
	if strings.Contains(err, "private") || !strings.Contains(err, "400") {
		t.Fatal("unsafe API error")
	}
}
func TestLongConnStreamReplacementFinalPayloadAndImage(t *testing.T) {
	bot, frames, _ := longConnFixture(t, nil, nil)
	waitLongReady(t, bot)
	source := make(chan Chunk)
	done := make(chan error, 1)
	go func() { done <- bot.consumeMessageChunksWithWindow("stream-req", source, time.Hour, time.Minute) }()
	source <- Chunk{Content: "thinking"}
	first := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	source <- Chunk{Replace: true, Content: "answer"}
	source <- Chunk{Content: "!"}
	imageData := base64.StdEncoding.EncodeToString([]byte("image-fixture"))
	source <- Chunk{Payload: BuildStreamReplyWithMsgItems("http-stream", "answer!", true, []MixedItem{{MsgType: "image", Image: &ImagePayload{Base64: imageData}}}), IsFinal: true}
	close(source)
	last := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	var a, z StreamReply
	json.Unmarshal(first.Body, &a)
	json.Unmarshal(last.Body, &z)
	if a.Stream.ID != z.Stream.ID || z.Stream.ID == "http-stream" || z.Stream.Content != "answer!" || !z.Stream.Finish || len(z.Stream.MsgItem) > 0 {
		t.Fatal("final payload did not replace same stream")
	}
	img := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	var m ImageMessage
	json.Unmarshal(img.Body, &m)
	if m.Image.MediaID != "image-1" {
		t.Fatal("image not uploaded and replied")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestLongConnStreamWindowClosesAndResumes(t *testing.T) {
	bot, frames, _ := longConnFixture(t, nil, nil)
	waitLongReady(t, bot)
	source := make(chan Chunk)
	done := make(chan error, 1)
	go func() { done <- bot.consumeMessageChunksWithWindow("window", source, time.Hour, 25*time.Millisecond) }()
	source <- Chunk{Content: "working"}
	first := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	ended := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	source <- Chunk{Replace: true, Content: "done", IsFinal: true}
	close(source)
	final := nextLongFrame(t, frames, LongConnCmdRespondMsg)
	var a, b, c StreamReply
	json.Unmarshal(first.Body, &a)
	json.Unmarshal(ended.Body, &b)
	json.Unmarshal(final.Body, &c)
	if a.Stream.ID != b.Stream.ID || !b.Stream.Finish || c.Stream.ID == a.Stream.ID || !c.Stream.Finish || c.Stream.Content != "done" {
		t.Fatal("stream window not respected")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestLongConnCoalescingAndFailureDrainsProducer(t *testing.T) {
	source := make(chan Chunk)
	out := coalesceLongConnChunks(source, time.Hour)
	source <- Chunk{Content: "one"}
	if c := <-out; c.Content != "one" {
		t.Fatal(c)
	}
	source <- Chunk{Content: "two"}
	source <- Chunk{Replace: true, Content: ""}
	source <- Chunk{Content: "three"}
	go func() { source <- Chunk{Content: "!", IsFinal: true}; close(source) }()
	if c := <-out; c.Content != "three!" || !c.IsFinal || !c.Replace {
		t.Fatal(c)
	}
	for range out {
	}
	bot, _ := NewLongConnBot("fixture", "fixture", nil)
	source = make(chan Chunk)
	done := make(chan error, 1)
	go func() { done <- bot.consumeMessageChunks("no-connection", source) }()
	source <- Chunk{Content: "hello"}
	source <- Chunk{Content: "next"}
	close(source)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("failure hidden")
		}
	case <-time.After(time.Second):
		t.Fatal("producer blocked")
	}
}
func TestLongConnMessageRateSharedAndCancelled(t *testing.T) {
	bot, _ := NewLongConnBot("fixture", "fixture", nil)
	body, _ := json.Marshal(Message{MsgID: "msg", ChatType: "single", From: MessageSender{UserID: "u"}})
	bot.acceptCallback(LongConnRawFrame{Cmd: LongConnCmdMsgCallback, Headers: LongConnHeaders{RequestID: "r"}, Body: body}, Message{MsgID: "msg", ChatType: "single", From: MessageSender{UserID: "u"}})
	for i := 0; i < 30; i++ {
		if err := bot.waitMessageRate(context.Background(), LongConnCmdRespondMsg, "r", nil); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := bot.waitMessageRate(ctx, LongConnCmdSendMsg, "push", LongConnPushMessage{ChatID: "u", ChatType: LongConnChatTypeSingle}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("passive/proactive did not share budget")
	}
	if len(bot.messageTimes["1:u"]) != 30 {
		t.Fatal("cancelled request reserved capacity")
	}
}

func TestLongConnReconnectCleansHeartbeatAndAuthFailureStops(t *testing.T) {
	for _, authFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "reconnect", true: "auth"}[authFailure], func(t *testing.T) {
			var connections, pings atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer c.Close()
				n := connections.Add(1)
				for {
					var f capturedLongConnRequest
					if c.ReadJSON(&f) != nil {
						return
					}
					if authFailure {
						c.WriteJSON(map[string]any{"headers": f.Headers, "errcode": 400, "errmsg": "private-credential"})
						return
					}
					c.WriteJSON(map[string]any{"headers": f.Headers, "errcode": 0})
					if f.Cmd == LongConnCmdSubscribe && n < 4 {
						return
					}
					if f.Cmd == LongConnCmdPing {
						pings.Add(1)
					}
				}
			}))
			defer server.Close()
			bot, _ := NewLongConnBotWithOptions("fixture", "fixture", nil, LongConnOptions{WebSocketURL: "ws" + strings.TrimPrefix(server.URL, "http"), PingInterval: 20 * time.Millisecond, ReconnectInterval: 5 * time.Millisecond, RequestTimeout: time.Second})
			defer bot.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- bot.Start(ctx) }()
			if authFailure {
				select {
				case err := <-done:
					if err == nil || strings.Contains(err.Error(), "private") {
						t.Fatal("auth failure not safely surfaced")
					}
				case <-time.After(time.Second):
					t.Fatal("auth retries")
				}
				if connections.Load() != 1 || bot.Ready() {
					t.Fatal("invalid auth state")
				}
				return
			}
			for until := time.Now().Add(time.Second); connections.Load() < 4 && time.Now().Before(until); {
				time.Sleep(time.Millisecond)
			}
			waitLongReady(t, bot)
			time.Sleep(125 * time.Millisecond)
			if n := pings.Load(); n < 3 || n > 8 {
				t.Fatalf("old heartbeat loops survived: %d", n)
			}
			if err := bot.Start(ctx); err == nil {
				t.Fatal("duplicate start accepted")
			}
			cancel()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("cancel did not stop")
			}
			if bot.Ready() {
				t.Fatal("closed subscription ready")
			}
		})
	}
}

func TestLongConnCoalescerContinuesPayloadSnapshot(t *testing.T) {
	source := make(chan Chunk, 3)
	source <- Chunk{Payload: BuildStreamReply("upstream", "start", false)}
	source <- Chunk{Content: " end", IsFinal: true}
	close(source)
	out := coalesceLongConnChunks(source, time.Hour)
	if c := <-out; c.Payload == nil {
		t.Fatal("payload lost")
	}
	if c := <-out; c.Content != "start end" || !c.IsFinal {
		t.Fatal("payload snapshot not retained")
	}
	for range out {
	}
}
