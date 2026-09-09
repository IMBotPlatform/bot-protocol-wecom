package wecom

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrLongConnReplaced means another client subscribed with the same Bot ID.
// Do not reconnect automatically and fight the new connection.
var ErrLongConnReplaced = errors.New("longconn replaced by another client")

// Ready reports a live, successfully authenticated subscription.
func (b *LongConnBot) Ready() bool {
	if b == nil {
		return false
	}
	b.connMu.RLock()
	defer b.connMu.RUnlock()
	return b.conn != nil && b.ready
}

type longConnCallback struct {
	requestID, chat string
	received        time.Time
}

func messageChat(msg Message) string {
	if msg.ChatType == "single" {
		return fmt.Sprintf("1:%s", msg.From.UserID)
	}
	return fmt.Sprintf("2:%s", msg.ChatID)
}

// Keep bounded deduplication metadata; never retain callback text or media keys.
func (b *LongConnBot) acceptCallback(frame LongConnRawFrame, msg Message) bool {
	key := frame.Cmd + ":" + msg.MsgID
	if msg.MsgID == "" {
		key = frame.Cmd + ":" + frame.Headers.RequestID
	}
	b.callbackMu.Lock()
	defer b.callbackMu.Unlock()
	if b.callbacks == nil {
		b.callbacks = make(map[string]longConnCallback)
	}
	now := time.Now()
	for k, v := range b.callbacks {
		if now.Sub(v.received) > 24*time.Hour {
			delete(b.callbacks, k)
		}
	}
	if _, ok := b.callbacks[key]; ok {
		return false
	}
	if len(b.callbacks) >= 10000 {
		return false
	}
	b.callbacks[key] = longConnCallback{frame.Headers.RequestID, messageChat(msg), now}
	return true
}

// Passive and proactive messages share WeCom's per-conversation 30/min, 1000/h budget.
func (b *LongConnBot) waitMessageRate(ctx context.Context, command, requestID string, body any) error {
	var key string
	switch command {
	case LongConnCmdSendMsg:
		if p, ok := body.(LongConnPushMessage); ok {
			key = fmt.Sprintf("%d:%s", p.ChatType, p.ChatID)
		}
	case LongConnCmdRespondMsg, LongConnCmdRespondWelcomeMsg, LongConnCmdRespondUpdateMsg:
		b.callbackMu.Lock()
		for _, c := range b.callbacks {
			if c.requestID == requestID {
				key = c.chat
				break
			}
		}
		b.callbackMu.Unlock()
	default:
		return nil
	}
	if key == "" {
		key = "request:" + requestID
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := time.Now()
		b.messageRateMu.Lock()
		if b.messageTimes == nil {
			b.messageTimes = make(map[string][]time.Time)
		}
		for k, times := range b.messageTimes {
			if len(times) == 0 || now.Sub(times[len(times)-1]) >= time.Hour {
				delete(b.messageTimes, k)
			}
		}
		times := b.messageTimes[key]
		for len(times) > 0 && now.Sub(times[0]) >= time.Hour {
			times = times[1:]
		}
		firstMinute := 0
		for firstMinute < len(times) && now.Sub(times[firstMinute]) >= time.Minute {
			firstMinute++
		}
		if len(times) < 1000 && len(times)-firstMinute < 30 {
			b.messageTimes[key] = append(times, now)
			b.messageRateMu.Unlock()
			return nil
		}
		var until time.Time
		if len(times)-firstMinute >= 30 {
			until = times[firstMinute].Add(time.Minute)
		}
		if len(times) >= 1000 && times[0].Add(time.Hour).After(until) {
			until = times[0].Add(time.Hour)
		}
		b.messageTimes[key] = times
		b.messageRateMu.Unlock()
		timer := time.NewTimer(time.Until(until))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-b.closedCh:
			timer.Stop()
			return errors.New("longconn bot closed")
		case <-timer.C:
		}
	}
}

// Drain producers even after a transport failure. Only text snapshots are coalesced;
// payloads and final chunks retain their ordering and are emitted immediately.
func coalesceLongConnChunks(source <-chan Chunk, interval time.Duration) <-chan Chunk {
	out := make(chan Chunk)
	go func() {
		defer close(out)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		text := ""
		dirty := false
		sent := false
		for {
			select {
			case c, ok := <-source:
				if !ok {
					if dirty {
						out <- Chunk{Content: text, Replace: true, IsFinal: true}
					}
					return
				}
				if c.Payload != nil {
					var stream *StreamReply
					switch v := c.Payload.(type) {
					case StreamReply:
						stream = &v
					case *StreamReply:
						stream = v
					}
					if stream != nil {
						text = stream.Stream.Content
						c.IsFinal = c.IsFinal || stream.Stream.Finish
					} else if dirty && c.Payload != NoResponse {
						out <- Chunk{Content: text, Replace: true}
					}
					dirty = false
					sent = true
					out <- c
					if c.IsFinal || c.Payload == NoResponse {
						for range source {
						}
						return
					}
					continue
				}
				if c.Replace {
					text = c.Content
				} else {
					text += c.Content
				}
				dirty = true
				if !sent || c.IsFinal || len(c.MsgItems) > 0 {
					c.Content = text
					c.Replace = true
					out <- c
					dirty = false
					sent = true
				}
				if c.IsFinal {
					for range source {
					}
					return
				}
			case <-tick.C:
				if dirty {
					out <- Chunk{Content: text, Replace: true}
					dirty = false
					sent = true
				}
			}
		}
	}()
	return out
}
