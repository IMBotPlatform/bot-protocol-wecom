# Runbook: Local Validation

## Default Check

```bash
go test ./...
```

## When To Re-run Full Suite

- 改动 `pkg/wecom/crypt.go`
- 改动 `pkg/wecom/config.go`
- 改动 `pkg/wecom/handler.go`
- 改动 `pkg/wecom/message.go`
- 改动 `pkg/wecom/longconn_bot.go`
- 改动 `pkg/wecom/longconn_message.go`

## Example Smoke Path

- 读 `example/echo/main.go` 验证最小接入方式是否仍然成立
- 若改动图片资源解密或 `msg_item` 组装逻辑，额外检查 `example/echo/main.go` 的“下载密文 -> 解密 -> BuildStreamImageItemFromBytes”路径

## Targeted Test Clues

- 长连接回调命令映射：`pkg/wecom/longconn_bot_test.go`
- 下载资源解密 helper：`pkg/wecom/crypt_test.go`
- 流式图片 `msg_item` 组装：`pkg/wecom/message_test.go`

## Downstream Reminder

- 若改动公共结构、构造函数或模式切换语义，同步检查 `IMBotCore/pkg/platform/wecom`
