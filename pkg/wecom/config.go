package wecom

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// envBotHTTPTimeout 控制主动回复 HTTP 请求的超时时间。
	envBotHTTPTimeout = "BOT_HTTP_TIMEOUT"
	// envBotStreamTTL 控制流式消息会话的最大存活时间。
	envBotStreamTTL = "BOT_STREAM_TTL"
	// envBotStreamWaitTimeout 控制刷新请求等待流水线片段的最大时长。
	envBotStreamWaitTimeout = "BOT_STREAM_WAIT_TIMEOUT"
	// envLongConnWSURL 控制长连接 WebSocket 地址。
	envLongConnWSURL = "BOT_LONG_CONN_WS_URL"
	// envLongConnPingInterval 控制长连接心跳间隔。
	envLongConnPingInterval = "BOT_LONG_CONN_PING_INTERVAL"
	// envLongConnReconnectInterval 控制长连接断线重连间隔。
	envLongConnReconnectInterval = "BOT_LONG_CONN_RECONNECT_INTERVAL"
	// envLongConnRequestTimeout 控制长连接单次请求等待响应的超时时间。
	envLongConnRequestTimeout = "BOT_LONG_CONN_REQUEST_TIMEOUT"
	// envLongConnWriteTimeout 控制长连接单次写入超时时间。
	envLongConnWriteTimeout = "BOT_LONG_CONN_WRITE_TIMEOUT"
)

// resolveDuration 解析时间配置，优先级为：参数值 > 环境变量 > 默认值。
// Parameters:
//   - paramVal: 显式传入的参数值（<=0 表示未设置）
//   - envKey: 环境变量名
//   - defaultVal: 默认值
//
// Returns:
//   - time.Duration: 解析后的时间值
func resolveDuration(paramVal time.Duration, envKey string, defaultVal time.Duration) time.Duration {
	// 优先使用显式传入的参数值。
	if paramVal > 0 {
		return paramVal
	}

	// 其次查找环境变量。
	if envStr := os.Getenv(envKey); envStr != "" {
		if secs, err := strconv.Atoi(envStr); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}

	// 最后返回默认值。
	return defaultVal
}

// resolveString 解析字符串配置，优先级为：参数值 > 环境变量 > 默认值。
// Parameters:
//   - paramVal: 显式传入的参数值（空字符串表示未设置）
//   - envKey: 环境变量名
//   - defaultVal: 默认值
//
// Returns:
//   - string: 解析后的字符串值
func resolveString(paramVal string, envKey string, defaultVal string) string {
	if trimmed := strings.TrimSpace(paramVal); trimmed != "" {
		return trimmed
	}

	if envStr := strings.TrimSpace(os.Getenv(envKey)); envStr != "" {
		return envStr
	}

	return defaultVal
}
