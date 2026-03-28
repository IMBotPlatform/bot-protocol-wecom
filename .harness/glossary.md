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
