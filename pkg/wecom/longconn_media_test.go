package wecom

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type capturedLongConnRequest struct {
	Cmd     string          `json:"cmd"`
	Headers LongConnHeaders `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
}

func TestLongConnUploadMediaEndToEnd(t *testing.T) {
	captured := make(chan capturedLongConnRequest, 8)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			var req capturedLongConnRequest
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			captured <- req

			response := map[string]any{
				"headers": req.Headers,
				"errcode": 0,
				"errmsg":  "ok",
			}
			switch req.Cmd {
			case LongConnCmdUploadMediaInit:
				response["body"] = map[string]any{"upload_id": "upload-1"}
			case LongConnCmdUploadMediaFinish:
				response["body"] = map[string]any{
					"type":       "file",
					"media_id":   "media-1",
					"created_at": "1380000000",
				}
			}
			if err := conn.WriteJSON(response); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	bot, err := NewLongConnBotWithOptions("bot-id", "secret", nil, LongConnOptions{
		RequestTimeout: 2 * time.Second,
		WriteTimeout:   2 * time.Second,
	})
	if err != nil {
		t.Fatalf("create bot: %v", err)
	}
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	bot.setConn(clientConn)
	readErrCh := make(chan error, 1)
	go bot.readLoop(clientConn, readErrCh)
	defer bot.Close()

	data := bytes.Repeat([]byte("a"), LongConnMediaChunkSize+37)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := bot.UploadMedia(ctx, LongConnMediaTypeFile, "report.bin", data)
	if err != nil {
		t.Fatalf("upload media: %v", err)
	}
	if result.Type != LongConnMediaTypeFile || result.MediaID != "media-1" || result.CreatedAt != 1380000000 {
		t.Fatalf("unexpected upload result: %+v", result)
	}

	requests := make([]capturedLongConnRequest, 0, 4)
	for len(requests) < 4 {
		select {
		case req := <-captured:
			requests = append(requests, req)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for request %d", len(requests)+1)
		}
	}
	wantCommands := []string{
		LongConnCmdUploadMediaInit,
		LongConnCmdUploadMediaChunk,
		LongConnCmdUploadMediaChunk,
		LongConnCmdUploadMediaFinish,
	}
	for i, want := range wantCommands {
		if requests[i].Cmd != want {
			t.Fatalf("request %d cmd: got=%s want=%s", i, requests[i].Cmd, want)
		}
	}

	var initBody LongConnUploadMediaInitBody
	if err := json.Unmarshal(requests[0].Body, &initBody); err != nil {
		t.Fatalf("decode init body: %v", err)
	}
	sum := md5.Sum(data)
	if initBody.Type != LongConnMediaTypeFile || initBody.TotalSize != len(data) || initBody.TotalChunks != 2 {
		t.Fatalf("unexpected init body: %+v", initBody)
	}
	if initBody.MD5 != hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected init md5: %s", initBody.MD5)
	}

	assembled := make([]byte, 0, len(data))
	for i, req := range requests[1:3] {
		var chunkBody LongConnUploadMediaChunkBody
		if err := json.Unmarshal(req.Body, &chunkBody); err != nil {
			t.Fatalf("decode chunk %d: %v", i, err)
		}
		if chunkBody.UploadID != "upload-1" || chunkBody.ChunkIndex != i {
			t.Fatalf("unexpected chunk %d body: %+v", i, chunkBody)
		}
		chunk, err := base64.StdEncoding.DecodeString(chunkBody.Base64Data)
		if err != nil {
			t.Fatalf("decode chunk %d data: %v", i, err)
		}
		assembled = append(assembled, chunk...)
	}
	if !bytes.Equal(assembled, data) {
		t.Fatalf("assembled upload data mismatch: got=%d want=%d", len(assembled), len(data))
	}
}

func TestValidateLongConnUploadInit(t *testing.T) {
	valid := LongConnUploadMediaInitBody{
		Type:        LongConnMediaTypeImage,
		Filename:    "photo.JPG",
		TotalSize:   1024,
		TotalChunks: 1,
	}
	if err := validateLongConnUploadInit(valid); err != nil {
		t.Fatalf("valid upload rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*LongConnUploadMediaInitBody)
	}{
		{
			name: "unsupported type",
			mutate: func(body *LongConnUploadMediaInitBody) {
				body.Type = "archive"
			},
		},
		{
			name: "wrong extension",
			mutate: func(body *LongConnUploadMediaInitBody) {
				body.Filename = "photo.bmp"
			},
		},
		{
			name: "too small",
			mutate: func(body *LongConnUploadMediaInitBody) {
				body.TotalSize = 4
			},
		},
		{
			name: "too few chunks",
			mutate: func(body *LongConnUploadMediaInitBody) {
				body.TotalSize = LongConnMediaChunkSize + 1
				body.TotalChunks = 1
			},
		},
		{
			name: "too many chunks",
			mutate: func(body *LongConnUploadMediaInitBody) {
				body.TotalChunks = LongConnMediaMaxChunks + 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := valid
			tt.mutate(&body)
			if err := validateLongConnUploadInit(body); err == nil {
				t.Fatalf("invalid body accepted: %+v", body)
			}
		})
	}
}

func TestLongConnMediaUploadRateLimitHonorsContext(t *testing.T) {
	originalWindow := longConnMediaRateWindow
	originalHourlyWindow := longConnMediaHourlyRateWindow
	longConnMediaRateWindow = 50 * time.Millisecond
	longConnMediaHourlyRateWindow = 100 * time.Millisecond
	defer func() {
		longConnMediaRateWindow = originalWindow
		longConnMediaHourlyRateWindow = originalHourlyWindow
	}()

	bot, err := NewLongConnBot("bot-id", "secret", nil)
	if err != nil {
		t.Fatalf("create bot: %v", err)
	}
	for i := 0; i < longConnMediaRateMaxRequests; i++ {
		if err := bot.waitMediaUploadRate(context.Background()); err != nil {
			t.Fatalf("reserve rate slot %d: %v", i, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if err := bot.waitMediaUploadRate(ctx); err == nil {
		t.Fatal("rate-limited call should honor context deadline")
	}

	time.Sleep(longConnMediaRateWindow)
	if err := bot.waitMediaUploadRate(context.Background()); err != nil {
		t.Fatalf("rate slot should recover after window: %v", err)
	}
}

func TestLongConnMediaUploadHourlyRateLimitHonorsContext(t *testing.T) {
	originalWindow := longConnMediaRateWindow
	originalHourlyWindow := longConnMediaHourlyRateWindow
	longConnMediaRateWindow = time.Millisecond
	longConnMediaHourlyRateWindow = 50 * time.Millisecond
	defer func() {
		longConnMediaRateWindow = originalWindow
		longConnMediaHourlyRateWindow = originalHourlyWindow
	}()

	bot, err := NewLongConnBot("bot-id", "secret", nil)
	if err != nil {
		t.Fatalf("create bot: %v", err)
	}
	requestTime := time.Now().Add(-2 * time.Millisecond)
	bot.mediaRequestTimes = make([]time.Time, longConnMediaRateMaxHourly)
	for i := range bot.mediaRequestTimes {
		bot.mediaRequestTimes[i] = requestTime
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if err := bot.waitMediaUploadRate(ctx); err == nil {
		t.Fatal("hourly rate-limited call should honor context deadline")
	}

	time.Sleep(longConnMediaHourlyRateWindow)
	if err := bot.waitMediaUploadRate(context.Background()); err != nil {
		t.Fatalf("hourly rate slot should recover after window: %v", err)
	}
}

func TestLongConnMediaUploadRateDoesNotReserveCanceledRequest(t *testing.T) {
	bot, err := NewLongConnBot("bot-id", "secret", nil)
	if err != nil {
		t.Fatalf("create bot: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := bot.waitMediaUploadRate(ctx); err == nil {
		t.Fatal("canceled upload request should be rejected")
	}
	if len(bot.mediaRequestTimes) != 0 {
		t.Fatalf("canceled upload reserved a rate slot: %d", len(bot.mediaRequestTimes))
	}
}
