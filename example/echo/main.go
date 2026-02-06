package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IMBotPlatform/bot-protocol-wecom/pkg/wecom"
)

const (
	// imageMaxBytes 为企业微信流式回复中图片（base64 前）的最大允许大小。
	// 文档要求不超过 10MB，示例侧提前做限制，避免下载超大文件导致内存占用。
	imageMaxBytes = 10 * 1024 * 1024

	// imageEncryptedMaxBytes 为下载企业微信回调里 image.url 的最大允许大小。
	// 注意：回调里的图片文件是加密后的字节流（AES-256-CBC + PKCS#7），密文长度会向上补齐到 32 字节倍数，
	// 因此这里在 10MB 基础上额外放宽一个 padding 块，避免“刚好 10MB”的图片被误判为超限。
	imageEncryptedMaxBytes = imageMaxBytes + 32

	// imageDownloadTimeout 控制图片下载的超时时间，避免卡死示例服务。
	imageDownloadTimeout = 10 * time.Second
)

// downloadURLBytes 下载指定 URL 的内容，并限制最大字节数。
//
// Parameters:
//   - ctx: 用于控制取消/超时的上下文
//   - urlStr: 要下载的资源 URL
//   - maxBytes: 最大允许下载字节数（超过则返回错误）
//
// Returns:
//   - []byte: 下载到的内容字节
//   - error: 下载失败、HTTP 非 200 或内容超限时返回
func downloadURLBytes(ctx context.Context, urlStr string, maxBytes int64) ([]byte, error) {
	if urlStr == "" {
		return nil, fmt.Errorf("url is empty")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("maxBytes must be > 0")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 使用带超时的 client，配合 ctx 避免长时间阻塞。
	client := &http.Client{Timeout: imageDownloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// 关键步骤：限制读取大小，避免下载超大文件导致内存占用。
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("content too large: %d bytes > %d bytes", len(data), maxBytes)
	}
	return data, nil
}

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

			// 兜底：理论上不会为空，但示例中仍做保护。
			if ctx.Message == nil {
				ch <- wecom.Chunk{Content: "未收到有效消息", IsFinal: true}
				return
			}

			// 图片消息：下载图片字节并回传为 stream.msg_item（仅在最后一次回复支持）。
			if ctx.Message.MsgType == "image" {
				imgURL := ""
				if ctx.Message.Image != nil {
					imgURL = ctx.Message.Image.URL
				}
				if imgURL == "" {
					ch <- wecom.Chunk{Content: "图片消息缺少 url，无法下载", IsFinal: true}
					return
				}
				if ctx.Bot == nil {
					// 兜底：Bot 为空时无法进行“下载文件解密”，直接返回文本提示避免 refresh 轮询无响应。
					ch <- wecom.Chunk{Content: "Bot 未初始化，无法解密下载图片", IsFinal: true}
					return
				}

				// 关键步骤：下载图片密文字节（企业微信回调的 image.url 返回的是已加密文件，不能直接作为图片使用）。
				// 为避免下载超大文件导致内存占用，这里限制最大下载大小（10MB + padding 余量）。
				dlCtx, cancel := context.WithTimeout(context.Background(), imageDownloadTimeout)
				defer cancel()

				encryptedBytes, err := downloadURLBytes(dlCtx, imgURL, imageEncryptedMaxBytes)
				if err != nil {
					// 错误兜底：下载失败时给出文本提示，避免企业微信轮询无响应。
					ch <- wecom.Chunk{Content: fmt.Sprintf("下载图片失败: %v", err), IsFinal: true}
					return
				}

				// 第二步：解密下载到的密文字节，得到“原始图片字节”（base64 前）。
				plainBytes, err := ctx.Bot.DecryptDownloadedFile(encryptedBytes)
				if err != nil {
					// 错误兜底：解密失败时给出文本提示。
					ch <- wecom.Chunk{Content: fmt.Sprintf("解密图片失败: %v", err), IsFinal: true}
					return
				}
				if len(plainBytes) > imageMaxBytes {
					// 兜底：解密后再校验一次明文大小，防止意外超限影响流式回复。
					ch <- wecom.Chunk{Content: fmt.Sprintf("图片过大: %d bytes > %d bytes", len(plainBytes), imageMaxBytes), IsFinal: true}
					return
				}

				// 第三步：将原始图片字节转换为企业微信 stream.msg_item 所需结构（image.base64 + image.md5）。
				// 注意：仅返回原始图片，不强制附带文本 content。
				item, err := wecom.BuildStreamImageItemFromBytes(plainBytes)
				if err != nil {
					// 错误兜底：构造 item 失败时给出文本提示。
					ch <- wecom.Chunk{Content: fmt.Sprintf("构造图片 item 失败: %v", err), IsFinal: true}
					return
				}

				// content 为空即可；IsFinal=true 触发最后一次回复携带 msg_item。
				ch <- wecom.Chunk{MsgItems: []wecom.MixedItem{item}, IsFinal: true}
				return
			}

			// 图文混排消息：同时包含文本和图片（图片 url 下载到的是加密文件，需要先解密）。
			if ctx.Message.MsgType == "mixed" {
				// mixed 中的文本和图片都在 mixed.msg_item[] 内，顶层不会有 ctx.Message.Text/Image。
				textParts := make([]string, 0, 4)
				msgItems := make([]wecom.MixedItem, 0, 2)

				if ctx.Message.Mixed != nil {
					for _, it := range ctx.Message.Mixed.Items {
						switch strings.TrimSpace(it.MsgType) {
						case "text":
							if it.Text != nil && strings.TrimSpace(it.Text.Content) != "" {
								textParts = append(textParts, it.Text.Content)
							}
						case "image":
							// mixed.image.url 同样是“下载密文”，需下载并解密后才能回传为图片。
							imgURL := ""
							if it.Image != nil {
								imgURL = strings.TrimSpace(it.Image.URL)
							}
							if imgURL == "" {
								continue
							}
							if ctx.Bot == nil {
								// 无 Bot 无法解密，直接跳过图片并兜底回复文本。
								continue
							}

							dlCtx, cancel := context.WithTimeout(context.Background(), imageDownloadTimeout)
							encryptedBytes, err := downloadURLBytes(dlCtx, imgURL, imageEncryptedMaxBytes)
							cancel()
							if err != nil {
								// 图片失败不阻断整体回复，尽量先把文本回出去。
								continue
							}

							plainBytes, err := ctx.Bot.DecryptDownloadedFile(encryptedBytes)
							if err != nil || len(plainBytes) > imageMaxBytes {
								continue
							}

							item, err := wecom.BuildStreamImageItemFromBytes(plainBytes)
							if err != nil {
								continue
							}
							msgItems = append(msgItems, item)
						}
					}
				}

				// content 仍沿用“回显”行为；图片通过 msg_item 回传（finish=true 时才会生效）。
				content := ""
				if len(textParts) > 0 {
					content = fmt.Sprintf("收到消息: %s", strings.Join(textParts, "\n"))
				}
				ch <- wecom.Chunk{Content: content, MsgItems: msgItems, IsFinal: true}
				return
			}

			// 获取用户输入（仅处理文本消息，其它类型将回显空字符串）
			text := ""
			if ctx.Message.Text != nil {
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
