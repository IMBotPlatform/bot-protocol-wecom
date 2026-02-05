// Package wecom tests cover crypt protocol compatibility and key behaviors.
package wecom

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestCalcSignatureDeterministic 验证签名算法具备确定性。
func TestCalcSignatureDeterministic(t *testing.T) {
	sig1 := CalcSignature("token", "12345", "nonce", "cipher")
	sig2 := CalcSignature("token", "12345", "nonce", "cipher")
	if sig1 != sig2 {
		t.Fatalf("signature mismatch: %s vs %s", sig1, sig2)
	}
}

// TestCryptEncryptDecryptRoundTrip 验证加解密流程能完整往返。
func TestCryptEncryptDecryptRoundTrip(t *testing.T) {
	rawKey := bytes.Repeat([]byte{0x11}, 32)
	encodingKey := base64.StdEncoding.EncodeToString(rawKey)
	// 企业微信 EncodingAESKey 为 43 字节 Base64 字符串，需去掉末尾 '='。
	encodingKey = strings.TrimRight(encodingKey, "=")
	crypt, err := NewCrypt("token", encodingKey, "corpID")
	if err != nil {
		t.Fatalf("create crypt: %v", err)
	}
	payload := BuildStreamReply("stream-id", "hello", false)
	ts := "1700000000"
	resp, err := crypt.EncryptResponse(payload, ts, "nonce")
	if err != nil {
		t.Fatalf("encrypt reply: %v", err)
	}
	msg, err := crypt.DecryptMessage(resp.MsgSignature, resp.Timestamp, resp.Nonce, EncryptedRequest{Encrypt: resp.Encrypt})
	if err != nil {
		t.Fatalf("decrypt message: %v", err)
	}
	if msg.MsgType != "stream" {
		t.Fatalf("unexpected msgtype: %s", msg.MsgType)
	}
	if msg.Stream == nil || msg.Stream.ID != "stream-id" {
		t.Fatalf("unexpected stream payload: %#v", msg.Stream)
	}
}

// TestVerifyURLHandlesDecodedQueryValue 验证 URL 解码导致的 '+' 还原场景。
func TestVerifyURLHandlesDecodedQueryValue(t *testing.T) {
	token := "token"
	rawKey := bytes.Repeat([]byte{0x34}, 32)
	encodingAESKey := strings.TrimRight(base64.StdEncoding.EncodeToString(rawKey), "=")
	corpID := "corp-id"
	crypt, err := NewCrypt(token, encodingAESKey, corpID)
	if err != nil {
		t.Fatalf("create crypt: %v", err)
	}

	var (
		echostr      string
		expectedBody string
	)
	for i := 0; i < 512; i++ {
		extra := fmt.Sprintf("payload-%d", i)
		enc, err := crypt.Encrypt([]byte(extra))
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if strings.Contains(enc, "+") {
			echostr = enc
			expectedBody = extra
			break
		}
	}
	if echostr == "" {
		t.Skip("unable to generate test data containing '+'; try rerun")
	}

	timestamp := "1761891968"
	nonce := "random-nonce"
	signature := CalcSignature(token, timestamp, nonce, echostr)

	values := url.Values{}
	values.Set("msg_signature", signature)
	values.Set("timestamp", timestamp)
	values.Set("nonce", nonce)
	values.Set("echostr", echostr)
	rawQuery := values.Encode() // 会对 '+' 进行 %2B 转义
	req, err := http.NewRequest(http.MethodGet, "/callback?"+rawQuery, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	decoded := req.URL.Query().Get("echostr")
	if decoded != echostr {
		t.Fatalf("unexpected decoded value: %q", decoded)
	}

	plain, err := crypt.VerifyURL(signature, timestamp, nonce, decoded)
	if err != nil {
		t.Fatalf("verify url: %v", err)
	}
	if plain != expectedBody {
		t.Fatalf("unexpected plaintext: %q", plain)
	}
}

// TestVerifyURLRoundTrip 验证 URL 验证流程的加解密往返。
func TestVerifyURLRoundTrip(t *testing.T) {
	token := "sample-token"
	rawKey := bytes.Repeat([]byte{0x44}, 32)
	encodingAESKey := strings.TrimRight(base64.StdEncoding.EncodeToString(rawKey), "=")
	corpID := "sample-corp-id"
	crypt, err := NewCrypt(token, encodingAESKey, corpID)
	if err != nil {
		t.Fatalf("create crypt: %v", err)
	}

	payload := []byte("roundtrip-payload")
	echoStr, err := crypt.Encrypt(payload)
	if err != nil {
		t.Fatalf("encrypt sample payload: %v", err)
	}

	timestamp := "1761891968"
	nonce := "nonce"
	signature := CalcSignature(token, timestamp, nonce, echoStr)

	plain, err := crypt.VerifyURL(signature, timestamp, nonce, echoStr)
	if err != nil {
		t.Fatalf("verify url: %v", err)
	}

	if plain != string(payload) {
		t.Fatalf("unexpected plaintext: %s", plain)
	}
}

// TestDecryptMessageWithDocSample 使用官方样例密文验证解密兼容性。
func TestDecryptMessageWithDocSample(t *testing.T) {
	token := "QDG6eK"
	encodingAESKey := "jWmYm7qr5nMoAUwZRjGtBxmz3KA1tkAj3ykkR6q2B2C"
	corpID := "wx5823bf96d3bd56c7"
	crypt, err := NewCrypt(token, encodingAESKey, corpID)
	if err != nil {
		t.Fatalf("create crypt: %v", err)
	}

	const cipherText = "RypEvHKD8QQKFhvQ6QleEB4J58tiPdvo+rtK1I9qca6aM/wvqnLSV5zEPeusUiX5L5X/0lWfrf0QADHHhGd3QczcdCUpj911L3vg3W/sYYvuJTs3TUUkSUXxaccAS0qhxchrRYt66wiSpGLYL42aM6A8dTT+6k4aSknmPj48kzJs8qLjvd4Xgpue06DOdnLxAUHzM6+kDZ+HMZfJYuR+LtwGc2hgf5gsijff0ekUNXZiqATP7PF5mZxZ3Izoun1s4zG4LUMnvw2r+KqCKIw+3IQH03v+BCA9nMELNqbSf6tiWSrXJB3LAVGUcallcrw8V2t9EL4EhzJWrQUax5wLVMNS0+rUPA3k22Ncx4XXZS9o0MBH27Bo6BpNelZpS+/uh9KsNlY6bHCmJU9p8g7m3fVKn28H3KDYA5Pl/T8Z1ptDAVe0lXdQ2YoyyH2uyPIGHBZZIs2pDBS8R07+qN+E7Q=="
	plain, err := crypt.Decrypt(cipherText)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	const expectedXML = `<xml><ToUserName><![CDATA[wx5823bf96d3bd56c7]]></ToUserName>
<FromUserName><![CDATA[mycreate]]></FromUserName>
<CreateTime>1409659813</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[hello]]></Content>
<MsgId>4561255354251345929</MsgId>
<AgentID>218</AgentID>
</xml>`
	if string(plain) != expectedXML {
		t.Fatalf("unexpected plaintext:\n%s", plain)
	}
}
