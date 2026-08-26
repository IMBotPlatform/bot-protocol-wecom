package wecom

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildStreamImageItemFromBytes(t *testing.T) {
	item, err := BuildStreamImageItemFromBytes([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.MsgType != "image" {
		t.Fatalf("unexpected msgtype: %s", item.MsgType)
	}
	if item.Image == nil {
		t.Fatalf("image payload is nil")
	}
	if item.Text != nil {
		t.Fatalf("text payload should be nil")
	}

	// hello -> base64: aGVsbG8=; md5: 5d41402abc4b2a76b9719d911017c592
	if item.Image.Base64 != "aGVsbG8=" {
		t.Fatalf("unexpected base64: %s", item.Image.Base64)
	}
	if item.Image.MD5 != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("unexpected md5: %s", item.Image.MD5)
	}
}

func TestBuildStreamReplyWithMsgItems_MsgItemOnlyOnFinish(t *testing.T) {
	items := []MixedItem{
		{
			MsgType: "image",
			Image: &ImagePayload{
				Base64: "BASE64",
				MD5:    "MD5",
			},
		},
	}

	reply := BuildStreamReplyWithMsgItems("stream-id", "content", false, items)
	if reply.Stream.Finish {
		t.Fatalf("finish should be false")
	}
	if len(reply.Stream.MsgItem) != 0 {
		t.Fatalf("msg_item should be empty when finish=false, got %d", len(reply.Stream.MsgItem))
	}
	data, err := json.Marshal(reply)
	if err != nil {
		t.Fatalf("marshal reply: %v", err)
	}
	if strings.Contains(string(data), "\"msg_item\"") {
		t.Fatalf("msg_item should be omitted when finish=false, got json=%s", string(data))
	}

	reply2 := BuildStreamReplyWithMsgItems("stream-id", "content", true, items)
	if !reply2.Stream.Finish {
		t.Fatalf("finish should be true")
	}
	if len(reply2.Stream.MsgItem) != 1 {
		t.Fatalf("msg_item should be present when finish=true, got %d", len(reply2.Stream.MsgItem))
	}
	data2, err := json.Marshal(reply2)
	if err != nil {
		t.Fatalf("marshal reply2: %v", err)
	}
	if !strings.Contains(string(data2), "\"msg_item\"") {
		t.Fatalf("msg_item should exist when finish=true, got json=%s", string(data2))
	}

	// 验证 slice 已拷贝（浅拷贝足够）：修改原 slice 元素不应影响 reply2。
	items[0] = MixedItem{
		MsgType: "text",
		Text: &TextPayload{
			Content: "changed",
		},
	}
	if reply2.Stream.MsgItem[0].MsgType != "image" {
		t.Fatalf("msg_item should not be affected by caller mutation, got %s", reply2.Stream.MsgItem[0].MsgType)
	}
}

func TestParseMessageWithQuoteVideo(t *testing.T) {
	raw := []byte(`{
		"msgid":"MSGID",
		"msgtype":"text",
		"text":{"content":"hello"},
		"quote":{
			"msgtype":"video",
			"video":{"url":"https://example.com/video","aeskey":"AESKEY"}
		}
	}`)

	msg, err := parseMessage(raw)
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	if msg.Quote == nil {
		t.Fatal("quote should not be nil")
	}
	if msg.Quote.MsgType != "video" {
		t.Fatalf("unexpected quote msgtype: %s", msg.Quote.MsgType)
	}
	if msg.Quote.Video == nil {
		t.Fatal("quote video should not be nil")
	}
	if msg.Quote.Video.URL != "https://example.com/video" {
		t.Fatalf("unexpected quote video url: %s", msg.Quote.Video.URL)
	}
	if msg.Quote.Video.AESKey != "AESKEY" {
		t.Fatalf("unexpected quote video aeskey: %s", msg.Quote.Video.AESKey)
	}
}

func TestMediaMessagesJSON(t *testing.T) {
	tests := []struct {
		name string
		msg  any
		want string
	}{
		{
			name: "file",
			msg: FileMessage{
				MsgType: "file",
				File:    MediaMessagePayload{MediaID: "media-file"},
			},
			want: `{"msgtype":"file","file":{"media_id":"media-file"}}`,
		},
		{
			name: "image",
			msg: ImageMessage{
				MsgType: "image",
				Image:   MediaMessagePayload{MediaID: "media-image"},
			},
			want: `{"msgtype":"image","image":{"media_id":"media-image"}}`,
		},
		{
			name: "voice",
			msg: VoiceMessage{
				MsgType: "voice",
				Voice:   MediaMessagePayload{MediaID: "media-voice"},
			},
			want: `{"msgtype":"voice","voice":{"media_id":"media-voice"}}`,
		},
		{
			name: "video",
			msg: VideoMessage{
				MsgType: "video",
				Video: VideoMessagePayload{
					MediaID:     "media-video",
					Title:       "title",
					Description: "description",
				},
			},
			want: `{"msgtype":"video","video":{"media_id":"media-video","title":"title","description":"description"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("marshal media message: %v", err)
			}
			if string(data) != tt.want {
				t.Fatalf("unexpected json: got=%s want=%s", string(data), tt.want)
			}
		})
	}
}
