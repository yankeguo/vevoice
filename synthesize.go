package volcvoice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yankeguo/rg"
	"github.com/yankeguo/volcvoice/internal/tts_wire"
)

const (
	AudioFormatAAC      = "aac"
	AudioFormatM4A      = "m4a"
	AudioFormatMP3      = "mp3"
	AudioFormatOGG      = "ogg"
	AudioFormatOGG_OPUS = "ogg_opus"
	AudioFormatPCM      = "pcm"
	AudioFormatWAV      = "wav"

	SampleRate8K  = 8000
	SampleRate16K = 16000
	SampleRate24K = 24000
	SampleRate32K = 32000
	SampleRate44K = 44100
	SampleRate48K = 48000
)

// SynthesizeStreamInput is a function to get input text chunk, the last chunk should be empty string and io.EOF error.
type SynthesizeStreamInput func(ctx context.Context) (chunk string, err error)

// SynthesizeStreamOutput is a function to process output audio chunk.
type SynthesizeStreamOutput func(ctx context.Context, chunk []byte) (err error)

const (
	// SynthesizeResourceTTS is a resource id for TTS service.
	SynthesizeResourceTTS = "volc.service_type.10029"
	// SynthesizeResourceVoiceClone2_0 is a resource id for VoiceClone 2.0 service.
	SynthesizeResourceVoiceClone2_0 = "volc.megatts.default"

	synthesizeNamespaceBidirectionalTTS = "BidirectionalTTS"
)

// SynthesizeService is a service to synthesize speech in bi-directional stream mode.
type SynthesizeService struct {
	c     *client
	proto *tts_wire.BinaryProtocol

	resourceID string
	requestID  string
	connectID  string

	userID     string
	speakerID  string
	format     string
	sampleRate int
	speechRate int
	pitchRate  int

	ssml bool

	input  SynthesizeStreamInput
	output SynthesizeStreamOutput
}

func newSynthesizeService(c *client) *SynthesizeService {
	s := &SynthesizeService{
		c:     c,
		proto: tts_wire.NewBinaryProtocol(),
	}

	s.proto.SetVersion(tts_wire.Version1)
	s.proto.SetHeaderSize(tts_wire.HeaderSize4)
	s.proto.SetSerialization(tts_wire.SerializationJSON)
	s.proto.SetCompression(tts_wire.CompressionNone, nil)
	s.proto.ContainsSequence = tts_wire.ContainsSequence

	return s
}

// SetResourceID sets the resource id
func (s *SynthesizeService) SetResourceID(id string) *SynthesizeService {
	s.resourceID = id
	return s
}

// SetRequestID sets the request id
func (s *SynthesizeService) SetRequestID(id string) *SynthesizeService {
	s.requestID = id
	return s
}

// SetConnectID sets the connect id
func (s *SynthesizeService) SetConnectID(id string) *SynthesizeService {
	s.connectID = id
	return s
}

// SetUserID sets the user id
func (s *SynthesizeService) SetUserID(id string) *SynthesizeService {
	s.userID = id
	return s
}

// SetSpeakerID sets the speaker id, for VoiceClone service, use the "S_" started speaker id.
func (s *SynthesizeService) SetSpeakerID(id string) *SynthesizeService {
	s.speakerID = id
	return s
}

// SetFormat sets the format of the synthesized speech
func (s *SynthesizeService) SetFormat(format string) *SynthesizeService {
	s.format = format
	return s
}

// SetSampleRate sets the sample rate of the synthesized speech
func (s *SynthesizeService) SetSampleRate(rate int) *SynthesizeService {
	s.sampleRate = rate
	return s
}

// SetSpeechRate sets the speech rate of the synthesized speech
func (s *SynthesizeService) SetSpeechRate(rate int) *SynthesizeService {
	s.speechRate = rate
	return s
}

// SetPitchRate sets the pitch rate of the synthesized speech
func (s *SynthesizeService) SetPitchRate(rate int) *SynthesizeService {
	s.pitchRate = rate
	return s
}

// SetSSML sets the ssml mode
func (s *SynthesizeService) SetSSML(ssml bool) *SynthesizeService {
	s.ssml = ssml
	return s
}

// SetInputFunc sets the input function
func (s *SynthesizeService) SetInput(input SynthesizeStreamInput) *SynthesizeService {
	s.input = input
	return s
}

// SetOutputFunc sets the output function
func (s *SynthesizeService) SetOutput(output SynthesizeStreamOutput) *SynthesizeService {
	s.output = output
	return s
}

func (s *SynthesizeService) dial(ctx context.Context) (conn *websocket.Conn, resp *http.Response, err error) {
	if s.resourceID == "" {
		err = errors.New("synthesize: resource id is required")
		return
	}
	if s.requestID == "" {
		err = errors.New("synthesize: request id is required")
		return
	}

	location := "wss://" + s.c.endpoint + "/api/v3/tts/bidirection"

	header := http.Header{}
	header.Set("X-Api-App-Key", s.c.appID)
	header.Set("X-Api-Access-Key", s.c.token)
	header.Set("X-Api-Resource-Id", s.resourceID)
	header.Set("X-Api-Request-Id", s.requestID)
	if s.connectID != "" {
		header.Set("X-Api-Connect-Id", s.connectID)
	}

	if conn, resp, err = s.c.ws.DialContext(ctx, location, header); err != nil {
		return
	}

	return
}

func (s *SynthesizeService) Do(ctx context.Context) (err error) {
	defer rg.Guard(&err)

	if s.input == nil {
		err = errors.New("synthesize: input function is required")
		return
	}

	if s.output == nil {
		err = errors.New("synthesize: output function is required")
		return
	}

	s.c.debug("synthesize: starting")

	conn, resp := rg.Must2(s.dial(ctx))
	defer conn.Close()

	s.c.debug("synthesize: websocket connected, LogID: ", resp.Header.Get("X-Tt-Logid"))

	rg.Must0(s.startConnection(ctx, conn))

	s.c.debug("synthesize: protocol connected")

	sessionID := rg.Must(uuid.NewV7()).String()

	rg.Must0(s.startTTSSession(
		ctx,
		conn,
		sessionID,
		synthesizeNamespaceBidirectionalTTS,
		&tts_wire.TTSReqParams{
			Speaker: s.speakerID,
			AudioParams: &tts_wire.AudioParams{
				Format:     s.format,
				SampleRate: int32(s.sampleRate),
				SpeechRate: int32(s.speechRate),
				PitchRate:  int32(s.pitchRate),
			},
		},
	))

	s.c.debug("synthesize: TTS session started:", sessionID)

	sCtx, sCancel := context.WithCancel(ctx)
	defer sCancel()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() (err error) {
		defer wg.Done()

	sendLoop:
		for {
			// break sendLoop if context is done
			if sCtx.Err() != nil {
				break sendLoop
			}

			var chunk string

			if chunk, err = s.input(sCtx); err != nil {
				break sendLoop
			}

			s.c.debug("synthesize: input chunk:", chunk)

			// break sendLoop if error
			if sCtx.Err() != nil {
				break sendLoop
			}

			var (
				text string
				ssml string
			)

			if s.ssml {
				ssml = chunk
			} else {
				text = chunk
			}

			if err = s.sendTTSMessage(
				sCtx,
				conn,
				sessionID,
				synthesizeNamespaceBidirectionalTTS,
				&tts_wire.TTSReqParams{
					Text:    text,
					Ssml:    ssml,
					Speaker: s.speakerID,
					AudioParams: &tts_wire.AudioParams{
						Format:     s.format,
						SampleRate: int32(s.sampleRate),
						SpeechRate: int32(s.speechRate),
						PitchRate:  int32(s.pitchRate),
					},
				},
			); err != nil {
				s.c.debug("synthesize: send TTS message error:", err)
				// break sendLoop if error
				break sendLoop
			} else {
				s.c.debug("synthesize: TTS message sent for chunk:", chunk)
			}
		}

		if err != nil {
			if err == io.EOF {
				s.c.debug("synthesize: send loop EOF")
				err = nil
			} else {
				s.c.debug("synthesize: send loop error:", err)
				sCancel()
			}
		}

		if err = s.finishSession(ctx, conn, sessionID); err != nil {
			s.c.debug("synthesize: finish session error:", err)
			sCancel()
		} else {
			s.c.debug("synthesize: TTS session finished")
		}

		return
	}()

	wg.Add(1)
	go func() (err error) {
		defer wg.Done()

	recvLoop:
		for {
			if sCtx.Err() != nil {
				break recvLoop
			}

			var msg *tts_wire.Message
			if msg, err = s.receiveMessage(sCtx, conn); err != nil {
				s.c.debug("synthesize: receive message error:", err)
				break recvLoop
			} else {
				s.c.debug("synthesize: received message:", msg.Type)
			}

			switch msg.Type {
			case tts_wire.MsgTypeFullServer:
				if msg.Event == int32(tts_wire.EventSessionFinished) {
					break recvLoop
				}
			case tts_wire.MsgTypeAudioOnlyServer:
				if err = s.output(sCtx, msg.Payload); err != nil {
					break recvLoop
				}
			case tts_wire.MsgTypeError:
				err = fmt.Errorf("synthesize: server error: (%d) %s", msg.ErrorCode, msg.Payload)
				break recvLoop
			default:
				err = fmt.Errorf("synthesize: unexpected message type: %d", msg.Type)
				break recvLoop
			}
		}

		if err != nil {
			s.c.debug("synthesize: recv loop error:", err)
			sCancel()
		} else {
			s.c.debug("synthesize: recv loop done")
		}

		return
	}()

	wg.Wait()

	s.c.debug("synthesize: read/recv goroutines done")

	if err = s.finishConnection(ctx, conn); err != nil {
		s.c.debug("synthesize: finish connection error:", err)
		return
	} else {
		s.c.debug("synthesize: protocol disconnected")
	}

	return
}

func (s *SynthesizeService) startConnection(ctx context.Context, conn *websocket.Conn) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(tts_wire.NewMessage(tts_wire.MsgTypeFullClient, tts_wire.MsgTypeFlagWithEvent))
	msg.Event = int32(tts_wire.EventStartConnection)
	msg.Payload = []byte("{}")

	frame := rg.Must(s.proto.Marshal(msg))

	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	mt, frame := rg.Must2(conn.ReadMessage())
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("synthesize.startConnection: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(tts_wire.Unmarshal(frame, s.proto.ContainsSequence))

	if msg.Type != tts_wire.MsgTypeFullServer {
		err = fmt.Errorf("synthesize.startConnection: unexpected message type: %d", msg.Type)
		return
	}

	if tts_wire.Event(msg.Event) != tts_wire.EventConnectionStarted {
		err = fmt.Errorf("synthesize.startConnection: unexpected event: %d", msg.Event)
		return
	}

	return
}

func (s *SynthesizeService) startTTSSession(ctx context.Context, conn *websocket.Conn, sessionID, namespace string, params *tts_wire.TTSReqParams) (err error) {
	defer rg.Guard(&err)

	req := tts_wire.TTSRequest{
		Event:     int32(tts_wire.EventStartSession),
		Namespace: namespace,
		ReqParams: params,
	}

	payload := rg.Must(json.Marshal(&req))

	msg := rg.Must(tts_wire.NewMessage(tts_wire.MsgTypeFullClient, tts_wire.MsgTypeFlagWithEvent))
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame := rg.Must(s.proto.Marshal(msg))

	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	mt, frame := rg.Must2(conn.ReadMessage())
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("synthesize.startTTSSession: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(tts_wire.Unmarshal(frame, s.proto.ContainsSequence))

	if msg.Type != tts_wire.MsgTypeFullServer {
		err = fmt.Errorf("synthesize.startTTSSession: unexpected message type: %d", msg.Type)
		return
	}
	if tts_wire.Event(msg.Event) != tts_wire.EventSessionStarted {
		err = fmt.Errorf("synthesize.startTTSSession: unexpected event: %d", msg.Event)
		return
	}

	return
}

func (s *SynthesizeService) sendTTSMessage(ctx context.Context, conn *websocket.Conn, sessionID, namespace string, params *tts_wire.TTSReqParams) (err error) {
	defer rg.Guard(&err)

	req := tts_wire.TTSRequest{
		Event:     int32(tts_wire.EventTaskRequest),
		Namespace: namespace,
		ReqParams: params,
	}

	payload := rg.Must(json.Marshal(&req))

	msg := rg.Must(tts_wire.NewMessage(tts_wire.MsgTypeFullClient, tts_wire.MsgTypeFlagWithEvent))
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame := rg.Must(s.proto.Marshal(msg))

	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	return
}

func (s *SynthesizeService) finishSession(ctx context.Context, conn *websocket.Conn, sessionID string) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(tts_wire.NewMessage(tts_wire.MsgTypeFullClient, tts_wire.MsgTypeFlagWithEvent))
	msg.Event = int32(tts_wire.EventFinishSession)
	msg.SessionID = sessionID
	msg.Payload = []byte("{}")

	frame := rg.Must(s.proto.Marshal(msg))
	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	return
}

func (s *SynthesizeService) receiveMessage(ctx context.Context, conn *websocket.Conn) (msg *tts_wire.Message, err error) {
	defer rg.Guard(&err)

	mt, frame := rg.Must2(conn.ReadMessage())
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("synthesize.receiveMessage: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(tts_wire.Unmarshal(frame, s.proto.ContainsSequence))
	return
}

func (s *SynthesizeService) finishConnection(ctx context.Context, conn *websocket.Conn) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(tts_wire.NewMessage(tts_wire.MsgTypeFullClient, tts_wire.MsgTypeFlagWithEvent))
	msg.Event = int32(tts_wire.EventFinishConnection)
	msg.Payload = []byte("{}")

	frame := rg.Must(s.proto.Marshal(msg))
	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	mt, frame := rg.Must2(conn.ReadMessage())

	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("synthesize.finishConnection: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(tts_wire.Unmarshal(frame, s.proto.ContainsSequence))

	if msg.Type != tts_wire.MsgTypeFullServer {
		err = fmt.Errorf("synthesize.finishConnection: unexpected message type: %d", msg.Type)
		return
	}
	if tts_wire.Event(msg.Event) != tts_wire.EventConnectionFinished {
		err = fmt.Errorf("synthesize.finishConnection: unexpected event: %d", msg.Event)
		return
	}

	return
}
