package volcvoice

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/yankeguo/volcvoice/internal/stream_wire"
)

const (
	StreamSynthesizeClusterV1           = "volcano_mega"
	StreamSynthesizeClusterV1Concurrent = "volcano_mega_concurr"
	StreamSynthesizeClusterV2           = "volcano_icl"
	StreamSynthesizeClusterV2Concurrent = "volcano_icl_concurr"
)

// StreamSynthesizeOutput is a function to process output audio chunk.
type StreamSynthesizeOutput func(ctx context.Context, chunk []byte) (err error)

// StreamSynthesizeService is the service for voice clone.
type StreamSynthesizeService struct {
	c *client

	cluster   string
	userID    string
	speakerID string
	format    string
	requestID string
	input     string
	ssml      bool

	output StreamSynthesizeOutput
}

// newStreamSynthesizeService creates a new voice clone service, in streaming mode.
func newStreamSynthesizeService(c *client) *StreamSynthesizeService {
	s := &StreamSynthesizeService{
		c: c,
	}
	return s
}

// SetCluster sets the cluster for the audio.
func (s *StreamSynthesizeService) SetCluster(cluster string) *StreamSynthesizeService {
	s.cluster = cluster
	return s
}

// SetUserID sets the user id for the audio.
func (s *StreamSynthesizeService) SetUserID(userID string) *StreamSynthesizeService {
	s.userID = userID
	return s
}

// SetSpeakerID sets the voice type for the audio, also known as the speaker id.
func (s *StreamSynthesizeService) SetSpeakerID(speakerID string) *StreamSynthesizeService {
	s.speakerID = speakerID
	return s
}

// SetFormat sets the encoding for the audio.
func (s *StreamSynthesizeService) SetFormat(format string) *StreamSynthesizeService {
	s.format = format
	return s
}

// SetRequestID sets the request id for the audio.
func (s *StreamSynthesizeService) SetRequestID(reqID string) *StreamSynthesizeService {
	s.requestID = reqID
	return s
}

// SetInput sets the text for the audio.
func (s *StreamSynthesizeService) SetInput(input string) *StreamSynthesizeService {
	s.input = input
	return s
}

// SetSSML sets the text type to SSML.
func (s *StreamSynthesizeService) SetSSML(ssml bool) *StreamSynthesizeService {
	s.ssml = ssml
	return s
}

// SetOutput sets the handler for the audio chunks.
func (s *StreamSynthesizeService) SetOutput(output StreamSynthesizeOutput) *StreamSynthesizeService {
	s.output = output
	return s
}

func (s *StreamSynthesizeService) buildBody() map[string]any {
	mApp := map[string]any{
		"appid":   s.c.appID,
		"token":   "placeholder",
		"cluster": s.cluster,
	}

	mUser := map[string]any{}

	if s.userID != "" {
		mUser["uid"] = s.userID
	}

	mAudio := map[string]any{
		"voice_type": s.speakerID,
	}
	if s.format != "" {
		mAudio["encoding"] = s.format
	}

	mRequest := map[string]any{
		"reqid":     s.requestID,
		"text":      s.input,
		"operation": "submit",
	}
	if s.ssml {
		mRequest["text_type"] = "ssml"
	} else {
		mRequest["text_type"] = "plain"
	}
	return map[string]any{
		"app":     mApp,
		"user":    mUser,
		"audio":   mAudio,
		"request": mRequest,
	}
}

// Do sends the audio request to the server, and stream audio chunks to handler.
func (s *StreamSynthesizeService) Do(ctx context.Context) (err error) {
	header := http.Header{}
	header.Add("Authorization", "Bearer;"+s.c.token)

	var conn *websocket.Conn
	if conn, _, err = s.c.ws.DialContext(
		ctx,
		"wss://"+s.c.endpoint+"/api/v1/tts/ws_binary",
		header,
	); err != nil {
		return
	}
	defer conn.Close()

	var body []byte
	if body, err = stream_wire.EncodeRequest(s.buildBody()); err != nil {
		return
	}

	if err = conn.WriteMessage(websocket.BinaryMessage, body); err != nil {
		return
	}

	for {
		var mt int
		var buf []byte
		if mt, buf, err = conn.ReadMessage(); err != nil {
			if io.EOF == err {
				err = nil
			}
			return
		}
		if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
			err = fmt.Errorf("unexpected message type: %d", mt)
			return
		}
		var res stream_wire.Response
		if res, err = stream_wire.DecodeResponse(buf); err != nil {
			return
		}
		if res.IsPayload {
			if s.output != nil {
				if err = s.output(ctx, res.PayloadData); err != nil {
					return
				}
			}
			if res.PayloadIndex < 0 {
				return
			}
		}
		if res.IsError {
			err = fmt.Errorf("%d: %s", res.ErrorCode, res.ErrorMessage)
			return
		}
	}
}
