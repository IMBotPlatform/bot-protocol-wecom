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
- `LongConnChatType`: 主动推送时显式选择自动、单聊或群聊解析
- `LongConnMediaType`: 长连接临时素材的文件、图片、语音或视频类型
- `NoResponse`: 显式放弃回复的哨兵语义

## Main Flows

- GET 验证：`VerifyURL` 解密 `echostr`
- POST 消息：解密请求 -> 自动解密 `image.url` 图片数据 -> 交给 `Handler` -> 构造被动回复或空包
- Webhook 最终回复：可在 `IsFinal=true` 的最后一个 `Chunk` 中携带 `msg_item`
- LongConn：订阅 -> 接收 `aibot_msg_callback` / `aibot_event_callback` -> 推断 `respond_msg` / `respond_welcome_msg` / `respond_update_msg` -> 转给 `Handler`
- LongConn 普通回复：支持流式、模板卡片、Markdown、文件、图片、语音和视频；拒绝组合流式卡片，最终图片自动转换为素材上传和独立图片回复
- LongConn 主动推送：按 `chat_type` 向单聊/群聊发送 Markdown、模板卡片或媒体消息
- LongConn 素材上传：init -> 512KB 分片 chunk -> finish -> 返回 3 天有效的 `media_id`

## Important Constraints

- 协议字段名与消息结构应保持向后兼容
- `Crypt` 加解密只返回结果或错误，不得记录解密后的请求、加密前的回复、`userid` 或其他业务明文
- 长连接请求 `req_id` 与 pending map 生命周期需要对应
- `msg_item` 只出现在 Webhook 线上帧；长连接 Handler 最终图片由 `longconn_stream.go` 转换
- 长连接不支持“流式消息 + 模板卡片”的组合回复
- 临时素材最多 100 片，上传会话 30 分钟有效，客户端执行每机器人 30 次/分钟和 1000 次/小时的滚动限流
- `feedback_event` 在 Webhook 模式下只允许空包快速返回，不进入普通业务回复流
- 图片资源会在协议层自动下载并解密；文件与视频资源仍由上层按需处理
- 官方 API 模式机器人文档描述的企业微信托管 MCP/API 工具不属于本协议 SDK

## Edge Cases / Common Misunderstandings

- `LongConnBot` 的错误既有可恢复错误，也有不可恢复错误
- `BOT_*` 环境变量只是在未显式传参时提供默认值
- `NoResponse` 与 `ErrNoResponse` 都表示“不要回复”，但触发路径不同

## Text snapshots and delivery acknowledgements

`Chunk.Replace=true` replaces the accumulated text, including an empty snapshot; the default remains append. Replacement snapshots supersede queued text so a producer can finish even after the client stops polling. HTTP streaming, long-connection streaming and one-shot replies share this meaning. `Replace` must not be combined with `Payload`. See `pkg/wecom/handler.go`, `stream.go`, `longconn_bot.go` and `snapshot_test.go`; downstream adapters must preserve the flag.

`Bot.Response` requires both HTTP 200 and a JSON acknowledgement containing `errcode=0`. Invalid/missing acknowledgements and transport failure do not prove delivery. Errors expose status/code only, without echoing response URLs or upstream bodies; callers must not blindly retry an uncertain delivery.

## Evidence

- `pkg/wecom/bot.go`
- `pkg/wecom/longconn_bot.go`
- `pkg/wecom/config.go`
- `pkg/wecom/longconn_message.go`
- `pkg/wecom/longconn_media.go`
- `docs/wecom_ai_bot/8_智能机器人长连接.md`
- `example/echo/main.go`

## 长连接装配合同

`Ready` 表示成功订阅；`ErrLongConnReplaced` 阻止多个实例相互抢连接。回调提供稳定 StreamID，异步业务不占用 ACK 读取循环；文本 4 秒合并、9 分钟结束当前流，发送采用共享会话预算。异步发送错误通过 `LongConnOptions.OnError` 上报，未知投递不重试。实现见 `pkg/wecom/longconn_state.go` / `longconn_stream.go`；验证见 `longconn_integration_test.go`。产品启用模式和凭据管理归下游。
