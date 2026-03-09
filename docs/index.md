# 附录：企业微信官方资料（仅引用）

更新时间：2026-03-09

本目录用于收纳与企业微信机器人接入相关的 **官方说明/存量资料**，便于查阅。

## 维护策略

- 仅做引用与归档：**不承诺持续同步更新**
- 若与官方最新文档有差异：**以官方为准**

## 资料索引

### 实现原理（基于 bot-protocol-wecom SDK）

- `docs/streaming.md`：流式输出实现原理
- `docs/interaction.md`：交互开发指南（消息类型、回复模式、事件路由）

### 官方文档（存量资料）

- `docs/wecom_ai_bot/1_概述.md`
- `docs/wecom_ai_bot/2_接收消息.md`
- `docs/wecom_ai_bot/3_接收事件.md`
- `docs/wecom_ai_bot/4_被动回复消息.md`
- `docs/wecom_ai_bot/5_模版卡片类型.md`
- `docs/wecom_ai_bot/6_回调和回复的加解密方案.md`
- `docs/wecom_ai_bot/7_主动回复消息.md`
- `docs/wecom_ai_bot/8_智能机器人长连接.md`

## 与 bot-protocol-wecom SDK 的关系

SDK 核心代码位于：

- `pkg/wecom/bot.go` — Bot HTTP 处理与流式响应
- `pkg/wecom/stream.go` — StreamManager 会话管理
- `pkg/wecom/handler.go` — Handler 接口定义
- `pkg/wecom/message.go` — 消息类型定义
- `pkg/wecom/crypt.go` — 加解密实现
- `pkg/wecom/template_card.go` — 模板卡片类型

建议阅读路径：

1) 先理解 SDK 的 Handler 接口：`pkg/wecom/handler.go`
2) 再看 Bot 的回调处理逻辑：`pkg/wecom/bot.go`
3) 最后按需查阅本附录中的官方资料
