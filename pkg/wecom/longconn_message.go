package wecom

import "encoding/json"

const (
	// LongConnCmdSubscribe 为长连接订阅命令。
	LongConnCmdSubscribe = "aibot_subscribe"
	// LongConnCmdPing 为长连接心跳命令。
	LongConnCmdPing = "ping"
	// LongConnCmdMsgCallback 为长连接消息回调命令。
	LongConnCmdMsgCallback = "aibot_msg_callback"
	// LongConnCmdEventCallback 为长连接事件回调命令。
	LongConnCmdEventCallback = "aibot_event_callback"
	// LongConnCmdRespondWelcomeMsg 为进入会话事件欢迎语回复命令。
	LongConnCmdRespondWelcomeMsg = "aibot_respond_welcome_msg"
	// LongConnCmdRespondMsg 为长连接普通消息回复命令。
	LongConnCmdRespondMsg = "aibot_respond_msg"
	// LongConnCmdRespondUpdateMsg 为模板卡片更新命令。
	LongConnCmdRespondUpdateMsg = "aibot_respond_update_msg"
	// LongConnCmdSendMsg 为主动推送消息命令。
	LongConnCmdSendMsg = "aibot_send_msg"
)

// LongConnHeaders 描述长连接消息头。
type LongConnHeaders struct {
	RequestID string `json:"req_id"`
}

// LongConnRequest 描述发往企业微信长连接服务端的请求帧。
type LongConnRequest struct {
	Cmd     string          `json:"cmd"`
	Headers LongConnHeaders `json:"headers"`
	Body    any             `json:"body,omitempty"`
}

// LongConnRawFrame 描述从长连接收到的原始帧。
// 该结构同时覆盖回调帧与请求响应帧。
type LongConnRawFrame struct {
	Cmd     string          `json:"cmd,omitempty"`
	Headers LongConnHeaders `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode *int            `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
}

// LongConnResponse 描述长连接命令响应。
type LongConnResponse struct {
	Headers LongConnHeaders `json:"headers"`
	ErrCode int             `json:"errcode"`
	ErrMsg  string          `json:"errmsg"`
}

// LongConnSubscribeBody 为长连接订阅请求体。
type LongConnSubscribeBody struct {
	BotID  string `json:"bot_id"`
	Secret string `json:"secret"`
}

// LongConnPushMessage 为主动推送消息请求体。
type LongConnPushMessage struct {
	ChatID       string           `json:"chatid"`
	MsgType      string           `json:"msgtype"`
	Markdown     *MarkdownPayload `json:"markdown,omitempty"`
	TemplateCard *TemplateCard    `json:"template_card,omitempty"`
}

// NewLongConnRequest 构造一个通用长连接请求帧。
func NewLongConnRequest(cmd, reqID string, body any) LongConnRequest {
	return LongConnRequest{
		Cmd: cmd,
		Headers: LongConnHeaders{
			RequestID: reqID,
		},
		Body: body,
	}
}

// BuildLongConnSubscribeRequest 构造订阅请求。
func BuildLongConnSubscribeRequest(reqID, botID, secret string) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdSubscribe,
		reqID,
		LongConnSubscribeBody{
			BotID:  botID,
			Secret: secret,
		},
	)
}

// BuildLongConnPingRequest 构造心跳请求。
func BuildLongConnPingRequest(reqID string) LongConnRequest {
	return NewLongConnRequest(LongConnCmdPing, reqID, nil)
}

// BuildLongConnSendMarkdownRequest 构造主动推送 Markdown 消息请求。
func BuildLongConnSendMarkdownRequest(reqID, chatID, content string) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdSendMsg,
		reqID,
		LongConnPushMessage{
			ChatID:  chatID,
			MsgType: "markdown",
			Markdown: &MarkdownPayload{
				Content: content,
			},
		},
	)
}

// BuildLongConnSendTemplateCardRequest 构造主动推送模板卡片请求。
func BuildLongConnSendTemplateCardRequest(reqID, chatID string, card *TemplateCard) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdSendMsg,
		reqID,
		LongConnPushMessage{
			ChatID:       chatID,
			MsgType:      "template_card",
			TemplateCard: card,
		},
	)
}

// HasAckResult 判断当前帧是否是命令响应帧。
func (f LongConnRawFrame) HasAckResult() bool {
	return f.ErrCode != nil || f.ErrMsg != ""
}

// IsCallback 判断当前帧是否是回调帧。
func (f LongConnRawFrame) IsCallback() bool {
	return (f.Cmd == LongConnCmdMsgCallback || f.Cmd == LongConnCmdEventCallback) && len(f.Body) > 0
}

// UnmarshalBody 将原始 body 解码到目标结构。
func (f LongConnRawFrame) UnmarshalBody(target any) error {
	if len(f.Body) == 0 {
		return nil
	}
	return json.Unmarshal(f.Body, target)
}
