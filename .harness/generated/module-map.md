# Generated Module Map

| Path | Kind | Responsibility |
| --- | --- | --- |
| `pkg/wecom/bot.go` | runtime entry | Webhook Bot 构造与 HTTP 回调 |
| `pkg/wecom/crypt.go` | security | 签名、加解密、下载文件解密 |
| `pkg/wecom/message.go` | contract | 消息结构、回复结构、流式回复构造 |
| `pkg/wecom/handler.go` | contract | `Handler` / `Context` / `Chunk` |
| `pkg/wecom/stream.go` | internal runtime | 流式会话管理 |
| `pkg/wecom/longconn_bot.go` | runtime entry | 长连接模式运行时 |
| `pkg/wecom/longconn_message.go` | contract | 长连接请求帧与响应帧 |
| `pkg/wecom/template_card.go` | contract | 模板卡片结构 |
| `example/echo/main.go` | example | 最小接入示例 |
