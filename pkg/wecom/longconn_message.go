package wecom

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
)

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
	// LongConnCmdUploadMediaInit 为临时素材上传初始化命令。
	LongConnCmdUploadMediaInit = "aibot_upload_media_init"
	// LongConnCmdUploadMediaChunk 为临时素材分片上传命令。
	LongConnCmdUploadMediaChunk = "aibot_upload_media_chunk"
	// LongConnCmdUploadMediaFinish 为临时素材上传结束命令。
	LongConnCmdUploadMediaFinish = "aibot_upload_media_finish"
)

// LongConnChatType 指定主动推送目标的会话类型。
type LongConnChatType uint32

const (
	// LongConnChatTypeAuto 兼容单聊和群聊，服务端优先按群聊解析。
	LongConnChatTypeAuto LongConnChatType = iota
	// LongConnChatTypeSingle 表示单聊，chatid 应填写用户 userid。
	LongConnChatTypeSingle
	// LongConnChatTypeGroup 表示群聊，chatid 应填写群聊 chatid。
	LongConnChatTypeGroup
)

// LongConnMediaType 表示长连接临时素材类型。
type LongConnMediaType string

const (
	// LongConnMediaTypeFile 表示普通文件。
	LongConnMediaTypeFile LongConnMediaType = "file"
	// LongConnMediaTypeImage 表示图片。
	LongConnMediaTypeImage LongConnMediaType = "image"
	// LongConnMediaTypeVoice 表示语音。
	LongConnMediaTypeVoice LongConnMediaType = "voice"
	// LongConnMediaTypeVideo 表示视频。
	LongConnMediaTypeVideo LongConnMediaType = "video"
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
	Body    json.RawMessage `json:"body,omitempty"`
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
	ChatID       string               `json:"chatid"`
	ChatType     LongConnChatType     `json:"chat_type,omitempty"`
	MsgType      string               `json:"msgtype"`
	Markdown     *MarkdownPayload     `json:"markdown,omitempty"`
	TemplateCard *TemplateCard        `json:"template_card,omitempty"`
	File         *MediaMessagePayload `json:"file,omitempty"`
	Image        *MediaMessagePayload `json:"image,omitempty"`
	Voice        *MediaMessagePayload `json:"voice,omitempty"`
	Video        *VideoMessagePayload `json:"video,omitempty"`
}

// LongConnUploadMediaInitBody 为临时素材上传初始化请求体。
type LongConnUploadMediaInitBody struct {
	Type        LongConnMediaType `json:"type"`
	Filename    string            `json:"filename"`
	TotalSize   int               `json:"total_size"`
	TotalChunks int               `json:"total_chunks"`
	MD5         string            `json:"md5,omitempty"`
}

// LongConnUploadMediaChunkBody 为临时素材分片上传请求体。
type LongConnUploadMediaChunkBody struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}

// LongConnUploadMediaFinishBody 为临时素材上传结束请求体。
type LongConnUploadMediaFinishBody struct {
	UploadID string `json:"upload_id"`
}

// LongConnUploadMediaInitResponseBody 为上传初始化响应体。
type LongConnUploadMediaInitResponseBody struct {
	UploadID string `json:"upload_id"`
}

// LongConnUploadMediaResult 为临时素材上传完成后的结果。
type LongConnUploadMediaResult struct {
	Type      LongConnMediaType `json:"type"`
	MediaID   string            `json:"media_id"`
	CreatedAt int64             `json:"created_at"`
}

// UnmarshalJSON 同时兼容 created_at 为 JSON 数字或数字字符串的服务端响应。
func (r *LongConnUploadMediaResult) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("longconn upload media result is nil")
	}

	var wire struct {
		Type      LongConnMediaType `json:"type"`
		MediaID   string            `json:"media_id"`
		CreatedAt json.RawMessage   `json:"created_at"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	var createdAt int64
	if len(wire.CreatedAt) > 0 && string(wire.CreatedAt) != "null" {
		if err := json.Unmarshal(wire.CreatedAt, &createdAt); err != nil {
			var text string
			if stringErr := json.Unmarshal(wire.CreatedAt, &text); stringErr != nil {
				return fmt.Errorf("decode created_at: %w", err)
			}
			parsed, parseErr := strconv.ParseInt(text, 10, 64)
			if parseErr != nil {
				return fmt.Errorf("decode created_at: %w", parseErr)
			}
			createdAt = parsed
		}
	}

	r.Type = wire.Type
	r.MediaID = wire.MediaID
	r.CreatedAt = createdAt
	return nil
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
	return BuildLongConnSendMarkdownRequestWithChatType(
		reqID,
		chatID,
		LongConnChatTypeAuto,
		content,
	)
}

// BuildLongConnSendMarkdownRequestWithChatType 构造指定会话类型的 Markdown 主动推送请求。
func BuildLongConnSendMarkdownRequestWithChatType(reqID, chatID string, chatType LongConnChatType, content string) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdSendMsg,
		reqID,
		LongConnPushMessage{
			ChatID:   chatID,
			ChatType: chatType,
			MsgType:  "markdown",
			Markdown: &MarkdownPayload{
				Content: content,
			},
		},
	)
}

// BuildLongConnSendTemplateCardRequest 构造主动推送模板卡片请求。
func BuildLongConnSendTemplateCardRequest(reqID, chatID string, card *TemplateCard) LongConnRequest {
	return BuildLongConnSendTemplateCardRequestWithChatType(
		reqID,
		chatID,
		LongConnChatTypeAuto,
		card,
	)
}

// BuildLongConnSendTemplateCardRequestWithChatType 构造指定会话类型的模板卡片主动推送请求。
func BuildLongConnSendTemplateCardRequestWithChatType(reqID, chatID string, chatType LongConnChatType, card *TemplateCard) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdSendMsg,
		reqID,
		LongConnPushMessage{
			ChatID:       chatID,
			ChatType:     chatType,
			MsgType:      "template_card",
			TemplateCard: card,
		},
	)
}

// BuildLongConnSendFileRequest 构造文件主动推送请求。
func BuildLongConnSendFileRequest(reqID, chatID, mediaID string) LongConnRequest {
	return BuildLongConnSendFileRequestWithChatType(reqID, chatID, LongConnChatTypeAuto, mediaID)
}

// BuildLongConnSendFileRequestWithChatType 构造指定会话类型的文件主动推送请求。
func BuildLongConnSendFileRequestWithChatType(reqID, chatID string, chatType LongConnChatType, mediaID string) LongConnRequest {
	return buildLongConnSendMediaRequest(reqID, chatID, chatType, "file", mediaID, "", "")
}

// BuildLongConnSendImageRequest 构造图片主动推送请求。
func BuildLongConnSendImageRequest(reqID, chatID, mediaID string) LongConnRequest {
	return BuildLongConnSendImageRequestWithChatType(reqID, chatID, LongConnChatTypeAuto, mediaID)
}

// BuildLongConnSendImageRequestWithChatType 构造指定会话类型的图片主动推送请求。
func BuildLongConnSendImageRequestWithChatType(reqID, chatID string, chatType LongConnChatType, mediaID string) LongConnRequest {
	return buildLongConnSendMediaRequest(reqID, chatID, chatType, "image", mediaID, "", "")
}

// BuildLongConnSendVoiceRequest 构造语音主动推送请求。
func BuildLongConnSendVoiceRequest(reqID, chatID, mediaID string) LongConnRequest {
	return BuildLongConnSendVoiceRequestWithChatType(reqID, chatID, LongConnChatTypeAuto, mediaID)
}

// BuildLongConnSendVoiceRequestWithChatType 构造指定会话类型的语音主动推送请求。
func BuildLongConnSendVoiceRequestWithChatType(reqID, chatID string, chatType LongConnChatType, mediaID string) LongConnRequest {
	return buildLongConnSendMediaRequest(reqID, chatID, chatType, "voice", mediaID, "", "")
}

// BuildLongConnSendVideoRequest 构造视频主动推送请求。
func BuildLongConnSendVideoRequest(reqID, chatID, mediaID, title, description string) LongConnRequest {
	return BuildLongConnSendVideoRequestWithChatType(
		reqID,
		chatID,
		LongConnChatTypeAuto,
		mediaID,
		title,
		description,
	)
}

// BuildLongConnSendVideoRequestWithChatType 构造指定会话类型的视频主动推送请求。
func BuildLongConnSendVideoRequestWithChatType(reqID, chatID string, chatType LongConnChatType, mediaID, title, description string) LongConnRequest {
	return buildLongConnSendMediaRequest(reqID, chatID, chatType, "video", mediaID, title, description)
}

// buildLongConnSendMediaRequest 构造媒体类型的主动推送请求。
func buildLongConnSendMediaRequest(reqID, chatID string, chatType LongConnChatType, msgType, mediaID, title, description string) LongConnRequest {
	body := LongConnPushMessage{
		ChatID:   chatID,
		ChatType: chatType,
		MsgType:  msgType,
	}
	media := &MediaMessagePayload{MediaID: mediaID}
	switch msgType {
	case "file":
		body.File = media
	case "image":
		body.Image = media
	case "voice":
		body.Voice = media
	case "video":
		body.Video = &VideoMessagePayload{
			MediaID:     mediaID,
			Title:       title,
			Description: description,
		}
	}
	return NewLongConnRequest(LongConnCmdSendMsg, reqID, body)
}

// BuildLongConnUploadMediaInitRequest 构造临时素材上传初始化请求。
func BuildLongConnUploadMediaInitRequest(reqID string, body LongConnUploadMediaInitBody) LongConnRequest {
	return NewLongConnRequest(LongConnCmdUploadMediaInit, reqID, body)
}

// BuildLongConnUploadMediaChunkRequest 构造临时素材分片上传请求。
func BuildLongConnUploadMediaChunkRequest(reqID, uploadID string, chunkIndex int, chunk []byte) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdUploadMediaChunk,
		reqID,
		LongConnUploadMediaChunkBody{
			UploadID:   uploadID,
			ChunkIndex: chunkIndex,
			Base64Data: base64.StdEncoding.EncodeToString(chunk),
		},
	)
}

// BuildLongConnUploadMediaFinishRequest 构造临时素材上传结束请求。
func BuildLongConnUploadMediaFinishRequest(reqID, uploadID string) LongConnRequest {
	return NewLongConnRequest(
		LongConnCmdUploadMediaFinish,
		reqID,
		LongConnUploadMediaFinishBody{UploadID: uploadID},
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
