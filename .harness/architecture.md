# Architecture

## Scope

本文件描述 `pkg/wecom` 当前结构与对外契约。

## System Shape

Observed fact:

- SDK 提供两种接入模式：Webhook 回调模式与 WebSocket 长连接模式
- 统一数据模型位于 `message.go`、`template_card.go`、`handler.go`
- HTTP 回调入口位于 `bot.go`
- 长连接入口位于 `longconn_bot.go`

## Major Modules

- `bot.go`: `Bot`、`StartOptions`、HTTP GET/POST 回调处理
- `crypt.go`: `Crypt`、签名计算、消息加解密
- `message.go`: 事件/消息结构、被动流式回复结构
- `handler.go`: `Handler`、`Context`、`Chunk`
- `stream.go`: 流式会话生命周期管理
- `longconn_bot.go`: 长连接连接管理与主动推送
- `longconn_message.go`: 长连接协议帧与请求构造
- `template_card.go`: 模板卡片结构体

## Dependency Directions

- `bot.go` 依赖 `Crypt`、`StreamManager`、`Handler`
- `IMBotCore/pkg/platform/wecom` 作为下游包装层依赖本仓库

## Key Flows

### Webhook

```text
GET/POST callback
  -> verify signature / decrypt
  -> normalize message
  -> call Handler
  -> emit passive stream reply or active response_url request
```

### Long Connection

```text
connect websocket
  -> subscribe
  -> read push frame
  -> call Handler
  -> send markdown/template card request
```

## High-Risk Areas

- `crypt.go`: 协议兼容性与安全性风险最高
- `message.go`: 公开结构体字段变更会影响下游
- `longconn_bot.go`: 重连、请求超时、挂起请求清理

## Evidence

- `pkg/wecom/bot.go`
- `pkg/wecom/crypt.go`
- `pkg/wecom/longconn_bot.go`
- `pkg/wecom/longconn_message.go`
- `pkg/wecom/*_test.go`

## Open Questions

- 未观察到仓库级 CI 配置
