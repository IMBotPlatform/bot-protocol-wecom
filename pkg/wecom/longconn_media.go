package wecom

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	// LongConnMediaChunkSize 是企业微信允许的单个素材分片最大原始字节数。
	LongConnMediaChunkSize = 512 * 1024
	// LongConnMediaMaxChunks 是一次临时素材上传允许的最大分片数。
	LongConnMediaMaxChunks = 100

	longConnMediaMinBytes        = 5
	longConnMediaMaxFileBytes    = 20 * 1024 * 1024
	longConnMediaMaxImageBytes   = 10 * 1024 * 1024
	longConnMediaMaxVoiceBytes   = 2 * 1024 * 1024
	longConnMediaMaxVideoBytes   = 10 * 1024 * 1024
	longConnMediaRateMaxRequests = 30
	longConnMediaRateMaxHourly   = 1000
)

var (
	longConnMediaRateWindow       = time.Minute
	longConnMediaHourlyRateWindow = time.Hour
)

// InitMediaUpload 初始化一次长连接临时素材上传并返回 upload_id。
// 调用方可以配合 UploadMediaChunk 和 FinishMediaUpload 实现断线续传或自定义分片顺序。
func (b *LongConnBot) InitMediaUpload(ctx context.Context, body LongConnUploadMediaInitBody) (string, error) {
	if b == nil {
		return "", errors.New("longconn bot is nil")
	}
	if err := validateLongConnUploadInit(body); err != nil {
		return "", err
	}

	req := BuildLongConnUploadMediaInitRequest(b.nextRequestID(), body)
	resp, err := b.sendMediaRequestWithTimeout(ctx, req)
	if err != nil {
		return "", err
	}

	var result LongConnUploadMediaInitResponseBody
	if len(resp.Body) == 0 {
		return "", errors.New("upload media init response body is empty")
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", fmt.Errorf("decode upload media init response: %w", err)
	}
	if strings.TrimSpace(result.UploadID) == "" {
		return "", errors.New("upload media init response missing upload_id")
	}
	return result.UploadID, nil
}

// UploadMediaChunk 上传一个临时素材分片。
func (b *LongConnBot) UploadMediaChunk(ctx context.Context, uploadID string, chunkIndex int, chunk []byte) error {
	if b == nil {
		return errors.New("longconn bot is nil")
	}
	if strings.TrimSpace(uploadID) == "" {
		return errors.New("upload id is required")
	}
	if chunkIndex < 0 || chunkIndex >= LongConnMediaMaxChunks {
		return fmt.Errorf("chunk index must be between 0 and %d", LongConnMediaMaxChunks-1)
	}
	if len(chunk) == 0 {
		return errors.New("media chunk is empty")
	}
	if len(chunk) > LongConnMediaChunkSize {
		return fmt.Errorf("media chunk exceeds %d bytes", LongConnMediaChunkSize)
	}

	req := BuildLongConnUploadMediaChunkRequest(b.nextRequestID(), uploadID, chunkIndex, chunk)
	_, err := b.sendMediaRequestWithTimeout(ctx, req)
	return err
}

// FinishMediaUpload 完成临时素材上传并返回 media_id 等结果。
func (b *LongConnBot) FinishMediaUpload(ctx context.Context, uploadID string) (LongConnUploadMediaResult, error) {
	if b == nil {
		return LongConnUploadMediaResult{}, errors.New("longconn bot is nil")
	}
	if strings.TrimSpace(uploadID) == "" {
		return LongConnUploadMediaResult{}, errors.New("upload id is required")
	}

	req := BuildLongConnUploadMediaFinishRequest(b.nextRequestID(), uploadID)
	resp, err := b.sendMediaRequestWithTimeout(ctx, req)
	if err != nil {
		return LongConnUploadMediaResult{}, err
	}
	if len(resp.Body) == 0 {
		return LongConnUploadMediaResult{}, errors.New("upload media finish response body is empty")
	}

	var result LongConnUploadMediaResult
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return LongConnUploadMediaResult{}, fmt.Errorf("decode upload media finish response: %w", err)
	}
	if strings.TrimSpace(result.MediaID) == "" {
		return LongConnUploadMediaResult{}, errors.New("upload media finish response missing media_id")
	}
	return result, nil
}

// UploadMedia 按企业微信协议自动完成初始化、512KB 分片上传和结束合并。
func (b *LongConnBot) UploadMedia(ctx context.Context, mediaType LongConnMediaType, filename string, data []byte) (LongConnUploadMediaResult, error) {
	if b == nil {
		return LongConnUploadMediaResult{}, errors.New("longconn bot is nil")
	}

	totalChunks := (len(data) + LongConnMediaChunkSize - 1) / LongConnMediaChunkSize
	sum := md5.Sum(data)
	initBody := LongConnUploadMediaInitBody{
		Type:        mediaType,
		Filename:    filename,
		TotalSize:   len(data),
		TotalChunks: totalChunks,
		MD5:         hex.EncodeToString(sum[:]),
	}
	uploadID, err := b.InitMediaUpload(ctx, initBody)
	if err != nil {
		return LongConnUploadMediaResult{}, err
	}

	for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
		start := chunkIndex * LongConnMediaChunkSize
		end := start + LongConnMediaChunkSize
		if end > len(data) {
			end = len(data)
		}
		if err := b.UploadMediaChunk(ctx, uploadID, chunkIndex, data[start:end]); err != nil {
			return LongConnUploadMediaResult{}, fmt.Errorf("upload media chunk %d: %w", chunkIndex, err)
		}
	}

	return b.FinishMediaUpload(ctx, uploadID)
}

// sendMediaRequestWithTimeout 应用上传频率限制并发送一条素材命令。
func (b *LongConnBot) sendMediaRequestWithTimeout(parent context.Context, req LongConnRequest) (LongConnResponse, error) {
	if err := b.waitMediaUploadRate(parent); err != nil {
		return LongConnResponse{}, err
	}
	ctx, cancel := b.newRequestContext(parent)
	defer cancel()
	return b.sendRequestAndWaitResponse(ctx, req.Cmd, req.Headers.RequestID, req.Body)
}

// waitMediaUploadRate 实现单机器人每分钟最多 30 次、每小时最多 1000 次素材上传命令的滚动窗口限制。
func (b *LongConnBot) waitMediaUploadRate(ctx context.Context) error {
	if b == nil {
		return errors.New("longconn bot is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.closedCh:
			return errors.New("longconn bot closed")
		default:
		}

		now := time.Now()
		minuteCutoff := now.Add(-longConnMediaRateWindow)
		hourCutoff := now.Add(-longConnMediaHourlyRateWindow)
		retentionCutoff := minuteCutoff
		if hourCutoff.Before(retentionCutoff) {
			retentionCutoff = hourCutoff
		}

		b.mediaRateMu.Lock()
		firstActive := 0
		for firstActive < len(b.mediaRequestTimes) && !b.mediaRequestTimes[firstActive].After(retentionCutoff) {
			firstActive++
		}
		if firstActive > 0 {
			copy(b.mediaRequestTimes, b.mediaRequestTimes[firstActive:])
			b.mediaRequestTimes = b.mediaRequestTimes[:len(b.mediaRequestTimes)-firstActive]
		}

		firstMinute := 0
		for firstMinute < len(b.mediaRequestTimes) && !b.mediaRequestTimes[firstMinute].After(minuteCutoff) {
			firstMinute++
		}
		firstHour := 0
		for firstHour < len(b.mediaRequestTimes) && !b.mediaRequestTimes[firstHour].After(hourCutoff) {
			firstHour++
		}
		minuteCount := len(b.mediaRequestTimes) - firstMinute
		hourCount := len(b.mediaRequestTimes) - firstHour

		if minuteCount < longConnMediaRateMaxRequests && hourCount < longConnMediaRateMaxHourly {
			b.mediaRequestTimes = append(b.mediaRequestTimes, now)
			b.mediaRateMu.Unlock()
			return nil
		}

		var retryAt time.Time
		if minuteCount >= longConnMediaRateMaxRequests {
			retryAt = b.mediaRequestTimes[firstMinute].Add(longConnMediaRateWindow)
		}
		if hourCount >= longConnMediaRateMaxHourly {
			hourRetryAt := b.mediaRequestTimes[firstHour].Add(longConnMediaHourlyRateWindow)
			if hourRetryAt.After(retryAt) {
				retryAt = hourRetryAt
			}
		}
		waitFor := time.Until(retryAt)
		b.mediaRateMu.Unlock()
		if waitFor <= 0 {
			continue
		}

		timer := time.NewTimer(waitFor)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-b.closedCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return errors.New("longconn bot closed")
		case <-timer.C:
		}
	}
}

// validateLongConnUploadInit 校验上传初始化参数与官网限制。
func validateLongConnUploadInit(body LongConnUploadMediaInitBody) error {
	if _, err := longConnMediaMaxBytes(body.Type); err != nil {
		return err
	}
	filename := strings.TrimSpace(body.Filename)
	if filename == "" {
		return errors.New("media filename is required")
	}
	if len([]byte(filename)) > 256 {
		return errors.New("media filename exceeds 256 bytes")
	}
	if err := validateLongConnMediaExtension(body.Type, filename); err != nil {
		return err
	}
	if body.TotalSize < longConnMediaMinBytes {
		return fmt.Errorf("media size must be at least %d bytes", longConnMediaMinBytes)
	}
	maxBytes, _ := longConnMediaMaxBytes(body.Type)
	if body.TotalSize > maxBytes {
		return fmt.Errorf("%s media exceeds %d bytes", body.Type, maxBytes)
	}
	if body.TotalChunks <= 0 || body.TotalChunks > LongConnMediaMaxChunks {
		return fmt.Errorf("total chunks must be between 1 and %d", LongConnMediaMaxChunks)
	}
	minimumChunks := (body.TotalSize + LongConnMediaChunkSize - 1) / LongConnMediaChunkSize
	if body.TotalChunks < minimumChunks {
		return fmt.Errorf("total chunks %d cannot carry %d bytes", body.TotalChunks, body.TotalSize)
	}
	return nil
}

// longConnMediaMaxBytes 返回不同素材类型的大小上限。
func longConnMediaMaxBytes(mediaType LongConnMediaType) (int, error) {
	switch mediaType {
	case LongConnMediaTypeFile:
		return longConnMediaMaxFileBytes, nil
	case LongConnMediaTypeImage:
		return longConnMediaMaxImageBytes, nil
	case LongConnMediaTypeVoice:
		return longConnMediaMaxVoiceBytes, nil
	case LongConnMediaTypeVideo:
		return longConnMediaMaxVideoBytes, nil
	default:
		return 0, fmt.Errorf("unsupported longconn media type: %q", mediaType)
	}
}

// validateLongConnMediaExtension 校验官网明确限定的图片、语音和视频格式。
func validateLongConnMediaExtension(mediaType LongConnMediaType, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := false
	switch mediaType {
	case LongConnMediaTypeFile:
		return nil
	case LongConnMediaTypeImage:
		allowed = ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif"
	case LongConnMediaTypeVoice:
		allowed = ext == ".amr"
	case LongConnMediaTypeVideo:
		allowed = ext == ".mp4"
	default:
		return fmt.Errorf("unsupported longconn media type: %q", mediaType)
	}
	if !allowed {
		return fmt.Errorf("unsupported %s media extension: %q", mediaType, ext)
	}
	return nil
}
