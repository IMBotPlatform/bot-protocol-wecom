package wecom

import (
	"os"
	"strconv"
	"time"
)

const (
	// envBotHTTPTimeout 控制主动回复 HTTP 请求的超时时间。
	envBotHTTPTimeout = "BOT_HTTP_TIMEOUT"
	// envBotStreamTTL 控制流式消息会话的最大存活时间。
	envBotStreamTTL = "BOT_STREAM_TTL"
	// envBotStreamWaitTimeout 控制刷新请求等待流水线片段的最大时长。
	envBotStreamWaitTimeout = "BOT_STREAM_WAIT_TIMEOUT"
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
