# bot-protocol-wecom

企业微信(WeCom) AI Bot SDK - 完整的 Go 实现。

## 功能

- **完整 Bot 能力**: HTTP 服务器、流式响应、主动回复
- **消息加解密**: AES-CBC 加解密，签名校验
- **消息类型**: 文本、图片、语音、文件、图文混排、流式消息、事件
- **模板卡片**: 完整的企业微信模板卡片类型支持

## 安装

```bash
go get github.com/IMBotPlatform/bot-protocol-wecom
```

## 快速开始

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

## 目录结构

```
bot-protocol-wecom/
├── pkg/wecom/           # SDK 核心代码
│   ├── bot.go           # Bot 结构体和 HTTP 处理
│   ├── crypt.go         # 加解密实现
│   ├── message.go       # 消息类型定义
│   ├── handler.go       # Handler 接口
│   ├── stream.go        # StreamManager
│   └── template_card.go # 模板卡片类型
├── example/             # 示例代码
│   └── echo/            # 简单回显示例
└── docs/                # 官方协议文档
```

## 主动回复

```go
handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
    // 使用 ctx.Bot 发送主动回复
    _ = ctx.Bot.ResponseMarkdown(ctx.ResponseURL, "**Hello!**")
    
    ch := make(chan wecom.Chunk)
    close(ch)
    return ch
})
```

## 文档

官方协议文档位于 `docs/wecom_ai_bot/` 目录。

## License

MIT
