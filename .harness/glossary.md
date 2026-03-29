# Glossary

## Webhook 模式

- Definition: 企业微信通过 HTTP 回调推送消息，SDK 负责验签、解密和回包
- Evidence: `pkg/wecom/bot.go`, `README.md`

## Long Connection 模式

- Definition: 机器人主动建立 WebSocket 长连接接收消息与发送响应
- Evidence: `pkg/wecom/longconn_bot.go`, `README.md`

## response_url

- Definition: 企业微信用于主动回复的 URL，由 SDK 客户端发起 HTTP 请求
- Evidence: `pkg/wecom/bot.go`, `pkg/wecom/handler.go`

## StreamManager

- Definition: 管理流式被动回复会话生命周期的内部组件
- Evidence: `pkg/wecom/stream.go`

## NoResponse

- Definition: 业务层显式放弃回复的哨兵语义；Webhook 首包可通过 `ErrNoResponse` 返回空包，流式与长连接可通过 `Chunk.Payload == NoResponse` 终止回复
- Evidence: `pkg/wecom/bot.go`, `pkg/wecom/handler.go`, `pkg/wecom/longconn_bot.go`

## msg_item

- Definition: 企业微信流式结束包里附带的 mixed 子消息列表，当前常见用法是把图片字节编码为 `image.base64 + image.md5`
- Evidence: `pkg/wecom/message.go`, `example/echo/main.go`
