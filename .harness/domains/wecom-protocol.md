# Domain: WeCom Protocol

## Responsibility

把企业微信协议细节封装为可复用的 Go SDK，并暴露回调加解密、长连接收发与下载资源解密 helper。

## Non-responsibility

- 不解析业务命令语义
- 不管理用户会话或产品级上下文

## Key Concepts

- `Bot`: HTTP 回调模式运行时
- `LongConnBot`: WebSocket 长连接运行时
- `Crypt`: 签名校验与加解密器
- `Handler`: 业务处理器抽象
- `Chunk`: 业务层输出片段
- `MixedItem`: 流式结束包里可附带的图文混排子消息
- `NoResponse`: 显式放弃回复的哨兵语义

## Main Flows

- GET 验证：`VerifyURL` 解密 `echostr`
- POST 消息：解密请求 -> 自动解密 `image.url` 图片数据 -> 交给 `Handler` -> 构造被动回复或空包
- POST / LongConn 最终回复：可在 `IsFinal=true` 的最后一个 `Chunk` 中携带 `msg_item`
- LongConn：订阅 -> 接收 `aibot_msg_callback` / `aibot_event_callback` -> 推断 `respond_msg` / `respond_welcome_msg` / `respond_update_msg` -> 转给 `Handler`

## Important Constraints

- 协议字段名与消息结构应保持向后兼容
- 长连接请求 `req_id` 与 pending map 生命周期需要对应
- `msg_item` 仅应出现在最终结束包
- `feedback_event` 在 Webhook 模式下只允许空包快速返回，不进入普通业务回复流
- 图片资源会在协议层自动下载并解密；文件与视频资源仍由上层按需处理

## Edge Cases / Common Misunderstandings

- `LongConnBot` 的错误既有可恢复错误，也有不可恢复错误
- `BOT_*` 环境变量只是在未显式传参时提供默认值
- `NoResponse` 与 `ErrNoResponse` 都表示“不要回复”，但触发路径不同

## Evidence

- `pkg/wecom/bot.go`
- `pkg/wecom/longconn_bot.go`
- `pkg/wecom/config.go`
- `pkg/wecom/longconn_message.go`
- `example/echo/main.go`
