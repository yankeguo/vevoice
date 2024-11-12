package volcvoice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yankeguo/rg"
	"github.com/yankeguo/volcvoice/internal/duplex_wire"
)

// StreamSynthesizeInput is a function to get input text chunk, the last chunk should be empty string and io.EOF error.
type StreamSynthesizeInput func(ctx context.Context) (chunk string, err error)

func StreamSynthesizeInputFromChannel(input chan string) StreamSynthesizeInput {
	return func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case chunk, more := <-input:
			if !more {
				return "", io.EOF
			}
			return chunk, nil
		}
	}
}

// StreamSynthesizeOutput is a function to handle output audio chunk.
func StreamSynthesizeInputFromSlice(input []string) StreamSynthesizeInput {
	if len(input) == 0 {
		return func(ctx context.Context) (string, error) {
			return "", io.EOF
		}
	}

	var _idx int64 = -1

	return func(ctx context.Context) (string, error) {
		idx := atomic.AddInt64(&_idx, 1)
		if idx >= int64(len(input)) {
			return "", io.EOF
		}
		return input[idx], nil
	}
}

const (
	// DuplexSynthesizeResourceStandard is a resource id for TTS service.
	DuplexSynthesizeResourceStandard = "volc.service_type.10029"

	// DuplexSynthesizeResourceVoiceCloneV2 is a resource id for VoiceClone 2.0 service.
	DuplexSynthesizeResourceVoiceCloneV2 = "volc.megatts.default"

	duplexSynthesizeNamespace = "BidirectionalTTS"
)

// DuplexSynthesizeService is a service to synthesize speech in bi-directional stream mode.
type DuplexSynthesizeService struct {
	c     *client
	proto *duplex_wire.BinaryProtocol

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

	input  StreamSynthesizeInput
	output StreamSynthesizeOutput
}

func newDuplexSynthesizeService(c *client) *DuplexSynthesizeService {
	s := &DuplexSynthesizeService{
		c:     c,
		proto: duplex_wire.NewBinaryProtocol(),
	}

	s.proto.SetVersion(duplex_wire.Version1)
	s.proto.SetHeaderSize(duplex_wire.HeaderSize4)
	s.proto.SetSerialization(duplex_wire.SerializationJSON)
	s.proto.SetCompression(duplex_wire.CompressionNone, nil)
	s.proto.ContainsSequence = duplex_wire.ContainsSequence

	return s
}

// SetResourceID sets the resource id
func (s *DuplexSynthesizeService) SetResourceID(id string) *DuplexSynthesizeService {
	s.resourceID = id
	return s
}

// SetRequestID sets the request id
func (s *DuplexSynthesizeService) SetRequestID(id string) *DuplexSynthesizeService {
	s.requestID = id
	return s
}

// SetConnectID sets the connect id
func (s *DuplexSynthesizeService) SetConnectID(id string) *DuplexSynthesizeService {
	s.connectID = id
	return s
}

// SetUserID sets the user id
func (s *DuplexSynthesizeService) SetUserID(id string) *DuplexSynthesizeService {
	s.userID = id
	return s
}

// SetSpeakerID sets the speaker id, for VoiceClone service, use the "S_" started speaker id.
func (s *DuplexSynthesizeService) SetSpeakerID(id string) *DuplexSynthesizeService {
	s.speakerID = id
	return s
}

// SetFormat sets the format of the synthesized speech
func (s *DuplexSynthesizeService) SetFormat(format string) *DuplexSynthesizeService {
	s.format = format
	return s
}

// SetSampleRate sets the sample rate of the synthesized speech
func (s *DuplexSynthesizeService) SetSampleRate(rate int) *DuplexSynthesizeService {
	s.sampleRate = rate
	return s
}

// SetSpeechRate sets the speech rate of the synthesized speech
func (s *DuplexSynthesizeService) SetSpeechRate(rate int) *DuplexSynthesizeService {
	s.speechRate = rate
	return s
}

// SetPitchRate sets the pitch rate of the synthesized speech
func (s *DuplexSynthesizeService) SetPitchRate(rate int) *DuplexSynthesizeService {
	s.pitchRate = rate
	return s
}

// SetSSML sets the ssml mode
func (s *DuplexSynthesizeService) SetSSML(ssml bool) *DuplexSynthesizeService {
	s.ssml = ssml
	return s
}

// SetInputFunc sets the input function
func (s *DuplexSynthesizeService) SetInput(input StreamSynthesizeInput) *DuplexSynthesizeService {
	s.input = input
	return s
}

// SetOutputFunc sets the output function
func (s *DuplexSynthesizeService) SetOutput(output StreamSynthesizeOutput) *DuplexSynthesizeService {
	s.output = output
	return s
}

func (s *DuplexSynthesizeService) dial(ctx context.Context) (conn *websocket.Conn, resp *http.Response, err error) {
	if s.resourceID == "" {
		err = errors.New("duplex_synthesize: resource id is required")
		return
	}
	if s.requestID == "" {
		err = errors.New("duplex_synthesize: request id is required")
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

func (s *DuplexSynthesizeService) Do(ctx context.Context) (err error) {
	defer rg.Guard(&err)

	if s.input == nil {
		err = errors.New("duplex_synthesize: input function is required")
		return
	}

	if s.output == nil {
		err = errors.New("duplex_synthesize: output function is required")
		return
	}

	s.c.debug("duplex_synthesize: starting")

	conn, resp := rg.Must2(s.dial(ctx))
	defer conn.Close()

	s.c.debug("duplex_synthesize: websocket connected, LogID: ", resp.Header.Get("X-Tt-Logid"))

	rg.Must0(s.startConnection(ctx, conn))

	s.c.debug("duplex_synthesize: protocol connected")

	sessionID := rg.Must(uuid.NewV7()).String()

	rg.Must0(s.startTTSSession(ctx, conn, sessionID))

	s.c.debug("duplex_synthesize: TTS session started:", sessionID)

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

			s.c.debug("duplex_synthesize: input chunk:", chunk)

			// break sendLoop if error
			if sCtx.Err() != nil {
				break sendLoop
			}

			if err = s.sendTTSMessage(sCtx, conn, sessionID, chunk); err != nil {
				s.c.debug("duplex_synthesize: send TTS message error:", err)
				// break sendLoop if error
				break sendLoop
			} else {
				s.c.debug("duplex_synthesize: TTS message sent for chunk:", chunk)
			}
		}

		if err != nil {
			if err == io.EOF {
				s.c.debug("duplex_synthesize: send loop EOF")
				err = nil
			} else {
				s.c.debug("duplex_synthesize: send loop error:", err)
				sCancel()
			}
		}

		if err = s.finishSession(ctx, conn, sessionID); err != nil {
			s.c.debug("duplex_synthesize: finish session error:", err)
			sCancel()
		} else {
			s.c.debug("duplex_synthesize: TTS session finished")
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

			var msg *duplex_wire.Message
			if msg, err = s.receiveMessage(sCtx, conn); err != nil {
				s.c.debug("duplex_synthesize: receive message error:", err)
				break recvLoop
			} else {
				s.c.debug("duplex_synthesize: received message:", msg.Type)
			}

			switch msg.Type {
			case duplex_wire.MsgTypeFullServer:
				if msg.Event == int32(duplex_wire.EventSessionFinished) {
					break recvLoop
				}
			case duplex_wire.MsgTypeAudioOnlyServer:
				if err = s.output(sCtx, msg.Payload); err != nil {
					break recvLoop
				}
			case duplex_wire.MsgTypeError:
				err = fmt.Errorf("duplex_synthesize: server error: (%d) %s", msg.ErrorCode, msg.Payload)
				break recvLoop
			default:
				err = fmt.Errorf("duplex_synthesize: unexpected message type: %d", msg.Type)
				break recvLoop
			}
		}

		if err != nil {
			s.c.debug("duplex_synthesize: recv loop error:", err)
			sCancel()
		} else {
			s.c.debug("duplex_synthesize: recv loop done")
		}

		return
	}()

	wg.Wait()

	s.c.debug("duplex_synthesize: read/recv goroutines done")

	if err = s.finishConnection(ctx, conn); err != nil {
		s.c.debug("duplex_synthesize: finish connection error:", err)
		return
	} else {
		s.c.debug("duplex_synthesize: protocol disconnected")
	}

	return
}

func (s *DuplexSynthesizeService) startConnection(ctx context.Context, conn *websocket.Conn) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(duplex_wire.NewMessage(duplex_wire.MsgTypeFullClient, duplex_wire.MsgTypeFlagWithEvent))
	msg.Event = int32(duplex_wire.EventStartConnection)
	msg.Payload = []byte("{}")

	frame := rg.Must(s.proto.Marshal(msg))

	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	mt, frame := rg.Must2(conn.ReadMessage())
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("duplex_synthesize.startConnection: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(duplex_wire.Unmarshal(frame, s.proto.ContainsSequence))

	if msg.Type != duplex_wire.MsgTypeFullServer {
		err = fmt.Errorf("duplex_synthesize.startConnection: unexpected message type: %d", msg.Type)
		return
	}

	if duplex_wire.Event(msg.Event) != duplex_wire.EventConnectionStarted {
		err = fmt.Errorf("duplex_synthesize.startConnection: unexpected event: %d", msg.Event)
		return
	}

	return
}

func (s *DuplexSynthesizeService) startTTSSession(ctx context.Context, conn *websocket.Conn, sessionID string) (err error) {
	defer rg.Guard(&err)

	req := duplex_wire.TTSRequest{
		User: &duplex_wire.TTSUser{
			Uid: s.userID,
		},
		Event:     int32(duplex_wire.EventStartSession),
		Namespace: duplexSynthesizeNamespace,
		ReqParams: &duplex_wire.TTSReqParams{
			Speaker: s.speakerID,
			AudioParams: &duplex_wire.AudioParams{
				Format:     s.format,
				SampleRate: int32(s.sampleRate),
				SpeechRate: int32(s.speechRate),
				PitchRate:  int32(s.pitchRate),
			},
		},
	}

	payload := rg.Must(json.Marshal(&req))

	msg := rg.Must(duplex_wire.NewMessage(duplex_wire.MsgTypeFullClient, duplex_wire.MsgTypeFlagWithEvent))
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame := rg.Must(s.proto.Marshal(msg))

	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	mt, frame := rg.Must2(conn.ReadMessage())
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("duplex_synthesize.startTTSSession: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(duplex_wire.Unmarshal(frame, s.proto.ContainsSequence))

	if msg.Type != duplex_wire.MsgTypeFullServer {
		err = fmt.Errorf("duplex_synthesize.startTTSSession: unexpected message type: %d", msg.Type)
		return
	}
	if duplex_wire.Event(msg.Event) != duplex_wire.EventSessionStarted {
		err = fmt.Errorf("duplex_synthesize.startTTSSession: unexpected event: %d", msg.Event)
		return
	}

	return
}

func (s *DuplexSynthesizeService) sendTTSMessage(ctx context.Context, conn *websocket.Conn, sessionID, content string) (err error) {
	defer rg.Guard(&err)

	var (
		text string
		ssml string
	)

	if s.ssml {
		ssml = content
	} else {
		text = content
	}

	req := duplex_wire.TTSRequest{
		Event:     int32(duplex_wire.EventTaskRequest),
		Namespace: duplexSynthesizeNamespace,
		ReqParams: &duplex_wire.TTSReqParams{
			Text:    text,
			Ssml:    ssml,
			Speaker: s.speakerID,
			AudioParams: &duplex_wire.AudioParams{
				Format:     s.format,
				SampleRate: int32(s.sampleRate),
				SpeechRate: int32(s.speechRate),
				PitchRate:  int32(s.pitchRate),
			},
		},
	}

	payload := rg.Must(json.Marshal(&req))

	msg := rg.Must(duplex_wire.NewMessage(duplex_wire.MsgTypeFullClient, duplex_wire.MsgTypeFlagWithEvent))
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame := rg.Must(s.proto.Marshal(msg))

	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	return
}

func (s *DuplexSynthesizeService) finishSession(ctx context.Context, conn *websocket.Conn, sessionID string) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(duplex_wire.NewMessage(duplex_wire.MsgTypeFullClient, duplex_wire.MsgTypeFlagWithEvent))
	msg.Event = int32(duplex_wire.EventFinishSession)
	msg.SessionID = sessionID
	msg.Payload = []byte("{}")

	frame := rg.Must(s.proto.Marshal(msg))
	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	return
}

func (s *DuplexSynthesizeService) receiveMessage(ctx context.Context, conn *websocket.Conn) (msg *duplex_wire.Message, err error) {
	defer rg.Guard(&err)

	mt, frame := rg.Must2(conn.ReadMessage())
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("duplex_synthesize.receiveMessage: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(duplex_wire.Unmarshal(frame, s.proto.ContainsSequence))
	return
}

func (s *DuplexSynthesizeService) finishConnection(ctx context.Context, conn *websocket.Conn) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(duplex_wire.NewMessage(duplex_wire.MsgTypeFullClient, duplex_wire.MsgTypeFlagWithEvent))
	msg.Event = int32(duplex_wire.EventFinishConnection)
	msg.Payload = []byte("{}")

	frame := rg.Must(s.proto.Marshal(msg))
	rg.Must0(conn.WriteMessage(websocket.BinaryMessage, frame))
	mt, frame := rg.Must2(conn.ReadMessage())

	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		err = fmt.Errorf("duplex_synthesize.finishConnection: unexpected message type: %d", mt)
		return
	}

	msg, _ = rg.Must2(duplex_wire.Unmarshal(frame, s.proto.ContainsSequence))

	if msg.Type != duplex_wire.MsgTypeFullServer {
		err = fmt.Errorf("duplex_synthesize.finishConnection: unexpected message type: %d", msg.Type)
		return
	}
	if duplex_wire.Event(msg.Event) != duplex_wire.EventConnectionFinished {
		err = fmt.Errorf("duplex_synthesize.finishConnection: unexpected event: %d", msg.Event)
		return
	}

	return
}
