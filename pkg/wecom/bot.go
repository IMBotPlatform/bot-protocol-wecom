package wecom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrNoResponse 表示业务层请求不进行任何被动回复（HTTP 200 OK 空包）。
	ErrNoResponse = errors.New("no response")
)

// Bot 集成企业微信回调处理与流式响应逻辑。
// Fields:
//   - streamMgr: 管理流式会话生命周期的 StreamManager
//   - crypto: 负责签名校验与加解密的 Crypt
//   - client: 主动回复客户端，负责向 response_url 发送消息
//   - handler: 业务处理器
type Bot struct {
	streamMgr *StreamManager
	crypto    *Crypt
	client    *http.Client
	handler   Handler
}

// StartOptions 控制 Bot 启动 HTTP 服务的参数。
// Fields:
//   - ListenAddr: HTTP 监听地址（当 Server 未提供 Addr 时使用）
//   - CallbackPath: 回调路径（为空则默认 /callback/command）
//   - Mux: 可选路由复用器（为空则内部创建新的 *http.ServeMux）
//   - Server: 可选 HTTP Server（为空则内部创建并使用 ListenAddr）
type StartOptions struct {
	ListenAddr   string
	CallbackPath string
	Mux          *http.ServeMux
	Server       *http.Server
}

// NewBot 根据给定参数创建 Bot。
// Parameters:
//   - token: 企业微信配置的消息校验 Token
//   - encodingAESKey: 企业微信后台生成的 43 字节 Base64 编码字符串
//   - corpID: 企业 ID，用于校验消息归属
//   - handler: 业务处理器（实现 Handler 接口）
//
// Returns:
//   - *Bot: 成功初始化的 Bot 实例
//   - error: 当加解密上下文初始化失败时返回错误
func NewBot(token, encodingAESKey, corpID string, handler Handler) (*Bot, error) {
	// 关键步骤：在构造 Bot 之前先初始化加解密上下文。
	crypto, err := NewCrypt(token, encodingAESKey, corpID)
	if err != nil {
		return nil, err
	}

	return &Bot{
		streamMgr: newStreamManager(0, 0),
		crypto:    crypto,
		client: &http.Client{
			Timeout: resolveDuration(0, envBotHTTPTimeout, 10*time.Second),
		},
		handler: handler,
	}, nil
}

// NewBotWithOptions 创建带自定义配置的 Bot。
// Parameters:
//   - token: 企业微信配置的消息校验 Token
//   - encodingAESKey: 企业微信后台生成的 43 字节 Base64 编码字符串
//   - corpID: 企业 ID，用于校验消息归属
//   - streamTTL: 流式会话最大存活时间（<=0 时使用默认值）
//   - streamWaitTimeout: 刷新请求等待时长（<=0 时使用默认值）
//   - handler: 业务处理器
//
// Returns:
//   - *Bot: 成功初始化的 Bot 实例
//   - error: 当加解密上下文初始化失败时返回错误
func NewBotWithOptions(token, encodingAESKey, corpID string, streamTTL, streamWaitTimeout time.Duration, handler Handler) (*Bot, error) {
	crypto, err := NewCrypt(token, encodingAESKey, corpID)
	if err != nil {
		return nil, err
	}

	return &Bot{
		streamMgr: newStreamManager(streamTTL, streamWaitTimeout),
		crypto:    crypto,
		client: &http.Client{
			Timeout: resolveDuration(0, envBotHTTPTimeout, 10*time.Second),
		},
		handler: handler,
	}, nil
}

// Start 启动 HTTP 服务并挂载 Bot 回调路由。
// Parameters:
//   - opts: 启动参数（包含监听地址、回调路径、可选的 Mux/Server）
//
// Returns:
//   - error: 启动失败或配置缺失时返回错误
func (b *Bot) Start(opts StartOptions) error {
	if b == nil {
		return errors.New("bot is nil")
	}

	// 关键步骤：确定回调路径（为空则使用默认值）。
	callbackPath := strings.TrimSpace(opts.CallbackPath)
	if callbackPath == "" {
		callbackPath = "/callback/command"
	}

	mux := opts.Mux
	if mux == nil {
		mux = http.NewServeMux()
	}
	// 关键步骤：注册回调路由到复用器。
	mux.Handle(callbackPath, b)

	srv := opts.Server
	if srv == nil {
		listenAddr := strings.TrimSpace(opts.ListenAddr)
		if listenAddr == "" {
			return errors.New("listen addr is required")
		}
		srv = &http.Server{
			Addr:    listenAddr,
			Handler: mux,
		}
	} else {
		if srv.Handler == nil {
			srv.Handler = mux
		}
		if strings.TrimSpace(srv.Addr) == "" {
			srv.Addr = strings.TrimSpace(opts.ListenAddr)
		}
		if strings.TrimSpace(srv.Addr) == "" {
			return errors.New("server addr is required")
		}
	}

	return srv.ListenAndServe()
}

// ServeHTTP 实现 http.Handler 接口，根据请求方法转发至不同处理逻辑。
// Parameters:
//   - w: http.ResponseWriter，用于写回响应
//   - r: *http.Request，请求上下文
func (b *Bot) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 根据请求方法执行对应的回调处理逻辑。
	switch r.Method {
	case http.MethodGet:
		// GET 请求用于企业微信 URL 验证。
		b.handleGet(w, r)
	case http.MethodPost:
		// POST 请求承载业务事件推送。
		b.handlePost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet 处理企业微信服务器验证 URL 的场景，需要校验签名并返回解密后的 echostr。
func (b *Bot) handleGet(w http.ResponseWriter, r *http.Request) {
	if b == nil || b.crypto == nil {
		http.Error(w, "server misconfigured", http.StatusInternalServerError)
		return
	}

	// 第一步：解析企业微信回调所需的查询参数。
	query := r.URL.Query()
	sig := query.Get("msg_signature")
	ts := query.Get("timestamp")
	nonce := query.Get("nonce")
	echostr := query.Get("echostr")
	if sig == "" || ts == "" || nonce == "" || echostr == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	// 第二步：调用加解密模块完成签名验证与明文解密。
	plain, err := b.crypto.VerifyURL(sig, ts, nonce, echostr)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// 第三步：以纯文本形式响应企业微信平台。
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(plain))
}

// handlePost 处理企业微信推送的业务回调，完成解密、业务响应构造与加密返回。
func (b *Bot) handlePost(w http.ResponseWriter, r *http.Request) {
	if b == nil || b.crypto == nil || b.streamMgr == nil {
		http.Error(w, "server misconfigured", http.StatusInternalServerError)
		return
	}

	// 第一步：请求开始前清理过期会话，避免资源堆积。
	b.cleanup()
	query := r.URL.Query()
	sig := query.Get("msg_signature")
	ts := query.Get("timestamp")
	nonce := query.Get("nonce")
	if sig == "" || ts == "" || nonce == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	// 第二步：解析请求体中的加密 JSON 数据。
	defer r.Body.Close()
	var req EncryptedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Encrypt == "" {
		http.Error(w, "missing encrypt", http.StatusBadRequest)
		return
	}

	// 第三步：解密业务消息，进入业务处理阶段。
	msg, err := b.crypto.DecryptMessage(sig, ts, nonce, req)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// 第四步：自动下载并解密消息中的图片数据。
	// 企业微信的 image.url 返回的是 AES-CBC 加密的密文，需要解密后才能使用。
	b.decryptMessageImages(msg)

	// 关键步骤：反馈事件仅支持空包回复，直接返回 200，避免进入 handler。
	if msg.MsgType == "event" && msg.Event != nil && (msg.Event.FeedbackEvent != nil || msg.Event.EventType == "feedback_event") {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 第四步：区分首次或刷新场景，由 Bot 内部流式逻辑产出响应体。
	var resp EncryptedResponse
	if msg.MsgType == "stream" {
		resp, err = b.refresh(msg, ts, nonce) // 流式刷新请求
	} else {
		resp, err = b.initial(msg, ts, nonce) // 首包或非流式请求
	}

	// 特殊处理：业务层明确要求不回复
	if errors.Is(err, ErrNoResponse) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 第五步：序列化加密响应并回写给企业微信平台。
	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

// initial 处理首次回调，创建流式会话并触发业务处理器。
func (b *Bot) initial(msg *Message, timestamp, nonce string) (EncryptedResponse, error) {
	// 第一步：创建或复用流式会话。
	stream, isNew := b.streamMgr.createOrGet(msg)

	// 关键步骤：首包只负责触发处理器并返回空 ACK，内容由 refresh 拉取。
	if isNew && b.handler != nil {
		ctx := Context{
			Message:     msg,
			StreamID:    stream.StreamID,
			ResponseURL: msg.ResponseURL,
			Bot:         b,
		}
		outCh := b.handler.Handle(ctx)
		if outCh != nil {
			// 后台消费处理器输出，由 refresh 统一返回内容。
			go b.doHandler(outCh, stream.StreamID)
		}
	}

	// 第二步：构造首包 ACK（始终为空内容）
	reply := BuildStreamReply(stream.StreamID, "", false)
	return b.crypto.EncryptResponse(reply, timestamp, nonce)
}

// refresh 处理企业微信的流式刷新请求。
func (b *Bot) refresh(msg *Message, timestamp, nonce string) (EncryptedResponse, error) {
	// 第一步：提取 streamID，判定是否为有效流式刷新请求。
	streamID := ""
	if msg.Stream != nil {
		streamID = msg.Stream.ID
	}
	if streamID == "" {
		// 无效 streamID 直接返回终止包，告知客户端结束。
		reply := BuildStreamReply("", "", true)
		return b.crypto.EncryptResponse(reply, timestamp, nonce)
	}

	// 第二步：从会话中获取最新累计片段。
	chunk := b.streamMgr.getLatestChunk(streamID)
	if chunk == nil {
		// 无片段可用，返回保持连接的空包。
		reply := BuildStreamReply(streamID, "", false)
		return b.crypto.EncryptResponse(reply, timestamp, nonce)
	}
	if chunk.IsFinal {
		// 最终片段需标记会话完成，避免资源泄露。
		b.streamMgr.markFinished(streamID)
	}

	// 第三步：将片段封装为流式响应并加密返回。
	// 优先处理携带的非流式 Payload
	if chunk.Payload != nil {
		return b.crypto.EncryptResponse(chunk.Payload, timestamp, nonce)
	}

	// 仅在流式结束（finish=true）时允许携带 msg_item（如图片）。
	if chunk.IsFinal && len(chunk.MsgItems) > 0 {
		reply := BuildStreamReplyWithMsgItems(streamID, chunk.Content, true, chunk.MsgItems)
		return b.crypto.EncryptResponse(reply, timestamp, nonce)
	}

	reply := BuildStreamReply(streamID, chunk.Content, chunk.IsFinal)
	return b.crypto.EncryptResponse(reply, timestamp, nonce)
}

// setFinalMessage 在非流式场景下尝试投递最终结果（仅在会话存在时生效）。
func (b *Bot) setFinalMessage(msgID, content string) {
	if msgID == "" {
		return
	}
	streamID, ok := b.streamMgr.getStreamIDByMsg(msgID)
	if !ok || streamID == "" {
		return
	}
	chunk := Chunk{Content: content, IsFinal: true}
	b.streamMgr.publish(streamID, chunk)
}

// cleanup 清理过期会话，防止 Session 过度累积。
func (b *Bot) cleanup() {
	if b == nil || b.streamMgr == nil {
		return
	}

	// 委托 StreamManager 移除超时会话。
	b.streamMgr.cleanup()
}

// doHandler 消费处理器输出并发布到流式会话。
func (b *Bot) doHandler(outCh <-chan Chunk, streamID string) {
	if outCh == nil {
		return
	}

	published := false
	for chunk := range outCh {
		// 关键步骤：NoResponse 在流式场景中等价于"立即结束"。
		if chunk.Payload == NoResponse {
			if b.streamMgr.publish(streamID, Chunk{Content: "", IsFinal: true}) {
				published = true
			}
			return
		}
		// 空 chunk 过滤：不能丢弃仅携带 MsgItems 的片段。
		if chunk.Content == "" && chunk.Payload == nil && len(chunk.MsgItems) == 0 && !chunk.IsFinal {
			continue
		}

		if b.streamMgr.publish(streamID, chunk) {
			published = true
		}
	}
	if !published {
		// 无输出也要结束会话，避免 refresh 无限轮询。
		b.streamMgr.publish(streamID, Chunk{Content: "", IsFinal: true})
	}
}

// Response 向指定的 response_url 发送主动回复消息。
// 对应文档：7_加解密说明.md - 如何主动回复消息
// 注意：response_url 有效期为 1 小时，且每个 url 仅可调用一次。
func (b *Bot) Response(responseURL string, msg any) error {
	if responseURL == "" {
		return fmt.Errorf("response_url is empty")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, responseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wecom api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ResponseMarkdown 发送 Markdown 消息。
func (b *Bot) ResponseMarkdown(responseURL, content string) error {
	msg := MarkdownMessage{
		MsgType: "markdown",
		Markdown: MarkdownPayload{
			Content: content,
		},
	}
	return b.Response(responseURL, msg)
}

// ResponseTemplateCard 发送模板卡片消息。
func (b *Bot) ResponseTemplateCard(responseURL string, card *TemplateCard) error {
	msg := TemplateCardMessage{
		MsgType:      "template_card",
		TemplateCard: card,
	}
	return b.Response(responseURL, msg)
}

// DecryptDownloadedFile 解密企业微信“下载文件”接口返回的二进制密文数据。
// Parameters:
//   - cipherData: 下载接口返回的密文字节（非 Base64 字符串）
//
// Returns:
//   - []byte: 解密后的明文字节
//   - error: 当 Bot/crypto 未初始化或解密失败时返回错误
func (b *Bot) DecryptDownloadedFile(cipherData []byte) ([]byte, error) {
	if b == nil {
		return nil, errors.New("bot is nil")
	}
	if b.crypto == nil {
		return nil, errors.New("bot crypto is nil")
	}
	return b.crypto.DecryptDownloadedFile(cipherData)
}

// decryptMessageImages 自动下载并解密消息中所有图片 URL，将解密后的数据填入 ImagePayload.Data。
// 企业微信回调的 image.url 返回的是 AES-CBC 加密密文，需要下载并解密后才能作为图片使用。
// 此方法会处理：
//   - MsgType=image 的顶层图片
//   - MsgType=mixed 中的 image 子消息
//
// 解密失败时会静默忽略（不中断流程）。
func (b *Bot) decryptMessageImages(msg *Message) {
	if msg == nil || b == nil {
		return
	}

	// 处理顶层 image 消息
	if msg.MsgType == "image" && msg.Image != nil {
		b.decryptImagePayload(msg.Image)
	}

	// 处理 mixed 消息中的 image 子消息
	if msg.MsgType == "mixed" && msg.Mixed != nil {
		for i := range msg.Mixed.Items {
			if msg.Mixed.Items[i].MsgType == "image" && msg.Mixed.Items[i].Image != nil {
				b.decryptImagePayload(msg.Mixed.Items[i].Image)
			}
		}
	}
}

// decryptImagePayload 下载并解密单个 ImagePayload。
func (b *Bot) decryptImagePayload(img *ImagePayload) {
	if img == nil || img.URL == "" {
		return
	}

	// 下载密文
	cipherData, err := b.downloadURL(img.URL)
	if err != nil {
		return
	}

	// 解密
	plainData, err := b.DecryptDownloadedFile(cipherData)
	if err != nil {
		return
	}

	// 填充解密后的数据
	img.Data = plainData
}

// downloadURL 下载指定 URL 的内容。
func (b *Bot) downloadURL(url string) ([]byte, error) {
	if b.client == nil {
		return nil, errors.New("http client is nil")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: status=%d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
