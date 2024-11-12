package volcvoice

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/yankeguo/volcvoice/internal/voiceclone_wire"
)

// VoiceCloneSynthesizeService is the service for voice clone.
type VoiceCloneSynthesizeService struct {
	c *client

	cluster   string
	userID    string
	speakerID string
	format    string
	requestID string
	input     string
	ssml      bool

	output SynthesizeStreamOutput
}

// newVoiceCloneSynthesizeService creates a new voice clone service, in streaming mode.
func newVoiceCloneSynthesizeService(c *client) *VoiceCloneSynthesizeService {
	s := &VoiceCloneSynthesizeService{
		c: c,
	}
	return s
}

// SetCluster sets the cluster for the audio.
func (s *VoiceCloneSynthesizeService) SetCluster(cluster string) *VoiceCloneSynthesizeService {
	s.cluster = cluster
	return s
}

// SetUserID sets the user id for the audio.
func (s *VoiceCloneSynthesizeService) SetUserID(userID string) *VoiceCloneSynthesizeService {
	s.userID = userID
	return s
}

// SetSpeakerID sets the voice type for the audio, also known as the speaker id.
func (s *VoiceCloneSynthesizeService) SetSpeakerID(speakerID string) *VoiceCloneSynthesizeService {
	s.speakerID = speakerID
	return s
}

// SetFormat sets the encoding for the audio.
func (s *VoiceCloneSynthesizeService) SetFormat(format string) *VoiceCloneSynthesizeService {
	s.format = format
	return s
}

// SetRequestID sets the request id for the audio.
func (s *VoiceCloneSynthesizeService) SetRequestID(reqID string) *VoiceCloneSynthesizeService {
	s.requestID = reqID
	return s
}

// SetInput sets the text for the audio.
func (s *VoiceCloneSynthesizeService) SetInput(input string) *VoiceCloneSynthesizeService {
	s.input = input
	return s
}

// SetSSML sets the text type to SSML.
func (s *VoiceCloneSynthesizeService) SetSSML(ssml bool) *VoiceCloneSynthesizeService {
	s.ssml = ssml
	return s
}

// SetOutput sets the handler for the audio chunks.
func (s *VoiceCloneSynthesizeService) SetOutput(output SynthesizeStreamOutput) *VoiceCloneSynthesizeService {
	s.output = output
	return s
}

func (s *VoiceCloneSynthesizeService) buildBody() map[string]any {
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
func (s *VoiceCloneSynthesizeService) Do(ctx context.Context) (err error) {
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
	if body, err = voiceclone_wire.EncodeRequest(s.buildBody()); err != nil {
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
		var res voiceclone_wire.Response
		if res, err = voiceclone_wire.DecodeResponse(buf); err != nil {
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
