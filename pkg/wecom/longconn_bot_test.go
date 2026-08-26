package wecom

import (
	"strings"
	"testing"
)

func TestLongConnBotResolveReplyCommand(t *testing.T) {
	bot := &LongConnBot{}

	msgCommand := bot.resolveReplyCommand(LongConnCmdMsgCallback, &Message{MsgType: "text"})
	if msgCommand != LongConnCmdRespondMsg {
		t.Fatalf("unexpected msg reply command: %s", msgCommand)
	}

	welcomeCommand := bot.resolveReplyCommand(
		LongConnCmdEventCallback,
		&Message{
			MsgType: "event",
			Event: &EventPayload{
				EventType: "enter_chat",
			},
		},
	)
	if welcomeCommand != LongConnCmdRespondWelcomeMsg {
		t.Fatalf("unexpected welcome reply command: %s", welcomeCommand)
	}

	updateCommand := bot.resolveReplyCommand(
		LongConnCmdEventCallback,
		&Message{
			MsgType: "event",
			Event: &EventPayload{
				EventType: "template_card_event",
			},
		},
	)
	if updateCommand != LongConnCmdRespondUpdateMsg {
		t.Fatalf("unexpected update reply command: %s", updateCommand)
	}
}

func TestNormalizeLongConnMessageBody(t *testing.T) {
	body, err := normalizeLongConnMessageBody("hello")
	if err != nil {
		t.Fatalf("normalize string payload: %v", err)
	}

	textMsg, ok := body.(TextMessage)
	if !ok {
		t.Fatalf("unexpected body type: %T", body)
	}
	if textMsg.Text == nil || textMsg.Text.Content != "hello" {
		t.Fatalf("unexpected text message: %+v", textMsg)
	}
}

func TestNormalizeLongConnMessageBodySupportsLatestMessageTypes(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name: "markdown",
			payload: MarkdownMessage{
				MsgType:  "markdown",
				Markdown: MarkdownPayload{Content: "hello"},
			},
		},
		{
			name: "file",
			payload: FileMessage{
				MsgType: "file",
				File:    MediaMessagePayload{MediaID: "media-file"},
			},
		},
		{
			name: "image",
			payload: ImageMessage{
				MsgType: "image",
				Image:   MediaMessagePayload{MediaID: "media-image"},
			},
		},
		{
			name: "voice",
			payload: VoiceMessage{
				MsgType: "voice",
				Voice:   MediaMessagePayload{MediaID: "media-voice"},
			},
		},
		{
			name: "video",
			payload: VideoMessage{
				MsgType: "video",
				Video:   VideoMessagePayload{MediaID: "media-video"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := normalizeLongConnMessageBody(tt.payload)
			if err != nil {
				t.Fatalf("normalize %s: %v", tt.name, err)
			}
			if body == nil {
				t.Fatal("normalized body is nil")
			}
		})
	}
}

func TestNormalizeLongConnMessageBodyRejectsUnsupportedCombinations(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		wantErr string
	}{
		{
			name: "stream with template card",
			payload: StreamWithTemplateCardMessage{
				MsgType: "stream",
			},
			wantErr: "does not support stream with template card",
		},
		{
			name: "stream msg_item",
			payload: StreamReply{
				MsgType: "stream",
				Stream: StreamReplyBody{
					ID:      "stream-1",
					MsgItem: []MixedItem{{MsgType: "image"}},
				},
			},
			wantErr: "does not support msg_item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeLongConnMessageBody(tt.payload)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLongConnChatType(t *testing.T) {
	for _, chatType := range []LongConnChatType{
		LongConnChatTypeAuto,
		LongConnChatTypeSingle,
		LongConnChatTypeGroup,
	} {
		if err := validateLongConnChatType(chatType); err != nil {
			t.Fatalf("valid chat type %d rejected: %v", chatType, err)
		}
	}
	if err := validateLongConnChatType(99); err == nil {
		t.Fatal("invalid chat type should be rejected")
	}
}

func TestLongConnSendValidation(t *testing.T) {
	var nilBot *LongConnBot
	if err := nilBot.SendMarkdown("chat-1", "hello"); err == nil {
		t.Fatal("nil bot should be rejected")
	}

	bot := &LongConnBot{}
	tests := []struct {
		name string
		send func() error
	}{
		{
			name: "blank markdown chat id",
			send: func() error { return bot.SendMarkdown(" \t", "hello") },
		},
		{
			name: "blank template card chat id",
			send: func() error { return bot.SendTemplateCard(" \n", &TemplateCard{}) },
		},
		{
			name: "blank media chat id",
			send: func() error { return bot.SendFile(" ", "media-1") },
		},
		{
			name: "blank media id",
			send: func() error { return bot.SendImage("chat-1", " \t") },
		},
		{
			name: "invalid chat type",
			send: func() error {
				return bot.SendVideoWithChatType("chat-1", 99, "media-1", "", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.send(); err == nil {
				t.Fatal("invalid send arguments should be rejected")
			}
		})
	}
}

func TestNormalizeLongConnOneShotBodyForTemplateUpdate(t *testing.T) {
	card := &TemplateCard{
		CardType: "button_interaction",
	}
	body, err := normalizeLongConnOneShotBody(LongConnCmdRespondUpdateMsg, card, "")
	if err != nil {
		t.Fatalf("normalize template update: %v", err)
	}

	updateMsg, ok := body.(UpdateTemplateCardMessage)
	if !ok {
		t.Fatalf("unexpected update body type: %T", body)
	}
	if updateMsg.ResponseType != "update_template_card" {
		t.Fatalf("unexpected response type: %s", updateMsg.ResponseType)
	}
	if updateMsg.TemplateCard != card {
		t.Fatalf("unexpected template card pointer: %+v", updateMsg.TemplateCard)
	}
}
