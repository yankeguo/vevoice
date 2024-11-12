package tts

import (
	"context"
	"errors"

	"golang.org/x/net/websocket"
)

const (
	// ResourceTTS is a resource id for TTS service.
	ResourceTTS = "volc.service_type.10029"
	// ResourceVoiceClone2_0 is a resource id for VoiceClone 2.0 service.
	ResourceVoiceClone2_0 = "volc.megatts.default"
)

// Service is a service to synthesize speech in bi-directional stream mode.
type Service struct {
	endpoint string
	appID    string
	token    string
	debug    bool

	resourceID string
	requestID  string
	connectID  string

	userID     string
	speakerID  string
	format     string
	sampleRate *int
	speechRate *int
	pitchRate  *int
}

func New() *Service {
	return &Service{}
}

// SetEndpoint sets the endpoint
func (s *Service) SetEndpoint(endpoint string) *Service {
	s.endpoint = endpoint
	return s
}

// SetAppID sets the app id
func (s *Service) SetAppID(appID string) *Service {
	s.appID = appID
	return s
}

// SetToken sets the token
func (s *Service) SetToken(token string) *Service {
	s.token = token
	return s
}

// SetDebug sets the debug mode
func (s *Service) SetDebug(debug bool) *Service {
	s.debug = debug
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
	s.sampleRate = &rate
	return s
}

// ClearSampleRate clears the sample rate of the synthesized speech, use the default value.
func (s *Service) ClearSampleRate() *Service {
	s.sampleRate = nil
	return s
}

// SetSpeechRate sets the speech rate of the synthesized speech
func (s *Service) SetSpeechRate(rate int) *Service {
	s.speechRate = &rate
	return s
}

// ClearSpeechRate clears the speech rate of the synthesized speech, use the default value.
func (s *Service) ClearSpeechRate() *Service {
	s.speechRate = nil
	return s
}

// SetPitchRate sets the pitch rate of the synthesized speech
func (s *Service) SetPitchRate(rate int) *Service {
	s.pitchRate = &rate
	return s
}

// ClearPitchRate clears the pitch rate of the synthesized speech, use the default value.
func (s *Service) ClearPitchRate() *Service {
	s.pitchRate = nil
	return s
}

func (s *Service) createWebsocketConfig() (cfg *websocket.Config, err error) {
	if s.resourceID == "" {
		err = errors.New("resource id is required")
		return
	}
	if s.requestID == "" {
		err = errors.New("request id is required")
		return
	}

	link := s.endpoint + "/api/v3/tts/bidirection"

	if cfg, err = websocket.NewConfig(
		"wss://"+link,
		"https://"+link,
	); err != nil {
		return
	}

	cfg.Header.Set("X-Api-App-Key", s.appID)
	cfg.Header.Set("X-Api-Access-Key", s.token)
	cfg.Header.Set("X-Api-Resource-Id", s.resourceID)
	cfg.Header.Set("X-Api-Request-Id", s.requestID)
	if s.connectID != "" {
		cfg.Header.Set("X-Api-Connect-Id", s.connectID)
	}
	return
}

func (s *Service) Do(ctx context.Context) (err error) {
	var cfg *websocket.Config

	if cfg, err = s.createWebsocketConfig(); err != nil {
		return
	}

	var conn *websocket.Conn
	if conn, err = cfg.DialContext(ctx); err != nil {
		return
	}
	defer conn.Close()

	return
}
