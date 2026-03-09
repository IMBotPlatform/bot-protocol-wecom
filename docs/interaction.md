# 企业微信交互开发指南

本文档说明企业微信的多种交互模式，包括被动回复（流式/非流式）、主动消息推送以及各类事件处理。

> 本文档为存量资料，仅供参考。若与官方最新文档有差异，请以官方为准。

## 1. 支持的消息类型矩阵

### 1.1 接收消息 (Bot 接收)
| 类型 | 字段 (`MsgType`) | 对应结构体 | 说明 |
| :--- | :--- | :--- | :--- |
| **文本** | `text` | `TextPayload` | 用户发送的普通文本 |
| **图片** | `image` | `ImagePayload` | 用户发送的图片 (含 URL，需解密) |
| **语音** | `voice` | `VoicePayload` | 用户发送的语音 (含转录文本) |
| **文件** | `file` | `FilePayload` | 用户发送的文件 (含下载 URL，需解密) |
| **图文混排** | `mixed` | `MixedPayload` | 包含文本和图片的组合消息 |
| **引用消息** | (无独立类型) | `QuotePayload` | 嵌套在上述消息中的 `quote` 字段 |

### 1.2 接收事件 (Bot 接收)
| 事件场景 | `EventType` | 对应结构体 | 说明 |
| :--- | :--- | :--- | :--- |
| **进入会话** | `enter_chat` | `EnterChat` | 用户当天首次进入单聊会话 |
| **模板卡片交互** | `template_card_event` | `TemplateCardEvent` | 用户点击卡片按钮 |
| **用户反馈** | `feedback_event` | `FeedbackEvent` | 用户对回复进行准确/不准确反馈 |

### 1.3 被动回复 (同步 HTTP 响应)
| 类型 | 字段 (`MsgType`) | 说明 |
| :--- | :--- | :--- |
| **流式文本** | `stream` | **核心功能**，AI 对话逐字输出 |
| **流式+图文** | `stream` | 流式结束 (`finish=true`) 时附带图片 |
| **普通文本** | `text` | 简单的非流式回复 |
| **模板卡片** | `template_card` | 发送欢迎语、功能菜单 |
| **流式+卡片** | `stream_with_template_card` | AI 回复的同时下发一张卡片 |
| **更新卡片** | `update_template_card` | 用户点击按钮后，更新原卡片状态 |

### 1.4 主动回复 (异步 HTTP 推送)
利用 `response_url` 在 1 小时内主动下发。

| 类型 | 字段 (`MsgType`) | 调用方式 |
| :--- | :--- | :--- |
| **Markdown** | `markdown` | `bot.ResponseMarkdown(responseURL, content)` |
| **模板卡片** | `template_card` | `bot.ResponseTemplateCard(responseURL, card)` |

---

## 2. 三种回复模式 (开发核心)

在 `Handler` 的实现中，你可以根据需求选择三种不同的回复模式。

### 2.1 模式一：流式文本回复 (默认)
最常用的模式。通过 `chan Chunk` 输出内容，SDK 会自动建立流式会话，将内容逐字推送到企业微信气泡中。

**代码示例:** 
```go
handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
    ch := make(chan wecom.Chunk)
    go func() {
        defer close(ch)
        ch <- wecom.Chunk{Content: "正在查询数据...", IsFinal: false}  // 中间帧
        // ... 耗时操作
        ch <- wecom.Chunk{Content: "查询完成: 100条", IsFinal: true}  // 最终帧
    }()
    return ch
})
```

### 2.2 模式二：静默执行 (No Response)
当你不希望 Bot 在对话框产生流式气泡，而是想完全依赖**主动推送**（如发一张独立的卡片）时，使用此模式。

**代码示例:** 
```go
handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
    // 主动推送 Markdown 消息
    _ = ctx.Bot.ResponseMarkdown(ctx.ResponseURL, "# 异步通知\n任务已后台开始")
    
    // 返回空 channel，不产生流式回复
    ch := make(chan wecom.Chunk)
    close(ch)
    return ch
})
```

### 2.3 模式三：非流式对象回复 (模板卡片)
当你需要同步回复模板卡片时（如 `enter_chat` 欢迎语或按钮交互回调），通过主动回复接口发送。

**代码示例:** 
```go
handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
    // 构造模板卡片
    card := &wecom.TemplateCard{
        TaskID:    "task_001",
        MainTitle: &wecom.MainTitle{Title: "欢迎使用"},
    }
    
    // 通过主动回复发送卡片
    _ = ctx.Bot.ResponseTemplateCard(ctx.ResponseURL, card)
    
    ch := make(chan wecom.Chunk)
    close(ch)
    return ch
})
```

---

## 3. 事件与路由机制

Bot 收到的报文通过 `Message` 结构体解析。开发者可根据 `MsgType` 和 `EventType` 字段进行分流处理。

### 3.1 事件映射表

| 原始事件类型 (`event_type`) | Bot 推荐处理方式 |
| :--- | :--- |
| `enter_chat` (进入会话) | 根据 `Message.EventType` 判断，回复欢迎卡片 |
| `template_card_event` (卡片交互) | 取 `EventKey` 执行对应业务逻辑 |
| `feedback_event` (用户反馈) | 记录反馈数据，返回空包即可 |

## 4. 开发场景示例

### 场景 A: 实现欢迎语
1.  在 Handler 中判断 `ctx.Message.MsgType == "event"` 且 `ctx.Message.EventType == "enter_chat"`。
2.  使用 `ctx.Bot.ResponseTemplateCard(ctx.ResponseURL, card)` 回复欢迎卡片。
3.  返回空 channel（不产生流式回复）。

### 场景 B: 卡片按钮交互
1.  发送一张包含按钮的卡片，按钮 Key 设为 `approve_order_123`。
2.  用户点击按钮，企业微信推送 `template_card_event`。
3.  Handler 根据 `EventKey` 执行业务逻辑。
4.  通过主动回复更新原卡片状态。

> 更多信息请参考 `docs/wecom_ai_bot/` 中的官方资料。
