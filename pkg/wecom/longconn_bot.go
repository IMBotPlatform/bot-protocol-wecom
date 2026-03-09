package wecom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const defaultLongConnWSURL = "wss://openws.work.weixin.qq.com"

// LongConnOptions 控制 LongConnBot 的连接行为。
// Fields:
//   - WebSocketURL: 企业微信长连接地址
//   - PingInterval: 心跳间隔
//   - ReconnectInterval: 断线重连间隔
//   - RequestTimeout: 单次命令等待响应的超时时间
//   - WriteTimeout: 单次写入超时时间
//   - Dialer: 可选自定义 WebSocket Dialer
type LongConnOptions struct {
	WebSocketURL      string
	PingInterval      time.Duration
	ReconnectInterval time.Duration
	RequestTimeout    time.Duration
	WriteTimeout      time.Duration
	Dialer            *websocket.Dialer
}

// LongConnBot 负责企业微信长连接模式的连接管理与消息收发。
// Fields:
//   - botID/secret: 长连接订阅所需的机器人凭证
//   - handler: 业务处理器，复用现有 Handler 抽象
//   - wsURL/pingInterval/reconnectInterval/requestTimeout/writeTimeout: 长连接运行参数
//   - dialer: WebSocket 建连器
//   - conn: 当前活跃连接
//   - pending: req_id -> 等待响应的调用方
//   - closedCh: 用于停止读写循环与重连循环
type LongConnBot struct {
	botID   string
	secret  string
	handler Handler

	wsURL             string
	pingInterval      time.Duration
	reconnectInterval time.Duration
	requestTimeout    time.Duration
	writeTimeout      time.Duration
	dialer            *websocket.Dialer

	connMu  sync.RWMutex
	conn    *websocket.Conn
	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan longConnAckResult

	closeOnce sync.Once
	closedCh  chan struct{}
}

// longConnAckResult 表示一次长连接命令的异步响应结果。
type longConnAckResult struct {
	response LongConnResponse
	err      error
}

// longConnAPIError 表示企业微信长连接服务端返回的业务错误。
type longConnAPIError struct {
	cmd       string
	requestID string
	errCode   int
	errMsg    string
}

// Error 返回企业微信长连接业务错误的可读描述。
func (e *longConnAPIError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf(
		"longconn api error: cmd=%s req_id=%s errcode=%d errmsg=%s",
		e.cmd,
		e.requestID,
		e.errCode,
		e.errMsg,
	)
}

// longConnPermanentError 表示不应重试的长连接错误。
// 例如：订阅鉴权失败，这类错误继续重连也无法恢复。
type longConnPermanentError struct {
	err error
}

// Error 返回不可恢复错误的描述。
func (e *longConnPermanentError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap 暴露底层错误，便于 errors.As / errors.Is 判断。
func (e *longConnPermanentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// NewLongConnBot 使用默认配置创建长连接机器人。
func NewLongConnBot(botID, secret string, handler Handler) (*LongConnBot, error) {
	return NewLongConnBotWithOptions(botID, secret, handler, LongConnOptions{})
}

// NewLongConnBotWithOptions 使用自定义配置创建长连接机器人。
func NewLongConnBotWithOptions(botID, secret string, handler Handler, opts LongConnOptions) (*LongConnBot, error) {
	if botID == "" {
		return nil, errors.New("bot id is required")
	}
	if secret == "" {
		return nil, errors.New("secret is required")
	}

	dialer := opts.Dialer
	if dialer == nil {
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: resolveDuration(opts.RequestTimeout, envLongConnRequestTimeout, 10*time.Second),
		}
	}

	return &LongConnBot{
		botID:             botID,
		secret:            secret,
		handler:           handler,
		wsURL:             resolveString(opts.WebSocketURL, envLongConnWSURL, defaultLongConnWSURL),
		pingInterval:      resolveDuration(opts.PingInterval, envLongConnPingInterval, 30*time.Second),
		reconnectInterval: resolveDuration(opts.ReconnectInterval, envLongConnReconnectInterval, 3*time.Second),
		requestTimeout:    resolveDuration(opts.RequestTimeout, envLongConnRequestTimeout, 5*time.Second),
		writeTimeout:      resolveDuration(opts.WriteTimeout, envLongConnWriteTimeout, 5*time.Second),
		dialer:            dialer,
		pending:           make(map[string]chan longConnAckResult),
		closedCh:          make(chan struct{}),
	}, nil
}

// Start 建立并维护企业微信长连接。
// 该方法会阻塞直到 ctx 结束、显式调用 Close，或遇到不可恢复错误。
func (b *LongConnBot) Start(ctx context.Context) error {
	if b == nil {
		return errors.New("longconn bot is nil")
	}

	for {
		// 关键步骤：每轮循环代表一次完整会话，断线后按重连间隔重新建立。
		select {
		case <-ctx.Done():
			return nil
		case <-b.closedCh:
			return nil
		default:
		}

		err := b.runSession(ctx)
		if err == nil {
			continue
		}

		var permanent *longConnPermanentError
		if errors.As(err, &permanent) {
			return permanent.err
		}

		// 仅对可恢复错误执行退避重连。
		select {
		case <-ctx.Done():
			return nil
		case <-b.closedCh:
			return nil
		case <-time.After(b.reconnectInterval):
		}
	}
}

// Close 关闭当前连接并停止后续重连。
func (b *LongConnBot) Close() error {
	if b == nil {
		return nil
	}
	b.closeOnce.Do(func() {
		close(b.closedCh)
	})

	conn := b.currentConn()
	if conn != nil {
		_ = conn.Close()
	}
	b.failAllPending(errors.New("longconn bot closed"))
	return nil
}

// SendMarkdown 主动推送 Markdown 消息到指定会话。
func (b *LongConnBot) SendMarkdown(chatID, content string) error {
	if chatID == "" {
		return errors.New("chat id is required")
	}

	req := BuildLongConnSendMarkdownRequest(b.nextRequestID(), chatID, content)
	ctx, cancel := b.newRequestContext(context.Background())
	defer cancel()

	return b.sendRequestAndWait(ctx, req.Cmd, req.Headers.RequestID, req.Body)
}

// SendTemplateCard 主动推送模板卡片消息到指定会话。
func (b *LongConnBot) SendTemplateCard(chatID string, card *TemplateCard) error {
	if chatID == "" {
		return errors.New("chat id is required")
	}
	if card == nil {
		return errors.New("template card is nil")
	}

	req := BuildLongConnSendTemplateCardRequest(b.nextRequestID(), chatID, card)
	ctx, cancel := b.newRequestContext(context.Background())
	defer cancel()

	return b.sendRequestAndWait(ctx, req.Cmd, req.Headers.RequestID, req.Body)
}

// runSession 完成一次完整的长连接会话生命周期。
// 步骤包括：建立 WebSocket、发送订阅、启动读循环、启动心跳循环，并等待任一环节退出。
func (b *LongConnBot) runSession(ctx context.Context) error {
	conn, _, err := b.dialer.DialContext(ctx, b.wsURL, nil)
	if err != nil {
		return err
	}

	// 关键步骤：连接建好后立即注册为当前活跃连接，便于主动发送复用。
	b.setConn(conn)
	defer b.releaseConn(conn, errors.New("longconn session closed"))

	readErrCh := make(chan error, 1)
	go b.readLoop(conn, readErrCh)

	// 关键步骤：订阅成功后企业微信才会开始推送回调。
	subscribeCtx, cancel := b.newRequestContext(ctx)
	err = b.sendRequestAndWait(
		subscribeCtx,
		LongConnCmdSubscribe,
		b.nextRequestID(),
		LongConnSubscribeBody{
			BotID:  b.botID,
			Secret: b.secret,
		},
	)
	cancel()
	if err != nil {
		var apiErr *longConnAPIError
		if errors.As(err, &apiErr) {
			return &longConnPermanentError{err: err}
		}
		return err
	}

	pingErrCh := make(chan error, 1)
	go b.pingLoop(ctx, pingErrCh)

	// 会话以“读循环退出 / 心跳退出 / 上下文结束 / 主动关闭”之一为结束条件。
	select {
	case <-ctx.Done():
		return nil
	case <-b.closedCh:
		return nil
	case err := <-readErrCh:
		return err
	case err := <-pingErrCh:
		return err
	}
}

// readLoop 持续读取长连接帧，并按“响应帧 / 回调帧”两类分发处理。
func (b *LongConnBot) readLoop(conn *websocket.Conn, errCh chan<- error) {
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}

		// 忽略无法解析的帧，避免单个异常包中断整条连接。
		var frame LongConnRawFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			continue
		}

		// 先消费命令响应，唤醒等待该 req_id 的发送方。
		if frame.HasAckResult() {
			response := LongConnResponse{
				Headers: frame.Headers,
				ErrMsg:  frame.ErrMsg,
			}
			if frame.ErrCode != nil {
				response.ErrCode = *frame.ErrCode
			}
			if b.completePending(frame.Headers.RequestID, longConnAckResult{response: response}) {
				continue
			}
		}

		if frame.IsCallback() {
			b.handleCallback(frame)
		}
	}
}

// pingLoop 定时发送心跳，确保长连接保持活跃。
// 任一心跳失败都视为当前会话失效，交由外层重连。
func (b *LongConnBot) pingLoop(ctx context.Context, errCh chan<- error) {
	ticker := time.NewTicker(b.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.closedCh:
			return
		case <-ticker.C:
			pingCtx, cancel := b.newRequestContext(ctx)
			err := b.sendRequestAndWait(pingCtx, LongConnCmdPing, b.nextRequestID(), nil)
			cancel()
			if err != nil {
				errCh <- err
				return
			}
		}
	}
}

// handleCallback 将企业微信回调转换为 Context，并把结果路由到对应回复命令。
func (b *LongConnBot) handleCallback(frame LongConnRawFrame) {
	if b.handler == nil {
		return
	}

	// 关键步骤：复用现有 Message 模型，避免长连接与 Webhook 两套业务输入结构。
	var msg Message
	if err := frame.UnmarshalBody(&msg); err != nil {
		return
	}

	ctx := Context{
		Message:   &msg,
		RequestID: frame.Headers.RequestID,
		LongConn:  b,
	}
	outCh := b.handler.Handle(ctx)
	if outCh == nil {
		return
	}

	// 根据回调类型推断后续要发送的长连接回复命令。
	switch b.resolveReplyCommand(frame.Cmd, &msg) {
	case LongConnCmdRespondMsg:
		go b.consumeMessageChunks(frame.Headers.RequestID, outCh)
	case LongConnCmdRespondWelcomeMsg, LongConnCmdRespondUpdateMsg:
		go b.consumeOneShotChunks(
			b.resolveReplyCommand(frame.Cmd, &msg),
			frame.Headers.RequestID,
			outCh,
		)
	default:
		return
	}
}

// resolveReplyCommand 根据回调类型与事件类型，推断应使用的回复命令。
func (b *LongConnBot) resolveReplyCommand(callbackCmd string, msg *Message) string {
	switch callbackCmd {
	case LongConnCmdMsgCallback:
		return LongConnCmdRespondMsg
	case LongConnCmdEventCallback:
		if msg == nil || msg.Event == nil {
			return ""
		}
		switch msg.Event.EventType {
		case "enter_chat":
			return LongConnCmdRespondWelcomeMsg
		case "template_card_event":
			return LongConnCmdRespondUpdateMsg
		default:
			return ""
		}
	default:
		return ""
	}
}

// consumeMessageChunks 消费业务层输出，并转为 `aibot_respond_msg` 命令。
// 流式文本会自动累积为企业微信要求的“完整内容”形式。
func (b *LongConnBot) consumeMessageChunks(requestID string, outCh <-chan Chunk) {
	streamID := generateStreamID()
	accumulated := ""
	sentAny := false
	finished := false

	for chunk := range outCh {
		// NoResponse 在长连接消息回复场景中表示业务层显式放弃回复。
		if chunk.Payload == NoResponse {
			return
		}

		// 若业务层直接返回了完整协议负载，则透传发送。
		if chunk.Payload != nil {
			body, err := normalizeLongConnMessageBody(chunk.Payload)
			if err != nil {
				return
			}
			if err := b.sendCallbackCommand(LongConnCmdRespondMsg, requestID, body); err != nil {
				return
			}
			sentAny = true
			if chunk.IsFinal {
				finished = true
				return
			}
			continue
		}

		if chunk.Content == "" && !chunk.IsFinal {
			continue
		}

		// 长连接模式要求 stream.content 始终传当前累积全文。
		accumulated += chunk.Content
		reply := BuildStreamReply(streamID, accumulated, chunk.IsFinal)
		if err := b.sendCallbackCommand(LongConnCmdRespondMsg, requestID, reply); err != nil {
			return
		}
		sentAny = true
		if chunk.IsFinal {
			finished = true
			return
		}
	}

	// 若业务提前关闭 channel 且已经发过流式片段，则补一个 finish=true 结束包。
	if sentAny && !finished {
		_ = b.sendCallbackCommand(
			LongConnCmdRespondMsg,
			requestID,
			BuildStreamReply(streamID, accumulated, true),
		)
	}
}

// consumeOneShotChunks 消费单次回复类输出，并转为欢迎语或卡片更新命令。
// 该路径用于 enter_chat / template_card_event 等非流式回调。
func (b *LongConnBot) consumeOneShotChunks(command string, requestID string, outCh <-chan Chunk) {
	var (
		lastPayload any
		accumulated string
	)

	for chunk := range outCh {
		if chunk.Payload == NoResponse {
			return
		}
		if chunk.Payload != nil {
			lastPayload = chunk.Payload
		}
		if chunk.Content != "" {
			accumulated += chunk.Content
		}
		if chunk.IsFinal {
			break
		}
	}

	body, err := normalizeLongConnOneShotBody(command, lastPayload, accumulated)
	if err != nil || body == nil {
		return
	}
	_ = b.sendCallbackCommand(command, requestID, body)
}

// normalizeLongConnMessageBody 将业务层 Payload 归一化为长连接普通消息回复体。
// 当前支持文本、模板卡片、流式消息，以及少量便捷输入（string / *TemplateCard）。
func normalizeLongConnMessageBody(payload any) (any, error) {
	switch body := payload.(type) {
	case TextMessage, *TextMessage,
		TemplateCardMessage, *TemplateCardMessage,
		StreamReply, *StreamReply,
		StreamWithTemplateCardMessage, *StreamWithTemplateCardMessage:
		return body, nil
	case *TemplateCard:
		return TemplateCardMessage{
			MsgType:      "template_card",
			TemplateCard: body,
		}, nil
	case string:
		return TextMessage{
			MsgType: "text",
			Text: &TextPayload{
				Content: body,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported longconn message payload: %T", payload)
	}
}

// normalizeLongConnOneShotBody 将业务层输出归一化为单次回复类命令体。
// 目前支持两类命令：
//   - aibot_respond_welcome_msg
//   - aibot_respond_update_msg
func normalizeLongConnOneShotBody(command string, payload any, content string) (any, error) {
	switch command {
	case LongConnCmdRespondWelcomeMsg:
		if payload == nil {
			if content == "" {
				return nil, nil
			}
			return TextMessage{
				MsgType: "text",
				Text: &TextPayload{
					Content: content,
				},
			}, nil
		}

		switch body := payload.(type) {
		case TextMessage, *TextMessage:
			return body, nil
		case string:
			return TextMessage{
				MsgType: "text",
				Text: &TextPayload{
					Content: body,
				},
			}, nil
		default:
			return nil, fmt.Errorf("unsupported longconn welcome payload: %T", payload)
		}
	case LongConnCmdRespondUpdateMsg:
		switch body := payload.(type) {
		case UpdateTemplateCardMessage, *UpdateTemplateCardMessage:
			return body, nil
		case TemplateCardMessage:
			return UpdateTemplateCardMessage{
				ResponseType: "update_template_card",
				TemplateCard: body.TemplateCard,
			}, nil
		case *TemplateCardMessage:
			return UpdateTemplateCardMessage{
				ResponseType: "update_template_card",
				TemplateCard: body.TemplateCard,
			}, nil
		case *TemplateCard:
			return UpdateTemplateCardMessage{
				ResponseType: "update_template_card",
				TemplateCard: body,
			}, nil
		default:
			return nil, fmt.Errorf("unsupported longconn update payload: %T", payload)
		}
	default:
		return nil, fmt.Errorf("unsupported longconn one-shot command: %s", command)
	}
}

// sendCallbackCommand 使用临时请求上下文发送一条回调关联命令。
func (b *LongConnBot) sendCallbackCommand(command string, requestID string, body any) error {
	ctx, cancel := b.newRequestContext(context.Background())
	defer cancel()
	return b.sendRequestAndWait(ctx, command, requestID, body)
}

// sendRequestAndWait 发送一条长连接命令，并等待同 req_id 的响应帧返回。
func (b *LongConnBot) sendRequestAndWait(ctx context.Context, command, requestID string, body any) error {
	if requestID == "" {
		return errors.New("request id is required")
	}

	conn := b.currentConn()
	if conn == nil {
		return errors.New("longconn websocket is not connected")
	}

	waiter := b.registerPending(requestID)
	defer b.unregisterPending(requestID, waiter)

	// 关键步骤：先登记 pending，再写请求，避免响应过快时丢失通知。
	req := NewLongConnRequest(command, requestID, body)
	if err := b.writeJSON(conn, req); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-waiter:
		if result.err != nil {
			return result.err
		}
		if result.response.ErrCode != 0 {
			return &longConnAPIError{
				cmd:       command,
				requestID: requestID,
				errCode:   result.response.ErrCode,
				errMsg:    result.response.ErrMsg,
			}
		}
		return nil
	}
}

// writeJSON 以串行方式写入 WebSocket，避免并发写导致底层连接状态损坏。
func (b *LongConnBot) writeJSON(conn *websocket.Conn, payload any) error {
	if conn == nil {
		return errors.New("websocket connection is nil")
	}

	b.writeMu.Lock()
	defer b.writeMu.Unlock()

	if err := conn.SetWriteDeadline(time.Now().Add(b.writeTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(payload)
}

// registerPending 为指定 req_id 注册一个等待响应的通道。
func (b *LongConnBot) registerPending(requestID string) chan longConnAckResult {
	waiter := make(chan longConnAckResult, 1)
	b.pendingMu.Lock()
	b.pending[requestID] = waiter
	b.pendingMu.Unlock()
	return waiter
}

// unregisterPending 在请求完成或超时后清理 pending 映射。
func (b *LongConnBot) unregisterPending(requestID string, waiter chan longConnAckResult) {
	b.pendingMu.Lock()
	current, ok := b.pending[requestID]
	if ok && current == waiter {
		delete(b.pending, requestID)
	}
	b.pendingMu.Unlock()
}

// completePending 将响应结果投递给等待指定 req_id 的调用方。
// 返回值表示是否成功命中等待者。
func (b *LongConnBot) completePending(requestID string, result longConnAckResult) bool {
	if requestID == "" {
		return false
	}

	b.pendingMu.Lock()
	waiter, ok := b.pending[requestID]
	if ok {
		delete(b.pending, requestID)
	}
	b.pendingMu.Unlock()
	if !ok {
		return false
	}

	waiter <- result
	close(waiter)
	return true
}

// failAllPending 在连接关闭时批量唤醒所有等待中的请求，避免永久阻塞。
func (b *LongConnBot) failAllPending(err error) {
	b.pendingMu.Lock()
	waiters := b.pending
	b.pending = make(map[string]chan longConnAckResult)
	b.pendingMu.Unlock()

	for _, waiter := range waiters {
		waiter <- longConnAckResult{err: err}
		close(waiter)
	}
}

// setConn 设置当前活跃连接。
func (b *LongConnBot) setConn(conn *websocket.Conn) {
	b.connMu.Lock()
	b.conn = conn
	b.connMu.Unlock()
}

// releaseConn 释放当前活跃连接，并将错误广播给所有等待中的调用方。
func (b *LongConnBot) releaseConn(conn *websocket.Conn, err error) {
	b.connMu.Lock()
	if b.conn == conn {
		b.conn = nil
	}
	b.connMu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	if err != nil {
		b.failAllPending(err)
	}
}

// currentConn 返回当前活跃连接快照。
func (b *LongConnBot) currentConn() *websocket.Conn {
	b.connMu.RLock()
	defer b.connMu.RUnlock()
	return b.conn
}

// nextRequestID 生成一条新的长连接请求标识。
func (b *LongConnBot) nextRequestID() string {
	return generateStreamID()
}

// newRequestContext 为单次长连接命令创建超时上下文。
func (b *LongConnBot) newRequestContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, b.requestTimeout)
}
