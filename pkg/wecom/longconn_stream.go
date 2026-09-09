package wecom

import (
	"context"
	"encoding/base64"
	"errors"
	"time"
)

// A transport owns its stream IDs. HTTP-style final payloads must update the
// same long-connection stream as earlier text chunks. Close idle streams before
// WeCom's ten-minute deadline; subsequent output starts a fresh stream.
func (b *LongConnBot) consumeMessageChunksWithWindow(requestID string, source <-chan Chunk, interval, window time.Duration) error {
	out := coalesceLongConnChunks(source, interval)
	defer func() {
		for range out {
		}
	}()
	var streamID, text string
	var feedback *FeedbackInfo
	var deadline <-chan time.Time
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	emit := func(final bool) error {
		if streamID == "" {
			streamID = generateStreamID()
			timer = time.NewTimer(window)
			deadline = timer.C
		}
		reply := BuildStreamReply(streamID, text, final)
		reply.Stream.Feedback = feedback
		err := b.sendCallbackCommand(LongConnCmdRespondMsg, requestID, reply)
		if final {
			timer.Stop()
			deadline = nil
			streamID = ""
		}
		return err
	}
	for {
		select {
		case <-deadline:
			if err := emit(true); err != nil {
				return err
			}
		case c, ok := <-out:
			if !ok {
				if streamID != "" {
					return emit(true)
				}
				return nil
			}
			if c.Replace && c.Payload != nil {
				return errors.New("replace cannot accompany payload")
			}
			if c.Payload == NoResponse {
				if streamID != "" {
					return emit(true)
				}
				return nil
			}
			var items []MixedItem
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
					feedback = stream.Stream.Feedback
					items = stream.Stream.MsgItem
					c.IsFinal = c.IsFinal || stream.Stream.Finish
				} else {
					body, err := normalizeLongConnMessageBody(c.Payload)
					if err != nil {
						return err
					}
					if streamID != "" {
						if err := emit(true); err != nil {
							return err
						}
					}
					if err := b.sendCallbackCommand(LongConnCmdRespondMsg, requestID, body); err != nil {
						return err
					}
					if c.IsFinal {
						return nil
					}
					continue
				}
			} else {
				// Coalescer output contains complete snapshots, including empty replacements.
				text = c.Content
				items = c.MsgItems
			}
			if len(items) > 0 && !c.IsFinal {
				return errors.New("stream media requires final chunk")
			}
			if err := emit(c.IsFinal); err != nil {
				return err
			}
			if err := b.sendStreamImages(requestID, items); err != nil {
				return err
			}
			if c.IsFinal {
				return nil
			}
		}
	}
}

// Long connections use media_id messages instead of webhook stream.msg_item.
func (b *LongConnBot) sendStreamImages(requestID string, items []MixedItem) error {
	for _, item := range items {
		if item.MsgType != "image" || item.Image == nil {
			return errors.New("unsupported stream media item")
		}
		data, err := base64.StdEncoding.DecodeString(item.Image.Base64)
		if err != nil {
			return errors.New("invalid stream image encoding")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		media, err := b.UploadMedia(ctx, LongConnMediaTypeImage, "image.png", data)
		cancel()
		if err != nil {
			return err
		}
		if err := b.sendCallbackCommand(LongConnCmdRespondMsg, requestID, ImageMessage{MsgType: "image", Image: MediaMessagePayload{MediaID: media.MediaID}}); err != nil {
			return err
		}
	}
	return nil
}
