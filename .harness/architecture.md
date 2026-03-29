# Architecture

## Scope

本文件描述 `pkg/wecom` 当前结构、配置层与对外契约。

## System Shape

Observed fact:

- SDK 提供两种接入模式：Webhook 回调模式与 WebSocket 长连接模式
- 回调加解密与下载资源解密能力位于 `crypt.go` 与 `bot.go`
- 统一数据模型位于 `message.go`、`template_card.go`、`handler.go`
- 运行参数解析与环境变量回退集中在 `config.go`
- HTTP 回调入口位于 `bot.go`
- 长连接入口位于 `longconn_bot.go`

## Major Modules

- `bot.go`: `Bot`、`StartOptions`、HTTP GET/POST 回调处理
- `crypt.go`: `Crypt`、签名计算、消息加解密
- `config.go`: `BOT_*` 运行参数解析与默认值回退
- `message.go`: 事件/消息结构、被动流式回复结构
- `handler.go`: `Handler`、`Context`、`Chunk`、`NoResponse`
- `stream.go`: 流式会话生命周期管理
- `longconn_bot.go`: 长连接连接管理、回调命令到响应命令映射、主动推送
- `longconn_message.go`: 长连接命令常量、协议帧与请求构造
- `template_card.go`: 模板卡片结构体

## Dependency Directions

- `bot.go` 依赖 `Crypt`、`StreamManager`、`Handler`、`config.go`
- `stream.go` 与 `longconn_bot.go` 依赖 `config.go` 解析运行参数默认值
- `example/echo/main.go` 演示图片下载密文的解密与 `msg_item` 构造
- `IMBotCore/pkg/platform/wecom` 作为下游包装层依赖本仓库

## Key Flows

### Webhook

```text
GET/POST callback
  -> verify signature / decrypt
  -> normalize message
  -> auto-decrypt image.url into ImagePayload.Data when present
  -> call Handler
  -> emit passive stream reply or active response_url request
```

### Long Connection

```text
connect websocket
  -> subscribe / ping / reconnect
  -> read callback frame
  -> map callback cmd to respond_msg / respond_welcome_msg / respond_update_msg
  -> call Handler
  -> send stream / markdown / template-card request
```

## High-Risk Areas

- `crypt.go`: 协议兼容性与安全性风险最高
- `message.go`: 公开结构体字段变更会影响下游
- `longconn_bot.go`: 重连、请求超时、挂起请求清理

## Evidence

- `pkg/wecom/bot.go`
- `pkg/wecom/crypt.go`
- `pkg/wecom/config.go`
- `pkg/wecom/longconn_bot.go`
- `example/echo/main.go`
- `pkg/wecom/longconn_message.go`
- `pkg/wecom/*_test.go`

## Open Questions

- 未观察到仓库级 CI 配置
