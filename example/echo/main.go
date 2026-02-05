package main

import (
	"fmt"
	"log"
	"os"

	"github.com/IMBotPlatform/bot-protocol-wecom/pkg/wecom"
)

func main() {
	// 从环境变量读取配置
	token := os.Getenv("WECOM_TOKEN")
	encodingAESKey := os.Getenv("WECOM_ENCODING_AES_KEY")
	corpID := os.Getenv("WECOM_CORP_ID")
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	if token == "" || encodingAESKey == "" || corpID == "" {
		log.Fatal("请设置环境变量: WECOM_TOKEN, WECOM_ENCODING_AES_KEY, WECOM_CORP_ID")
	}

	// 创建业务处理器
	handler := wecom.HandlerFunc(func(ctx wecom.Context) <-chan wecom.Chunk {
		ch := make(chan wecom.Chunk)
		go func() {
			defer close(ch)

			// 获取用户输入
			text := ""
			if ctx.Message != nil && ctx.Message.Text != nil {
				text = ctx.Message.Text.Content
			}

			// 简单的回显示例
			reply := fmt.Sprintf("收到消息: %s", text)
			ch <- wecom.Chunk{Content: reply, IsFinal: true}
		}()
		return ch
	})

	// 创建 Bot 实例
	bot, err := wecom.NewBot(token, encodingAESKey, corpID, handler)
	if err != nil {
		log.Fatalf("创建 Bot 失败: %v", err)
	}

	// 启动 HTTP 服务
	log.Printf("WeCom Bot 启动中，监听地址: %s", listenAddr)
	if err := bot.Start(wecom.StartOptions{ListenAddr: listenAddr}); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
