# Conventions

## API Boundary

- 对外契约主要由导出类型与构造函数组成
- 新增公共字段前，优先考虑是否会被 `IMBotCore` 或外部调用方序列化/反序列化使用

## Runtime Configuration

- 默认值优先由显式参数决定，再回退环境变量，再回退内置默认值
- 相关环境变量集中在 `pkg/wecom/config.go`

## Design Constraints

- SDK 不应引入产品侧命令或部署语义
- 长连接和 Webhook 模式共享 `Handler` 抽象，避免双份业务逻辑

## Testing Bias

- 任何加解密、签名、请求帧构造变更都要优先靠单元测试锁定行为
