# 附录：企业微信智能机器人官方资料

官网目录：<https://developer.work.weixin.qq.com/document/path/101039>
本地同步日期：2026-08-25

本目录保存企业微信“智能机器人”官方文档的本地 Markdown 快照，便于协议实现、评审与离线查阅。

## 维护策略

- 每篇快照在 frontmatter 中记录官网 `path_id`、内部 `doc_id`、官网更新时间和本地同步日期。
- 本地文件是同步时点的快照，不代表企业微信后续不会再次修改。
- 若本地快照与官网内容存在差异，**以官网当前文档为准**。

## 资料索引

### 实现原理（基于 bot-protocol-wecom SDK）

- `docs/streaming.md`：流式输出实现原理
- `docs/interaction.md`：交互开发指南（消息类型、回复模式、事件路由）

### 官方文档快照

| 序号 | 文档 | Path ID | doc_id | 官网更新时间 | 本地文件 |
|---:|---|---:|---:|---|---|
| 1 | 概述 | 101039 | 59174 | 2026-04-09 | [1_概述.md](wecom_ai_bot/1_概述.md) |
| 2 | 接收消息 | 100719 | 57141 | 2026-05-18 | [2_接收消息.md](wecom_ai_bot/2_接收消息.md) |
| 3 | 接收事件 | 101027 | 59058 | 2026-05-20 | [3_接收事件.md](wecom_ai_bot/3_接收事件.md) |
| 4 | 被动回复消息 | 101031 | 59068 | 2026-04-20 | [4_被动回复消息.md](wecom_ai_bot/4_被动回复消息.md) |
| 5 | 模板卡片类型 | 101032 | 59098 | 2026-02-06 | [5_模版卡片类型.md](wecom_ai_bot/5_模版卡片类型.md) |
| 6 | 回调和回复的加解密方案 | 101033 | 59137 | 2025-07-23 | [6_回调和回复的加解密方案.md](wecom_ai_bot/6_回调和回复的加解密方案.md) |
| 7 | 主动回复消息 | 101138 | 59947 | 2026-03-24 | [7_主动回复消息.md](wecom_ai_bot/7_主动回复消息.md) |
| 8 | 智能机器人长连接 | 101463 | 60904 | 2026-05-25 | [8_智能机器人长连接.md](wecom_ai_bot/8_智能机器人长连接.md) |
| 9 | API模式机器人文档使用说明 | 101468 | 60967 | 2026-03-13 | [9_API模式机器人文档使用说明.md](wecom_ai_bot/9_API模式机器人文档使用说明.md) |

## 与 bot-protocol-wecom SDK 的关系

SDK 核心代码位于：

- `pkg/wecom/bot.go` — Bot HTTP 处理与流式响应
- `pkg/wecom/stream.go` — StreamManager 会话管理
- `pkg/wecom/handler.go` — Handler 接口定义
- `pkg/wecom/message.go` — 消息类型定义
- `pkg/wecom/crypt.go` — 加解密实现
- `pkg/wecom/template_card.go` — 模板卡片类型
- `pkg/wecom/longconn_bot.go` — 长连接运行时、回调回复与主动推送
- `pkg/wecom/longconn_message.go` — 长连接命令、消息与上传协议结构
- `pkg/wecom/longconn_media.go` — 临时素材初始化、分片、完成与自动上传

第 9 篇“API 模式机器人文档使用说明”描述企业微信托管 MCP/API 能力，不属于本协议 SDK 的实现范围。

建议阅读路径：

1) 先理解 SDK 的 Handler 接口：`pkg/wecom/handler.go`
2) 再看 Bot 的回调处理逻辑：`pkg/wecom/bot.go`
3) 最后按需查阅本附录中的官方资料
