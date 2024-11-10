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

type VoiceCloneHandler func(buf []byte)

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
	return s
}

// SetCluster sets the cluster for the audio.
func (s *VoiceCloneService) SetCluster(cluster string) {
	s.opts.App.Cluster = cluster
}

// SetUID sets the user id for the audio.
func (s *VoiceCloneService) SetUID(uid string) {
	s.opts.User.UID = uid
}

// SetVoiceType sets the voice type for the audio, also known as the speaker id.
func (s *VoiceCloneService) SetVoiceType(voiceType string) {
	s.opts.Audio.VoiceType = voiceType
}

// SetEncoding sets the encoding for the audio.
func (s *VoiceCloneService) SetEncoding(encoding string) {
	s.opts.Audio.Encoding = encoding
}

// SetRequestID sets the request id for the audio.
func (s *VoiceCloneService) SetRequestID(reqID string) {
	s.opts.Request.ReqID = reqID
}

// SetText sets the text for the audio.
func (s *VoiceCloneService) SetText(text string) {
	s.opts.Request.Text = text
}

// SetTextType sets the text type for the audio.
func (s *VoiceCloneService) SetTextType(textType string) {
	s.opts.Request.TextType = textType
}

// SetHandler sets the handler for the audio chunks.
func (s *VoiceCloneService) SetHandler(h VoiceCloneHandler) {
	s.h = h
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

	h := http.Header{}
	h.Add("Authorization", "Bearer;"+s.c.opts.token)

	cfg := &websocket.Config{
		Location: l,
		Header:   h,
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
				s.h(res.PayloadData)
			}
		}
		if res.IsError {
			err = fmt.Errorf("%d: %s", res.ErrorCode, res.ErrorMessage)
			return
		}
	}
}
