package wecom

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReplaceSnapshotKeepsFinalAndEmptyWithoutBackpressure(t *testing.T) {
	m := newStreamManager(time.Minute, time.Millisecond)
	s, _ := m.createOrGet(&Message{MsgID: "snapshot"})
	m.publish(s.StreamID, Chunk{Content: "first"})
	m.publish(s.StreamID, Chunk{Content: " second"})
	if v := m.getLatestChunk(s.StreamID); v.Content != "first second" {
		t.Fatal(v)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			m.publish(s.StreamID, Chunk{Content: fmt.Sprint(i), Replace: true})
		}
		m.publish(s.StreamID, Chunk{Replace: true})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("obsolete snapshots blocked publisher")
	}
	if v := m.getLatestChunk(s.StreamID); v.Content != "" || !v.Replace {
		t.Fatal(v)
	}
	m.publish(s.StreamID, Chunk{Content: "answer", Replace: true, IsFinal: true})
	if v := m.getLatestChunk(s.StreamID); v.Content != "answer" || !v.IsFinal {
		t.Fatal(v)
	}
	if m.publish(s.StreamID, Chunk{Replace: true, Payload: "mixed"}) {
		t.Fatal("mixed replacement accepted")
	}
}
func TestResponseChecksBusinessAcknowledgementWithoutLeakingSecrets(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		ok     bool
	}{{200, `{"errcode":0}`, true}, {200, `{"errcode":40014,"errmsg":"PRIVATE_SENTINEL"}`, false}, {500, `PRIVATE_SENTINEL`, false}, {200, `{}`, false}, {200, `invalid PRIVATE_SENTINEL`, false}} {
		t.Run(fmt.Sprint(tc.status, tc.body), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			b := &Bot{client: server.Client()}
			err := b.ResponseMarkdown(server.URL+"?token=PRIVATE_SENTINEL", "secret")
			if (err == nil) != tc.ok {
				t.Fatal(err)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
				t.Fatal("sensitive response leaked")
			}
		})
	}
}
