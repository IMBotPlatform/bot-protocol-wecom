# AGENTS.md

## Mission / Scope

这个仓库负责企业微信协议 SDK。

- owns: 回调验签、消息加解密、流式回复、长连接连接管理、模板卡片结构
- not owns: 命令系统、产品侧 skills、部署编排、Claude CLI 适配

## Start Here

1. `.harness/README.md`
2. `.harness/generated/module-map.md`
3. `.harness/generated/repo-manifest.yaml`
4. `.harness/generated/contract-index.yaml`
5. `.harness/generated/validation-index.yaml`
6. `.harness/evolution-policy.yaml`
7. `.harness/generated/api-index.md`
8. `README.md`

## Source of Truth

- 代码：`pkg/wecom/*.go`
- 测试：`pkg/wecom/*_test.go`
- 使用示例：`example/echo/main.go`

## Important Directories

- `pkg/wecom/`: SDK 主体
- `example/echo/`: 最小回显示例
- `docs/wecom_ai_bot/`: 协议资料

## Hard Constraints

- 保持企业微信回调协议兼容性
- 不把上层业务概念引入 SDK 包
- 变更公开消息结构、`Bot` 构造函数、长连接请求结构时，需提示下游 `IMBotCore`

## Validation Expectations

- 首选：`go test ./...`
- 改动 `pkg/wecom/crypt.go`、`pkg/wecom/message.go`、`pkg/wecom/longconn_*` 后必须跑全量测试

## High-Risk Areas

- `pkg/wecom/crypt.go`
- `pkg/wecom/bot.go`
- `pkg/wecom/longconn_bot.go`
