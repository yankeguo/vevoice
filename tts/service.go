package tts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/yankeguo/rg"
	"golang.org/x/net/websocket"
)

// InputFunc is a function to get input text chunk, the last chunk should be empty string and io.EOF error.
type InputFunc func(ctx context.Context) (chunk string, err error)

// OutputFunc is a function to process output audio chunk.
type OutputFunc func(ctx context.Context, chunk []byte) (err error)

const (
	// ResourceTTS is a resource id for TTS service.
	ResourceTTS = "volc.service_type.10029"
	// ResourceVoiceClone2_0 is a resource id for VoiceClone 2.0 service.
	ResourceVoiceClone2_0 = "volc.megatts.default"

	// NamespaceTTS is a namespace for TTS service.
	NamespaceBidirectionalTTS = "BidirectionalTTS"
)

// Service is a service to synthesize speech in bi-directional stream mode.
type Service struct {
	debug bool

	apiEndpoint string
	apiPath     string
	apiAppID    string
	apiToken    string

	resourceID string
	requestID  string
	connectID  string

	userID     string
	speakerID  string
	format     string
	sampleRate int
	speechRate int
	pitchRate  int

	protocol *BinaryProtocol

	ssml bool

	input  InputFunc
	output OutputFunc
}

func New() *Service {
	s := &Service{}
	s.protocol = NewBinaryProtocol()
	s.protocol.SetVersion(Version1)
	s.protocol.SetHeaderSize(HeaderSize4)
	s.protocol.SetSerialization(SerializationJSON)
	s.protocol.SetCompression(CompressionNone, nil)
	s.protocol.ContainsSequence = ContainsSequence
	return s
}

// SetDebug sets the debug mode
func (s *Service) SetDebug(debug bool) *Service {
	s.debug = debug
	return s
}

// SetAPIEndpoint sets the api endpoint
func (s *Service) SetAPIEndpoint(endpoint string) *Service {
	s.apiEndpoint = endpoint
	return s
}

// SetAPIPath sets the path
func (s *Service) SetAPIPath(path string) *Service {
	s.apiPath = path
	return s
}

// SetAPIAppID sets the app id
func (s *Service) SetAPIAppID(appID string) *Service {
	s.apiAppID = appID
	return s
}

// SetAPIToken sets the token
func (s *Service) SetAPIToken(token string) *Service {
	s.apiToken = token
	return s
}

// SetResourceID sets the resource id
func (s *Service) SetResourceID(id string) *Service {
	s.resourceID = id
	return s
}

// SetRequestID sets the request id
func (s *Service) SetRequestID(id string) *Service {
	s.requestID = id
	return s
}

// SetConnectID sets the connect id
func (s *Service) SetConnectID(id string) *Service {
	s.connectID = id
	return s
}

// SetUserID sets the user id
func (s *Service) SetUserID(id string) *Service {
	s.userID = id
	return s
}

// SetSpeakerID sets the speaker id, for VoiceClone service, use the "S_" started speaker id.
func (s *Service) SetSpeakerID(id string) *Service {
	s.speakerID = id
	return s
}

// SetFormat sets the format of the synthesized speech
func (s *Service) SetFormat(format string) *Service {
	s.format = format
	return s
}

// SetSampleRate sets the sample rate of the synthesized speech
func (s *Service) SetSampleRate(rate int) *Service {
	s.sampleRate = rate
	return s
}

// SetSpeechRate sets the speech rate of the synthesized speech
func (s *Service) SetSpeechRate(rate int) *Service {
	s.speechRate = rate
	return s
}

// SetPitchRate sets the pitch rate of the synthesized speech
func (s *Service) SetPitchRate(rate int) *Service {
	s.pitchRate = rate
	return s
}

// SetSSML sets the ssml mode
func (s *Service) SetSSML(ssml bool) *Service {
	s.ssml = ssml
	return s
}

// SetInputFunc sets the input function
func (s *Service) SetInput(input InputFunc) *Service {
	s.input = input
	return s
}

// SetOutputFunc sets the output function
func (s *Service) SetOutput(output OutputFunc) *Service {
	s.output = output
	return s
}

func (s *Service) createWebsocketConfig() (cfg *websocket.Config, err error) {
	if s.apiAppID == "" {
		err = errors.New("tts.Service: app id is required")
		return
	}
	if s.apiToken == "" {
		err = errors.New("tts.Service: token is required")
		return
	}
	if s.apiEndpoint == "" {
		err = errors.New("tts.Service: endpoint is required")
		return
	}
	if s.apiPath == "" {
		err = errors.New("tts.Service: path is required")
		return
	}
	if s.resourceID == "" {
		err = errors.New("tts.Service: resource id is required")
		return
	}
	if s.requestID == "" {
		err = errors.New("tts.Service: request id is required")
		return
	}

	base := strings.TrimSuffix(s.apiEndpoint, "/") + "/" + strings.TrimPrefix(s.apiPath, "/")

	if cfg, err = websocket.NewConfig("wss://"+base, "https://"+base); err != nil {
		return
	}

	cfg.Header.Set("X-Api-App-Key", s.apiAppID)
	cfg.Header.Set("X-Api-Access-Key", s.apiToken)
	cfg.Header.Set("X-Api-Resource-Id", s.resourceID)
	cfg.Header.Set("X-Api-Request-Id", s.requestID)
	if s.connectID != "" {
		cfg.Header.Set("X-Api-Connect-Id", s.connectID)
	}
	return
}

func (s *Service) Do(ctx context.Context) (err error) {
	defer rg.Guard(&err)

	if s.input == nil {
		err = errors.New("tts.Service: input function is required")
		return
	}

	if s.output == nil {
		err = errors.New("tts.Service: output function is required")
		return
	}

	cfg := rg.Must(s.createWebsocketConfig())

	if s.debug {
		log.Println("tts.Service: websocket dialing to", cfg.Location.String())
	}

	conn := rg.Must(cfg.DialContext(ctx))
	defer conn.Close()

	if s.debug {
		log.Println("tts.Service: websocket connected")
	}

	rg.Must0(s.protocol.startConnection(ctx, conn))

	if s.debug {
		log.Println("tts.Service: protocol connected")
	}

	sessionID := rg.Must(uuid.NewV7()).String()

	rg.Must0(s.protocol.startTTSSession(
		ctx,
		conn,
		sessionID,
		NamespaceBidirectionalTTS,
		&TTSReqParams{
			Speaker: s.speakerID,
			AudioParams: &AudioParams{
				Format:     s.format,
				SampleRate: int32(s.sampleRate),
				SpeechRate: int32(s.speechRate),
				PitchRate:  int32(s.pitchRate),
			},
		},
	))

	if s.debug {
		log.Println("tts.Service: TTS session started:", sessionID)
	}

	sCtx, sCancel := context.WithCancel(ctx)
	defer sCancel()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() (err error) {
		defer wg.Done()

	sendLoop:
		for {
			// break sendLoop if context is done
			if ctx.Err() != nil {
				break sendLoop
			}

			var chunk string

			if chunk, err = s.input(sCtx); err != nil {
				break sendLoop
			}

			if s.debug {
				log.Println("tts.Service: input chunk:", chunk)
			}

			// break sendLoop if error
			if ctx.Err() != nil {
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

			if err = s.protocol.sendTTSMessage(
				sCtx,
				conn,
				sessionID,
				NamespaceBidirectionalTTS,
				&TTSReqParams{
					Text:    text,
					Ssml:    ssml,
					Speaker: s.speakerID,
					AudioParams: &AudioParams{
						Format:     s.format,
						SampleRate: int32(s.sampleRate),
						SpeechRate: int32(s.speechRate),
						PitchRate:  int32(s.pitchRate),
					},
				},
			); err != nil {
				if s.debug {
					log.Println("tts.Service: send TTS message error:", err)
				}
				// break sendLoop if error
				break sendLoop
			} else {
				if s.debug {
					log.Println("tts.Service: TTS message sent for chunk:", chunk)
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				if s.debug {
					log.Println("tts.Service: input EOF")
				}
				err = nil
			} else {
				if s.debug {
					log.Println("tts.Service: input error:", err)
				}
				sCancel()
			}
		}

		if err = s.protocol.finishSession(ctx, conn, sessionID); err != nil {
			if s.debug {
				log.Println("tts.Service: finish session error:", err)
			}
			sCancel()
		} else {
			if s.debug {
				log.Println("tts.Service: TTS session finished")
			}
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

			var msg *Message
			if msg, err = s.protocol.receiveMessage(sCtx, conn); err != nil {
				if s.debug {
					log.Println("tts.Service: receive message error:", err)
				}
				break recvLoop
			} else {
				if s.debug {
					log.Println("tts.Service: received message:", msg.Type)
				}
			}

			switch msg.Type {
			case MsgTypeFullServer:
				if msg.Event == int32(EventSessionFinished) {
					break recvLoop
				}
			case MsgTypeAudioOnlyServer:
				if err = s.output(sCtx, msg.Payload); err != nil {
					break recvLoop
				}
			case MsgTypeError:
				err = fmt.Errorf("tts.Service: server error: (%d) %s", msg.ErrorCode, msg.Payload)
				break recvLoop
			default:
				err = fmt.Errorf("tts.Service: unexpected message type: %d", msg.Type)
				break recvLoop
			}
		}

		if err != nil {
			if s.debug {
				log.Println("tts.Service: receive error:", err)
			}
			sCancel()
		}

		return
	}()

	wg.Wait()

	if err = s.protocol.finishConnection(ctx, conn); err != nil {
		if s.debug {
			log.Println("tts.Service: finish connection error:", err)
		}
		return
	} else {
		if s.debug {
			log.Println("tts.Service: protocol disconnected")
		}
	}

	return
}
