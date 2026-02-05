# bot-protocol-wecom

企业微信(WeCom) AI Bot 通信协议 Go 实现。

## 功能

- **消息加解密**: AES-CBC 加解密，签名校验
- **消息类型**: 文本、图片、语音、文件、图文混排、流式消息、事件
- **模板卡片**: 完整的企业微信模板卡片类型支持
- **流式响应**: 流式消息构造与解析

## 安装

```bash
go get github.com/IMBotPlatform/bot-protocol-wecom
```

## 快速开始

```go
package main

import (
    "log"
    wecom "github.com/IMBotPlatform/bot-protocol-wecom"
)

func main() {
    // 创建加解密器
    crypt, err := wecom.NewCrypt("TOKEN", "ENCODING_AES_KEY", "CORP_ID")
    if err != nil {
        log.Fatal(err)
    }

    // 解密消息
    msg, err := crypt.DecryptMessage(msgSignature, timestamp, nonce, req)
    if err != nil {
        log.Fatal(err)
    }

    // 构造流式回复
    reply := wecom.BuildStreamReply("stream-id", "Hello", false)

    // 加密响应
    resp, err := crypt.EncryptResponse(reply, timestamp, nonce)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 文档

官方协议文档位于 `docs/wecom-official/` 目录。

## License

MIT
