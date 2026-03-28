# Domain: WeCom Protocol

## Responsibility

把企业微信协议细节封装为可复用的 Go SDK。

## Non-responsibility

- 不解析业务命令语义
- 不管理用户会话或产品级上下文

## Key Concepts

- `Bot`: HTTP 回调模式运行时
- `LongConnBot`: WebSocket 长连接运行时
- `Crypt`: 签名校验与加解密器
- `Handler`: 业务处理器抽象
- `Chunk`: 业务层输出片段

## Main Flows

- GET 验证：`VerifyURL` 解密 `echostr`
- POST 消息：解密请求 -> 交给 `Handler` -> 构造被动回复
- LongConn：订阅 -> 接收推送 -> 转给 `Handler` -> 发送命令帧

## Important Constraints

- 协议字段名与消息结构应保持向后兼容
- 长连接请求 `req_id` 与 pending map 生命周期需要对应

## Edge Cases / Common Misunderstandings

- `LongConnBot` 的错误既有可恢复错误，也有不可恢复错误
- `BOT_*` 环境变量只是在未显式传参时提供默认值

## Evidence

- `pkg/wecom/bot.go`
- `pkg/wecom/longconn_bot.go`
- `pkg/wecom/config.go`
- `pkg/wecom/longconn_message.go`
