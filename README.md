# bot-protocol-wecom

[![Go Reference](https://pkg.go.dev/badge/github.com/IMBotPlatform/bot-protocol-wecom.svg)](https://pkg.go.dev/github.com/IMBotPlatform/bot-protocol-wecom)
[![Go Report Card](https://goreportcard.com/badge/github.com/IMBotPlatform/bot-protocol-wecom)](https://goreportcard.com/report/github.com/IMBotPlatform/bot-protocol-wecom)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> 🤖 企业微信(WeCom) AI Bot SDK - 完整的 Go 实现

## ✨ 功能特性

| 功能 | 描述 |
|------|------|
| 🌐 **双接入模式** | 支持 Webhook 回调模式与 WebSocket 长连接模式 |
| 🤖 **完整 Bot 能力** | 流式响应、主动回复、模板卡片、事件处理 |
| 🔐 **消息加解密** | AES-CBC 加解密，签名校验 |
| 💬 **消息类型** | 文本、图片、语音、文件、图文混排、流式消息、事件 |
| 🎴 **模板卡片** | 完整的企业微信模板卡片类型支持 |

## 📦 安装

```bash
go get github.com/IMBotPlatform/bot-protocol-wecom
```

## 🧭 模式选择

| 模式 | 入口 | 适用场景 | 关键凭证 |
|------|------|------|------|
| Webhook 回调 | `wecom.NewBot(...)` | 已有公网回调地址、沿用现有企业微信回调模式 | `TOKEN` / `ENCODING_AES_KEY` / `CORP_ID` |
| WebSocket 长连接 | `wecom.NewLongConnBot(...)` | 无公网 IP、高实时性、希望由服务端主动保持连接 | `BOT_ID` / `SECRET` |

## 🚀 快速开始

### 1. Webhook 回调模式

```go
package main

import (
    "log"
    "github.com/IMBotPlatform/bot-protocol-wecom/pkg/wecom"
)

func main() {
    // 定义业务处理器
    handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
        ch := make(chan wecom.Chunk)
        go func() {
            defer close(ch)
            ch <- wecom.Chunk{Content: "Hello, ", IsFinal: false}
            ch <- wecom.Chunk{Content: "World!", IsFinal: true}
        }()
        return ch
    })

    // 创建并启动 Bot
    bot, err := wecom.NewBot("TOKEN", "ENCODING_AES_KEY", "CORP_ID", handler)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Bot starting on :8080...")
    if err := bot.Start(wecom.StartOptions{ListenAddr: ":8080"}); err != nil {
        log.Fatal(err)
    }
}
```

### 2. WebSocket 长连接模式

```go
package main

import (
    "context"
    "log"

    "github.com/IMBotPlatform/bot-protocol-wecom/pkg/wecom"
)

func main() {
    handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
        ch := make(chan wecom.Chunk, 1)
        ch <- wecom.Chunk{Content: "收到长连接消息", IsFinal: true}
        close(ch)
        return ch
    })

    bot, err := wecom.NewLongConnBot("BOT_ID", "SECRET", handler)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    log.Println("LongConn Bot starting...")
    if err := bot.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## 📁 目录结构

```
bot-protocol-wecom/
├── pkg/wecom/           # SDK 核心代码
│   ├── bot.go           # Bot 结构体和 HTTP 处理
│   ├── crypt.go         # 加解密实现
│   ├── longconn_bot.go  # 长连接连接管理与回调分发
│   ├── longconn_message.go # 长连接协议结构
│   ├── message.go       # 消息类型定义
│   ├── handler.go       # Handler 接口
│   ├── stream.go        # StreamManager
│   └── template_card.go # 模板卡片类型
├── example/             # 示例代码
│   └── echo/            # 简单回显示例
└── docs/                # 官方协议文档
```

## 📤 主动回复

```go
handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
    // 使用 ctx.Bot 发送主动回复
    _ = ctx.Bot.ResponseMarkdown(ctx.ResponseURL, "**Hello!**")
    
    ch := make(chan wecom.Chunk)
    close(ch)
    return ch
})
```

## 📡 长连接主动推送

```go
package main

import (
    "context"
    "log"

    "github.com/IMBotPlatform/bot-protocol-wecom/pkg/wecom"
)

func main() {
    bot, err := wecom.NewLongConnBot("BOT_ID", "SECRET", nil)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        if err := bot.Start(ctx); err != nil {
            log.Printf("longconn stopped: %v", err)
        }
    }()

    // 主动推送消息到单聊 userid 或群聊 chatid。
    _ = bot.SendMarkdown("CHAT_ID", "**hello from longconn**")
}
```

## ⚙️ 常用环境变量

| 环境变量 | 说明 |
|------|------|
| `BOT_HTTP_TIMEOUT` | Webhook 模式主动回复 HTTP 超时 |
| `BOT_STREAM_TTL` | Webhook 模式流式会话保留时间 |
| `BOT_STREAM_WAIT_TIMEOUT` | Webhook 模式流式刷新等待时间 |
| `BOT_LONG_CONN_WS_URL` | 长连接 WebSocket 地址，默认 `wss://openws.work.weixin.qq.com` |
| `BOT_LONG_CONN_PING_INTERVAL` | 长连接心跳间隔，单位秒 |
| `BOT_LONG_CONN_RECONNECT_INTERVAL` | 长连接断线重连间隔，单位秒 |
| `BOT_LONG_CONN_REQUEST_TIMEOUT` | 长连接单次命令超时，单位秒 |
| `BOT_LONG_CONN_WRITE_TIMEOUT` | 长连接单次写入超时，单位秒 |

## 📝 长连接说明

- 长连接模式会自动完成订阅、心跳和断线重连。
- 普通消息回调会自动映射到 `aibot_respond_msg`。
- `enter_chat` 事件会自动映射到 `aibot_respond_welcome_msg`。
- `template_card_event` 会自动映射到 `aibot_respond_update_msg`。
- 长连接资源消息目前会透出 `aeskey` 字段，便于调用方自行做图片/文件解密下载。

## 📖 文档

官方协议文档位于 [`docs/wecom_ai_bot/`](./docs/wecom_ai_bot/) 目录。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

[MIT](LICENSE) © IMBotPlatform
