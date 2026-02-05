package wecom

// Chunk 描述流式输出片段。
// Fields:
//   - Content: 文本内容（企业微信要求为累积完整内容）
//   - Payload: 扩展负载（模板卡片等非流式回复）
//   - IsFinal: 是否为最终片段
type Chunk struct {
	Content string
	Payload any
	IsFinal bool
}

// NoResponse 是一个哨兵值，用于标记不需要被动回复。
// 当 Chunk.Payload == NoResponse 时，Bot 层直接返回 HTTP 200 OK 空包。
var NoResponse = struct{}{}

// Context 承载单次请求的上下文信息。
// Fields:
//   - Message: 解密后的企业微信消息
//   - StreamID: 流式会话 ID
//   - ResponseURL: 主动回复 URL（有效期 1 小时）
//   - Bot: Bot 实例，用于主动回复
type Context struct {
	Message     *Message
	StreamID    string
	ResponseURL string
	Bot         *Bot
}

// Handler 定义业务处理器接口，由用户实现。
// 返回的 channel 用于流式输出响应内容。
type Handler interface {
	Handle(ctx Context) <-chan Chunk
}

// HandlerFunc 便于将函数直接作为 Handler 使用。
type HandlerFunc func(ctx Context) <-chan Chunk

// Handle 实现 Handler 接口。
func (f HandlerFunc) Handle(ctx Context) <-chan Chunk {
	if f == nil {
		return nil
	}
	return f(ctx)
}
