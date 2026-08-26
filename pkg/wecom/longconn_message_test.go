package wecom

import (
	"encoding/base64"
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

func TestBuildLongConnSendMarkdownRequestWithChatType(t *testing.T) {
	req := BuildLongConnSendMarkdownRequestWithChatType(
		"req-send-single",
		"user-1",
		LongConnChatTypeSingle,
		"hello",
	)
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal markdown request: %v", err)
	}
	want := `{"cmd":"aibot_send_msg","headers":{"req_id":"req-send-single"},"body":{"chatid":"user-1","chat_type":1,"msgtype":"markdown","markdown":{"content":"hello"}}}`
	if string(data) != want {
		t.Fatalf("unexpected json: got=%s want=%s", string(data), want)
	}
}

func TestBuildLongConnSendMediaRequests(t *testing.T) {
	tests := []struct {
		name      string
		request   LongConnRequest
		msgType   string
		mediaID   string
		chatType  LongConnChatType
		wantTitle string
		wantDesc  string
	}{
		{
			name:     "file",
			request:  BuildLongConnSendFileRequestWithChatType("req-file", "group-1", LongConnChatTypeGroup, "media-file"),
			msgType:  "file",
			mediaID:  "media-file",
			chatType: LongConnChatTypeGroup,
		},
		{
			name:     "image",
			request:  BuildLongConnSendImageRequest("req-image", "chat-1", "media-image"),
			msgType:  "image",
			mediaID:  "media-image",
			chatType: LongConnChatTypeAuto,
		},
		{
			name:     "voice",
			request:  BuildLongConnSendVoiceRequest("req-voice", "chat-1", "media-voice"),
			msgType:  "voice",
			mediaID:  "media-voice",
			chatType: LongConnChatTypeAuto,
		},
		{
			name:      "video",
			request:   BuildLongConnSendVideoRequest("req-video", "chat-1", "media-video", "title", "description"),
			msgType:   "video",
			mediaID:   "media-video",
			chatType:  LongConnChatTypeAuto,
			wantTitle: "title",
			wantDesc:  "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.request.Cmd != LongConnCmdSendMsg {
				t.Fatalf("unexpected cmd: %s", tt.request.Cmd)
			}
			body, ok := tt.request.Body.(LongConnPushMessage)
			if !ok {
				t.Fatalf("unexpected body type: %T", tt.request.Body)
			}
			if body.MsgType != tt.msgType || body.ChatType != tt.chatType {
				t.Fatalf("unexpected push body: %+v", body)
			}

			var gotMediaID string
			switch tt.msgType {
			case "file":
				if body.File == nil {
					t.Fatal("file payload is nil")
				}
				gotMediaID = body.File.MediaID
			case "image":
				if body.Image == nil {
					t.Fatal("image payload is nil")
				}
				gotMediaID = body.Image.MediaID
			case "voice":
				if body.Voice == nil {
					t.Fatal("voice payload is nil")
				}
				gotMediaID = body.Voice.MediaID
			case "video":
				if body.Video == nil {
					t.Fatal("video payload is nil")
				}
				gotMediaID = body.Video.MediaID
				if body.Video.Title != tt.wantTitle || body.Video.Description != tt.wantDesc {
					t.Fatalf("unexpected video payload: %+v", body.Video)
				}
			}
			if gotMediaID != tt.mediaID {
				t.Fatalf("unexpected media id: got=%s want=%s", gotMediaID, tt.mediaID)
			}
		})
	}
}

func TestBuildLongConnUploadMediaRequests(t *testing.T) {
	initReq := BuildLongConnUploadMediaInitRequest("req-init", LongConnUploadMediaInitBody{
		Type:        LongConnMediaTypeFile,
		Filename:    "report.pdf",
		TotalSize:   1024,
		TotalChunks: 1,
		MD5:         "md5",
	})
	if initReq.Cmd != LongConnCmdUploadMediaInit {
		t.Fatalf("unexpected init cmd: %s", initReq.Cmd)
	}

	chunk := []byte("chunk-data")
	chunkReq := BuildLongConnUploadMediaChunkRequest("req-chunk", "upload-1", 2, chunk)
	if chunkReq.Cmd != LongConnCmdUploadMediaChunk {
		t.Fatalf("unexpected chunk cmd: %s", chunkReq.Cmd)
	}
	chunkBody, ok := chunkReq.Body.(LongConnUploadMediaChunkBody)
	if !ok {
		t.Fatalf("unexpected chunk body type: %T", chunkReq.Body)
	}
	decoded, err := base64.StdEncoding.DecodeString(chunkBody.Base64Data)
	if err != nil {
		t.Fatalf("decode chunk: %v", err)
	}
	if string(decoded) != string(chunk) || chunkBody.ChunkIndex != 2 {
		t.Fatalf("unexpected chunk body: %+v", chunkBody)
	}

	finishReq := BuildLongConnUploadMediaFinishRequest("req-finish", "upload-1")
	if finishReq.Cmd != LongConnCmdUploadMediaFinish {
		t.Fatalf("unexpected finish cmd: %s", finishReq.Cmd)
	}
}

func TestLongConnUploadMediaResultAcceptsStringAndNumberTimestamp(t *testing.T) {
	for _, raw := range []string{
		`{"type":"file","media_id":"media-1","created_at":"1380000000"}`,
		`{"type":"file","media_id":"media-1","created_at":1380000000}`,
	} {
		var result LongConnUploadMediaResult
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatalf("unmarshal result %s: %v", raw, err)
		}
		if result.Type != LongConnMediaTypeFile || result.MediaID != "media-1" || result.CreatedAt != 1380000000 {
			t.Fatalf("unexpected result: %+v", result)
		}
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
