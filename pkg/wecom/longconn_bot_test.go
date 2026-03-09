package wecom

import "testing"

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
