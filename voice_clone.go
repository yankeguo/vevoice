package volcvoice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/net/websocket"
)

const (
	ClusterICL           = "volcano_icl"
	ClusterICLConcurrent = "volcano_icl_concurr"

	EncodingWAV = "wav"
	EncodingMP3 = "mp3"
	EncodingOGG = "ogg_opus"
	EncodingPCM = "pcm"

	TextTypePlain = "plain"
	TextTypeSSML  = "ssml"

	OperationQuery  = "query"
	OperationSubmit = "submit"
)

type voiceCloneOptions struct {
	App struct {
		AppID   string `json:"appid"`
		Token   string `json:"token"` // random non-empty string, since real token is sent by header
		Cluster string `json:"cluster"`
	} `json:"app"`
	User struct {
		UID string `json:"uid"`
	} `json:"user"`
	Audio struct {
		VoiceType string `json:"voice_type"`
		Encoding  string `json:"encoding,omitempty"`
	} `json:"audio"`
	Request struct {
		ReqID     string `json:"reqid"`
		Text      string `json:"text"`
		TextType  string `json:"text_type"`
		Operation string `json:"operation"`
	} `json:"request"`
}

// VoiceCloneHandler is the handler for the audio chunks.
type VoiceCloneHandler func(buf []byte) (err error)

// VoiceCloneService is the service for voice clone.
type VoiceCloneService struct {
	c    *client
	opts *voiceCloneOptions
	h    VoiceCloneHandler
}

// NewVoiceCloneService creates a new voice clone service, in streaming mode.
func NewVoiceCloneService(c *client) *VoiceCloneService {
	s := &VoiceCloneService{
		c:    c,
		opts: &voiceCloneOptions{},
	}
	s.opts.App.AppID = c.opts.appID
	s.opts.App.Token = "placeholder"
	s.opts.Request.Operation = OperationSubmit
	return s
}

// SetCluster sets the cluster for the audio.
func (s *VoiceCloneService) SetCluster(cluster string) *VoiceCloneService {
	s.opts.App.Cluster = cluster
	return s
}

// SetUID sets the user id for the audio.
func (s *VoiceCloneService) SetUID(uid string) *VoiceCloneService {
	s.opts.User.UID = uid
	return s
}

// SetVoiceType sets the voice type for the audio, also known as the speaker id.
func (s *VoiceCloneService) SetVoiceType(voiceType string) *VoiceCloneService {
	s.opts.Audio.VoiceType = voiceType
	return s
}

// SetEncoding sets the encoding for the audio.
func (s *VoiceCloneService) SetEncoding(encoding string) *VoiceCloneService {
	s.opts.Audio.Encoding = encoding
	return s
}

// SetRequestID sets the request id for the audio.
func (s *VoiceCloneService) SetRequestID(reqID string) *VoiceCloneService {
	s.opts.Request.ReqID = reqID
	return s
}

// SetText sets the text for the audio.
func (s *VoiceCloneService) SetText(text string) *VoiceCloneService {
	s.opts.Request.Text = text
	return s
}

// SetTextType sets the text type for the audio.
func (s *VoiceCloneService) SetTextType(textType string) *VoiceCloneService {
	s.opts.Request.TextType = textType
	return s
}

// SetHandler sets the handler for the audio chunks.
func (s *VoiceCloneService) SetHandler(h VoiceCloneHandler) *VoiceCloneService {
	s.h = h
	return s
}

// Do sends the audio request to the server, and stream audio chunks to handler.
func (s *VoiceCloneService) Do(ctx context.Context) (err error) {
	var req []byte
	if req, err = encodeVoiceCloneRequest(s.opts); err != nil {
		return
	}

	var l *url.URL
	if l, err = url.Parse(fmt.Sprintf("wss://%s/api/v1/tts/ws_binary", s.c.opts.endpoint)); err != nil {
		return
	}

	var o *url.URL
	if o, err = url.Parse(l.String()); err != nil {
		return
	}

	h := http.Header{}
	h.Add("Authorization", "Bearer;"+s.c.opts.token)

	cfg := &websocket.Config{
		Location: l,
		Origin:   o,
		Header:   h,
		Version:  websocket.ProtocolVersionHybi,
	}

	var conn *websocket.Conn
	if conn, err = cfg.DialContext(ctx); err != nil {
		return
	}
	defer conn.Close()

	if err = websocket.Message.Send(conn, req); err != nil {
		return
	}

	for {
		var buf []byte
		if err = websocket.Message.Receive(conn, &buf); err != nil {
			if io.EOF == err {
				err = nil
			}
			return
		}
		var res voiceCloneResponse
		if res, err = decodeVoiceCloneResponse(buf); err != nil {
			return
		}
		if res.IsPayload {
			if s.h != nil {
				if err = s.h(res.PayloadData); err != nil {
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
