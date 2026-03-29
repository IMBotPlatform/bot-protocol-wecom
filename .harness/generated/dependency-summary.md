# Generated Dependency Summary

## Direct Module Dependencies

- `github.com/gorilla/websocket`

## Internal Runtime Dependencies

- `bot.go` 依赖 `stream.go`、`crypt.go`、`handler.go`、`config.go`
- `stream.go` 依赖 `config.go` 解析流式 TTL 与等待超时默认值
- `longconn_bot.go` 依赖 `longconn_message.go`、`handler.go`、`template_card.go`、`config.go`
- `example/echo/main.go` 消费 `Bot.DecryptDownloadedFile` 与 `BuildStreamImageItemFromBytes` 演示图片回放

## Downstream Consumers Observed In Workspace

- `IMBotCore/pkg/platform/wecom`
- 通过 `IMBotCore` 间接被 `wechataibot` 使用
