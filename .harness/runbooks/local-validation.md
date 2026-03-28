# Runbook: Local Validation

## Default Check

```bash
go test ./...
```

## When To Re-run Full Suite

- 改动 `pkg/wecom/crypt.go`
- 改动 `pkg/wecom/message.go`
- 改动 `pkg/wecom/longconn_bot.go`
- 改动 `pkg/wecom/longconn_message.go`

## Example Smoke Path

- 读 `example/echo/main.go` 验证最小接入方式是否仍然成立

## Downstream Reminder

- 若改动公共结构、构造函数或模式切换语义，同步检查 `IMBotCore/pkg/platform/wecom`
