# Project Harness

## Purpose

为 agent 提供 SDK 级导航层，帮助其快速判断：

- 入口 API 在哪里
- 哪些类型属于公开契约
- 哪些改动会影响下游 `IMBotCore`

## Reading Order

1. `generated/repo-manifest.yaml`
2. `generated/module-map.md`
3. `generated/api-index.md`
4. `architecture.md`
5. `domains/wecom-protocol.md`
6. `runbooks/local-validation.md`

## Evidence Priority

1. `pkg/wecom/*.go`
2. `pkg/wecom/*_test.go`
3. `README.md`
4. `example/echo/main.go`
